// Package client is a thin, dependency-free JSON-RPC 2.0 client for the i-doit
// API. Every call is an HTTP POST against a single endpoint
// (<url>/src/jsonrpc.php). The client handles API-key and optional session
// authentication, per-attempt request timeouts and retries with exponential
// backoff.
package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	rpcPath       = "/src/jsonrpc.php"
	rpcVersion    = "2.0"
	sessionHeader = "X-RPC-Auth-Session"
	userHeader    = "X-RPC-Auth-Username"
	passHeader    = "X-RPC-Auth-Password"
)

// Config holds everything required to build a Client.
type Config struct {
	// URL is the base i-doit URL without the /src/jsonrpc.php suffix.
	URL string
	// APIKey is the mandatory API key configured in i-doit.
	APIKey string
	// Username and Password enable session authentication (idoit.login).
	Username string
	Password string
	// Language is passed as the "language" parameter on every request.
	Language string

	RequestTimeout     time.Duration
	MaxRetries         int
	InsecureSkipVerify bool

	UserAgent string
}

// Client is safe for concurrent use.
type Client struct {
	endpoint   string
	apiKey     string
	username   string
	password   string
	language   string
	timeout    time.Duration
	maxRetries int
	userAgent  string

	httpClient *http.Client

	idCounter uint64

	mu      sync.Mutex
	session string
}

// New validates cfg and returns a ready to use Client.
func New(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.URL) == "" {
		return nil, errors.New("i-doit URL must not be empty")
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("i-doit API key must not be empty")
	}

	base := strings.TrimRight(strings.TrimSpace(cfg.URL), "/")
	base = strings.TrimSuffix(base, rpcPath)
	endpoint := base + rpcPath

	timeout := cfg.RequestTimeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	retries := cfg.MaxRetries
	if retries < 0 {
		retries = 0
	}
	lang := cfg.Language
	if lang == "" {
		lang = "en"
	}
	ua := cfg.UserAgent
	if ua == "" {
		ua = "terraform-provider-i-doit"
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.InsecureSkipVerify {
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{} //nolint:gosec // opt-in via provider config
		}
		transport.TLSClientConfig.InsecureSkipVerify = true
	}

	return &Client{
		endpoint:   endpoint,
		apiKey:     cfg.APIKey,
		username:   cfg.Username,
		password:   cfg.Password,
		language:   lang,
		timeout:    timeout,
		maxRetries: retries,
		userAgent:  ua,
		httpClient: &http.Client{Transport: transport},
	}, nil
}

// Endpoint returns the fully qualified JSON-RPC endpoint.
func (c *Client) Endpoint() string { return c.endpoint }

type rpcRequest struct {
	Version string         `json:"version"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params"`
	ID      uint64         `json:"id"`
}

type rpcResponse struct {
	Version string          `json:"version"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *RPCError       `json:"error"`
}

// Request performs a single JSON-RPC call. The API key and language parameter
// are injected automatically; params may be nil. On success the raw "result"
// value is unmarshalled into out (unless out is nil).
func (c *Client) Request(ctx context.Context, method string, params map[string]any, out any) error {
	merged := make(map[string]any, len(params)+2)
	for k, v := range params {
		merged[k] = v
	}
	merged["apikey"] = c.apiKey
	if _, ok := merged["language"]; !ok {
		merged["language"] = c.language
	}
	return c.call(ctx, method, merged, nil, out)
}

// call runs one JSON-RPC method with retry/backoff around the transport.
func (c *Client) call(ctx context.Context, method string, params map[string]any, extraHeaders map[string]string, out any) error {
	body := rpcRequest{
		Version: rpcVersion,
		Method:  method,
		Params:  params,
		ID:      atomic.AddUint64(&c.idCounter, 1),
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			wait := backoff(attempt)
			tflog.Debug(ctx, "retrying i-doit API call", map[string]any{
				"method": method, "attempt": attempt, "backoff_ms": wait.Milliseconds(),
			})
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		result, err := c.doRequest(ctx, method, payload, extraHeaders)
		if err == nil {
			if out == nil || len(result) == 0 || string(result) == "null" {
				return nil
			}
			if err := json.Unmarshal(result, out); err != nil {
				return fmt.Errorf("decode result of %s: %w", method, err)
			}
			return nil
		}

		lastErr = err
		if !retryable(err) {
			return err
		}
	}
	return fmt.Errorf("i-doit API call %s failed after %d attempt(s): %w", method, c.maxRetries+1, lastErr)
}

// doRequest performs exactly one HTTP round trip.
func (c *Client) doRequest(ctx context.Context, method string, payload []byte, extraHeaders map[string]string) (json.RawMessage, error) {
	attemptCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(attemptCtx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	if _, overridden := extraHeaders[userHeader]; !overridden {
		if s := c.currentSession(); s != "" {
			req.Header.Set(sessionHeader, s)
		}
	}
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	tflog.Debug(ctx, "i-doit API request", map[string]any{"method": method, "endpoint": c.endpoint})

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &retryNetError{err: err}
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &retryNetError{err: fmt.Errorf("read response body: %w", err)}
	}

	if s := resp.Header.Get(sessionHeader); s != "" {
		c.setSession(s)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &transportError{StatusCode: resp.StatusCode, Status: resp.Status, Body: string(raw)}
	}

	var parsed rpcResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("decode response of %s: %w (body: %s)", method, err, truncate(string(raw), 512))
	}
	if parsed.Error != nil {
		return nil, parsed.Error
	}
	return parsed.Result, nil
}

func (c *Client) currentSession() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.session
}

func (c *Client) setSession(s string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.session = s
}

// retryNetError marks a transport level failure as retryable.
type retryNetError struct{ err error }

func (e *retryNetError) Error() string { return e.err.Error() }
func (e *retryNetError) Unwrap() error { return e.err }

func retryable(err error) bool {
	var te *transportError
	if errors.As(err, &te) {
		return te.retryable()
	}
	var ne *retryNetError
	return errors.As(err, &ne)
}

func backoff(attempt int) time.Duration {
	const base = 500 * time.Millisecond
	const maxWait = 30 * time.Second
	d := time.Duration(float64(base) * math.Pow(2, float64(attempt-1)))
	if d > maxWait {
		d = maxWait
	}
	jitter := time.Duration(rand.Int63n(int64(base))) //nolint:gosec // jitter only
	return d + jitter
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

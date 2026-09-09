package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// rpcCall is the decoded shape of an incoming JSON-RPC request body.
type rpcCall struct {
	Version string         `json:"version"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params"`
}

// testServer spins up an httptest server that decodes each JSON-RPC request and
// hands it to fn, which returns the raw JSON to place in the "result" field (or
// a *RPCError to return an error object, or an int HTTP status to fail the
// transport).
func testServer(t *testing.T, fn func(call rpcCall) any) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != rpcPath {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		var call rpcCall
		if err := json.NewDecoder(r.Body).Decode(&call); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		out := fn(call)
		switch v := out.(type) {
		case int:
			w.WriteHeader(v)
			_, _ = w.Write([]byte(`{"error":"boom"}`))
		case *RPCError:
			resp := map[string]any{"version": "2.0", "id": 1, "error": v}
			_ = json.NewEncoder(w).Encode(resp)
		case json.RawMessage:
			resp := map[string]any{"version": "2.0", "id": 1, "result": v}
			_ = json.NewEncoder(w).Encode(resp)
		case string:
			resp := map[string]any{"version": "2.0", "id": 1, "result": json.RawMessage(v)}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			resp := map[string]any{"version": "2.0", "id": 1, "result": v}
			_ = json.NewEncoder(w).Encode(resp)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newTestClient(t *testing.T, srv *httptest.Server, cfg Config) *Client {
	t.Helper()
	cfg.URL = srv.URL
	if cfg.APIKey == "" {
		cfg.APIKey = "secret"
	}
	c, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func shrinkBackoff(t *testing.T) {
	t.Helper()
	old := backoffBase
	backoffBase = time.Millisecond
	t.Cleanup(func() { backoffBase = old })
}

func TestRequestInjectsAuthAndDecodes(t *testing.T) {
	srv := testServer(t, func(call rpcCall) any {
		if call.Method != "idoit.version" {
			t.Errorf("method = %q", call.Method)
		}
		if call.Params["apikey"] != "secret" {
			t.Errorf("apikey = %v", call.Params["apikey"])
		}
		if call.Params["language"] != "en" {
			t.Errorf("language = %v", call.Params["language"])
		}
		if call.Version != rpcVersion {
			t.Errorf("version = %q", call.Version)
		}
		return json.RawMessage(`{"version":"25","type":"pro"}`)
	})
	c := newTestClient(t, srv, Config{})

	v, err := c.ReadVersion(context.Background())
	if err != nil {
		t.Fatalf("ReadVersion: %v", err)
	}
	if v.Version != "25" || v.Type != "pro" {
		t.Fatalf("unexpected version payload: %+v", v)
	}
}

func TestRequestRetriesOnServerError(t *testing.T) {
	shrinkBackoff(t)
	var calls atomic.Int64
	srv := testServer(t, func(call rpcCall) any {
		if calls.Add(1) <= 2 {
			return http.StatusServiceUnavailable
		}
		return json.RawMessage(`{"version":"25"}`)
	})
	c := newTestClient(t, srv, Config{MaxRetries: 3})

	if _, err := c.ReadVersion(context.Background()); err != nil {
		t.Fatalf("expected success after retries, got %v", err)
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("call count = %d, want 3", got)
	}
}

func TestRequestGivesUpAfterMaxRetries(t *testing.T) {
	shrinkBackoff(t)
	var calls atomic.Int64
	srv := testServer(t, func(call rpcCall) any {
		calls.Add(1)
		return http.StatusInternalServerError
	})
	c := newTestClient(t, srv, Config{MaxRetries: 2})

	if _, err := c.ReadVersion(context.Background()); err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("call count = %d, want 3 (1 try + 2 retries)", got)
	}
}

func TestRPCErrorNotRetried(t *testing.T) {
	shrinkBackoff(t)
	var calls atomic.Int64
	srv := testServer(t, func(call rpcCall) any {
		calls.Add(1)
		return &RPCError{Code: -32602, Message: "invalid params"}
	})
	c := newTestClient(t, srv, Config{MaxRetries: 3})

	_, err := c.ReadVersion(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) {
		t.Fatalf("error is not *RPCError: %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("call count = %d, want 1 (RPC errors are not retried)", got)
	}
}

func TestSessionHeaderCaptureAndReuse(t *testing.T) {
	var seen []string
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Header.Get(sessionHeader))
		mu.Unlock()
		w.Header().Set(sessionHeader, "sess-1")
		_ = json.NewEncoder(w).Encode(map[string]any{"version": "2.0", "id": 1, "result": json.RawMessage(`{"version":"25"}`)})
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv, Config{})

	if _, err := c.ReadVersion(context.Background()); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if _, err := c.ReadVersion(context.Background()); err != nil {
		t.Fatalf("second call: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(seen) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(seen))
	}
	if seen[0] != "" {
		t.Errorf("first request should carry no session header, got %q", seen[0])
	}
	if seen[1] != "sess-1" {
		t.Errorf("second request should reuse captured session, got %q", seen[1])
	}
}

func TestMaxConcurrentRequests(t *testing.T) {
	const limit = 2
	var inFlight atomic.Int32
	var peak atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := inFlight.Add(1)
		for {
			p := peak.Load()
			if n <= p || peak.CompareAndSwap(p, n) {
				break
			}
		}
		time.Sleep(25 * time.Millisecond)
		inFlight.Add(-1)
		_ = json.NewEncoder(w).Encode(map[string]any{"version": "2.0", "id": 1, "result": json.RawMessage(`{"version":"25"}`)})
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv, Config{MaxConcurrentRequests: limit})

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.ReadVersion(context.Background()); err != nil {
				t.Errorf("ReadVersion: %v", err)
			}
		}()
	}
	wg.Wait()

	if got := peak.Load(); got > limit {
		t.Fatalf("observed %d concurrent requests, limit is %d", got, limit)
	}
}

func TestContextCancelledBeforeRequest(t *testing.T) {
	srv := testServer(t, func(call rpcCall) any { return json.RawMessage(`{"version":"25"}`) })
	c := newTestClient(t, srv, Config{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.ReadVersion(ctx); err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

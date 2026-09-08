package client

import (
	"context"
	"errors"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// HasCredentials reports whether username and password are configured for
// session authentication.
func (c *Client) HasCredentials() bool {
	return c.username != "" && c.password != ""
}

// Login performs an idoit.login handshake. The resulting session id is cached
// and sent as the X-RPC-Auth-Session header on all subsequent requests. It is
// a no-op when no credentials are configured or a session is already active.
func (c *Client) Login(ctx context.Context) error {
	if !c.HasCredentials() {
		return nil
	}
	if c.currentSession() != "" {
		return nil
	}

	params := map[string]any{"apikey": c.apiKey, "language": c.language}
	headers := map[string]string{
		userHeader: c.username,
		passHeader: c.password,
	}

	var result struct {
		SessionID string `json:"session-id"`
		Name      string `json:"name"`
		Username  string `json:"username"`
	}
	if err := c.call(ctx, "idoit.login", params, headers, &result); err != nil {
		return err
	}

	// doRequest already caches the session from the response header; fall back
	// to the body field if the header was absent.
	if c.currentSession() == "" {
		if strings.TrimSpace(result.SessionID) == "" {
			return errors.New("idoit.login succeeded but returned no session id")
		}
		c.setSession(result.SessionID)
	}

	tflog.Debug(ctx, "i-doit session established", map[string]any{"user": result.Username})
	return nil
}

// Logout terminates the cached session, if any.
func (c *Client) Logout(ctx context.Context) error {
	session := c.currentSession()
	if session == "" {
		return nil
	}

	params := map[string]any{"apikey": c.apiKey, "language": c.language}
	headers := map[string]string{sessionHeader: session}
	err := c.call(ctx, "idoit.logout", params, headers, nil)
	c.setSession("")
	return err
}

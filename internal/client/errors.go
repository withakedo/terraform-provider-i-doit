package client

import (
	"encoding/json"
	"fmt"
)

// RPCError represents a JSON-RPC 2.0 error object as returned by the i-doit API.
// It carries the numeric code, the human readable message and an optional,
// method specific data payload.
type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *RPCError) Error() string {
	if len(e.Data) > 0 && string(e.Data) != "null" && string(e.Data) != `""` {
		return fmt.Sprintf("i-doit API error %d: %s (data: %s)", e.Code, e.Message, string(e.Data))
	}
	return fmt.Sprintf("i-doit API error %d: %s", e.Code, e.Message)
}

// transportError wraps a non-2xx HTTP response. It is retryable for status
// codes 429 and 5xx.
type transportError struct {
	StatusCode int
	Status     string
	Body       string
}

func (e *transportError) Error() string {
	body := e.Body
	if len(body) > 512 {
		body = body[:512] + "..."
	}
	return fmt.Sprintf("i-doit API HTTP %s: %s", e.Status, body)
}

func (e *transportError) retryable() bool {
	return e.StatusCode == 429 || e.StatusCode >= 500
}

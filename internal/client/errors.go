package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

// notFoundHints are lower-case substrings that identify an i-doit RPC error as
// "the addressed record does not exist" rather than a genuine failure such as an
// authentication, permission or server-side error. i-doit has no dedicated
// error code for this, so message matching is the only portable signal.
var notFoundHints = []string{
	"not found",
	"does not exist",
	"doesn't exist",
	"unknown object",
	"no object",
	"non-existing",
	"non existing",
	"could not be loaded",
	"object not exist",
}

// IsNotFound reports whether err is an i-doit RPCError that indicates the
// requested object or category entry is absent. Callers use it to decide
// whether a resource should be removed from Terraform state; any other error
// (transport failure, auth rejection, server error) must be surfaced instead so
// a healthy resource is never silently dropped.
func IsNotFound(err error) bool {
	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) {
		return false
	}
	msg := strings.ToLower(rpcErr.Message)
	if len(rpcErr.Data) > 0 {
		msg += " " + strings.ToLower(string(rpcErr.Data))
	}
	for _, h := range notFoundHints {
		if strings.Contains(msg, h) {
			return true
		}
	}
	return false
}

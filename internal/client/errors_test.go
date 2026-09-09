package client

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestRPCErrorError(t *testing.T) {
	e := &RPCError{Code: -32602, Message: "invalid params"}
	if got := e.Error(); got != "i-doit API error -32602: invalid params" {
		t.Errorf("got %q", got)
	}

	e = &RPCError{Code: 42, Message: "boom", Data: json.RawMessage(`{"x":1}`)}
	if got := e.Error(); got != `i-doit API error 42: boom (data: {"x":1})` {
		t.Errorf("got %q", got)
	}
}

func TestTransportErrorRetryable(t *testing.T) {
	cases := map[int]bool{
		429: true,
		500: true,
		503: true,
		400: false,
		404: false,
		401: false,
	}
	for code, want := range cases {
		te := &transportError{StatusCode: code}
		if te.retryable() != want {
			t.Errorf("status %d: retryable = %v, want %v", code, te.retryable(), want)
		}
	}
}

func TestRetryableClassifier(t *testing.T) {
	if !retryable(&transportError{StatusCode: 502}) {
		t.Error("5xx transportError should be retryable")
	}
	if retryable(&transportError{StatusCode: 400}) {
		t.Error("4xx transportError should not be retryable")
	}
	if !retryable(&retryNetError{err: fmt.Errorf("dial tcp: connection refused")}) {
		t.Error("retryNetError should be retryable")
	}
	if retryable(&RPCError{Code: -1, Message: "x"}) {
		t.Error("RPCError should not be retryable")
	}
	if retryable(fmt.Errorf("plain")) {
		t.Error("plain error should not be retryable")
	}
}

func TestIsNotFound(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"object not found", &RPCError{Code: -1, Message: "Object not found."}, true},
		{"does not exist", &RPCError{Code: -1, Message: "The entry does not exist"}, true},
		{"could not be loaded", &RPCError{Code: -1, Message: "Object could not be loaded"}, true},
		{"hint in data", &RPCError{Code: -1, Message: "error", Data: json.RawMessage(`"unknown object"`)}, true},
		{"wrapped", fmt.Errorf("read failed: %w", &RPCError{Code: -1, Message: "not found"}), true},
		{"permission denied", &RPCError{Code: -1, Message: "Permission denied"}, false},
		{"generic", &RPCError{Code: -32603, Message: "Internal error"}, false},
		{"transport", &transportError{StatusCode: 500}, false},
		{"nil", nil, false},
		{"plain", fmt.Errorf("boom"), false},
	}
	for _, tc := range cases {
		if got := IsNotFound(tc.err); got != tc.want {
			t.Errorf("%s: IsNotFound = %v, want %v", tc.name, got, tc.want)
		}
	}
}

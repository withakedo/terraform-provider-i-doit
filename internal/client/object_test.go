package client

import (
	"context"
	"encoding/json"
	"testing"
)

func TestReadObjectFound(t *testing.T) {
	srv := testServer(t, func(call rpcCall) any {
		if call.Method != "cmdb.object.read" {
			t.Errorf("method = %q", call.Method)
		}
		return json.RawMessage(`{"id":"42","title":"web01","sysid":"SYSID_42","type":"C__OBJTYPE__SERVER"}`)
	})
	c := newTestClient(t, srv, Config{})

	obj, err := c.ReadObject(context.Background(), 42)
	if err != nil {
		t.Fatalf("ReadObject: %v", err)
	}
	if obj == nil {
		t.Fatal("expected object, got nil")
	}
	if obj.ID.Int64() != 42 || obj.Title != "web01" || obj.SysID != "SYSID_42" {
		t.Fatalf("unexpected object: %+v", obj)
	}
}

func TestReadObjectEmptySentinels(t *testing.T) {
	for _, body := range []string{`null`, `false`, `[]`, `{}`} {
		srv := testServer(t, func(call rpcCall) any { return json.RawMessage(body) })
		c := newTestClient(t, srv, Config{})
		obj, err := c.ReadObject(context.Background(), 1)
		if err != nil {
			t.Errorf("body %s: unexpected error %v", body, err)
		}
		if obj != nil {
			t.Errorf("body %s: expected nil object", body)
		}
	}
}

func TestReadObjectZeroID(t *testing.T) {
	srv := testServer(t, func(call rpcCall) any { return json.RawMessage(`{"id":0,"title":""}`) })
	c := newTestClient(t, srv, Config{})
	obj, err := c.ReadObject(context.Background(), 1)
	if err != nil || obj != nil {
		t.Fatalf("expected (nil, nil), got (%v, %v)", obj, err)
	}
}

func TestReadObjectNotFoundErrorIsAbsent(t *testing.T) {
	srv := testServer(t, func(call rpcCall) any {
		return &RPCError{Code: -1, Message: "Object not found."}
	})
	c := newTestClient(t, srv, Config{})
	obj, err := c.ReadObject(context.Background(), 1)
	if err != nil {
		t.Fatalf("not-found RPC error should be swallowed, got %v", err)
	}
	if obj != nil {
		t.Fatal("expected nil object")
	}
}

func TestReadObjectOtherErrorPropagates(t *testing.T) {
	srv := testServer(t, func(call rpcCall) any {
		return &RPCError{Code: -32603, Message: "Permission denied"}
	})
	c := newTestClient(t, srv, Config{})
	if _, err := c.ReadObject(context.Background(), 1); err == nil {
		t.Fatal("non-not-found RPC error must be surfaced, got nil")
	}
}

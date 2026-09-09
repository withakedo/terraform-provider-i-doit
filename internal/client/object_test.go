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

func TestCreateObjectFullForwardsStatusAndTemplate(t *testing.T) {
	srv := testServer(t, func(call rpcCall) any {
		if call.Method != "cmdb.object.create" {
			t.Errorf("method = %q", call.Method)
		}
		if call.Params["status"] != "C__CMDB_STATUS__IN_OPERATION" {
			t.Errorf("status = %v", call.Params["status"])
		}
		if call.Params["template"] == nil {
			t.Errorf("template not forwarded: %+v", call.Params)
		}
		return json.RawMessage(`{"id":"99","success":true}`)
	})
	c := newTestClient(t, srv, Config{})

	id, err := c.CreateObjectFull(context.Background(), "C__OBJTYPE__SERVER", "web", "C__CMDB_STATUS__IN_OPERATION", 12)
	if err != nil || id != 99 {
		t.Fatalf("got (%d, %v), want (99, nil)", id, err)
	}
}

func TestCreateObjectOmitsEmptyStatusAndTemplate(t *testing.T) {
	srv := testServer(t, func(call rpcCall) any {
		if _, ok := call.Params["status"]; ok {
			t.Errorf("status should be omitted when empty")
		}
		if _, ok := call.Params["template"]; ok {
			t.Errorf("template should be omitted when zero")
		}
		return json.RawMessage(`{"id":1}`)
	})
	c := newTestClient(t, srv, Config{})
	if _, err := c.CreateObject(context.Background(), "C__OBJTYPE__SERVER", "web"); err != nil {
		t.Fatalf("CreateObject: %v", err)
	}
}

func TestUpdateObject(t *testing.T) {
	srv := testServer(t, func(call rpcCall) any {
		if call.Method != "cmdb.object.update" {
			t.Errorf("method = %q", call.Method)
		}
		if call.Params["id"] == nil || call.Params["status"] != "C__CMDB_STATUS__RETIRED" {
			t.Errorf("unexpected params: %+v", call.Params)
		}
		return json.RawMessage(`{"success":true}`)
	})
	c := newTestClient(t, srv, Config{})
	if err := c.UpdateObject(context.Background(), 7, map[string]any{"status": "C__CMDB_STATUS__RETIRED"}); err != nil {
		t.Fatalf("UpdateObject: %v", err)
	}
	if err := c.UpdateObject(context.Background(), 7, nil); err != nil {
		t.Fatalf("UpdateObject(nil) should be a no-op, got %v", err)
	}
}

package client

import (
	"context"
	"encoding/json"
	"testing"
)

func TestReadCategoryArray(t *testing.T) {
	srv := testServer(t, func(call rpcCall) any {
		if call.Method != "cmdb.category.read" {
			t.Errorf("method = %q", call.Method)
		}
		return json.RawMessage(`[{"id":"5","hostname":"web01"},{"id":"6","hostname":"web02"}]`)
	})
	c := newTestClient(t, srv, Config{})

	entries, err := c.ReadCategory(context.Background(), 42, "C__CATG__IP")
	if err != nil {
		t.Fatalf("ReadCategory: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].EntryID() != 5 || entries[1].EntryID() != 6 {
		t.Fatalf("entry ids = %d, %d", entries[0].EntryID(), entries[1].EntryID())
	}
}

func TestReadCategoryBareObject(t *testing.T) {
	srv := testServer(t, func(call rpcCall) any {
		return json.RawMessage(`{"id":7,"description":"single"}`)
	})
	c := newTestClient(t, srv, Config{})

	entries, err := c.ReadCategory(context.Background(), 42, "C__CATS__NET")
	if err != nil {
		t.Fatalf("ReadCategory: %v", err)
	}
	if len(entries) != 1 || entries[0].EntryID() != 7 {
		t.Fatalf("unexpected entries: %+v", entries)
	}
}

func TestReadCategorySentinels(t *testing.T) {
	for _, body := range []string{`null`, `false`, `[]`, `{}`} {
		srv := testServer(t, func(call rpcCall) any { return json.RawMessage(body) })
		c := newTestClient(t, srv, Config{})
		entries, err := c.ReadCategory(context.Background(), 1, "C__CATG__IP")
		if err != nil || entries != nil {
			t.Errorf("body %s: got (%v, %v), want (nil, nil)", body, entries, err)
		}
	}
}

func TestReadCategoryNotFoundSwallowed(t *testing.T) {
	srv := testServer(t, func(call rpcCall) any {
		return &RPCError{Code: -1, Message: "object does not exist"}
	})
	c := newTestClient(t, srv, Config{})
	entries, err := c.ReadCategory(context.Background(), 1, "C__CATG__IP")
	if err != nil || entries != nil {
		t.Fatalf("got (%v, %v), want (nil, nil)", entries, err)
	}
}

func TestReadCategoryOtherErrorPropagates(t *testing.T) {
	srv := testServer(t, func(call rpcCall) any {
		return &RPCError{Code: -32603, Message: "Permission denied"}
	})
	c := newTestClient(t, srv, Config{})
	if _, err := c.ReadCategory(context.Background(), 1, "C__CATG__IP"); err == nil {
		t.Fatal("expected error to propagate")
	}
}

func TestSaveCategoryObjectResult(t *testing.T) {
	srv := testServer(t, func(call rpcCall) any {
		if call.Params["object"] == nil || call.Params["category"] == nil || call.Params["data"] == nil {
			t.Errorf("missing save params: %+v", call.Params)
		}
		return json.RawMessage(`{"entry":"9","success":true}`)
	})
	c := newTestClient(t, srv, Config{})

	id, err := c.SaveCategory(context.Background(), 42, "C__CATG__IP", map[string]any{"hostname": "x"}, 0)
	if err != nil {
		t.Fatalf("SaveCategory: %v", err)
	}
	if id != 9 {
		t.Fatalf("entry id = %d, want 9", id)
	}
}

func TestSaveCategoryBareID(t *testing.T) {
	srv := testServer(t, func(call rpcCall) any { return json.RawMessage(`12`) })
	c := newTestClient(t, srv, Config{})
	id, err := c.SaveCategory(context.Background(), 42, "C__CATG__IP", map[string]any{"x": "y"}, 0)
	if err != nil || id != 12 {
		t.Fatalf("got (%d, %v), want (12, nil)", id, err)
	}
}

func TestSaveCategoryFallsBackToEntryID(t *testing.T) {
	srv := testServer(t, func(call rpcCall) any {
		if call.Params["entry"] == nil {
			t.Error("expected entry param on update")
		}
		return json.RawMessage(`{"success":true}`)
	})
	c := newTestClient(t, srv, Config{})
	id, err := c.SaveCategory(context.Background(), 42, "C__CATG__IP", map[string]any{"x": "y"}, 3)
	if err != nil || id != 3 {
		t.Fatalf("got (%d, %v), want (3, nil)", id, err)
	}
}

func TestDeleteCategoryEntrySwallowsNotFound(t *testing.T) {
	srv := testServer(t, func(call rpcCall) any {
		return &RPCError{Code: -1, Message: "entry does not exist"}
	})
	c := newTestClient(t, srv, Config{})
	if err := c.DeleteCategoryEntry(context.Background(), 42, "C__CATG__IP", 5); err != nil {
		t.Fatalf("not-found delete should be a no-op, got %v", err)
	}
}

func TestDeleteCategoryEntryPropagatesOtherErrors(t *testing.T) {
	srv := testServer(t, func(call rpcCall) any {
		return &RPCError{Code: -32603, Message: "Permission denied"}
	})
	c := newTestClient(t, srv, Config{})
	if err := c.DeleteCategoryEntry(context.Background(), 42, "C__CATG__IP", 5); err == nil {
		t.Fatal("expected error to propagate")
	}
}

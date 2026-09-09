package client

import (
	"context"
	"encoding/json"
	"testing"
)

func TestReadDialogArray(t *testing.T) {
	srv := testServer(t, func(call rpcCall) any {
		if call.Method != "cmdb.dialog.read" {
			t.Errorf("method = %q", call.Method)
		}
		return json.RawMessage(`[{"id":"5","title":"Dell","const":"C__X__DELL"},{"id":6,"title":"HP"}]`)
	})
	c := newTestClient(t, srv, Config{})

	vals, err := c.ReadDialog(context.Background(), "C__CATG__MODEL", "manufacturer")
	if err != nil {
		t.Fatalf("ReadDialog: %v", err)
	}
	if len(vals) != 2 || vals[0].ID.Int64() != 5 || vals[0].ConstValue() != "C__X__DELL" || vals[1].Title != "HP" {
		t.Fatalf("unexpected values: %+v", vals)
	}
}

func TestReadDialogObjectMap(t *testing.T) {
	srv := testServer(t, func(call rpcCall) any {
		return json.RawMessage(`{"5":{"title":"Dell"},"6":{"title":"HP"}}`)
	})
	c := newTestClient(t, srv, Config{})

	vals, err := c.ReadDialog(context.Background(), "C__CATG__MODEL", "manufacturer")
	if err != nil {
		t.Fatalf("ReadDialog: %v", err)
	}
	if len(vals) != 2 {
		t.Fatalf("expected 2 values, got %d", len(vals))
	}
	byID := map[int64]string{}
	for _, v := range vals {
		byID[v.ID.Int64()] = v.Title
	}
	if byID[5] != "Dell" || byID[6] != "HP" {
		t.Fatalf("id was not backfilled from the map key: %+v", vals)
	}
}

func TestReadDialogSentinels(t *testing.T) {
	for _, body := range []string{`null`, `[]`, `{}`, `false`} {
		srv := testServer(t, func(call rpcCall) any { return json.RawMessage(body) })
		c := newTestClient(t, srv, Config{})
		vals, err := c.ReadDialog(context.Background(), "C__CATG__MODEL", "manufacturer")
		if err != nil || vals != nil {
			t.Errorf("body %s: got (%v, %v), want (nil, nil)", body, vals, err)
		}
	}
}

func TestReadDialogNotFoundSwallowed(t *testing.T) {
	srv := testServer(t, func(call rpcCall) any {
		return &RPCError{Code: -1, Message: "property does not exist"}
	})
	c := newTestClient(t, srv, Config{})
	vals, err := c.ReadDialog(context.Background(), "C__CATG__MODEL", "nope")
	if err != nil || vals != nil {
		t.Fatalf("got (%v, %v), want (nil, nil)", vals, err)
	}
}

func TestCreateDialogValueWithParent(t *testing.T) {
	srv := testServer(t, func(call rpcCall) any {
		if call.Method != "cmdb.dialog.create" {
			t.Errorf("method = %q", call.Method)
		}
		if call.Params["parent"] == nil {
			t.Errorf("parent not forwarded: %+v", call.Params)
		}
		return json.RawMessage(`{"entry_id":"42","success":true}`)
	})
	c := newTestClient(t, srv, Config{})

	id, err := c.CreateDialogValue(context.Background(), "C__CATG__MODEL", "manufacturer", "Acme", 7)
	if err != nil || id != 42 {
		t.Fatalf("got (%d, %v), want (42, nil)", id, err)
	}
}

func TestUpdateDialog(t *testing.T) {
	srv := testServer(t, func(call rpcCall) any {
		if call.Method != "cmdb.dialog.update" {
			t.Errorf("method = %q", call.Method)
		}
		if call.Params["value"] != "Renamed" {
			t.Errorf("value = %v", call.Params["value"])
		}
		return json.RawMessage(`{"success":true}`)
	})
	c := newTestClient(t, srv, Config{})
	if err := c.UpdateDialog(context.Background(), "C__CATG__MODEL", "manufacturer", 42, "Renamed"); err != nil {
		t.Fatalf("UpdateDialog: %v", err)
	}
}

func TestDeleteDialogSwallowsNotFound(t *testing.T) {
	srv := testServer(t, func(call rpcCall) any {
		return &RPCError{Code: -1, Message: "entry does not exist"}
	})
	c := newTestClient(t, srv, Config{})
	if err := c.DeleteDialog(context.Background(), "C__CATG__MODEL", "manufacturer", 42); err != nil {
		t.Fatalf("expected not-found delete to be a no-op, got %v", err)
	}
}

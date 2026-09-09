package client

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// DialogValue is one selectable value of a dialog / dialog+ attribute.
type DialogValue struct {
	ID       FlexInt `json:"id"`
	Title    string  `json:"title"`
	Const    string  `json:"const"`
	Constant string  `json:"constant"`
	ParentID FlexInt `json:"parent"`
}

// ConstValue returns whichever constant field i-doit populated.
func (d DialogValue) ConstValue() string {
	if d.Const != "" {
		return d.Const
	}
	return d.Constant
}

// ReadDialog returns the values configured for a dialog attribute
// (cmdb.dialog.read). property is the attribute key, e.g. "manufacturer".
// The result may be a JSON array or an id-keyed object; both are handled.
func (c *Client) ReadDialog(ctx context.Context, category, property string) ([]DialogValue, error) {
	params := map[string]any{
		"category": category,
		"property": property,
	}
	var raw json.RawMessage
	if err := c.Request(ctx, "cmdb.dialog.read", params, &raw); err != nil {
		if IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	trimmed := strings.TrimSpace(string(raw))
	switch {
	case trimmed == "", trimmed == "null", trimmed == "false", trimmed == "[]", trimmed == "{}":
		return nil, nil
	case strings.HasPrefix(trimmed, "["):
		var vals []DialogValue
		if err := json.Unmarshal(raw, &vals); err != nil {
			return nil, err
		}
		return vals, nil
	case strings.HasPrefix(trimmed, "{"):
		var m map[string]DialogValue
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
		out := make([]DialogValue, 0, len(m))
		for k, v := range m {
			if v.ID == 0 {
				v.ID = FlexInt(parseInt64(k))
			}
			out = append(out, v)
		}
		return out, nil
	default:
		return nil, errors.New("unexpected cmdb.dialog.read result shape")
	}
}

type dialogWriteResult struct {
	EntryID FlexInt `json:"entry_id"`
	ID      FlexInt `json:"id"`
	Success bool    `json:"success"`
}

// CreateDialog adds a new value to a dialog attribute (cmdb.dialog.create).
func (c *Client) CreateDialog(ctx context.Context, category, property, value string) (int64, error) {
	return c.CreateDialogValue(ctx, category, property, value, 0)
}

// CreateDialogValue adds a new value to a dialog attribute, optionally nested
// under parentID for a hierarchical dialog+ attribute. It returns the new id.
func (c *Client) CreateDialogValue(ctx context.Context, category, property, value string, parentID int64) (int64, error) {
	params := map[string]any{
		"category": category,
		"property": property,
		"value":    value,
	}
	if parentID > 0 {
		params["parent"] = parentID
	}
	var res dialogWriteResult
	if err := c.Request(ctx, "cmdb.dialog.create", params, &res); err != nil {
		return 0, err
	}
	if res.EntryID != 0 {
		return res.EntryID.Int64(), nil
	}
	return res.ID.Int64(), nil
}

// UpdateDialog renames an existing dialog value (cmdb.dialog.update).
func (c *Client) UpdateDialog(ctx context.Context, category, property string, entryID int64, value string) error {
	params := map[string]any{
		"category": category,
		"property": property,
		"entry_id": entryID,
		"value":    value,
	}
	return c.Request(ctx, "cmdb.dialog.update", params, nil)
}

// DeleteDialog removes a dialog value (cmdb.dialog.delete). A missing value is
// treated as already gone.
func (c *Client) DeleteDialog(ctx context.Context, category, property string, entryID int64) error {
	params := map[string]any{
		"category": category,
		"property": property,
		"entry_id": entryID,
	}
	err := c.Request(ctx, "cmdb.dialog.delete", params, nil)
	if err != nil && IsNotFound(err) {
		return nil
	}
	return err
}

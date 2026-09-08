package client

import "context"

// DialogValue is one selectable value of a dialog / dialog+ attribute.
type DialogValue struct {
	ID       FlexInt `json:"id"`
	Title    string  `json:"title"`
	Const    string  `json:"const"`
	Constant string  `json:"constant"`
}

// ReadDialog returns the values configured for a dialog attribute
// (cmdb.dialog.read). property is the attribute key, e.g. "manufacturer".
func (c *Client) ReadDialog(ctx context.Context, category, property string) ([]DialogValue, error) {
	params := map[string]any{
		"category": category,
		"property": property,
	}
	var values []DialogValue
	if err := c.Request(ctx, "cmdb.dialog.read", params, &values); err != nil {
		return nil, err
	}
	return values, nil
}

type createDialogResult struct {
	EntryID FlexInt `json:"entry_id"`
	ID      FlexInt `json:"id"`
	Success bool    `json:"success"`
}

// CreateDialog adds a new value to a dialog attribute (cmdb.dialog.create) and
// returns the id of the created value.
func (c *Client) CreateDialog(ctx context.Context, category, property, value string) (int64, error) {
	params := map[string]any{
		"category": category,
		"property": property,
		"value":    value,
	}
	var res createDialogResult
	if err := c.Request(ctx, "cmdb.dialog.create", params, &res); err != nil {
		return 0, err
	}
	if res.EntryID != 0 {
		return res.EntryID.Int64(), nil
	}
	return res.ID.Int64(), nil
}

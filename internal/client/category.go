package client

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

// CategoryEntry is a single category record. Field values are kept as raw
// interface values because their shape depends on the attribute type (scalar
// for text/number fields, object with id/title/const for dialog fields).
type CategoryEntry map[string]any

// EntryID extracts the numeric identifier of a category entry ("id" key).
func (e CategoryEntry) EntryID() int64 {
	switch v := e["id"].(type) {
	case float64:
		return int64(v)
	case string:
		return parseInt64(v)
	}
	return 0
}

func parseInt64(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if v, err := strconv.ParseInt(s, 10, 64); err == nil {
		return v
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return int64(f)
	}
	return 0
}

// ReadCategory returns every entry i-doit currently stores for the given
// category on the given object. Single-value categories may be returned as a
// bare object instead of a one element array; both shapes are handled.
func (c *Client) ReadCategory(ctx context.Context, objectID int64, category string) ([]CategoryEntry, error) {
	params := map[string]any{
		"objID":    objectID,
		"category": category,
	}
	var raw json.RawMessage
	if err := c.Request(ctx, "cmdb.category.read", params, &raw); err != nil {
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
		var entries []CategoryEntry
		if err := json.Unmarshal(raw, &entries); err != nil {
			return nil, err
		}
		return entries, nil
	case strings.HasPrefix(trimmed, "{"):
		var entry CategoryEntry
		if err := json.Unmarshal(raw, &entry); err != nil {
			return nil, err
		}
		return []CategoryEntry{entry}, nil
	default:
		return nil, errors.New("unexpected cmdb.category.read result shape")
	}
}

type saveCategoryResult struct {
	Entry   FlexInt `json:"entry"`
	ID      FlexInt `json:"id"`
	Success bool    `json:"success"`
	Message string  `json:"message"`
}

// SaveCategory creates (entryID == 0) or updates (entryID > 0) a category entry
// via cmdb.category.save and returns the resulting entry id.
func (c *Client) SaveCategory(ctx context.Context, objectID int64, category string, data map[string]any, entryID int64) (int64, error) {
	params := map[string]any{
		"object":   objectID,
		"category": category,
		"data":     data,
	}
	if entryID > 0 {
		params["entry"] = entryID
	}

	var raw json.RawMessage
	if err := c.Request(ctx, "cmdb.category.save", params, &raw); err != nil {
		return 0, err
	}

	// Newer i-doit returns an object; some versions return a bare id.
	var res saveCategoryResult
	if err := json.Unmarshal(raw, &res); err == nil {
		switch {
		case res.Entry != 0:
			return res.Entry.Int64(), nil
		case res.ID != 0:
			return res.ID.Int64(), nil
		}
	}
	var bare FlexInt
	if err := json.Unmarshal(raw, &bare); err == nil && bare != 0 {
		return bare.Int64(), nil
	}

	if entryID > 0 {
		return entryID, nil
	}
	return 0, errors.New("cmdb.category.save returned no entry id")
}

// DeleteCategoryEntry archives a single category entry (reversible).
func (c *Client) DeleteCategoryEntry(ctx context.Context, objectID int64, category string, entryID int64) error {
	params := map[string]any{
		"object":   objectID,
		"category": category,
		"entry":    entryID,
	}
	err := c.Request(ctx, "cmdb.category.delete", params, nil)
	if err != nil && IsNotFound(err) {
		// Already gone.
		return nil
	}
	return err
}

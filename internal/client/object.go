package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Version holds the payload of idoit.version.
type Version struct {
	Version string `json:"version"`
	Step    string `json:"step"`
	Type    string `json:"type"`
	Login   struct {
		Userid   string `json:"userid"`
		Name     string `json:"name"`
		Username string `json:"username"`
		Mail     string `json:"mail"`
		Language string `json:"language"`
	} `json:"login"`
}

// ReadVersion calls idoit.version. It doubles as the provider reachability and
// authentication probe.
func (c *Client) ReadVersion(ctx context.Context) (*Version, error) {
	var v Version
	if err := c.Request(ctx, "idoit.version", nil, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// Object is the subset of cmdb.object.read we surface to Terraform.
type Object struct {
	ID              FlexInt `json:"id"`
	Title           string  `json:"title"`
	SysID           string  `json:"sysid"`
	TypeConst       string  `json:"type"`
	TypeTitle       string  `json:"type_title"`
	ObjectType      FlexInt `json:"objecttype"`
	Status          FlexInt `json:"status"`
	CmdbStatus      FlexInt `json:"cmdb_status"`
	CmdbStatusTitle string  `json:"cmdb_status_title"`
	Created         string  `json:"created"`
	Updated         string  `json:"updated"`
}

// StatusLabel returns the most descriptive status string available.
func (o *Object) StatusLabel() string {
	if o.CmdbStatusTitle != "" {
		return o.CmdbStatusTitle
	}
	if o.CmdbStatus != 0 {
		return fmt.Sprintf("%d", o.CmdbStatus.Int64())
	}
	if o.Status != 0 {
		return fmt.Sprintf("%d", o.Status.Int64())
	}
	return ""
}

type createObjectResult struct {
	ID      FlexInt `json:"id"`
	Message string  `json:"message"`
	Success bool    `json:"success"`
}

// CreateObject creates a CMDB object and returns its numeric id.
func (c *Client) CreateObject(ctx context.Context, objType, title string) (int64, error) {
	params := map[string]any{
		"type":  objType,
		"title": title,
	}
	var res createObjectResult
	if err := c.Request(ctx, "cmdb.object.create", params, &res); err != nil {
		return 0, err
	}
	if res.ID == 0 {
		return 0, errors.New("cmdb.object.create returned no object id")
	}
	return res.ID.Int64(), nil
}

// ReadObject returns the object with the given id, or (nil, nil) when it does
// not exist.
func (c *Client) ReadObject(ctx context.Context, id int64) (*Object, error) {
	params := map[string]any{"id": id}

	var raw json.RawMessage
	if err := c.Request(ctx, "cmdb.object.read", params, &raw); err != nil {
		// Only a genuine "object does not exist" error means the resource is
		// gone; auth / permission / server errors must be surfaced so a
		// healthy object is never dropped from state.
		if IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	trimmed := strings.TrimSpace(string(raw))
	switch trimmed {
	case "", "null", "false", "[]", "{}":
		return nil, nil
	}

	var obj Object
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, fmt.Errorf("decode cmdb.object.read result: %w", err)
	}
	if obj.ID == 0 {
		return nil, nil
	}
	return &obj, nil
}

// UpdateObjectTitle updates the object title (the only mutable core attribute).
func (c *Client) UpdateObjectTitle(ctx context.Context, id int64, title string) error {
	params := map[string]any{"id": id, "title": title}
	return c.Request(ctx, "cmdb.object.update", params, nil)
}

// ArchiveObject sets the object to the "archived" record status (reversible).
func (c *Client) ArchiveObject(ctx context.Context, id int64) error {
	return c.Request(ctx, "cmdb.object.archive", map[string]any{"object": id}, nil)
}

// PurgeObject removes the object from the database irreversibly.
func (c *Client) PurgeObject(ctx context.Context, id int64) error {
	return c.Request(ctx, "cmdb.object.purge", map[string]any{"object": id}, nil)
}

// ObjectsFilter maps to the "filter" parameter of cmdb.objects.read.
type ObjectsFilter struct {
	Type  string
	Title string
	IDs   []int64
	SysID string
}

func (f ObjectsFilter) toMap() map[string]any {
	m := map[string]any{}
	if f.Type != "" {
		m["type"] = f.Type
	}
	if f.Title != "" {
		m["title"] = f.Title
	}
	if f.SysID != "" {
		m["sysid"] = f.SysID
	}
	if len(f.IDs) > 0 {
		m["ids"] = f.IDs
	}
	return m
}

// ObjectListItem is one entry of cmdb.objects.read.
type ObjectListItem struct {
	ID        FlexInt `json:"id"`
	Title     string  `json:"title"`
	SysID     string  `json:"sysid"`
	TypeConst string  `json:"type"`
	TypeTitle string  `json:"type_title"`
	Status    FlexInt `json:"status"`
}

// ListObjects queries cmdb.objects.read. A limit <= 0 means "no limit".
func (c *Client) ListObjects(ctx context.Context, filter ObjectsFilter, limit int) ([]ObjectListItem, error) {
	params := map[string]any{}
	if f := filter.toMap(); len(f) > 0 {
		params["filter"] = f
	}
	if limit > 0 {
		params["limit"] = limit
	}
	var items []ObjectListItem
	if err := c.Request(ctx, "cmdb.objects.read", params, &items); err != nil {
		return nil, err
	}
	return items, nil
}

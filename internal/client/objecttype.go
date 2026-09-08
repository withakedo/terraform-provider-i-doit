package client

import "context"

// ObjectType is one entry of cmdb.object_types.
type ObjectType struct {
	ID             FlexInt `json:"id"`
	Title          string  `json:"title"`
	Const          string  `json:"const"`
	TitleLang      string  `json:"title_lang"`
	TypeGroup      FlexInt `json:"type_group"`
	Container      FlexInt `json:"container"`
	Status         FlexInt `json:"status"`
	ObjectCount    FlexInt `json:"object_count"`
	ObjectCountOld FlexInt `json:"objectcount"`
}

// Count returns the number of objects of this type, tolerating both the modern
// and the legacy response key.
func (t ObjectType) Count() int64 {
	if t.ObjectCount != 0 {
		return t.ObjectCount.Int64()
	}
	return t.ObjectCountOld.Int64()
}

// ListObjectTypes returns all configured object types. i-doit does not support
// filtering by constant on every version, so callers filter client side.
func (c *Client) ListObjectTypes(ctx context.Context) ([]ObjectType, error) {
	var types []ObjectType
	if err := c.Request(ctx, "cmdb.object_types", map[string]any{"countobjects": true}, &types); err != nil {
		return nil, err
	}
	return types, nil
}

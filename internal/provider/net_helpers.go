package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/withakedo/terraform-provider-i-doit/internal/client"
)

// Object type and category constants used by the typed network resources.
const (
	objTypeLayer3Net = "C__OBJTYPE__LAYER3_NET"
	objTypeLayer2Net = "C__OBJTYPE__LAYER2_NET"

	catL3Net = "C__CATS__NET"
	catL2Net = "C__CATS__LAYER2_NET"
	catIP    = "C__CATG__IP"

	netTypeIPv4Const = "C__CATS_NET_TYPE__IPV4"
	netTypeIPv6Const = "C__CATS_NET_TYPE__IPV6"
)

// putStr adds key=value to m when the framework string is set (non-null,
// non-unknown, non-empty).
func putStr(m map[string]any, key string, v types.String) {
	if !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
		m[key] = v.ValueString()
	}
}

// putBool adds key to m as an i-doit yes/no value ("1"/"0") when set.
func putBool(m map[string]any, key string, v types.Bool) {
	if v.IsNull() || v.IsUnknown() {
		return
	}
	if v.ValueBool() {
		m[key] = "1"
	} else {
		m[key] = "0"
	}
}

// putStrList adds key to m as a []string when the list has elements.
func putStrList(ctx context.Context, m map[string]any, key string, v types.List) diag.Diagnostics {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	var out []string
	d := v.ElementsAs(ctx, &out, false)
	if !d.HasError() && len(out) > 0 {
		m[key] = out
	}
	return d
}

// mergeExtra folds an Optional map(string) into a category data map. Explicit
// typed fields already in m win on key collision.
func mergeExtra(ctx context.Context, m map[string]any, extra types.Map) diag.Diagnostics {
	ex, d := mapToStringMap(ctx, extra)
	for k, v := range ex {
		if _, taken := m[k]; !taken {
			m[k] = v
		}
	}
	return d
}

// entryStr coerces a single category field to a string. ok is false when the
// field is absent or cannot be represented as a scalar.
func entryStr(entry map[string]any, key string) (string, bool) {
	raw, ok := entry[key]
	if !ok {
		return "", false
	}
	return coerceScalar(raw)
}

// setStrFromEntry writes the coerced API value of key into dst when present.
func setStrFromEntry(dst *types.String, entry map[string]any, key string) {
	if s, ok := entryStr(entry, key); ok {
		*dst = types.StringValue(s)
	}
}

// entryRefID extracts the numeric id of a reference field (an object, or a
// one-element array of objects, each carrying an "id").
func entryRefID(entry map[string]any, key string) (string, bool) {
	raw, ok := entry[key]
	if !ok {
		return "", false
	}
	switch t := raw.(type) {
	case map[string]any:
		if id, ok := coerceScalar(t["id"]); ok && id != "" && id != "0" {
			return id, true
		}
		return "", false
	case []any:
		if len(t) == 0 {
			return "", false
		}
		if m, ok := t[0].(map[string]any); ok {
			if id, ok := coerceScalar(m["id"]); ok && id != "" && id != "0" {
				return id, true
			}
		}
		return "", false
	default:
		return coerceScalar(raw)
	}
}

// setRefFromEntry writes the referenced object id of key into dst when present.
func setRefFromEntry(dst *types.String, entry map[string]any, key string) {
	if s, ok := entryRefID(entry, key); ok {
		*dst = types.StringValue(s)
	}
}

// setBoolFromEntry interprets an i-doit yes/no field.
func setBoolFromEntry(dst *types.Bool, entry map[string]any, key string) {
	if s, ok := entryStr(entry, key); ok {
		s = strings.ToLower(strings.TrimSpace(s))
		*dst = types.BoolValue(s == "1" || s == "true" || s == "yes")
	}
}

// nullUnknownStr collapses any still-unknown optional+computed string to null so
// the value is known when written to state after apply.
func nullUnknownStr(ps ...*types.String) {
	for _, p := range ps {
		if p.IsUnknown() {
			*p = types.StringNull()
		}
	}
}

// nullUnknownBool does the same for optional+computed bools.
func nullUnknownBool(ps ...*types.Bool) {
	for _, p := range ps {
		if p.IsUnknown() {
			*p = types.BoolNull()
		}
	}
}

// netTypeToConst maps "ipv4"/"ipv6" (any case, or a raw constant) to the i-doit
// C__CATS__NET type dialog constant.
func netTypeToConst(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "ipv6", "6", "c__cats_net_type__ipv6":
		return netTypeIPv6Const
	default:
		return netTypeIPv4Const
	}
}

// constToNetType maps the net-type dialog value of a C__CATS__NET entry back to
// "ipv4"/"ipv6".
func constToNetType(entry map[string]any) string {
	s, _ := entryStr(entry, "type")
	if strings.Contains(strings.ToUpper(s), "IPV6") || strings.TrimSpace(s) == "2" {
		return "ipv6"
	}
	return "ipv4"
}

// firstCategoryEntry returns the single entry of a single-value category, or nil.
func firstCategoryEntry(ctx context.Context, c *client.Client, objectID int64, category string) (client.CategoryEntry, error) {
	entries, err := c.ReadCategory(ctx, objectID, category)
	if err != nil || len(entries) == 0 {
		return nil, err
	}
	return entries[0], nil
}

// findCategoryEntry returns the entry with the given id, falling back to the
// sole entry when there is exactly one and the id did not match.
func findCategoryEntry(ctx context.Context, c *client.Client, objectID int64, category string, entryID int64) (client.CategoryEntry, error) {
	entries, err := c.ReadCategory(ctx, objectID, category)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.EntryID() == entryID {
			return e, nil
		}
	}
	if len(entries) == 1 {
		return entries[0], nil
	}
	return nil, nil
}

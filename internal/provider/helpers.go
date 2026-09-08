package provider

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// mapToStringMap converts a framework string map into a Go map. A null or
// unknown map yields an empty (non-nil) map.
func mapToStringMap(ctx context.Context, m types.Map) (map[string]string, diag.Diagnostics) {
	out := map[string]string{}
	if m.IsNull() || m.IsUnknown() {
		return out, nil
	}
	diags := m.ElementsAs(ctx, &out, false)
	return out, diags
}

func toAnyMap(m map[string]string) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func keysOf(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// coerceScalar reduces an arbitrary JSON value returned by cmdb.category.read to
// a single string. ok is false when the value cannot be represented sensibly as
// a scalar (nested arrays, objects without an obvious label), in which case the
// caller keeps the prior state value.
func coerceScalar(v any) (string, bool) {
	switch t := v.(type) {
	case nil:
		return "", true
	case string:
		return t, true
	case bool:
		if t {
			return "1", true
		}
		return "0", true
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64), true
	case map[string]any:
		for _, key := range []string{"title", "value", "const", "constant", "ref_title", "id"} {
			if raw, exists := t[key]; exists {
				if s, ok := coerceScalar(raw); ok && s != "" {
					return s, true
				}
			}
		}
		return "", false
	default:
		return "", false
	}
}

func categoryEntryID(objectID int64, category string, entryID int64) string {
	return fmt.Sprintf("%d/%s/%d", objectID, category, entryID)
}

// reconcileData builds the state value of the data map for the given managed
// keys: use the coerced API value when the field is present and representable,
// otherwise fall back to the prior state value so unrelated formatting quirks do
// not cause perpetual diffs.
func reconcileData(entry map[string]any, managed []string, prior map[string]string) map[string]string {
	out := make(map[string]string, len(managed))
	for _, k := range managed {
		raw, present := entry[k]
		if !present {
			if pv, ok := prior[k]; ok {
				out[k] = pv
			}
			continue
		}
		if v, ok := coerceScalar(raw); ok {
			out[k] = v
			continue
		}
		if pv, ok := prior[k]; ok {
			out[k] = pv
		}
	}
	return out
}

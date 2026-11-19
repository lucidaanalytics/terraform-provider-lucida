package defsync

import "reflect"

// cleanWebhookSource drops what the API echoes but a file leaves out: zero values inside mapping (hookmap
// serializes distinct_id_path unconditionally) and auth.has_secret / zero auth fields.
func cleanWebhookSource(o Object) {
	for _, f := range []string{"mapping", "auth"} {
		if m, ok := o[f].(map[string]any); ok {
			delete(m, "has_secret")
			for k, v := range m {
				if isZero(v) {
					delete(m, k)
				}
			}
			if len(m) == 0 {
				delete(o, f)
			}
		}
	}
}

func isZero(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case string:
		return x == ""
	case bool:
		return !x
	case float64:
		return x == 0
	case []any:
		return len(x) == 0
	case map[string]any:
		return len(x) == 0
	}
	return false
}

// DropDefaults keeps only the keys of a settings object that differ from its `defaults` block (which goes
// too), so a pulled file states choices, not the API's defaults.
func DropDefaults(o Object) Object {
	defs, _ := o["defaults"].(map[string]any)
	out := Object{}
	for k, v := range o {
		if k == "defaults" {
			continue
		}
		if d, has := defs[k]; has && k != "section" && reflect.DeepEqual(d, v) {
			continue
		}
		out[k] = v
	}
	return out
}

// revenueReadOnly are the feed status fields the API echoes but never accepts.
var revenueReadOnly = []string{"fx_feed_error", "fx_feed_synced_at", "project_id", "updated_at"}

func cleanRevenue(o Object) {
	for _, f := range revenueReadOnly {
		delete(o, f)
	}
}

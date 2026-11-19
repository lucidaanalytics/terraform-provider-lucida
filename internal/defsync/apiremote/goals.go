package apiremote

import "lucida/internal/defsync"

// goalTarget folds the goal the API serves (event / pathname / source) back into the create shape
// (target + warehouse columns), so a pulled goal compares equal to its file.
func goalTarget(g defsync.Object) {
	src, _ := g["source"].(map[string]any)
	set := func(field string, v any) {
		if v != nil {
			g[field] = v
		}
	}
	switch g["kind"] {
	case "event":
		set("target", g["event"])
	case "pageview":
		set("target", g["pathname"])
	case "action":
		set("target", src["id"])
	case "warehouse":
		set("target", src["table"])
		set("timestamp_field", src["timestamp_field"])
		set("distinct_id_field", src["distinct_id_field"])
	}
}

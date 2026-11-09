package apiremote

import (
	"context"
	"fmt"
	"net/url"

	"lucida/internal/defsync"
)

// listEndpoints fills each endpoint's query and variables from its current version and uses the
// slug as the id, since the endpoint routes are addressed by slug (P-M23).
func (r *Remote) listEndpoints(ctx context.Context, rows []defsync.Object) ([]defsync.Object, error) {
	r.endpoints = map[string]defsync.Object{}
	for _, e := range rows {
		slug := fmt.Sprint(e["slug"])
		var versions struct {
			Data []struct {
				Version   float64 `json:"version"`
				Query     any     `json:"query"`
				Variables any     `json:"variables"`
			} `json:"data"`
		}
		if err := r.api.Do(ctx, "GET", "/endpoints/"+url.PathEscape(slug)+"/versions", nil, nil, &versions); err != nil {
			return nil, err
		}
		for _, v := range versions.Data {
			if cur, _ := e["current_version"].(float64); v.Version == cur {
				e["query"] = v.Query
				if vars, ok := v.Variables.([]any); ok && len(vars) > 0 {
					e["variables"] = vars
				}
			}
		}
		e["id"] = slug
		r.endpoints[slug] = e
	}
	return rows, nil
}

// updateEndpoint PATCHes name/description/refresh (the route takes the full trio) and posts a new
// version when the query or variables changed, which becomes the current version.
func (r *Remote) updateEndpoint(ctx context.Context, slug string, o defsync.Object) error {
	cur := r.endpoints[slug]
	meta := false
	patch := defsync.Object{}
	for _, f := range []string{"name", "description", "refresh_every_minutes"} {
		if v, ok := o[f]; ok {
			patch[f], meta = v, true
		} else if v, ok := cur[f]; ok && v != nil {
			patch[f] = v
		}
	}
	if meta {
		if err := r.api.Do(ctx, "PATCH", "/endpoints/"+url.PathEscape(slug), nil, patch, nil); err != nil {
			return err
		}
	}
	_, q := o["query"]
	_, vars := o["variables"]
	if !q && !vars {
		return nil
	}
	ver := defsync.Object{"query": cur["query"], "note": "lucida apply"}
	if v, ok := cur["variables"]; ok {
		ver["variables"] = v
	}
	for _, f := range []string{"query", "variables"} {
		if v, ok := o[f]; ok {
			ver[f] = v
		}
	}
	return r.api.Do(ctx, "POST", "/endpoints/"+url.PathEscape(slug)+"/versions", nil, ver, nil)
}

// Package apiremote is the lc-api adapter behind defsync.Remote. The CLI (`lucida pull|plan|apply`)
// and the Terraform provider both use it, so route quirks (alerts under insights, dashboard tiles,
// endpoint versions, key secrets) live in one place and the two cannot drift.
package apiremote

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"

	"lucida/internal/defsync"
)

// API is the project-scoped JSON transport (lcapi.Client satisfies it). Paths are relative to the
// project, e.g. "/flags"; list answers are {data, next_cursor}; failures are returned as errors.
type API interface {
	Do(ctx context.Context, method, suffix string, q url.Values, body, out any) error
}

// KeySink receives a new key's secret, which the API shows exactly once.
type KeySink func(name, secret string) error

type page struct {
	Data       []json.RawMessage `json:"data"`
	NextCursor string            `json:"next_cursor"`
}

// Remote adapts lc-api to defsync.Remote. Dashboard tiles name their insight; ids are resolved here.
type Remote struct {
	api   API
	onKey KeySink
	// literalRefs sends credential references as given instead of resolving env:NAME (Terraform
	// already holds them as sensitive values).
	literalRefs bool
	insights    map[string]string         // insight id -> name, loaded lazily
	endpoints   map[string]defsync.Object // slug -> listed endpoint, for full-body PATCHes
}

// New returns a Remote; onKey may be nil only when no keys are created.
func New(api API, onKey KeySink) *Remote { return &Remote{api: api, onKey: onKey} }

func (r *Remote) list(ctx context.Context, route string) ([]defsync.Object, error) {
	var out []defsync.Object
	q := url.Values{"limit": {"100"}}
	for {
		var p page
		if err := r.api.Do(ctx, "GET", "/"+route, q, nil, &p); err != nil {
			return nil, err
		}
		for _, raw := range p.Data {
			var v defsync.Object
			if err := json.Unmarshal(raw, &v); err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		if p.NextCursor == "" || ctx.Err() != nil {
			return out, ctx.Err()
		}
		q.Set("cursor", p.NextCursor)
	}
}

// LiteralRefs makes credential references travel as given, not as env:NAME lookups. The CLI never
// calls it.
func (r *Remote) LiteralRefs() *Remote { r.literalRefs = true; return r }

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (r *Remote) insightNames(ctx context.Context) (map[string]string, error) {
	if r.insights == nil {
		rows, err := r.list(ctx, "insights")
		if err != nil {
			return nil, err
		}
		r.insights = map[string]string{}
		for _, o := range rows {
			r.insights[fmt.Sprint(o["id"])] = fmt.Sprint(o["name"])
		}
	}
	return r.insights, nil
}

func (r *Remote) List(ctx context.Context, k defsync.Kind) ([]defsync.Object, error) {
	if k.Singleton {
		return r.listSingletons(ctx, k)
	}
	rows, err := r.list(ctx, k.Path())
	if err != nil {
		return nil, err
	}
	switch k.Name {
	case "experiments": // archived is what prune leaves behind
		live := rows[:0]
		for _, e := range rows {
			if e["status"] != "archived" {
				live = append(live, e)
			}
		}
		slices.SortStableFunc(live, func(a, b defsync.Object) int { return b2i(a["status"] == "stopped") - b2i(b["status"] == "stopped") })
		return live, nil
	case "endpoints":
		return r.listEndpoints(ctx, rows)
	case "web-goals":
		for _, g := range rows {
			goalTarget(g)
		}
		return rows, nil
	case "alerts", "dashboards":
	default:
		return rows, nil
	}
	names, err := r.insightNames(ctx)
	if err != nil {
		return nil, err
	}
	if k.Name == "alerts" {
		for _, a := range rows {
			a["insight"] = names[fmt.Sprint(a["insight_id"])]
		}
		return rows, nil
	}
	for _, d := range rows {
		var detail struct {
			Tiles []struct {
				InsightID *string         `json:"insight_id"`
				TextBody  *string         `json:"text_body"`
				Layout    json.RawMessage `json:"layout"`
				Position  int             `json:"position"`
			} `json:"tiles"`
		}
		if err := r.api.Do(ctx, "GET", "/dashboards/"+url.PathEscape(fmt.Sprint(d["id"])), nil, nil, &detail); err != nil {
			return nil, err
		}
		tiles := []any{}
		for _, t := range detail.Tiles {
			tile := map[string]any{"position": t.Position}
			if t.InsightID != nil {
				tile["insight"] = names[*t.InsightID]
			}
			if t.TextBody != nil {
				tile["text_body"] = *t.TextBody
			}
			var layout map[string]any
			if json.Unmarshal(t.Layout, &layout) == nil && len(layout) > 0 {
				tile["layout"] = layout
			}
			tiles = append(tiles, tile)
		}
		d["tiles"] = tiles
	}
	return rows, nil
}

func (r *Remote) Create(ctx context.Context, k defsync.Kind, o defsync.Object) error {
	if k.Singleton {
		return r.putSingleton(ctx, k, o)
	}
	if k.Name == "webhook-sources" {
		var err error
		if o, err = resolveHookSecret(o); err != nil {
			return err
		}
	}
	if k.Name == "alerts" {
		return r.createAlert(ctx, o)
	}
	if k.Name == "keys" {
		return r.createKey(ctx, o)
	}
	if k.Name == "export-schedules" && !r.literalRefs {
		var err error
		if o, err = ResolveExportCreds(o); err != nil {
			return err
		}
	}
	body, tiles := splitTiles(o)
	var created struct {
		ID string `json:"id"`
	}
	if err := r.api.Do(ctx, "POST", "/"+k.Path(), nil, body, &created); err != nil {
		return err
	}
	if k.Name == "insights" && r.insights != nil {
		r.insights[created.ID] = fmt.Sprint(o["name"])
	}
	if tiles == nil {
		return nil
	}
	return r.addTiles(ctx, created.ID, tiles)
}

func (r *Remote) Update(ctx context.Context, k defsync.Kind, id string, o defsync.Object) error {
	if k.Singleton {
		return r.putSingleton(ctx, k, defsync.Object{"section": id, "name": id, "body": o})
	}
	if k.Name == "endpoints" {
		return r.updateEndpoint(ctx, id, o)
	}
	body, tiles := splitTiles(o)
	method := "PATCH"
	if k.Put {
		method = "PUT"
	}
	if err := r.api.Do(ctx, method, "/"+k.Path()+"/"+url.PathEscape(id), nil, body, nil); err != nil {
		return err
	}
	if tiles == nil {
		return nil
	}
	var detail struct {
		Tiles []struct {
			ID string `json:"id"`
		} `json:"tiles"`
	}
	if err := r.api.Do(ctx, "GET", "/dashboards/"+url.PathEscape(id), nil, nil, &detail); err != nil {
		return err
	}
	for _, t := range detail.Tiles { // tiles are replaced wholesale: they have no natural key
		if err := r.api.Do(ctx, "DELETE", "/dashboards/"+url.PathEscape(id)+"/tiles/"+url.PathEscape(t.ID), nil, nil, nil); err != nil {
			return err
		}
	}
	return r.addTiles(ctx, id, tiles)
}

func (r *Remote) Delete(ctx context.Context, k defsync.Kind, id string) error {
	if k.Name == "experiments" { // experiments have no delete: archiving hides them
		return r.api.Do(ctx, "POST", "/experiments/"+url.PathEscape(id)+"/archive", nil, nil, nil)
	}
	return r.api.Do(ctx, "DELETE", "/"+k.Path()+"/"+url.PathEscape(id), nil, nil, nil)
}

// createAlert posts under the alert's insight (named in the file) and then applies `enabled`,
// which the create route does not take.
func (r *Remote) createAlert(ctx context.Context, o defsync.Object) error {
	names, err := r.insightNames(ctx)
	if err != nil {
		return err
	}
	insight := ""
	for id, n := range names {
		if n == o["insight"] {
			insight = id
		}
	}
	if insight == "" {
		return fmt.Errorf("alert references unknown insight %v (add it under insights/)", o["insight"])
	}
	body := defsync.Object{}
	for f, v := range o {
		if f != "insight" && f != "enabled" {
			body[f] = v
		}
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := r.api.Do(ctx, "POST", "/insights/"+url.PathEscape(insight)+"/alerts", nil, body, &created); err != nil {
		return err
	}
	if o["enabled"] != false {
		return nil
	}
	return r.api.Do(ctx, "PATCH", "/alerts/"+url.PathEscape(created.ID), nil, defsync.Object{"enabled": false}, nil)
}

func (r *Remote) addTiles(ctx context.Context, dashboard string, tiles []any) error {
	names, err := r.insightNames(ctx)
	if err != nil {
		return err
	}
	ids := map[string]string{}
	for id, n := range names {
		ids[n] = id
	}
	for _, raw := range tiles {
		t, _ := raw.(map[string]any)
		body := map[string]any{"position": t["position"], "layout": t["layout"], "text_body": t["text_body"]}
		if n, ok := t["insight"].(string); ok {
			id, found := ids[n]
			if !found {
				return fmt.Errorf("dashboard tile references unknown insight %q (add it under insights/)", n)
			}
			body["insight_id"] = id
		}
		if body["layout"] == nil {
			delete(body, "layout")
		}
		if err := r.api.Do(ctx, "POST", "/dashboards/"+url.PathEscape(dashboard)+"/tiles", nil, body, nil); err != nil {
			return err
		}
	}
	return nil
}

// splitTiles separates dashboard tiles (synced through the tiles routes) from the object body.
func splitTiles(o defsync.Object) (defsync.Object, []any) {
	tiles, ok := o["tiles"].([]any)
	if !ok {
		return o, nil
	}
	body := defsync.Object{}
	for k, v := range o {
		if k != "tiles" {
			body[k] = v
		}
	}
	return body, tiles
}

// Package defsync is definitions as code: project definitions (flags, actions, cohorts, insights,
// metrics, experiments, dashboards with tiles, insight alerts) live as YAML files, and Diff/Apply reconcile them with lc-api
// through the Remote port. Objects are matched by a natural key (flag key or name), never by id,
// so the same files apply to any project.
package defsync

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"sort"
)

// Object is one definition in its API JSON shape.
type Object = map[string]any

// Kind is one definition type. Name is both the directory under the root and the API route.
type Kind struct {
	Name   string
	Key    string   // natural key field
	Fields []string // fields managed as code; everything else (ids, timestamps) is ignored
	Fixed  []string // sent on create only (the API cannot change them)
	Put    bool     // update replaces the object (PUT) instead of PATCH
	Frozen []string // sent on create only and never compared afterwards (the API cannot change them)
	Refs   []string // credential references: sent on create only, never compared, pulled or planned (values are env:NAME, resolved by the adapter)
	Once   bool     // create only: an existing object is never compared or changed (API keys)
	// Open: every key of the object is managed (settings sections), except id and defaults. NoPrune: prune never
	// deletes these (the remote has fixed singletons, not a collection). Clean: drops what the API echoes but
	// a file never writes (zero values), applied to both sides before comparing.
	Open, NoPrune bool
	Clean         func(Object)
	// Route is the API collection when it differs from Name ("web/goals"); Singleton: one fixed object per
	// project (settings sections, toolbar origins, revenue config): created and updated by PUT, never deleted.
	Route     string
	Singleton bool
}

// Path is the project-relative API route of the kind.
func (k Kind) Path() string {
	if k.Route != "" {
		return k.Route
	}
	return k.Name
}

// Kinds in apply order: insights before dashboards and alerts, which reference insights by name.
// Experiments are archived on prune (the API has no delete); alert Slack URLs pull redacted, so a
// file only changes channels when it restates every channel with its url. Endpoints come last.
var Kinds = []Kind{
	{Name: "actions", Key: "name", Fields: []string{"name", "description", "steps"}},
	{Name: "cohorts", Key: "name", Fields: []string{"name", "description", "kind", "filters"}, Fixed: []string{"kind"}},
	{Name: "flags", Key: "key", Fields: []string{"key", "name", "description", "kind", "filters", "variants", "payloads", "stickiness", "tags", "enabled"}, Fixed: []string{"key"}},
	{Name: "insights", Key: "name", Fields: []string{"name", "description", "query", "settings", "tags", "folder"}},
	{Name: "metrics", Key: "name", Fields: []string{"name", "display_name", "description", "unit", "kind", "definition", "tags"}, Put: true},
	{Name: "experiments", Key: "name", Fields: []string{"name", "hypothesis", "kind", "variants", "audience", "traffic_pct", "schedule", "exposure", "metrics", "stats_config", "requires_approval"}, Fixed: []string{"kind"}},
	{Name: "dashboards", Key: "name", Fields: []string{"name", "description", "filters", "tags", "pinned", "tiles"}},
	{Name: "alerts", Key: "name", Fields: []string{"name", "insight", "condition", "check_interval", "channels", "enabled"}, Fixed: []string{"insight"}},
	// Endpoints are keyed by slug; query and variables are the current version, so a change to
	// either becomes a new version (the adapter posts /versions), the rest a PATCH.
	// Keys are created once; the file never holds a secret (create returns it once, see the CLI).
	{Name: "keys", Key: "name", Fields: []string{"name", "kind", "scopes", "revoked"}, Once: true}, // revoked: written by pull --include-revoked
	// Batch export schedules: only name, interval and enabled change in place; where and what they
	// export is frozen at create (delete and recreate to change it). Credentials are env:NAME references.
	{Name: "export-schedules", Key: "name", Fields: []string{"name", "stream", "format", "destination", "warehouse", "interval_secs", "chunk_secs", "options", "enabled"},
		Frozen: []string{"stream", "format", "destination", "warehouse", "chunk_secs", "options"}, Refs: []string{"credentials"}},
	// Settings: one object per section (ingest, sdk) over /settings/<section>; the API serves effective
	// values plus `defaults`, pull writes only keys that differ from them and prune never resets a section.
	{Name: "settings", Key: "section", Open: true, NoPrune: true, Singleton: true},
	// The origins the toolbar may run on: one object, {name: origins, origins: [...]}.
	{Name: "toolbar-origins", Key: "name", Fields: []string{"name", "origins"}, NoPrune: true, Singleton: true},
	// Webhook sources by name. The signing secret never leaves the server: auth_secret is an env:NAME
	// reference read on create only (see apiremote); a pulled file has auth mode, header and tolerance only.
	{Name: "webhook-sources", Key: "name", Fields: []string{"name", "mapping", "enabled", "log_payloads", "log_keep", "auth"}, Refs: []string{"auth_secret"}, Clean: cleanWebhookSource},
	// Tracking plans (event schema + enforcement mode) by name.
	{Name: "tracking-plans", Key: "name", Fields: []string{"name", "mode", "schema"}},
	// Web goals by name. The API has no update: everything but the name is sent on create only (delete and recreate to change).
	{Name: "web-goals", Route: "web/goals", Key: "name", Fields: []string{"name", "kind", "target", "value", "value_property", "timestamp_field", "distinct_id_field"},
		Frozen: []string{"kind", "target", "value", "value_property", "timestamp_field", "distinct_id_field"}},
	// Revenue config: one object {name: revenue, ...mapping}; the API serves the defaults until it is saved.
	{Name: "revenue", Route: "web/revenue/config", Key: "name", Open: true, NoPrune: true, Singleton: true, Clean: cleanRevenue},
	{Name: "endpoints", Key: "slug", Fields: []string{"slug", "name", "description", "refresh_every_minutes", "query", "variables"}, Fixed: []string{"slug"}},
}

// KindByName finds a kind.
func KindByName(n string) (Kind, bool) {
	for _, k := range Kinds {
		if k.Name == n {
			return k, true
		}
	}
	return Kind{}, false
}

// Remote is the port to lc-api. Dashboard tiles travel as Object["tiles"] = [{insight, text_body,
// layout, position}] with the insight named, and the adapter maps names to ids.
type Remote interface {
	List(ctx context.Context, k Kind) ([]Object, error)
	Create(ctx context.Context, k Kind, o Object) error
	Update(ctx context.Context, k Kind, id string, o Object) error
	Delete(ctx context.Context, k Kind, id string) error
}

// Op is what Apply will do to one object.
type Op string

const (
	OpCreate Op = "create"
	OpUpdate Op = "update"
	OpDelete Op = "delete"
	OpSame   Op = "unchanged"
)

// Change is one planned operation.
type Change struct {
	Kind Kind
	Key  string
	ID   string // remote id (update, delete)
	Op   Op
	Body Object   // managed fields to send (create, update)
	Diff []string // update: names of the fields that differ (values are never shown)
}

// Normalize keeps only the managed fields, JSON-normalized so YAML ints and API floats compare equal.
func Normalize(k Kind, o Object) Object {
	out := Object{}
	if k.Open {
		for f, v := range o {
			if v != nil && f != "id" && f != "defaults" {
				out[f] = v
			}
		}
	}
	for _, f := range k.Fields {
		if v, ok := o[f]; ok && v != nil {
			out[f] = v
		}
	}
	raw, _ := json.Marshal(out) // YAML/JSON-decoded values always marshal
	var norm Object
	_ = json.Unmarshal(raw, &norm)
	if k.Clean != nil {
		k.Clean(norm)
	}
	return norm
}

// Dedupe keeps the first object per key and returns the keys it dropped.
func Dedupe(k Kind, objs []Object) (kept []Object, dropped []string) {
	seen := map[string]bool{}
	for _, o := range objs {
		key := keyOf(k, o)
		if seen[key] {
			dropped = append(dropped, key)
			continue
		}
		seen[key] = true
		kept = append(kept, o)
	}
	return kept, dropped
}

func keyOf(k Kind, o Object) string { s, _ := o[k.Key].(string); return s }

// Diff plans the changes that make remote match local for one kind. A local field that is absent
// is left alone (not cleared); prune also deletes remote objects missing from local.
func Diff(k Kind, local, remote []Object, prune bool) ([]Change, error) {
	byKey := map[string]Object{}
	for _, r := range remote {
		if _, dup := byKey[keyOf(k, r)]; !dup { // names can repeat remotely: the first is the match
			byKey[keyOf(k, r)] = r
		}
	}
	seen := map[string]bool{}
	var out []Change
	for _, l := range local {
		key := keyOf(k, l)
		if key == "" {
			return nil, fmt.Errorf("%s: an object has no %q", k.Name, k.Key)
		}
		if seen[key] {
			return nil, fmt.Errorf("%s: %q is defined twice", k.Name, key)
		}
		seen[key] = true
		want := Normalize(k, l)
		r, ok := byKey[key]
		if !ok {
			for _, f := range k.Refs {
				if v, ok := l[f]; ok {
					want[f] = v
				}
			}
			out = append(out, Change{Kind: k, Key: key, Op: OpCreate, Body: want})
			continue
		}
		id := fmt.Sprint(r["id"])
		have := Normalize(k, r)
		op := OpSame
		if k.Once {
			out = append(out, Change{Kind: k, Key: key, ID: id, Op: OpSame})
			continue
		}
		var diff []string
		body := want // PUT replaces, so it sends every managed field
		if !k.Put {
			body = Object{} // PATCH sends only what changed (started experiments refuse design fields)
		}
		for f, v := range want {
			if slices.Contains(k.Frozen, f) {
				continue
			}
			if !reflect.DeepEqual(have[f], v) {
				op = OpUpdate
				diff = append(diff, f)
				body[f] = v
			}
		}
		if !k.Put {
			for _, f := range k.Fixed {
				delete(body, f)
			}
		}
		out = append(out, Change{Kind: k, Key: key, ID: id, Op: op, Body: body, Diff: sortedCopy(diff)})
	}
	if prune && !k.Once && !k.NoPrune { // never delete keys: one may be the key the CLI signs in with
		var gone []string
		for key := range byKey {
			if !seen[key] {
				gone = append(gone, key)
			}
		}
		sort.Strings(gone)
		for _, key := range gone {
			out = append(out, Change{Kind: k, Key: key, ID: fmt.Sprint(byKey[key]["id"]), Op: OpDelete})
		}
	}
	return out, nil
}

// Apply executes changes in order and stops at the first error (already-applied changes stay;
// rerunning converges because matching is by key).
func Apply(ctx context.Context, r Remote, changes []Change) error {
	for _, c := range changes {
		var err error
		switch c.Op {
		case OpCreate:
			err = r.Create(ctx, c.Kind, c.Body)
		case OpUpdate:
			err = r.Update(ctx, c.Kind, c.ID, c.Body)
		case OpDelete:
			err = r.Delete(ctx, c.Kind, c.ID)
		}
		if err != nil {
			return fmt.Errorf("%s %s %q: %w", c.Op, c.Kind.Name, c.Key, err)
		}
	}
	return nil
}

func sortedCopy(s []string) []string { sort.Strings(s); return s }

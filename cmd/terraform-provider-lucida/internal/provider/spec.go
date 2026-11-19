package provider

import (
	"strings"

	"lucida/internal/defsync"
)

// Spec is one Terraform resource. Almost all of it is the defsync.Kind, so a kind added to defsync
// becomes a resource here with no code; only the exceptions below need a line.
type Spec struct {
	TypeSuffix string // lucida_<suffix>
	Kind       defsync.Kind
	Secret     bool                      // the create answer carries a one-time secret (keys)
	Live       func(defsync.Object) bool // nil: every listed object exists
	Summary    string
	// KeyValue is the fixed natural key of a singleton (the settings section, "origins", "revenue"): the
	// key attribute is then computed, not written. Delete only removes the resource from state.
	KeyValue string
}

// Skipped lists defsync kinds the provider deliberately does not expose, with the reason.
// TestEveryKindHasAResource fails for a kind that is neither a resource nor listed here.
var Skipped = map[string]string{
	"webhook-sources": "sources carry a one-time token and signing secret: managed with lucida plan/apply",
}

// extraKinds are API collections Terraform manages that defsync (the CLI) does not carry yet. Moving
// one into defsync.Kinds makes it a CLI kind too and removes it from here.
var extraKinds = []Spec{
	{TypeSuffix: "connector", Summary: "A connector instance (source or destination), keyed by name. `credentials` are sent on create only.",
		Kind: defsync.Kind{Name: "connectors", Key: "name", Fields: []string{"name", "kind", "config", "sync_interval_secs"}, Fixed: []string{"kind"}, Refs: []string{"credentials"}}},
	{TypeSuffix: "destination", Summary: "A destination connector (the same API object as `lucida_connector`; pick a destination `kind`).",
		Kind: defsync.Kind{Name: "connectors", Key: "name", Fields: []string{"name", "kind", "config", "sync_interval_secs"}, Fixed: []string{"kind"}, Refs: []string{"credentials"}}},
	{TypeSuffix: "traffic_exclusion", Summary: "A traffic exclusion rule. The note is its identity; only `enabled` can change in place.",
		Kind: defsync.Kind{Name: "traffic-exclusions", Key: "note", Fields: []string{"note", "kind", "key", "value", "enabled"}, Fixed: []string{"kind", "key", "value"}}},
}

var summaries = map[string]string{
	"actions":          "A named group of event steps (an action), keyed by name.",
	"cohorts":          "A cohort, keyed by name. `kind` is fixed at create.",
	"flags":            "A feature flag, keyed by its flag key. `kind` is fixed at create.",
	"insights":         "A saved insight, keyed by name.",
	"metrics":          "A metric definition, keyed by name.",
	"experiments":      "An experiment, keyed by name. Destroying archives it (the API has no delete).",
	"dashboards":       "A dashboard with its tiles; tiles name their insight.",
	"alerts":           "An insight alert, keyed by name; `insight` names the insight it watches.",
	"keys":             "A project API key. The secret is returned once as the sensitive `secret` attribute; destroying revokes the key.",
	"export-schedules": "A batch export schedule. Where and what it exports is fixed at create; `credentials` are sent on create only.",
	"tracking-plans":   "A tracking plan (event schema and enforcement mode `off|warn|enforce`), keyed by name.",
	"web-goals":        "A web goal, keyed by name. The API cannot edit a goal: changing anything but the name replaces it.",
	"toolbar-origins":  "The origins the toolbar may run on (a singleton: create adopts the existing list, destroy only removes it from state).",
	"revenue":          "The project's revenue mapping (a singleton: create adopts the saved config, destroy only removes it from state).",
	"endpoints":        "A query endpoint, keyed by slug; a change to query or variables creates a new version.",
}

// Specs returns one spec per defsync kind (in defsync order) followed by the extras.
func Specs() []Spec {
	var out []Spec
	for _, k := range defsync.Kinds {
		if _, skip := Skipped[k.Name]; skip {
			continue
		}
		s := Spec{TypeSuffix: singular(k.Name), Kind: k, Summary: summaries[k.Name]}
		if k.Name == "settings" { // one resource per section
			for _, sec := range []string{"ingest", "sdk"} {
				out = append(out, Spec{TypeSuffix: sec + "_settings", Kind: k, KeyValue: sec,
					Summary: "The `" + sec + "` settings section (a singleton: set only the keys you manage; destroy only removes it from state)."})
			}
			continue
		}
		if k.Singleton {
			s.KeyValue = map[string]string{"toolbar-origins": "origins", "revenue": "revenue"}[k.Name]
		}
		if k.Name == "keys" {
			s.Secret = true
			s.Live = func(o defsync.Object) bool { return o["revoked_at"] == nil }
		}
		out = append(out, s)
	}
	return append(out, extraKinds...)
}

// singular turns a kind directory name into a type suffix: "export-schedules" -> "export_schedule".
func singular(name string) string {
	return strings.TrimSuffix(strings.ReplaceAll(name, "-", "_"), "s")
}

// Attribute names the resource adds to the key attribute (named after Kind.Key).
const (
	attrID         = "id"
	attrDefinition = "definition"
	attrSecret     = "secret"
)

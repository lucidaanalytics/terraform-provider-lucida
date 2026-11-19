package provider

import (
	"context"
	"testing"

	"lucida/internal/defsync"
)

// Singletons exist before Terraform: create adopts them with a PUT, destroy never calls the API, and a
// second plan over what was applied has nothing to change.
func TestSingletonResources(t *testing.T) {
	f, api := newFake(t)
	f.rows["settings"] = []map[string]any{
		{"id": "ingest", "hot_distinct_id_per_minute": 6000.0, "defaults": map[string]any{}},
		{"id": "sdk", "consent_mode": "off", "defaults": map[string]any{}},
	}
	f.rows["toolbar"] = []map[string]any{{"id": "origins", "origins": []any{}}}
	f.rows["revenue"] = []map[string]any{{"id": "config", "base_currency": "USD", "minor_units": false}}
	ctx := context.Background()
	for _, tc := range []struct {
		suffix, key string
		def         defsync.Object
	}{
		{"sdk_settings", "sdk", defsync.Object{"consent_mode": "opt_in"}},
		{"toolbar_origin", "origins", defsync.Object{"origins": []any{"https://a.example"}}},
		{"revenue", "revenue", defsync.Object{"minor_units": true}},
	} {
		c := Crud{Spec: spec(t, tc.suffix), API: api}
		e, err := c.Create(ctx, Entity{Def: tc.def})
		if err != nil || e.Key != tc.key || e.ID != c.Spec.KeyValue && tc.suffix != "revenue" {
			t.Fatalf("%s create: %v %+v", tc.suffix, err, e)
		}
		got, found, err := c.Read(ctx, e.ID, &Entity{Def: tc.def})
		if err != nil || !found || canonical(got.Def) != canonical(tc.def) {
			t.Fatalf("%s read after create: %v %v %v", tc.suffix, err, found, got.Def)
		}
		before := f.count("PUT")
		if err := c.Update(ctx, Entity{ID: e.ID, Key: tc.key, Def: tc.def}); err != nil || f.count("PUT") != before {
			t.Fatalf("%s: applying the same definition must not PUT again: %v", tc.suffix, err)
		}
		before = f.count("DELETE")
		if err := c.Delete(ctx, e.ID); err != nil || f.count("DELETE") != before {
			t.Fatalf("%s: destroy must not call the API: %v", tc.suffix, err)
		}
	}
	if f.row("revenue", "id", "config")["base_currency"] != "USD" {
		t.Fatal("revenue PUT must keep the saved fields it does not manage")
	}
}

func TestSingletonSpecs(t *testing.T) {
	for _, s := range []string{"ingest_settings", "sdk_settings", "toolbar_origin", "revenue"} {
		if spec(t, s).KeyValue == "" {
			t.Errorf("%s must be a keyed singleton", s)
		}
	}
	if _, skipped := Skipped["webhook-sources"]; !skipped {
		t.Error("webhook-sources stays out of Terraform (secrets)")
	}
}

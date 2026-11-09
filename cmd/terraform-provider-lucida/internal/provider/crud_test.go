package provider

import (
	"context"
	"strings"
	"testing"

	"lucida/internal/defsync"
)

func spec(t *testing.T, suffix string) Spec {
	t.Helper()
	for _, s := range Specs() {
		if s.TypeSuffix == suffix {
			return s
		}
	}
	t.Fatalf("no spec %s", suffix)
	return Spec{}
}

func flagEntity() Entity {
	return Entity{Key: "new-checkout", Def: defsync.Object{"name": "New checkout", "enabled": true, "filters": map[string]any{"rollout": 50}}}
}

func TestCreateReadUpdateDelete(t *testing.T) {
	f, api := newFake(t)
	c := Crud{Spec: spec(t, "flag"), API: api}
	ctx := context.Background()

	// Pagination: three other flags make the list span two pages.
	for _, k := range []string{"a", "b", "c"} {
		if _, err := c.Create(ctx, Entity{Key: k, Def: defsync.Object{"name": k}}); err != nil {
			t.Fatal(err)
		}
	}
	e, err := c.Create(ctx, flagEntity())
	if err != nil || e.ID == "" {
		t.Fatalf("create: %v %+v", err, e)
	}
	if _, err := c.Create(ctx, flagEntity()); err == nil || !strings.Contains(err.Error(), "import") {
		t.Fatalf("a second create must point at import: %v", err)
	}

	got, found, err := c.Read(ctx, e.ID, &Entity{Def: flagEntity().Def})
	if err != nil || !found {
		t.Fatal(err, found)
	}
	if _, ok := got.Def["tags"]; ok || got.Def["enabled"] != true {
		t.Fatalf("only configured fields are read, server defaults (tags) are not drift: %v", got.Def)
	}

	// Drift: someone flips the flag in the UI.
	f.row("flags", "key", "new-checkout")["enabled"] = false
	got, _, _ = c.Read(ctx, e.ID, &Entity{Def: flagEntity().Def})
	if got.Def["enabled"] != false {
		t.Fatalf("drift must show: %v", got.Def)
	}

	// Update sends only the changed field; the key is never PATCHed.
	e.Def = defsync.Object{"name": "New checkout", "enabled": true, "filters": map[string]any{"rollout": 75}}
	if err := c.Update(ctx, e); err != nil {
		t.Fatal(err)
	}
	row := f.row("flags", "key", "new-checkout")
	if row["enabled"] != true || row["filters"].(map[string]any)["rollout"] != 75.0 {
		t.Fatalf("update not applied: %v", row)
	}
	before := f.count("PATCH")
	if err := c.Update(ctx, e); err != nil || f.count("PATCH") != before {
		t.Fatalf("an unchanged definition must not PATCH: %v", err)
	}

	// Import reads everything the kind manages.
	imp, found, err := c.Read(ctx, e.ID, nil)
	if err != nil || !found || imp.Key != "new-checkout" || imp.Def["name"] != "New checkout" {
		t.Fatalf("import: %v %v %+v", err, found, imp)
	}
	if _, ok := imp.Def["created_at"]; ok {
		t.Fatal("unmanaged fields must not be imported")
	}

	if err := c.Delete(ctx, e.ID); err != nil {
		t.Fatal(err)
	}
	if _, found, _ := c.Read(ctx, e.ID, nil); found {
		t.Fatal("deleted flag still found")
	}
	if err := c.Delete(ctx, e.ID); err != nil {
		t.Fatalf("deleting a gone object is success: %v", err)
	}
}

func TestKeySecretIsReturnedOnceAndNeverRead(t *testing.T) {
	f, api := newFake(t)
	c := Crud{Spec: spec(t, "key"), API: api}
	ctx := context.Background()
	e, err := c.Create(ctx, Entity{Key: "ci", Def: defsync.Object{"kind": "project", "scopes": []any{"read"}}})
	if err != nil || !strings.HasPrefix(e.Secret, "lcsk_secret_") {
		t.Fatalf("secret not captured: %v", err)
	}
	got, _, _ := c.Read(ctx, e.ID, &Entity{Def: e.Def, Secret: e.Secret})
	if got.Secret != e.Secret {
		t.Fatal("the stored secret must be carried through reads")
	}
	if imp, _, _ := c.Read(ctx, e.ID, nil); imp.Secret != "" {
		t.Fatal("an import has no secret")
	}
	if err := c.Delete(ctx, e.ID); err != nil {
		t.Fatal(err)
	}
	if _, found, _ := c.Read(ctx, e.ID, nil); found {
		t.Fatal("a revoked key is gone")
	}
	if f.row("keys", "name", "ci")["revoked_at"] == nil {
		t.Fatal("destroy revokes")
	}
}

func TestErrorsAreProblemDetailsAndRetried(t *testing.T) {
	f, api := newFake(t)
	c := Crud{Spec: spec(t, "cohort"), API: api}
	f.fail["GET cohorts"] = 503 // one transient failure is retried
	if _, err := c.Create(context.Background(), Entity{Key: "vip", Def: defsync.Object{"name": "x", "kind": "static"}}); err != nil {
		t.Fatalf("5xx must be retried: %v", err)
	}
	api.APIKey = "wrong"
	_, err := c.Create(context.Background(), Entity{Key: "vip2", Def: defsync.Object{}})
	if err == nil || !strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "wrong") {
		t.Fatalf("auth failure must report the status and never the key: %v", err)
	}
}

func TestReplaceRules(t *testing.T) {
	flags := spec(t, "flag").Kind
	if !needsReplace(spec(t, "cohort").Kind, defsync.Object{"kind": "static"}, defsync.Object{"kind": "dynamic"}) {
		t.Fatal("a fixed field cannot change in place")
	}
	if needsReplace(flags, defsync.Object{"kind": "boolean", "enabled": true}, defsync.Object{"kind": "boolean", "enabled": false}) {
		t.Fatal("enabled changes in place")
	}
	if !needsReplace(spec(t, "key").Kind, defsync.Object{"scopes": []any{"a"}}, defsync.Object{"scopes": []any{"b"}}) {
		t.Fatal("keys are create-only")
	}
}

func TestValidateDefinition(t *testing.T) {
	flags := spec(t, "flag").Kind
	for _, bad := range []string{`[1]`, `nope`, `{"key":"x"}`, `{"colour":"red"}`} {
		if validateDefinition(flags, bad) == nil {
			t.Errorf("%s must be rejected", bad)
		}
	}
	if err := validateDefinition(flags, `{"name":"n","enabled":true}`); err != nil {
		t.Fatal(err)
	}
	if validateDefinition(spec(t, "export_schedule").Kind, `{"credentials":{}}`) == nil {
		t.Fatal("credentials is its own sensitive attribute")
	}
}

package provider

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"lucida/internal/defsync"
)

var protoV6 = map[string]func() (tfprotov6.ProviderServer, error){
	"lucida": providerserver.NewProtocol6WithError(New("test")()),
}

// A kind added to defsync must become a resource, or be skipped here with a reason.
func TestEveryKindHasAResource(t *testing.T) {
	have := map[string]bool{}
	for _, s := range Specs() {
		have[s.Kind.Name] = true
	}
	for _, k := range defsync.Kinds {
		if _, skipped := Skipped[k.Name]; !have[k.Name] && !skipped {
			t.Errorf("defsync kind %q has no Terraform resource: add a Spec or list it in Skipped with a reason", k.Name)
		}
		if why := Skipped[k.Name]; have[k.Name] && why != "" {
			t.Errorf("kind %q is both a resource and skipped (%s)", k.Name, why)
		}
	}
	for name, why := range Skipped {
		if _, ok := defsync.KindByName(name); !ok || strings.TrimSpace(why) == "" {
			t.Errorf("Skipped[%q] must name a real kind and carry a reason", name)
		}
	}
	// The kinds the product promises, regardless of what defsync grows.
	for _, want := range []string{"flag", "experiment", "cohort", "action", "key", "alert", "export_schedule", "connector", "destination", "traffic_exclusion"} {
		found := false
		for _, s := range Specs() {
			found = found || s.TypeSuffix == want
		}
		if !found {
			t.Errorf("missing lucida_%s", want)
		}
	}
}

func TestSpecsAreWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range Specs() {
		if seen[s.TypeSuffix] {
			t.Errorf("duplicate type %s", s.TypeSuffix)
		}
		seen[s.TypeSuffix] = true
		if s.Summary == "" {
			t.Errorf("%s has no summary: add one to summaries", s.TypeSuffix)
		}
		if _, ok := examples[s.TypeSuffix]; !ok {
			t.Errorf("%s has no example in docs.go", s.TypeSuffix)
		}
		if !s.Kind.Open && !strings.Contains(strings.Join(s.Kind.Fields, ","), s.Kind.Key) {
			t.Errorf("%s: key %q is not one of its fields", s.TypeSuffix, s.Kind.Key)
		}
	}
}

// The provider server must accept every generated schema (valid attribute names, no collisions).
func TestSchemasValidate(t *testing.T) {
	srv, err := protoV6["lucida"]()
	if err != nil {
		t.Fatal(err)
	}
	resp, err := srv.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range resp.Diagnostics {
		t.Errorf("%s: %s", d.Summary, d.Detail)
	}
	if len(resp.ResourceSchemas) != len(Specs()) || len(resp.DataSourceSchemas) != 2 {
		t.Fatalf("schemas: %d resources, %d data sources", len(resp.ResourceSchemas), len(resp.DataSourceSchemas))
	}
	var secrets int
	for _, a := range resp.ResourceSchemas["lucida_key"].Block.Attributes {
		if a.Name == "secret" && a.Sensitive {
			secrets++
		}
	}
	if secrets != 1 {
		t.Fatal("lucida_key.secret must be a sensitive attribute")
	}
}

// docs/ and examples/ are generated: run `UPDATE_DOCS=1 go test ./cmd/terraform-provider-lucida/...` after a change.
func TestGeneratedFilesAreCurrent(t *testing.T) {
	root := filepath.Join("..", "..") // the monorepo keeps docs/ and examples/ beside main.go
	if _, err := os.Stat(filepath.Join(root, "docs")); err != nil {
		root = filepath.Join("..", "..", "..", "..") // the public mirror keeps them at the repo root
	}
	files := Files()
	for _, rel := range sortedKeys(files) {
		p := filepath.Join(root, rel)
		if os.Getenv("UPDATE_DOCS") != "" {
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(files[rel]), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		got, err := os.ReadFile(p)
		if err != nil || string(got) != files[rel] {
			t.Errorf("%s is stale or missing: run UPDATE_DOCS=1 go test ./cmd/terraform-provider-lucida/...", rel)
		}
	}
}

package defsync

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

type memRemote struct {
	objs map[string][]Object
	log  []string
}

func (m *memRemote) List(_ context.Context, k Kind) ([]Object, error) { return m.objs[k.Name], nil }
func (m *memRemote) Create(_ context.Context, k Kind, o Object) error {
	m.log = append(m.log, "create "+k.Name+" "+keyOf(k, o))
	return nil
}
func (m *memRemote) Update(_ context.Context, k Kind, id string, o Object) error {
	if _, has := o["key"]; has && k.Name == "flags" {
		m.log = append(m.log, "BAD: key sent on update")
	}
	m.log = append(m.log, "update "+k.Name+" "+id)
	return nil
}
func (m *memRemote) Delete(_ context.Context, k Kind, id string) error {
	m.log = append(m.log, "delete "+k.Name+" "+id)
	return nil
}

func TestDiffCreateUpdateSamePrune(t *testing.T) {
	k, _ := KindByName("flags")
	remote := []Object{
		{"id": "1", "key": "same", "enabled": true, "version": 3.0, "tags": []any{"a"}},
		{"id": "2", "key": "changed", "enabled": false},
		{"id": "3", "key": "orphan"},
	}
	local := []Object{
		{"key": "same", "enabled": true, "tags": []any{"a"}},
		{"key": "changed", "enabled": true},
		{"key": "new", "filters": map[string]any{"groups": []any{map[string]any{"rollout": 50}}}},
	}
	ch, err := Diff(k, local, remote, true)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]Op{"same": OpSame, "changed": OpUpdate, "new": OpCreate, "orphan": OpDelete}
	if len(ch) != 4 {
		t.Fatalf("%v", ch)
	}
	for _, c := range ch {
		if want[c.Key] != c.Op {
			t.Errorf("%s: %s, want %s", c.Key, c.Op, want[c.Key])
		}
	}
	noPrune, _ := Diff(k, local, remote, false)
	if len(noPrune) != 3 {
		t.Fatal("without prune nothing is deleted")
	}
	r := &memRemote{}
	if err := Apply(context.Background(), r, ch); err != nil {
		t.Fatal(err)
	}
	if len(r.log) != 3 || r.log[0] != "update flags 2" || r.log[1] != "create flags new" || r.log[2] != "delete flags 3" {
		t.Fatalf("%v", r.log)
	}
}

func TestDiffRejectsMissingAndDuplicateKeys(t *testing.T) {
	k, _ := KindByName("insights")
	if _, err := Diff(k, []Object{{"description": "x"}}, nil, false); err == nil {
		t.Fatal("missing key must error")
	}
	if _, err := Diff(k, []Object{{"name": "a"}, {"name": "a"}}, nil, false); err == nil {
		t.Fatal("duplicate key must error")
	}
}

func TestWriteLoadRoundTrip(t *testing.T) {
	root := t.TempDir()
	k, _ := KindByName("dashboards")
	objs := []Object{
		{"id": "d1", "name": "Growth KPIs", "pinned": true, "created_at": "x",
			"tiles": []any{map[string]any{"insight": "Signups", "position": 0}}},
		{"id": "d2", "name": "growth kpis"}, // same file stem
	}
	paths, err := Write(root, k, objs)
	if err != nil || len(paths) != 2 {
		t.Fatalf("%v %v", paths, err)
	}
	_ = os.WriteFile(filepath.Join(root, "dashboards", "README.md"), []byte("ignored"), 0o644)
	got, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, has := got["flags"]; has {
		t.Fatal("kinds without a directory must not load (prune must not touch them)")
	}
	if len(got["dashboards"]) != 2 {
		t.Fatalf("%v", got)
	}
	ch, _ := Diff(k, got["dashboards"], objs, true)
	for _, c := range ch {
		if c.Op != OpSame {
			t.Fatalf("round trip must be unchanged, got %v %s", c.Key, c.Op)
		}
	}
	if _, has := got["dashboards"][0]["created_at"]; has {
		t.Fatal("unmanaged fields must not be written")
	}
	if FileName("  Checkout / Funnel!! ") != "checkout-funnel" || FileName("***") != "unnamed" {
		t.Fatal(FileName("  Checkout / Funnel!! "))
	}
}

func TestPatchSendsOnlyChangedFields(t *testing.T) {
	k, _ := KindByName("experiments")
	remote := []Object{{"id": "e1", "name": "Pricing", "kind": "ab", "variants": []any{"a", "b"}, "hypothesis": "x"}}
	local := []Object{{"name": "Pricing", "kind": "ab", "variants": []any{"a", "b"}, "hypothesis": "y"}}
	ch, err := Diff(k, local, remote, false)
	if err != nil || len(ch) != 1 || ch[0].Op != OpUpdate {
		t.Fatalf("%v %v", ch, err)
	}
	if len(ch[0].Body) != 1 || ch[0].Body["hypothesis"] != "y" {
		t.Fatalf("a running experiment refuses unchanged design fields, so only the change is sent: %v", ch[0].Body)
	}
	m, _ := KindByName("metrics")
	ch, _ = Diff(m, []Object{{"name": "rev", "unit": "usd", "kind": "sum"}}, []Object{{"id": "m1", "name": "rev", "unit": "eur", "kind": "sum"}}, false)
	if len(ch[0].Body) != 3 {
		t.Fatalf("PUT kinds still send every managed field: %v", ch[0].Body)
	}
	if _, ok := KindByName("alerts"); !ok {
		t.Fatal("alerts kind missing")
	}
}

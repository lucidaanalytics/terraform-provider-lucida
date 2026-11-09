package apiremote

import (
	"strings"
	"testing"

	"lucida/internal/defsync"
)

func TestResolveExportCredsReadsEnvAndRefusesValues(t *testing.T) {
	t.Setenv("LC_TEST_SNOW_KEY", "pem-data")
	o := defsync.Object{"name": "n", "credentials": map[string]any{"warehouse": map[string]any{"private_key": "env:LC_TEST_SNOW_KEY"}}}
	got, err := ResolveExportCreds(o)
	if err != nil {
		t.Fatal(err)
	}
	if got["credentials"].(map[string]any)["warehouse"].(map[string]string)["private_key"] != "pem-data" {
		t.Fatalf("not resolved: %v", got)
	}
	if o["credentials"].(map[string]any)["warehouse"].(map[string]any)["private_key"] != "env:LC_TEST_SNOW_KEY" {
		t.Fatal("the input was mutated")
	}
	lit := defsync.Object{"name": "n", "credentials": map[string]any{"dest": map[string]any{"dsn": "postgres://u:p@h/db"}}}
	if _, err := ResolveExportCreds(lit); err == nil || strings.Contains(err.Error(), "postgres://") {
		t.Fatalf("a literal credential must be refused without echoing it: %v", err)
	}
	if _, err := ResolveExportCreds(defsync.Object{"name": "n", "credentials": map[string]any{"query_key": "env:LC_TEST_UNSET_X"}}); err == nil {
		t.Fatal("an unset variable must fail")
	}
}

func TestExportScheduleKindFreezesDestination(t *testing.T) {
	k, ok := defsync.KindByName("export-schedules")
	if !ok {
		t.Fatal("kind missing")
	}
	local := []defsync.Object{{"name": "wh", "stream": "events", "destination": map[string]any{"kind": "s3"}, "interval_secs": 3600, "credentials": map[string]any{"dest": map[string]any{"k": "env:X"}}}}
	remote := []defsync.Object{{"id": "s1", "name": "wh", "stream": "events", "destination": map[string]any{"kind": "gcs"}, "interval_secs": 7200}}
	ch, err := defsync.Diff(k, local, remote, false)
	if err != nil || len(ch) != 1 || ch[0].Op != defsync.OpUpdate || len(ch[0].Diff) != 1 || ch[0].Diff[0] != "interval_secs" {
		t.Fatalf("only the interval may change in place: %v %v", ch, err)
	}
	ch, _ = defsync.Diff(k, local, nil, false)
	if ch[0].Op != defsync.OpCreate || ch[0].Body["credentials"] == nil {
		t.Fatalf("create carries the credential references: %v", ch)
	}
}

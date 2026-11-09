package apiremote

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"lucida/internal/defsync"
)

// ResolveExportCreds turns the credentials of an export-schedules file entry into the API shape. A
// file never holds a secret: every value must be `env:NAME`, read from the environment at apply time
// (an unset variable is an error naming the variable, never its value).
func ResolveExportCreds(o defsync.Object) (defsync.Object, error) {
	raw, ok := o["credentials"]
	if !ok {
		return o, nil
	}
	creds, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("export-schedules/%v: credentials must be a map", o["name"])
	}
	out := map[string]any{}
	for group, v := range creds {
		if s, ok := v.(string); ok { // query_key
			val, err := envRef(o, group, s)
			if err != nil {
				return nil, err
			}
			out[group] = val
			continue
		}
		m, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("export-schedules/%v: credentials.%s must be an env:NAME reference or a map of them", o["name"], group)
		}
		res := map[string]string{}
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			s, _ := m[k].(string)
			val, err := envRef(o, group+"."+k, s)
			if err != nil {
				return nil, err
			}
			res[k] = val
		}
		out[group] = res
	}
	cp := defsync.Object{}
	for k, v := range o {
		cp[k] = v
	}
	cp["credentials"] = out
	return cp, nil
}

func envRef(o defsync.Object, field, ref string) (string, error) {
	name, ok := strings.CutPrefix(ref, "env:")
	if !ok || name == "" {
		return "", fmt.Errorf("export-schedules/%v: credentials.%s must be an env:NAME reference, not a value", o["name"], field)
	}
	v := os.Getenv(name)
	if v == "" {
		return "", fmt.Errorf("export-schedules/%v: credentials.%s: environment variable %s is not set", o["name"], field, name)
	}
	return v, nil
}

package defsync

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load reads <root>/<kind>/*.yaml|*.yml|*.json. Only kinds with a directory are returned, so prune
// never touches a kind the repository does not manage.
func Load(root string) (map[string][]Object, error) {
	out := map[string][]Object{}
	for _, k := range Kinds {
		dir := filepath.Join(root, k.Name)
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		out[k.Name] = []Object{}
		for _, e := range entries {
			ext := filepath.Ext(e.Name())
			if e.IsDir() || (ext != ".yaml" && ext != ".yml" && ext != ".json") {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				return nil, err
			}
			var o Object
			if err := yaml.Unmarshal(raw, &o); err != nil { // YAML is a superset of JSON
				return nil, fmt.Errorf("%s/%s: %w", k.Name, e.Name(), err)
			}
			out[k.Name] = append(out[k.Name], o)
		}
	}
	return out, nil
}

// Write replaces <root>/<kind>/ with one YAML file per object (managed fields only).
func Write(root string, k Kind, objs []Object) ([]string, error) {
	dir := filepath.Join(root, k.Name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	old, _ := os.ReadDir(dir)
	for _, e := range old { // pull mirrors the remote: files of deleted objects go too
		if ext := filepath.Ext(e.Name()); !e.IsDir() && (ext == ".yaml" || ext == ".yml" || ext == ".json") {
			if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
				return nil, err
			}
		}
	}
	sort.Slice(objs, func(i, j int) bool { return keyOf(k, objs[i]) < keyOf(k, objs[j]) })
	var paths []string
	used := map[string]bool{}
	for _, o := range objs {
		name := FileName(keyOf(k, o))
		for i := 2; used[name]; i++ { // two keys can slug the same ("A b" / "a-b")
			name = fmt.Sprintf("%s-%d", FileName(keyOf(k, o)), i)
		}
		used[name] = true
		raw, err := yaml.Marshal(Normalize(k, o))
		if err != nil {
			return nil, err
		}
		p := filepath.Join(dir, name+".yaml")
		if err := os.WriteFile(p, raw, 0o644); err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}
	return paths, nil
}

// FileName turns a key into a safe file stem.
func FileName(key string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(key) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	s := strings.TrimRight(b.String(), "-")
	if s == "" {
		return "unnamed"
	}
	return s
}

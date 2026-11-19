package provider

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"lucida/internal/defsync"
)

// parseObject decodes a JSON object attribute.
func parseObject(s string) (defsync.Object, error) {
	var o defsync.Object
	if err := json.Unmarshal([]byte(s), &o); err != nil || o == nil {
		return nil, fmt.Errorf("must be a JSON object (use jsonencode({...}))")
	}
	return o, nil
}

// canonical renders an object with sorted keys, the form stored in state.
func canonical(o defsync.Object) string {
	b, _ := json.Marshal(o) // decoded JSON always marshals
	return string(b)
}

// sameJSON reports whether two JSON documents are equal after decoding (key order and number
// spelling do not matter).
func sameJSON(a, b string) bool {
	var x, y any
	if json.Unmarshal([]byte(a), &x) != nil || json.Unmarshal([]byte(b), &y) != nil {
		return a == b
	}
	return reflect.DeepEqual(x, y)
}

// validateDefinition checks a definition against the kind at plan time: it must be an object of
// managed fields only, without the key (its own attribute) or the credential references (theirs).
func validateDefinition(k defsync.Kind, s string) error {
	o, err := parseObject(s)
	if err != nil {
		return err
	}
	for f := range o {
		switch {
		case f == k.Key:
			return fmt.Errorf("%q is the resource's own attribute, remove it from the definition", f)
		case slices.Contains(k.Refs, f):
			return fmt.Errorf("%q is a separate sensitive attribute, remove it from the definition", f)
		case !k.Open && !slices.Contains(k.Fields, f):
			return fmt.Errorf("unknown field %q; %s manages %s", f, k.Name, strings.Join(definable(k), ", "))
		}
	}
	return nil
}

// fieldList describes the fields a definition may carry, for schema text.
func fieldList(k defsync.Kind) string {
	if k.Open {
		return "any key of the object (the API's own names)"
	}
	return "`" + strings.Join(definable(k), "`, `") + "`"
}

// definable lists the fields a definition may carry.
func definable(k defsync.Kind) []string {
	var out []string
	for _, f := range k.Fields {
		if f != k.Key {
			out = append(out, f)
		}
	}
	return out
}

// needsReplace reports whether a definition change cannot be made in place: a create-only kind
// (keys) or a changed fixed/frozen field.
func needsReplace(k defsync.Kind, old, planned defsync.Object) bool {
	if sameJSON(canonical(old), canonical(planned)) {
		return false
	}
	if k.Once {
		return true
	}
	for _, f := range slices.Concat(k.Fixed, k.Frozen) {
		if !sameJSON(canonical(defsync.Object{"v": old[f]}), canonical(defsync.Object{"v": planned[f]})) {
			return true
		}
	}
	return false
}

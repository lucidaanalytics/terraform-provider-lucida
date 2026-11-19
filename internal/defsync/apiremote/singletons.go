package apiremote

import (
	"context"
	"fmt"
	"os"
	"strings"

	"lucida/internal/defsync"
)

var settingsSections = []string{"ingest", "sdk"}

const toolbarRoute = "/toolbar/origins"

// listSingletons reads the fixed objects of the settings and toolbar-origins kinds. A settings object is
// {section, ...effective values, defaults}; the id is the section so Update can address it.
func (r *Remote) listSingletons(ctx context.Context, k defsync.Kind) ([]defsync.Object, error) {
	if k.Name == "revenue" {
		var o defsync.Object
		if err := r.api.Do(ctx, "GET", "/"+k.Path(), nil, nil, &o); err != nil {
			return nil, err
		}
		o["id"], o["name"] = "revenue", "revenue"
		return []defsync.Object{o}, nil
	}
	if k.Name == "toolbar-origins" {
		var o defsync.Object
		if err := r.api.Do(ctx, "GET", toolbarRoute, nil, nil, &o); err != nil {
			return nil, err
		}
		return []defsync.Object{{"id": "origins", "name": "origins", "origins": o["origins"]}}, nil
	}
	var out []defsync.Object
	for _, sec := range settingsSections {
		var o defsync.Object
		if err := r.api.Do(ctx, "GET", "/settings/"+sec, nil, nil, &o); err != nil {
			return nil, err
		}
		o["section"], o["id"] = sec, sec
		out = append(out, o)
	}
	return out, nil
}

// putSingleton PUTs only the written fields. Create passes the whole object, Update wraps the changed
// fields in "body" with the section as id.
func (r *Remote) putSingleton(ctx context.Context, k defsync.Kind, o defsync.Object) error {
	body, _ := o["body"].(defsync.Object)
	if body == nil {
		body = o
	}
	out := defsync.Object{}
	for f, v := range body {
		if f != "section" && f != "name" && f != "id" && f != "defaults" {
			out[f] = v
		}
	}
	route := toolbarRoute
	if k.Name == "revenue" { // PUT replaces the whole config: overlay the written fields on what is saved
		route = "/" + k.Path()
		var cur defsync.Object
		if err := r.api.Do(ctx, "GET", route, nil, nil, &cur); err != nil {
			return err
		}
		for f, v := range out {
			cur[f] = v
		}
		out = cur
		for _, f := range []string{"fx_feed_error", "fx_feed_synced_at"} {
			delete(out, f)
		}
	}
	if k.Name == "settings" {
		sec := fmt.Sprint(o["section"])
		known := false
		for _, s := range settingsSections {
			known = known || s == sec
		}
		if !known {
			return fmt.Errorf("settings: unknown section %q (ingest or sdk)", sec)
		}
		route = "/settings/" + sec
	}
	return r.api.Do(ctx, "PUT", route, nil, out, nil)
}

// resolveHookSecret moves auth_secret (an env:NAME reference, never a value) into auth.secret for the
// create call. An unset variable is an error naming the variable, never a value.
func resolveHookSecret(o defsync.Object) (defsync.Object, error) {
	ref, has := o["auth_secret"]
	cp := defsync.Object{}
	for k, v := range o {
		if k != "auth_secret" {
			cp[k] = v
		}
	}
	if !has {
		return cp, nil
	}
	name, ok := strings.CutPrefix(fmt.Sprint(ref), "env:")
	if !ok || name == "" {
		return nil, fmt.Errorf("webhook-sources/%v: auth_secret must be an env:NAME reference, not a value", o["name"])
	}
	val := os.Getenv(name)
	if val == "" {
		return nil, fmt.Errorf("webhook-sources/%v: environment variable %s is not set", o["name"], name)
	}
	auth, _ := cp["auth"].(map[string]any)
	next := map[string]any{"secret": val}
	for k, v := range auth {
		next[k] = v
	}
	cp["auth"] = next
	return cp, nil
}

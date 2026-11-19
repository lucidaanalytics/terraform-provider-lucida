package provider

import (
	"context"
	"fmt"
	"slices"

	"lucida/internal/defsync"
	"lucida/internal/defsync/apiremote"
)

// Entity is the provider's view of one managed object, independent of the framework types.
type Entity struct {
	ID     string
	Key    string
	Def    defsync.Object // managed fields except the key and the credential references
	Refs   defsync.Object // credential references, sent on create only and never read back
	Secret string         // one-time secret (keys)
}

// Crud runs one Spec against the API through the same adapter and Diff the CLI uses.
type Crud struct {
	Spec Spec
	API  apiremote.API
}

func (c Crud) remote(sink apiremote.KeySink) *apiremote.Remote {
	return apiremote.New(c.API, sink).LiteralRefs()
}

// local is the defsync object an Entity stands for.
func (c Crud) local(e Entity) defsync.Object {
	o := defsync.Object{}
	for k, v := range e.Def {
		o[k] = v
	}
	for k, v := range e.Refs {
		o[k] = v
	}
	o[c.Spec.Kind.Key] = e.Key
	return o
}

func (c Crud) list(ctx context.Context, r *apiremote.Remote) ([]defsync.Object, error) {
	rows, err := r.List(ctx, c.Spec.Kind)
	if err != nil {
		return nil, err
	}
	if c.Spec.Live != nil {
		rows = slices.DeleteFunc(rows, func(o defsync.Object) bool { return !c.Spec.Live(o) })
	}
	return rows, nil
}

func idOf(o defsync.Object) string { return fmt.Sprint(o["id"]) }

// Create makes the object and returns it with its id. An existing object with the same natural key
// is an error (import it), never silently adopted.
func (c Crud) Create(ctx context.Context, e Entity) (Entity, error) {
	if c.Spec.Kind.Singleton {
		return c.adopt(ctx, e)
	}
	var secret string
	r := c.remote(func(_, s string) error { secret = s; return nil })
	before, err := c.list(ctx, r)
	if err != nil {
		return e, err
	}
	changes, err := defsync.Diff(c.Spec.Kind, []defsync.Object{c.local(e)}, before, false)
	if err != nil {
		return e, err
	}
	if changes[0].Op != defsync.OpCreate {
		return e, fmt.Errorf("%s %q already exists (id %s): import it with `terraform import` instead of creating it", c.Spec.Kind.Name, e.Key, changes[0].ID)
	}
	if err := r.Create(ctx, c.Spec.Kind, changes[0].Body); err != nil {
		return e, err
	}
	after, err := c.list(ctx, r)
	if err != nil {
		return e, err
	}
	known := map[string]bool{}
	for _, o := range before {
		known[idOf(o)] = true
	}
	for _, o := range after {
		if !known[idOf(o)] && o[c.Spec.Kind.Key] == e.Key {
			e.ID, e.Secret = idOf(o), secret
			return e, nil
		}
	}
	return e, fmt.Errorf("%s %q was created but could not be found afterwards", c.Spec.Kind.Name, e.Key)
}

// adopt "creates" a singleton: it always exists, so create is an update of the written fields.
func (c Crud) adopt(ctx context.Context, e Entity) (Entity, error) {
	e.Key = c.Spec.KeyValue
	e.ID = c.Spec.KeyValue
	return e, c.Update(ctx, e)
}

// Read fetches the object by id. found is false when it is gone. prior is the last known definition
// (nil on import): only fields the configuration manages are reported, and a changed or removed
// one shows as drift. Fields the API never returns (frozen ones) keep their prior value.
func (c Crud) Read(ctx context.Context, id string, prior *Entity) (e Entity, found bool, err error) {
	rows, err := c.list(ctx, c.remote(nil))
	if err != nil {
		return e, false, err
	}
	i := slices.IndexFunc(rows, func(o defsync.Object) bool { return idOf(o) == id })
	if i < 0 {
		return e, false, nil
	}
	k := c.Spec.Kind
	have := defsync.Normalize(k, rows[i])
	e = Entity{ID: id, Def: defsync.Object{}}
	e.Key, _ = have[k.Key].(string)
	if prior == nil {
		for f, v := range have {
			if f != k.Key {
				e.Def[f] = v
			}
		}
		return e, true, nil
	}
	e.Refs, e.Secret = prior.Refs, prior.Secret
	for f, old := range prior.Def {
		if v, ok := have[f]; ok {
			e.Def[f] = v
		} else if slices.Contains(k.Frozen, f) {
			e.Def[f] = old
		}
	}
	return e, true, nil
}

// Update applies the fields that differ, using the CLI's Diff so fixed, frozen and PUT/PATCH rules agree.
func (c Crud) Update(ctx context.Context, e Entity) error {
	r := c.remote(nil)
	rows, err := c.list(ctx, r)
	if err != nil {
		return err
	}
	i := slices.IndexFunc(rows, func(o defsync.Object) bool { return idOf(o) == e.ID })
	if i < 0 {
		return &APIError{Status: 404, Detail: c.Spec.Kind.Name + " " + e.ID + " no longer exists"}
	}
	changes, err := defsync.Diff(c.Spec.Kind, []defsync.Object{c.local(e)}, rows[i:i+1], false)
	if err != nil {
		return err
	}
	if changes[0].Op != defsync.OpUpdate {
		return nil
	}
	return r.Update(ctx, c.Spec.Kind, e.ID, changes[0].Body)
}

// Delete removes the object; one that is already gone is success.
func (c Crud) Delete(ctx context.Context, id string) error {
	if c.Spec.Kind.Singleton {
		return nil // fixed object: destroy forgets it, the API value stays
	}
	err := c.remote(nil).Delete(ctx, c.Spec.Kind, id)
	if IsNotFound(err) {
		return nil
	}
	return err
}

package apiremote

import (
	"context"
	"errors"
	"fmt"

	"lucida/internal/defsync"
)

// createKey makes a project key. The secret comes back once and goes only to the KeySink, never
// to the definitions, the plan or an error.
func (r *Remote) createKey(ctx context.Context, o defsync.Object) error {
	var created struct {
		Secret string `json:"secret"`
	}
	if err := r.api.Do(ctx, "POST", "/keys", nil, o, &created); err != nil {
		return err
	}
	if created.Secret == "" {
		return errors.New("the key was created but the answer carried no secret")
	}
	if r.onKey == nil {
		return fmt.Errorf("key %v was created but nothing was set up to receive its secret", o["name"])
	}
	return r.onKey(fmt.Sprint(o["name"]), created.Secret)
}

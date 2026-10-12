// Package ownedrecords runs the SDK-owned Neutron record calls for
// collections that Gophercloud v2.15.0 does not bind. It reuses the
// neutronfind Source engine (descriptors, local Body filters, *_links paging,
// GET-then-list find and guarded mutations) so each leaf package only
// declares its pinned Resource profile.
package ownedrecords

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Spec describes one owned collection. Resource names the operation error
// context. Bound is the positional proxy argument name (for example
// "flavor") that update **attrs cannot repeat.
type Spec struct {
	Resource string
	Bound    string
	Profile  neutronfind.Profile
}

// Record is the Source view plus the actual HTTP receipt.
type Record = neutronfind.Record

// Target binds a Spec to an API's selected client. Client is nil for a nil API.
type Target struct {
	Spec   *Spec
	Client func() *gophercloud.ServiceClient
}

// Invalid reports an SDK-side input error before or instead of HTTP.
func Invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

// Wrap adds the operation and resource context.
func (t Target) Wrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, t.Spec.Resource, cloudread.ContextError(ctx, err))
}

// Engine captures the selected client for one call. The guard fails when the
// API's client is replaced while the call runs.
func (t Target) Engine(ctx context.Context, profile neutronfind.Profile) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if t.Client == nil {
		return nil, Invalid("configured %s API is required", t.Spec.Resource)
	}
	selected := t.Client()
	return neutronfind.New(ctx, selected, profile, func(context.Context) error {
		if t.Client() != selected {
			return Invalid("selected %s client changed", t.Spec.Resource)
		}
		return nil
	})
}

// ValidateID accepts one literal URL segment.
func ValidateID(label, id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." || !utf8.ValidString(id) {
		return Invalid("nonblank UTF-8 literal %s is required", label)
	}
	for _, character := range id {
		if unicode.IsControl(character) {
			return Invalid("%s contains controls", label)
		}
	}
	return nil
}

// Apply runs options in order on owned copies; a nil option fails before HTTP.
func Apply[T any, O ~func(*T) error](ctx context.Context, engine *neutronfind.Engine, initial T, clone func(T) T, options []O) (T, error) {
	policy := clone(initial)
	for _, apply := range options {
		if err := engine.Check(ctx); err != nil {
			return policy, err
		}
		if apply == nil {
			return policy, Invalid("nil option")
		}
		next := clone(policy)
		err := apply(&next)
		if err = errors.Join(err, engine.Check(ctx)); err != nil {
			return policy, err
		}
		policy = clone(next)
	}
	return policy, nil
}

// Text returns a string descriptor value, or "" when it is not a string.
func Text(r *Record, key string) string {
	if value := Field[string](r, key); value != nil {
		return *value
	}
	return ""
}

// Field decodes one Source descriptor value. It is nil when the value is
// missing, null or of another JSON type; Resource keeps the raw value.
func Field[T any](r *Record, key string) *T {
	if r == nil || r.Resource == nil {
		return nil
	}
	raw := bytes.TrimSpace(r.Resource.Body[key])
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil
	}
	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil
	}
	return &value
}

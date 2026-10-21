// Package manilafind implements Python Resource.find for Manila resources:
// a direct GET by identity, a list fallback after 400/403/404, exact ID/name
// matching and duplicate detection.
package manilafind

import (
	"context"
	"fmt"
	"iter"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Spec supplies one resource's operations. List must be the summary
// collection (GET /<collection>), not the detail listing.
type Spec[T any] struct {
	Kind          string
	Identity      string
	IgnoreMissing bool
	// NameQuery is set to the identity unless Query already has "name".
	NameQuery bool
	Query     url.Values
	Get       func(context.Context, string) (*T, error)
	List      func(context.Context, url.Values) iter.Seq2[*T, error]
	ID, Name  func(*T) string
}

func Find[T any](ctx context.Context, spec Spec[T]) (*T, error) {
	if ctx == nil {
		return nil, fmt.Errorf("%w: context is required", resource.ErrInvalidOption)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := resource.Name(spec.Identity).Validate(); err != nil {
		return nil, err
	}
	if resource.ID(spec.Identity).Validate() == nil {
		value, err := spec.Get(ctx, spec.Identity)
		if err == nil {
			return value, nil
		}
		fallback := false
		for _, code := range []int{http.StatusBadRequest, http.StatusForbidden, http.StatusNotFound} {
			if gophercloud.ResponseCodeIs(err, code) {
				fallback = true
			}
		}
		if !fallback {
			return nil, err
		}
	}
	query := url.Values{}
	for key, values := range spec.Query {
		query[key] = append([]string(nil), values...)
	}
	if spec.NameQuery && !query.Has("name") {
		query.Set("name", spec.Identity)
	}
	var found *T
	for value, err := range spec.List(ctx, query) {
		if err != nil {
			return nil, err
		}
		if spec.ID(value) != spec.Identity && spec.Name(value) != spec.Identity {
			continue
		}
		if found != nil {
			return nil, &resource.AmbiguousError{Resource: spec.Kind, Name: spec.Identity, IDs: []string{spec.ID(found), spec.ID(value)}}
		}
		found = value
	}
	if found == nil && !spec.IgnoreMissing {
		return nil, &resource.NotFoundError{Resource: spec.Kind, Reference: spec.Identity}
	}
	return found, nil
}

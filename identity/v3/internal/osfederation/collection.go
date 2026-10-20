package osfederation

import (
	"context"
	"errors"
	"iter"
	"net/http"
	"net/url"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Post implements a Python create_* proxy call for a class whose
// create_method is POST: POST {collection} with {resource_key: attrs}. Every
// given declared attribute, including id, stays in the body as Resource.create
// sends the dirty attributes; the response overlays the requested values.
func Post[O ~func(*WriteOpts) error](ctx context.Context, t Target, options []O) (*Record, error) {
	const op = "Create"
	p, err := t.Prepare(ctx)
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	policy, err := Apply(ctx, p.guard, WriteOpts{}, CopyWrite, slices.Clone(options))
	if err == nil {
		err = p.Headers(policy.Headers)
	}
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	fields, err := t.Spec.fields(policy.Fields, false)
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	body := map[string]any{t.Spec.ResourceKey: cloneFields(fields)}
	response, err := rest.DoJSONGuardedRejections(ctx, &p.source.Client, p.guard, http.MethodPost, p.source.Client.ServiceURL(t.Path), body, nil, rest.RejectionPolicy{Codes: rejectedCodes(), PreserveCleanRetry: true}, Codes()...)
	if err != nil {
		return evidence(response), t.Wrap(ctx, op, err)
	}
	value, err := p.record(response, p.seed(fields, ""))
	return value, t.Wrap(ctx, op, err)
}

// Named describes the name lookup of a class whose name is its own Body key.
// Key is the Body key compared with name_or_id. Query is the wire name that
// Resource.find adds to the fallback list when the class maps a name query;
// empty means the fallback list has no query.
type Named struct{ Key, Query string }

// FindNamed implements a Python find_* proxy call for a class with a separate
// name: GET {collection}/{name_or_id} first and, after a clean 400, 403 or
// 404 only, the complete list (with the name query when the class maps one)
// matched by exact ID or name.
func FindNamed[O ~func(*FindOpts) error](ctx context.Context, t Target, nameOrID string, named Named, options []O) (*Record, error) {
	const op = "Find"
	p, err := t.Prepare(ctx)
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	policy, err := Apply(ctx, p.guard, FindOpts{}, CopyFind, slices.Clone(options))
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	var query url.Values
	if named.Query != "" {
		query = url.Values{named.Query: {nameOrID}}
	}
	p.acceptJSON()
	ignore := policy.IgnoreMissing == nil || *policy.IgnoreMissing
	collection := resource.NewCollection(resource.Adapter[Record]{Kind: t.Spec.Kind, IdentityFind: true,
		ValidateID:        func(id string) error { return ValidateID("ID", id) },
		IdentityDirectGet: func(id string) error { return ValidateID("ID", id) },
		IdentityResponseID: func(v *Record) (string, error) {
			if v == nil {
				return "", Invalid("find response missing")
			}
			return v.Text("id"), nil
		},
		Name: func(v *Record) string { return v.Text(named.Key) },
		Get:  func(_ context.Context, id string) (*Record, error) { return p.Get(id, nil) },
		Iterate: func(context.Context, url.Values) iter.Seq2[*Record, error] {
			return p.List(query, nil, rest.ListControl{})
		}})
	value, err := collection.FindIdentity(ctx, nameOrID, resource.WithIdentityFindIgnoreMissing(ignore))
	return value, t.Wrap(ctx, op, errors.Join(err, p.guard(ctx)))
}

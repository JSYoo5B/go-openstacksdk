package workflows

import (
	"context"
	"iter"
	"net/url"

	resource "github.com/JSYoo5B/go-openstacksdk/resource"
)

// FindIdentity resolves one Workflow by identity the way openstacksdk's
// find_workflow does: it first sends GET with the literal identity and returns the
// result, then falls back to a complete list only after a clean 400, 403 or
// 404 (see resource.WithIdentityFindFallback for stricter policies). The list
// fallback matches the identity against the exact ID or name of every row on
// every page, so a second match is an ambiguity error instead of a silent
// choice. A missing result is (nil, nil) by default, like the Python
// ignore_missing=True default; pass resource.WithIdentityFindIgnoreMissing(false)
// for a not-found error. Transport errors, 401/5xx statuses, cancellation and
// list failures are never treated as absence. Nothing is mutated or retried.
func (a *API) FindIdentity(ctx context.Context, identity string, options ...resource.IdentityFindOption) (*Workflow, error) {
	return a.identityResources().FindIdentity(ctx, identity, options...)
}

func (a *API) identityResources() *resource.Collection[Workflow] {
	return resource.NewCollection(resource.Adapter[Workflow]{
		Kind:         "workflows",
		IdentityFind: true,
		Get:          func(ctx context.Context, id string) (*Workflow, error) { return a.Get(ctx, id) },
		ID:           func(v *Workflow) string { return v.ID },
		Name:         func(v *Workflow) string { return v.Name },
		IterateControlled: func(ctx context.Context, q url.Values, control resource.ListControl) iter.Seq2[*Workflow, error] {
			options := make([]ListOption, 0, len(q))
			for key, values := range q {
				for _, value := range values {
					options = append(options, WithListQuery(key, value))
				}
			}
			return a.listWithControl(ctx, control, options...)
		}})
}

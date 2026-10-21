package crontriggers

import (
	"context"
	"iter"
	"net/url"

	resource "github.com/JSYoo5B/go-openstacksdk/resource"
)

// FindIdentity resolves one CronTrigger by identity the way openstacksdk's
// find_cron_trigger does: it first sends GET with the literal identity and returns the
// result, then falls back to a complete list only after a clean 400, 403 or
// 404 (see resource.WithIdentityFindFallback for stricter policies). The list
// fallback matches the identity against the exact ID or name of every row on
// every page, so a second match is an ambiguity error instead of a silent
// choice. The list request carries the identity as the name filter, like Python. A missing result is (nil, nil) by default, like the Python
// ignore_missing=True default; pass resource.WithIdentityFindIgnoreMissing(false)
// for a not-found error. Transport errors, 401/5xx statuses, cancellation and
// list failures are never treated as absence. Nothing is mutated or retried.
func (a *API) FindIdentity(ctx context.Context, identity string, options ...resource.IdentityFindOption) (*CronTrigger, error) {
	return a.identityResources(false).FindIdentity(ctx, identity, options...)
}

// FindIdentityAllProjects is FindIdentity with Python's all_projects=True: the
// direct GET is unchanged and the list fallback adds all_projects=1, which
// needs an administrator. The SDK reserves the all_projects key in
// resource.WithIdentityFindQuery, so this separate method carries the flag.
func (a *API) FindIdentityAllProjects(ctx context.Context, identity string, options ...resource.IdentityFindOption) (*CronTrigger, error) {
	return a.identityResources(true).FindIdentity(ctx, identity, options...)
}

func (a *API) identityResources(allProjects bool) *resource.Collection[CronTrigger] {
	return resource.NewCollection(resource.Adapter[CronTrigger]{
		Kind:         "crontriggers",
		IdentityFind: true,
		Get:          func(ctx context.Context, id string) (*CronTrigger, error) { return a.Get(ctx, id) },
		ID:           func(v *CronTrigger) string { return v.ID },
		Name:         func(v *CronTrigger) string { return v.Name },
		NameQuery:    func(name string) string { return name },
		IterateControlled: func(ctx context.Context, q url.Values, control resource.ListControl) iter.Seq2[*CronTrigger, error] {
			options := make([]ListOption, 0, len(q)+1)
			if allProjects {
				options = append(options, WithListOptions(ListOpts{AllProjects: 1}))
			}
			for key, values := range q {
				for _, value := range values {
					options = append(options, WithListQuery(key, value))
				}
			}
			return a.listWithControl(ctx, control, options...)
		}})
}

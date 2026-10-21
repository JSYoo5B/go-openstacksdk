package tlds

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Find resolves a resource.ID (one GET) or an exact resource.Name (list with
// name=<name>; duplicates fail). A missing TLD is an error unless
// resource.WithIgnoreMissing() is supplied.
func (a *API) Find(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) (*TLD, error) {
	return a.Resources.Find(ctx, ref, options...)
}

// FindIdentity is Python find_tld: GET by the identifier first, then a list
// filtered by name, matching the identifier against ID or name. Absence
// defaults to nil, nil; use resource.WithIdentityFindIgnoreMissing(false) for
// an error.
func (a *API) FindIdentity(ctx context.Context, identity string, options ...resource.IdentityFindOption) (*TLD, error) {
	return a.Resources.FindIdentity(ctx, identity, options...)
}

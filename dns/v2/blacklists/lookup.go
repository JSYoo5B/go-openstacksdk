package blacklists

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Find resolves an explicit ID (one GET). Blacklists have no name attribute,
// so a resource.Name reference is not supported; match patterns with ListOpts.
func (a *API) Find(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) (*Blacklist, error) {
	return a.Resources.Find(ctx, ref, options...)
}

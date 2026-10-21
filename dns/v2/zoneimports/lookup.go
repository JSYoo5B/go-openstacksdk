package zoneimports

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Find resolves an explicit ID with one GET. Imports have no name.
func (a *API) Find(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) (*ZoneImport, error) {
	return a.Resources.Find(ctx, ref, options...)
}

// All collects the import list with common list options.
func (a *API) All(ctx context.Context, options ...resource.ListOption) ([]*ZoneImport, error) {
	return a.Resources.All(ctx, options...)
}

// Remove deletes an import; a missing import is ignored unless
// resource.WithMissingError() is supplied (Python ignore_missing=True).
func (a *API) Remove(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) error {
	return a.Resources.Delete(ctx, ref, options...)
}

// WaitFor polls GET until status matches case-insensitively; ERROR fails.
func (a *API) WaitFor(ctx context.Context, ref resource.Ref, status string, options ...resource.WaitOption) (*ZoneImport, error) {
	return a.Resources.Wait(ctx, ref, status, options...)
}

// WaitForDeletion polls until GET returns 404 or the status is DELETED.
func (a *API) WaitForDeletion(ctx context.Context, ref resource.Ref, options ...resource.WaitOption) error {
	return a.Resources.WaitDeleted(ctx, ref, options...)
}

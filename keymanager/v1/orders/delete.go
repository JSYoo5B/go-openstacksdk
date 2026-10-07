package orders

import (
	"context"

	"github.com/JSYoo5B/gophercloudsdk/internal/keymanagerread"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// deleteOwned is only the collection binding. Native Delete keeps its public
// string-ID signature, native status policy and compatibility result handling.
func (a *API) deleteOwned(ctx context.Context, id string) error {
	var client *gophercloud.ServiceClient
	if a != nil {
		client = a.client
	}
	return request.Wrap("Delete", "orders", keymanagerread.Delete(ctx, client, "orders", id))
}

func (a *API) removeOwned(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) error {
	var client *gophercloud.ServiceClient
	var collection *resource.Collection[Order]
	if a != nil {
		client, collection = a.client, a.Resources
	}
	return request.Wrap("Remove", "orders", keymanagerread.Remove(ctx, client, "orders", collection, ref, options...))
}

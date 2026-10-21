package replicas

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// ForceRemove sends the force_delete action for one ID. A 404 is ignored by
// default, like Remove; pass resource.WithMissingError to report
// resource.ErrNotFound. The ID is never resolved by name, nothing is retried and
// acknowledgement does not imply the resource is gone: use WaitForDeletion.
func (a *API) ForceRemove(ctx context.Context, id string, options ...resource.LookupOption) error {
	if err := resource.ID(id).Validate(); err != nil {
		return request.Wrap("ForceRemove", "replicas", err)
	}
	ignore, err := resource.IgnoresMissing(true, options...)
	if err != nil {
		return request.Wrap("ForceRemove", "replicas", err)
	}
	err = a.ForceDelete(ctx, id)
	if err != nil && ctx.Err() == nil && gophercloud.ResponseCodeIs(err, 404) {
		if ignore {
			return nil
		}
		return request.Wrap("ForceRemove", "replicas", &resource.NotFoundError{Resource: "replicas", Reference: id, Cause: err})
	}
	return err
}

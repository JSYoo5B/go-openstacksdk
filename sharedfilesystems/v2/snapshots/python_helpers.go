package snapshots

import (
	"context"
	"net/url"

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
		return request.Wrap("ForceRemove", "snapshots", err)
	}
	ignore, err := resource.IgnoresMissing(true, options...)
	if err != nil {
		return request.Wrap("ForceRemove", "snapshots", err)
	}
	err = a.ForceDelete(ctx, id)
	if err != nil && ctx.Err() == nil && gophercloud.ResponseCodeIs(err, 404) {
		if ignore {
			return nil
		}
		return request.Wrap("ForceRemove", "snapshots", &resource.NotFoundError{Resource: "snapshots", Reference: id, Cause: err})
	}
	return err
}

// GetMetadata reads GET /snapshots/{id}/metadata and returns the metadata map.
// Only HTTP 200 succeeds; an absent or null metadata object yields an empty,
// non-nil map. A 404 stays the native response error.
func (a *API) GetMetadata(ctx context.Context, id string) (map[string]string, error) {
	if err := resource.ID(id).Validate(); err != nil {
		return nil, request.Wrap("GetMetadata", "snapshots", err)
	}
	var out struct {
		Metadata map[string]string `json:"metadata"`
	}
	_, err := a.client.Get(ctx, a.client.ServiceURL("snapshots", id, "metadata"), &out, &gophercloud.RequestOpts{OkCodes: []int{200}})
	if err != nil {
		return nil, request.Wrap("GetMetadata", "snapshots", err)
	}
	if out.Metadata == nil {
		out.Metadata = map[string]string{}
	}
	return out.Metadata, nil
}

// GetMetadatum reads GET /snapshots/{id}/metadata/{key} and returns the
// service's "meta" map. Only HTTP 200 succeeds and an empty key fails before
// any request.
func (a *API) GetMetadatum(ctx context.Context, id string, key string) (map[string]string, error) {
	if err := resource.ID(id).Validate(); err != nil {
		return nil, request.Wrap("GetMetadatum", "snapshots", err)
	}
	if key == "" {
		return nil, request.Wrap("GetMetadatum", "snapshots", resource.ErrInvalidOption)
	}
	var out struct {
		Meta map[string]string `json:"meta"`
	}
	_, err := a.client.Get(ctx, a.client.ServiceURL("snapshots", id, "metadata", url.PathEscape(key)), &out, &gophercloud.RequestOpts{OkCodes: []int{200}})
	if err != nil {
		return nil, request.Wrap("GetMetadatum", "snapshots", err)
	}
	return out.Meta, nil
}

// DeleteMetadata deletes each key with its own DELETE
// /snapshots/{id}/metadata/{key}, in order. The first failure of any kind,
// including 404, stops the loop and is returned; later keys are not attempted
// and earlier deletions are not rolled back. Only HTTP 200 succeeds.
func (a *API) DeleteMetadata(ctx context.Context, id string, keys []string) error {
	if err := resource.ID(id).Validate(); err != nil {
		return request.Wrap("DeleteMetadata", "snapshots", err)
	}
	for _, key := range keys {
		if key == "" {
			return request.Wrap("DeleteMetadata", "snapshots", resource.ErrInvalidOption)
		}
	}
	for _, key := range keys {
		_, err := a.client.Delete(ctx, a.client.ServiceURL("snapshots", id, "metadata", url.PathEscape(key)), &gophercloud.RequestOpts{OkCodes: []int{200}})
		if err != nil {
			return request.Wrap("DeleteMetadata", "snapshots", err)
		}
	}
	return nil
}

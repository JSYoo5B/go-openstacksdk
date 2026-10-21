package sharetypes

import (
	"context"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/sharetypes"
)

// UnsetExtraSpecProperty deletes one extra spec with DELETE
// /types/{id}/extra_specs/{key}. Unlike Remove, a 404 is reported by default
// (resource.ErrNotFound wrapping the native response error); pass
// resource.WithIgnoreMissing to treat an absent share type or key as success.
// Other failures, including 403, remain errors and nothing is retried.
func (a *API) UnsetExtraSpecProperty(ctx context.Context, id string, key string, options ...resource.LookupOption) error {
	if err := resource.ID(id).Validate(); err != nil {
		return request.Wrap("UnsetExtraSpecProperty", "sharetypes", err)
	}
	if strings.TrimSpace(key) == "" {
		return request.Wrap("UnsetExtraSpecProperty", "sharetypes", resource.ErrInvalidOption)
	}
	ignore, err := resource.IgnoresMissing(false, options...)
	if err != nil {
		return request.Wrap("UnsetExtraSpecProperty", "sharetypes", err)
	}
	err = a.UnsetExtraSpecs(ctx, id, key)
	if err != nil && ctx.Err() == nil && gophercloud.ResponseCodeIs(err, 404) {
		if ignore {
			return nil
		}
		return request.Wrap("UnsetExtraSpecProperty", "sharetypes", &resource.NotFoundError{Resource: "sharetypes", Reference: id + "/" + key, Cause: err})
	}
	return err
}

// CreateSpecsOpts describes a share type whose extra specs are sent verbatim.
// Unlike CreateOpts, whose driver_handles_share_servers field is required and
// therefore cannot carry false, every ExtraSpecs value (false, zero, empty
// string and nil) is preserved on the wire.
type CreateSpecsOpts struct {
	// Name is required.
	Name string
	// Description is omitted when nil and sent, even when empty, otherwise.
	Description *string
	// IsPublic is omitted when nil and sent as os-share-type-access:is_public
	// otherwise.
	IsPublic *bool
	// ExtraSpecs is omitted when nil and sent as given otherwise, including an
	// empty map. Keys must not be blank.
	ExtraSpecs map[string]any
}

// CreateWithExtraSpecs sends POST /types with one share_type envelope built
// from opts. HTTP 200 and 202 succeed; other statuses keep the native error.
// The call is never replayed and acknowledgement does not prove the type is
// usable by every project.
func (a *API) CreateWithExtraSpecs(ctx context.Context, opts CreateSpecsOpts) (*ShareType, error) {
	if strings.TrimSpace(opts.Name) == "" {
		return nil, request.Wrap("CreateWithExtraSpecs", "sharetypes", resource.ErrInvalidOption)
	}
	for key := range opts.ExtraSpecs {
		if strings.TrimSpace(key) == "" {
			return nil, request.Wrap("CreateWithExtraSpecs", "sharetypes", resource.ErrInvalidOption)
		}
	}
	body := map[string]any{"name": opts.Name}
	if opts.Description != nil {
		body["description"] = *opts.Description
	}
	if opts.IsPublic != nil {
		body["os-share-type-access:is_public"] = *opts.IsPublic
	}
	if opts.ExtraSpecs != nil {
		body["extra_specs"] = opts.ExtraSpecs
	}
	var result upstream.CreateResult
	resp, err := a.client.Post(ctx, a.client.ServiceURL("types"), map[string]any{"share_type": body}, &result.Body,
		&gophercloud.RequestOpts{OkCodes: []int{200, 202}})
	_, result.Header, result.Err = gophercloud.ParseResponse(resp, err)
	value, err := result.Extract()
	if err != nil {
		return nil, request.Wrap("CreateWithExtraSpecs", "sharetypes", err)
	}
	return value, nil
}

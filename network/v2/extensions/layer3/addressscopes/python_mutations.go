package addressscopes

import (
	"context"
	"errors"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// AddressScope inherits id and name from Resource. The pinned Source adds
// ip_version, project_id, tenant_id and is_shared.
var neutronBodyFieldNames = map[string]string{
	"id": "id", "name": "name", "ip_version": "ip_version",
	"project_id": "project_id", "tenant_id": "tenant_id", "is_shared": "shared",
}

// AttributeOption supplies a JSON Body attribute for the dedicated Source
// mutation profile. Use WithAttribute to snapshot values before a request.
type AttributeOption = request.Option[struct{}]

// WithAttribute accepts canonical Python Body names or wire names. Unknown
// extension attributes remain in the body. Later options replace earlier ones.
// Omission, null, false, zero, empty strings and empty lists stay distinct.
func WithAttribute(key string, value any) AttributeOption {
	if wire, ok := neutronBodyFieldNames[key]; ok {
		key = wire
	}
	return request.WithField[struct{}](key, value)
}

func neutronMutationAttributes(ctx context.Context, client *gophercloud.ServiceClient, options []AttributeOption) (map[string]any, error) {
	if ctx == nil || client == nil || client.ProviderClient == nil {
		return nil, fmt.Errorf("%w: a context and configured client are required", resource.ErrInvalidOption)
	}
	if ctx.Err() != nil {
		return nil, neutronMutationContextError(ctx, nil)
	}
	config, err := request.Apply(struct{}{}, append([]AttributeOption(nil), options...)...)
	if err == nil {
		err = request.ValidateCapabilities(config, true, false, false)
	}
	if err != nil {
		if ctx.Err() != nil {
			err = neutronMutationContextError(ctx, err)
		}
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, neutronMutationContextError(ctx, nil)
	}
	for _, key := range []string{"base_path", "microversion", "connection", "_synchronized", "resource_type", "__conflicting_attrs", "location"} {
		if _, present := config.Fields[key]; present {
			return nil, fmt.Errorf("%w: %q is a Resource control attribute, not a Body extension", resource.ErrInvalidOption, key)
		}
	}
	return request.MergeFields(map[string]any{}, config.Fields)
}

func neutronMutationContextError(ctx context.Context, err error) error {
	cause := context.Cause(ctx)
	if cause == ctx.Err() {
		cause = nil
	}
	return errors.Join(err, ctx.Err(), cause)
}

func neutronMutationCodes() []int {
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = 200 + i
	}
	return codes
}

// CreateWithAttributes creates from supplied Body attributes without the
// native builder's required-field checks or synthesized default fields. No
// attributes sends an empty resource envelope. The server validates the body.
// Successful HTTP statuses are 200..399; result decoding uses the native model.
func (a *API) CreateWithAttributes(ctx context.Context, options ...AttributeOption) (*AddressScope, error) {
	var client *gophercloud.ServiceClient
	if a != nil {
		client = a.client
	}
	attributes, err := neutronMutationAttributes(ctx, client, options)
	if err != nil {
		return nil, request.Wrap("CreateWithAttributes", "addressscopes", err)
	}
	var result CreateResult
	response, err := client.Post(ctx, client.ServiceURL("address-scopes"), map[string]any{"address_scope": attributes}, &result.Body, &gophercloud.RequestOpts{OkCodes: neutronMutationCodes()})
	_, result.Header, result.Err = gophercloud.ParseResponse(response, err)
	if ctx.Err() != nil {
		result.Err = neutronMutationContextError(ctx, result.Err)
	}
	value, err := result.Extract()
	return value, request.Wrap("CreateWithAttributes", "addressscopes", err)
}

// UpdateWithAttributes commits supplied Body attributes for a literal ID.
// No attributes returns a native model containing that ID without HTTP. It
// does not maintain a mutable Resource's cache or track prior field values.
func (a *API) UpdateWithAttributes(ctx context.Context, id string, options ...AttributeOption) (*AddressScope, error) {
	var client *gophercloud.ServiceClient
	if a != nil {
		client = a.client
	}
	attributes, err := neutronMutationAttributes(ctx, client, options)
	if err != nil {
		return nil, request.Wrap("UpdateWithAttributes", "addressscopes", err)
	}
	if _, present := attributes["id"]; present {
		return nil, request.Wrap("UpdateWithAttributes", "addressscopes", fmt.Errorf("%w: update ID must be supplied as the method argument", resource.ErrInvalidOption))
	}
	if len(attributes) == 0 {
		return &AddressScope{ID: id}, nil
	}
	var result UpdateResult
	response, err := client.Put(ctx, client.ServiceURL("address-scopes", id), map[string]any{"address_scope": attributes}, &result.Body, &gophercloud.RequestOpts{OkCodes: neutronMutationCodes()})
	_, result.Header, result.Err = gophercloud.ParseResponse(response, err)
	if ctx.Err() != nil {
		result.Err = neutronMutationContextError(ctx, result.Err)
	}
	value, err := result.Extract()
	return value, request.Wrap("UpdateWithAttributes", "addressscopes", err)
}

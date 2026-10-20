package trunks

import (
	"context"
	"errors"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

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
func (a *API) CreateWithAttributes(ctx context.Context, options ...AttributeOption) (*Trunk, error) {
	var client *gophercloud.ServiceClient
	if a != nil {
		client = a.client
	}
	attributes, err := neutronMutationAttributes(ctx, client, options)
	if err != nil {
		return nil, request.Wrap("CreateWithAttributes", "trunks", err)
	}
	var result CreateResult
	response, err := client.Post(ctx, client.ServiceURL("trunks"), map[string]any{"trunk": attributes}, &result.Body, &gophercloud.RequestOpts{OkCodes: neutronMutationCodes()})
	_, result.Header, result.Err = gophercloud.ParseResponse(response, err)
	if ctx.Err() != nil {
		result.Err = neutronMutationContextError(ctx, result.Err)
	}
	value, err := result.Extract()
	return value, request.Wrap("CreateWithAttributes", "trunks", err)
}

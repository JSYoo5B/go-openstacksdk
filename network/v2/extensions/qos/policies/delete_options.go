package policies

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// DeleteOpts selects the pinned Python proxy's missing-resource behavior.
// Nil IgnoreMissing means true. These Resource classes have no revision option.
type DeleteOpts struct{ IgnoreMissing *bool }

// DeleteOption is one option of DeleteWithOptions and DeleteSourceRecord:
// WithDeleteOptions, request.IgnoreMissing or request.Custom over
// *DeleteOpts. Other shared calls fail with resource.ErrInvalidOption before
// HTTP.
type DeleteOption = request.DeleteCall

var deleteCalls = calls.Spec[DeleteOpts]{
	IgnoreMissing: calls.Pointer(func(target *DeleteOpts) **bool { return &target.IgnoreMissing }),
}

// WithDeleteOptions snapshots the configuration and returns new pointers when
// reused, using the common Neutron delete option helper.
func WithDeleteOptions(value DeleteOpts) DeleteOption {
	apply := request.WithNeutronDeleteOptions(request.NeutronDeleteOpts{IgnoreMissing: value.IgnoreMissing})
	return request.Custom(func(target *DeleteOpts) error {
		var config request.NeutronDeleteOpts
		if err := apply(&config); err != nil {
			return err
		}
		target.IgnoreMissing = config.IgnoreMissing
		return nil
	})
}

// DeleteWithOptions deletes a literal ID without a lookup. Missing resources
// are ignored by default, and later options override earlier ones. It accepts
// HTTP 200..399 without decoding a successful body. Ordinary Delete and Remove
// retain their native status policy.
func (a *API) DeleteWithOptions(ctx context.Context, id string, options ...DeleteOption) error {
	if a == nil || a.client == nil {
		return request.Wrap("DeleteWithOptions", "policies", fmt.Errorf("%w: a configured client is required", resource.ErrInvalidOption))
	}
	shared := make([]request.NeutronDeleteOption, 0, len(options))
	for _, apply := range calls.Translate(options, deleteCalls) {
		if apply == nil {
			shared = append(shared, nil)
			continue
		}
		shared = append(shared, func(config *request.NeutronDeleteOpts) error {
			target := DeleteOpts{IgnoreMissing: config.IgnoreMissing}
			if err := apply(&target); err != nil {
				return err
			}
			config.IgnoreMissing = target.IgnoreMissing
			return nil
		})
	}
	return request.Wrap("DeleteWithOptions", "policies", request.DeleteNeutron(ctx, a.client, a.client.ServiceURL("qos/policies", id), shared...))
}

package quotas

import (
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// UpdateOpts replaces project overrides. Omitted values use server defaults;
// zero and negative integer values are present. Explicit null is invalid under
// Barbican's published integer-only schema. An empty UpdateOpts sends {}.
type UpdateOpts struct {
	Secrets    request.Optional[int64] `json:"secrets,omitzero"`
	Orders     request.Optional[int64] `json:"orders,omitzero"`
	Containers request.Optional[int64] `json:"containers,omitzero"`
	Consumers  request.Optional[int64] `json:"consumers,omitzero"`
	CAs        request.Optional[int64] `json:"cas,omitzero"`
}

type GetOpts struct{}
type DeleteOpts struct {
	IgnoreMissing *bool `json:"ignore_missing,omitempty"`
}

// GetOption is one option of Get: request.Header or request.Custom over
// *request.Config[GetOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type GetOption = request.GetCall

// UpdateOption is one option of Update: WithUpdateOptions, WithUpdateSecrets,
// WithUpdateOrders, WithUpdateContainers, WithUpdateConsumers, WithUpdateCAs,
// request.Header or request.Custom over *request.Config[UpdateOpts]. Other
// shared calls fail with resource.ErrInvalidOption before HTTP.
type UpdateOption = request.WriteCall

// DeleteOption is one option of Delete: WithDeleteOptions,
// request.IgnoreMissing (nil means true), request.Header or request.Custom
// over *request.Config[DeleteOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type DeleteOption = request.DeleteCall

var (
	getCalls    = calls.Spec[request.Config[GetOpts]]{Header: calls.ConfigHeader[GetOpts]}
	updateCalls = calls.Spec[request.Config[UpdateOpts]]{Header: calls.ConfigHeader[UpdateOpts]}
	deleteCalls = calls.Spec[request.Config[DeleteOpts]]{
		Header:        calls.ConfigHeader[DeleteOpts],
		IgnoreMissing: calls.Pointer(func(c *request.Config[DeleteOpts]) **bool { return &c.Options.IgnoreMissing }),
	}
)

func snapshot[T any](opts T) request.Option[T] {
	encoded, err := json.Marshal(opts)
	return func(config *request.Config[T]) error {
		if err != nil {
			return fmt.Errorf("%w: quota options: %v", resource.ErrInvalidOption, err)
		}
		var copied T
		if err := json.Unmarshal(encoded, &copied); err != nil {
			return fmt.Errorf("%w: quota options: %v", resource.ErrInvalidOption, err)
		}
		config.Options = copied
		return nil
	}
}

func WithUpdateOptions(opts UpdateOpts) UpdateOption { return request.Custom(snapshot(opts)) }
func WithDeleteOptions(opts DeleteOpts) DeleteOption { return request.Custom(snapshot(opts)) }
func WithUpdateSecrets(value int64) UpdateOption {
	return request.Custom(func(c *request.Config[UpdateOpts]) error { c.Options.Secrets = request.Present(value); return nil })
}
func WithUpdateOrders(value int64) UpdateOption {
	return request.Custom(func(c *request.Config[UpdateOpts]) error { c.Options.Orders = request.Present(value); return nil })
}
func WithUpdateContainers(value int64) UpdateOption {
	return request.Custom(func(c *request.Config[UpdateOpts]) error { c.Options.Containers = request.Present(value); return nil })
}
func WithUpdateConsumers(value int64) UpdateOption {
	return request.Custom(func(c *request.Config[UpdateOpts]) error { c.Options.Consumers = request.Present(value); return nil })
}
func WithUpdateCAs(value int64) UpdateOption {
	return request.Custom(func(c *request.Config[UpdateOpts]) error { c.Options.CAs = request.Present(value); return nil })
}

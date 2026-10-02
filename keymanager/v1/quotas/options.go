package quotas

import (
	"encoding/json"
	"fmt"

	"gophercloudsdk/request"
	"gophercloudsdk/resource"
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
type GetOption = request.Option[GetOpts]
type UpdateOption = request.Option[UpdateOpts]
type DeleteOption = request.Option[DeleteOpts]

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

func WithUpdateOptions(opts UpdateOpts) UpdateOption { return snapshot(opts) }
func WithDeleteOptions(opts DeleteOpts) DeleteOption { return snapshot(opts) }
func WithUpdateSecrets(value int64) UpdateOption {
	return func(c *request.Config[UpdateOpts]) error { c.Options.Secrets = request.Present(value); return nil }
}
func WithUpdateOrders(value int64) UpdateOption {
	return func(c *request.Config[UpdateOpts]) error { c.Options.Orders = request.Present(value); return nil }
}
func WithUpdateContainers(value int64) UpdateOption {
	return func(c *request.Config[UpdateOpts]) error { c.Options.Containers = request.Present(value); return nil }
}
func WithUpdateConsumers(value int64) UpdateOption {
	return func(c *request.Config[UpdateOpts]) error { c.Options.Consumers = request.Present(value); return nil }
}
func WithUpdateCAs(value int64) UpdateOption {
	return func(c *request.Config[UpdateOpts]) error { c.Options.CAs = request.Present(value); return nil }
}
func WithGetHeader(key, value string) GetOption { return request.WithHeader[GetOpts](key, value) }
func WithUpdateHeader(key, value string) UpdateOption {
	return request.WithHeader[UpdateOpts](key, value)
}
func WithDeleteHeader(key, value string) DeleteOption {
	return request.WithHeader[DeleteOpts](key, value)
}
func WithDeleteIgnoreMissing(value bool) DeleteOption {
	return func(c *request.Config[DeleteOpts]) error {
		copied := value
		c.Options.IgnoreMissing = &copied
		return nil
	}
}

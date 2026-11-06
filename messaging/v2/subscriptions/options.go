package subscriptions

import (
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// RequestIdentity supplies an explicit unauthenticated project or client ID.
// Empty strings preserve the configured source values. Headers are never added
// to the shared client, and no auth-result project is inferred.
type RequestIdentity struct{ ClientID, ProjectID string }
type CreateOpts struct {
	RequestIdentity `json:"-"`
	Subscriber      string                  `json:"subscriber"`
	TTL             request.Optional[int64] `json:"ttl,omitzero"`
	Options         json.RawMessage         `json:"options,omitempty"`
}
type GetOpts struct{ RequestIdentity }
type DeleteOpts struct {
	RequestIdentity
	IgnoreMissing *bool
}
type ListOpts struct {
	RequestIdentity
	Limit     int
	Marker    string
	MaxItems  int
	Paginated *bool
}

// CreateOption is one option of Create: WithCreateOptions, the WithCreate*
// setters, request.Field (an extension body member that cannot replace a
// typed one), request.Header or request.Custom over
// *request.Config[CreateOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type CreateOption = request.WriteCall

// GetOption is one option of Get: WithGetOptions, the WithGet* identity
// setters, request.Header or request.Custom over *request.Config[GetOpts].
// Other shared calls fail with resource.ErrInvalidOption before HTTP.
type GetOption = request.GetCall

// DeleteOption is one option of Delete: WithDeleteOptions, the WithDelete*
// identity setters, request.IgnoreMissing, request.Header or request.Custom
// over *request.Config[DeleteOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type DeleteOption = request.DeleteCall

// ListOption is one option of List and All: WithListOptions, the WithList*
// identity setters, request.MaxItems, request.Paginated, request.Query with a
// string or []string value (it may carry limit and marker but no scope or SDK
// control), request.Header or request.Custom over *request.Config[ListOpts].
// The page limit and marker are ListOpts members; request.Limit,
// request.Marker and other shared calls fail with resource.ErrInvalidOption
// before HTTP.
type ListOption = request.ListCall

var (
	createCalls = calls.Spec[request.Config[CreateOpts]]{Field: calls.ConfigField[CreateOpts], Header: calls.ConfigHeader[CreateOpts]}
	getCalls    = calls.Spec[request.Config[GetOpts]]{Header: calls.ConfigHeader[GetOpts]}
	deleteCalls = calls.Spec[request.Config[DeleteOpts]]{
		Header:        calls.ConfigHeader[DeleteOpts],
		IgnoreMissing: calls.Pointer(func(c *request.Config[DeleteOpts]) **bool { return &c.Options.IgnoreMissing }),
	}
	listCalls = calls.Spec[request.Config[ListOpts]]{
		Header: calls.ConfigHeader[ListOpts],
		Query:  calls.ConfigQuery[ListOpts],
		Paging: calls.Paging[request.Config[ListOpts]]{
			MaxItems:  calls.Value(func(c *request.Config[ListOpts]) *int { return &c.Options.MaxItems }),
			Paginated: calls.Pointer(func(c *request.Config[ListOpts]) **bool { return &c.Options.Paginated }),
		},
	}
)

func copyCreate(value CreateOpts) CreateOpts {
	value.Options = append(json.RawMessage(nil), value.Options...)
	return value
}
func copyList(value ListOpts) ListOpts {
	if value.Paginated != nil {
		owned := *value.Paginated
		value.Paginated = &owned
	}
	return value
}
func WithCreateOptions(value CreateOpts) CreateOption {
	owned := copyCreate(value)
	return request.Custom(func(c *request.Config[CreateOpts]) error { c.Options = copyCreate(owned); return nil })
}
func WithGetOptions(value GetOpts) GetOption {
	return request.Custom(func(c *request.Config[GetOpts]) error { c.Options = value; return nil })
}
func WithDeleteOptions(value DeleteOpts) DeleteOption {
	if value.IgnoreMissing != nil {
		owned := *value.IgnoreMissing
		value.IgnoreMissing = &owned
	}
	return request.Custom(func(c *request.Config[DeleteOpts]) error {
		c.Options = value
		if value.IgnoreMissing != nil {
			owned := *value.IgnoreMissing
			c.Options.IgnoreMissing = &owned
		}
		return nil
	})
}
func WithListOptions(value ListOpts) ListOption {
	owned := copyList(value)
	return request.Custom(func(c *request.Config[ListOpts]) error { c.Options = copyList(owned); return nil })
}
func WithCreateSubscriber(value string) CreateOption {
	return request.Custom(func(c *request.Config[CreateOpts]) error { c.Options.Subscriber = value; return nil })
}
func WithCreateTTL(value int64) CreateOption {
	return request.Custom(func(c *request.Config[CreateOpts]) error { c.Options.TTL = request.Present(value); return nil })
}
func WithCreateDeliveryOptions(value any) CreateOption {
	encoded, err := json.Marshal(value)
	return request.Custom(func(c *request.Config[CreateOpts]) error {
		if err != nil {
			return fmt.Errorf("%w: subscription options: %v", resource.ErrInvalidOption, err)
		}
		c.Options.Options = append(json.RawMessage(nil), encoded...)
		return nil
	})
}
func WithCreateClientID(value string) CreateOption {
	return request.Custom(func(c *request.Config[CreateOpts]) error { c.Options.ClientID = value; return nil })
}
func WithCreateProjectID(value string) CreateOption {
	return request.Custom(func(c *request.Config[CreateOpts]) error { c.Options.ProjectID = value; return nil })
}
func WithGetClientID(value string) GetOption {
	return request.Custom(func(c *request.Config[GetOpts]) error { c.Options.ClientID = value; return nil })
}
func WithGetProjectID(value string) GetOption {
	return request.Custom(func(c *request.Config[GetOpts]) error { c.Options.ProjectID = value; return nil })
}
func WithDeleteClientID(value string) DeleteOption {
	return request.Custom(func(c *request.Config[DeleteOpts]) error { c.Options.ClientID = value; return nil })
}
func WithDeleteProjectID(value string) DeleteOption {
	return request.Custom(func(c *request.Config[DeleteOpts]) error { c.Options.ProjectID = value; return nil })
}
func WithListClientID(value string) ListOption {
	return request.Custom(func(c *request.Config[ListOpts]) error { c.Options.ClientID = value; return nil })
}
func WithListProjectID(value string) ListOption {
	return request.Custom(func(c *request.Config[ListOpts]) error { c.Options.ProjectID = value; return nil })
}

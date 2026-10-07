package subscriptions

import (
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
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
type CreateOption = request.Option[CreateOpts]
type GetOption = request.Option[GetOpts]
type DeleteOption = request.Option[DeleteOpts]
type ListOption = request.Option[ListOpts]

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
	return func(c *request.Config[CreateOpts]) error { c.Options = copyCreate(owned); return nil }
}
func WithGetOptions(value GetOpts) GetOption {
	return func(c *request.Config[GetOpts]) error { c.Options = value; return nil }
}
func WithDeleteOptions(value DeleteOpts) DeleteOption {
	if value.IgnoreMissing != nil {
		owned := *value.IgnoreMissing
		value.IgnoreMissing = &owned
	}
	return func(c *request.Config[DeleteOpts]) error {
		c.Options = value
		if value.IgnoreMissing != nil {
			owned := *value.IgnoreMissing
			c.Options.IgnoreMissing = &owned
		}
		return nil
	}
}
func WithListOptions(value ListOpts) ListOption {
	owned := copyList(value)
	return func(c *request.Config[ListOpts]) error { c.Options = copyList(owned); return nil }
}
func WithCreateSubscriber(value string) CreateOption {
	return func(c *request.Config[CreateOpts]) error { c.Options.Subscriber = value; return nil }
}
func WithCreateTTL(value int64) CreateOption {
	return func(c *request.Config[CreateOpts]) error { c.Options.TTL = request.Present(value); return nil }
}
func WithCreateDeliveryOptions(value any) CreateOption {
	encoded, err := json.Marshal(value)
	return func(c *request.Config[CreateOpts]) error {
		if err != nil {
			return fmt.Errorf("%w: subscription options: %v", resource.ErrInvalidOption, err)
		}
		c.Options.Options = append(json.RawMessage(nil), encoded...)
		return nil
	}
}
func WithCreateField(key string, value any) CreateOption {
	return request.WithField[CreateOpts](key, value)
}
func WithCreateHeader(key, value string) CreateOption {
	return request.WithHeader[CreateOpts](key, value)
}
func WithGetHeader(key, value string) GetOption { return request.WithHeader[GetOpts](key, value) }
func WithDeleteHeader(key, value string) DeleteOption {
	return request.WithHeader[DeleteOpts](key, value)
}
func WithListHeader(key, value string) ListOption { return request.WithHeader[ListOpts](key, value) }
func WithListQuery(key, value string) ListOption  { return request.WithQuery[ListOpts](key, value) }
func WithCreateClientID(value string) CreateOption {
	return func(c *request.Config[CreateOpts]) error { c.Options.ClientID = value; return nil }
}
func WithCreateProjectID(value string) CreateOption {
	return func(c *request.Config[CreateOpts]) error { c.Options.ProjectID = value; return nil }
}
func WithGetClientID(value string) GetOption {
	return func(c *request.Config[GetOpts]) error { c.Options.ClientID = value; return nil }
}
func WithGetProjectID(value string) GetOption {
	return func(c *request.Config[GetOpts]) error { c.Options.ProjectID = value; return nil }
}
func WithDeleteClientID(value string) DeleteOption {
	return func(c *request.Config[DeleteOpts]) error { c.Options.ClientID = value; return nil }
}
func WithDeleteProjectID(value string) DeleteOption {
	return func(c *request.Config[DeleteOpts]) error { c.Options.ProjectID = value; return nil }
}
func WithListClientID(value string) ListOption {
	return func(c *request.Config[ListOpts]) error { c.Options.ClientID = value; return nil }
}
func WithListProjectID(value string) ListOption {
	return func(c *request.Config[ListOpts]) error { c.Options.ProjectID = value; return nil }
}
func WithDeleteIgnoreMissing(value bool) DeleteOption {
	return func(c *request.Config[DeleteOpts]) error {
		owned := value
		c.Options.IgnoreMissing = &owned
		return nil
	}
}
func WithListMaxItems(value int) ListOption {
	return func(c *request.Config[ListOpts]) error { c.Options.MaxItems = value; return nil }
}
func WithListPaginated(value bool) ListOption {
	return func(c *request.Config[ListOpts]) error { owned := value; c.Options.Paginated = &owned; return nil }
}

package shareaccessrules

import (
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/manilaown"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// AllowOpts supplies access identity; an omitted access level uses Manila's default.
// Optional pointer fields distinguish omission from explicit false or empty input.
type AllowOpts struct {
	AccessType     string             `json:"access_type"`
	AccessTo       string             `json:"access_to"`
	AccessLevel    string             `json:"access_level,omitempty"`
	Metadata       *map[string]string `json:"metadata,omitempty"`
	LockVisibility *bool              `json:"lock_visibility,omitempty"`
	LockDeletion   *bool              `json:"lock_deletion,omitempty"`
	LockReason     *string            `json:"lock_reason,omitempty"`
}

// AllowOption is one option of Allow and Create: WithAllowOptions, the typed
// setters below, request.Header, request.Field or request.Custom over
// *request.Config[AllowOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type AllowOption = request.WriteCall

var allowCalls = manilaown.HeaderFieldCalls[AllowOpts]()

func WithAllowOptions(value AllowOpts) AllowOption { return request.Custom(request.WithOptions(value)) }
func WithAccessLevel(value string) AllowOption {
	return request.Custom(func(c *request.Config[AllowOpts]) error { c.Options.AccessLevel = value; return nil })
}
func WithMetadata(value map[string]string) AllowOption {
	copy := make(map[string]string, len(value))
	for key, item := range value {
		copy[key] = item
	}
	return request.Custom(func(c *request.Config[AllowOpts]) error { c.Options.Metadata = &copy; return nil })
}
func WithLockVisibility(value bool) AllowOption {
	return request.Custom(func(c *request.Config[AllowOpts]) error { c.Options.LockVisibility = &value; return nil })
}
func WithLockDeletion(value bool) AllowOption {
	return request.Custom(func(c *request.Config[AllowOpts]) error { c.Options.LockDeletion = &value; return nil })
}
func WithLockReason(value string) AllowOption {
	return request.Custom(func(c *request.Config[AllowOpts]) error { c.Options.LockReason = &value; return nil })
}

// DenyOpts configures access revocation. Nil IgnoreMissing means true.
type DenyOpts struct {
	Unrestrict    *bool `json:"unrestrict,omitempty"`
	IgnoreMissing *bool `json:"-"`
}

// DenyOption is one option of Deny: WithDenyOptions, WithUnrestrict,
// WithDenyField, request.Header, request.IgnoreMissing or request.Custom over
// *request.Config[DenyOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type DenyOption = request.DeleteCall

var denyCalls = func() calls.Spec[request.Config[DenyOpts]] {
	spec := manilaown.IgnoreMissingCalls(func(o *DenyOpts) **bool { return &o.IgnoreMissing })
	spec.Header = calls.ConfigHeader[DenyOpts]
	return spec
}()

func WithDenyOptions(value DenyOpts) DenyOption { return request.Custom(request.WithOptions(value)) }

// WithDenyField adds an extension member to the deny_access body; access_id
// and unrestrict are protected. No delete category admits request.Field, so
// this option stays.
func WithDenyField(key string, value any) DenyOption {
	return request.Custom(request.WithField[DenyOpts](key, value))
}
func WithUnrestrict(value bool) DenyOption {
	return request.Custom(func(c *request.Config[DenyOpts]) error { c.Options.Unrestrict = &value; return nil })
}

type GetOpts struct{}

// GetOption is one option of Get: request.Header or request.Custom over
// *request.Config[GetOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type GetOption = request.GetCall

var getCalls = manilaown.HeaderCalls[GetOpts]()

type ListOpts struct{}

// ListOption is one option of List and All: request.Query (a string or
// []string value), request.Header, request.MaxItems, request.Paginated,
// WithListBodyFilter or request.Custom over *request.Config[ListOpts]. Other
// shared calls fail with resource.ErrInvalidOption before HTTP.
type ListOption = request.ListCall

package shareaccessrules

import "github.com/JSYoo5B/gophercloudsdk/request"

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

type AllowOption = request.Option[AllowOpts]

func WithAllowOptions(value AllowOpts) AllowOption { return request.WithOptions(value) }
func WithAllowField(key string, value any) AllowOption {
	return request.WithField[AllowOpts](key, value)
}
func WithAllowHeader(key, value string) AllowOption {
	return request.WithHeader[AllowOpts](key, value)
}
func WithAccessLevel(value string) AllowOption {
	return func(c *request.Config[AllowOpts]) error { c.Options.AccessLevel = value; return nil }
}
func WithMetadata(value map[string]string) AllowOption {
	copy := make(map[string]string, len(value))
	for key, item := range value {
		copy[key] = item
	}
	return func(c *request.Config[AllowOpts]) error { c.Options.Metadata = &copy; return nil }
}
func WithLockVisibility(value bool) AllowOption {
	return func(c *request.Config[AllowOpts]) error { c.Options.LockVisibility = &value; return nil }
}
func WithLockDeletion(value bool) AllowOption {
	return func(c *request.Config[AllowOpts]) error { c.Options.LockDeletion = &value; return nil }
}
func WithLockReason(value string) AllowOption {
	return func(c *request.Config[AllowOpts]) error { c.Options.LockReason = &value; return nil }
}

// DenyOpts configures access revocation. Nil IgnoreMissing means true.
type DenyOpts struct {
	Unrestrict    *bool `json:"unrestrict,omitempty"`
	IgnoreMissing *bool `json:"-"`
}

type DenyOption = request.Option[DenyOpts]

func WithDenyOptions(value DenyOpts) DenyOption { return request.WithOptions(value) }
func WithDenyField(key string, value any) DenyOption {
	return request.WithField[DenyOpts](key, value)
}
func WithDenyHeader(key, value string) DenyOption {
	return request.WithHeader[DenyOpts](key, value)
}
func WithUnrestrict(value bool) DenyOption {
	return func(c *request.Config[DenyOpts]) error { c.Options.Unrestrict = &value; return nil }
}
func WithDenyIgnoreMissing(value bool) DenyOption {
	return func(c *request.Config[DenyOpts]) error { c.Options.IgnoreMissing = &value; return nil }
}

type GetOpts struct{}
type GetOption = request.Option[GetOpts]

func WithGetHeader(key, value string) GetOption { return request.WithHeader[GetOpts](key, value) }

type ListOpts struct{}
type ListOption = request.Option[ListOpts]

func WithListQuery(key, value string) ListOption  { return request.WithQuery[ListOpts](key, value) }
func WithListHeader(key, value string) ListOption { return request.WithHeader[ListOpts](key, value) }

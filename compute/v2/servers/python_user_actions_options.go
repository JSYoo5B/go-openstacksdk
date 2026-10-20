package servers

import (
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// UserActionOpts carries headers and JSON extensions for direct actions.
// The action methods own literal network/address inputs and prevent replacement.
type UserActionOpts struct{}
type UserActionOption = request.Option[UserActionOpts]

func WithUserActionHeader(key, value string) UserActionOption {
	return request.WithHeader[UserActionOpts](key, value)
}
func WithUserActionField(key string, value any) UserActionOption {
	return request.WithField[UserActionOpts](key, value)
}

// LockServerOpts preserves an omitted reason and an explicitly empty reason.
// A nil reason matches locked_reason=None in Python.
type LockServerOpts struct {
	LockedReason *string `json:"locked_reason"`
}
type LockServerOption = request.Option[LockServerOpts]

func WithLockedReason(value string) LockServerOption {
	return func(c *request.Config[LockServerOpts]) error {
		owned := value
		c.Options.LockedReason = &owned
		return nil
	}
}

// WithoutLockedReason restores the Python default when composing options.
func WithoutLockedReason() LockServerOption {
	return func(c *request.Config[LockServerOpts]) error { c.Options.LockedReason = nil; return nil }
}
func WithLockServerHeader(key, value string) LockServerOption {
	return request.WithHeader[LockServerOpts](key, value)
}
func WithLockServerField(key string, value any) LockServerOption {
	return request.WithField[LockServerOpts](key, value)
}

// UnshelveServerOpts separates the default zone from an explicit null zone.
// ZoneSet=true, AvailabilityZone=nil unpins the zone (microversion 2.91).
// An empty Host is omitted, matching Python's truthiness check.
type UnshelveServerOpts struct {
	Host             string  `json:"host"`
	AvailabilityZone *string `json:"availability_zone"`
	ZoneSet          bool    `json:"-"`
}
type UnshelveServerOption = request.Option[UnshelveServerOpts]

func WithUnshelveHost(value string) UnshelveServerOption {
	return func(c *request.Config[UnshelveServerOpts]) error { c.Options.Host = value; return nil }
}
func WithUnshelveAvailabilityZone(value string) UnshelveServerOption {
	return func(c *request.Config[UnshelveServerOpts]) error {
		owned := value
		c.Options.AvailabilityZone = &owned
		c.Options.ZoneSet = true
		return nil
	}
}
func WithUnshelveUnpinnedAvailabilityZone() UnshelveServerOption {
	return func(c *request.Config[UnshelveServerOpts]) error {
		c.Options.AvailabilityZone = nil
		c.Options.ZoneSet = true
		return nil
	}
}
func WithUnshelveDefaultAvailabilityZone() UnshelveServerOption {
	return func(c *request.Config[UnshelveServerOpts]) error {
		c.Options.AvailabilityZone = nil
		c.Options.ZoneSet = false
		return nil
	}
}
func WithUnshelveServerHeader(key, value string) UnshelveServerOption {
	return request.WithHeader[UnshelveServerOpts](key, value)
}
func WithUnshelveServerField(key string, value any) UnshelveServerOption {
	return request.WithField[UnshelveServerOpts](key, value)
}

// FloatingIPServerOpts retains an explicitly empty fixed_address. Nil means
// omitted, matching fixed_address=None in Python.
type FloatingIPServerOpts struct {
	FixedAddress *string `json:"fixed_address"`
}
type FloatingIPServerOption = request.Option[FloatingIPServerOpts]

func WithFloatingIPFixedAddress(value string) FloatingIPServerOption {
	return func(c *request.Config[FloatingIPServerOpts]) error {
		owned := value
		c.Options.FixedAddress = &owned
		return nil
	}
}
func WithoutFloatingIPFixedAddress() FloatingIPServerOption {
	return func(c *request.Config[FloatingIPServerOpts]) error { c.Options.FixedAddress = nil; return nil }
}
func WithFloatingIPServerHeader(key, value string) FloatingIPServerOption {
	return request.WithHeader[FloatingIPServerOpts](key, value)
}
func WithFloatingIPServerField(key string, value any) FloatingIPServerOption {
	return request.WithField[FloatingIPServerOpts](key, value)
}

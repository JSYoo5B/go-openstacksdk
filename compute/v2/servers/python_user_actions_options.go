package servers

import (
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// UserActionOpts carries headers and JSON extensions for direct actions.
// The action methods own literal network/address inputs and prevent replacement.
type UserActionOpts struct{}

// UserActionOption is one option of the direct server actions:
// request.Header, request.Field (a JSON extension inside the action object;
// actions without a body reject it) or request.Custom over
// *request.Config[UserActionOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type UserActionOption = request.WriteCall

// LockServerOpts preserves an omitted reason and an explicitly empty reason.
// A nil reason matches locked_reason=None in Python.
type LockServerOpts struct {
	LockedReason *string `json:"locked_reason"`
}

// LockServerOption is one option of LockWithOptions: WithLockedReason,
// WithoutLockedReason, request.Header, request.Field or request.Custom over
// *request.Config[LockServerOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type LockServerOption = request.WriteCall

func WithLockedReason(value string) LockServerOption {
	return request.Custom(func(c *request.Config[LockServerOpts]) error {
		owned := value
		c.Options.LockedReason = &owned
		return nil
	})
}

// WithoutLockedReason restores the Python default when composing options.
func WithoutLockedReason() LockServerOption {
	return request.Custom(func(c *request.Config[LockServerOpts]) error { c.Options.LockedReason = nil; return nil })
}

// UnshelveServerOpts separates the default zone from an explicit null zone.
// ZoneSet=true, AvailabilityZone=nil unpins the zone (microversion 2.91).
// An empty Host is omitted, matching Python's truthiness check.
type UnshelveServerOpts struct {
	Host             string  `json:"host"`
	AvailabilityZone *string `json:"availability_zone"`
	ZoneSet          bool    `json:"-"`
}

// UnshelveServerOption is one option of UnshelveWithOptions: WithUnshelveHost,
// WithUnshelveAvailabilityZone, WithUnshelveUnpinnedAvailabilityZone,
// WithUnshelveDefaultAvailabilityZone, request.Header, request.Field or
// request.Custom over *request.Config[UnshelveServerOpts]. Other shared calls
// fail with resource.ErrInvalidOption before HTTP.
type UnshelveServerOption = request.WriteCall

func WithUnshelveHost(value string) UnshelveServerOption {
	return request.Custom(func(c *request.Config[UnshelveServerOpts]) error { c.Options.Host = value; return nil })
}
func WithUnshelveAvailabilityZone(value string) UnshelveServerOption {
	return request.Custom(func(c *request.Config[UnshelveServerOpts]) error {
		owned := value
		c.Options.AvailabilityZone = &owned
		c.Options.ZoneSet = true
		return nil
	})
}
func WithUnshelveUnpinnedAvailabilityZone() UnshelveServerOption {
	return request.Custom(func(c *request.Config[UnshelveServerOpts]) error {
		c.Options.AvailabilityZone = nil
		c.Options.ZoneSet = true
		return nil
	})
}
func WithUnshelveDefaultAvailabilityZone() UnshelveServerOption {
	return request.Custom(func(c *request.Config[UnshelveServerOpts]) error {
		c.Options.AvailabilityZone = nil
		c.Options.ZoneSet = false
		return nil
	})
}

// FloatingIPServerOpts retains an explicitly empty fixed_address. Nil means
// omitted, matching fixed_address=None in Python.
type FloatingIPServerOpts struct {
	FixedAddress *string `json:"fixed_address"`
}

// FloatingIPServerOption is one option of AddFloatingIP:
// WithFloatingIPFixedAddress, WithoutFloatingIPFixedAddress, request.Header,
// request.Field or request.Custom over *request.Config[FloatingIPServerOpts].
// Other shared calls fail with resource.ErrInvalidOption before HTTP.
type FloatingIPServerOption = request.WriteCall

func WithFloatingIPFixedAddress(value string) FloatingIPServerOption {
	return request.Custom(func(c *request.Config[FloatingIPServerOpts]) error {
		owned := value
		c.Options.FixedAddress = &owned
		return nil
	})
}
func WithoutFloatingIPFixedAddress() FloatingIPServerOption {
	return request.Custom(func(c *request.Config[FloatingIPServerOpts]) error { c.Options.FixedAddress = nil; return nil })
}

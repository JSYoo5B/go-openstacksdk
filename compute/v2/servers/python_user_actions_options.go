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

// LockServerOption is one option of LockWithOptions: request.Header,
// request.Field or request.Custom over *request.Config[LockServerOpts] (for
// example to set LockedReason, or to clear LockedReason). Other shared calls
// fail with resource.ErrInvalidOption before HTTP.
type LockServerOption = request.WriteCall

// UnshelveServerOpts separates the default zone from an explicit null zone.
// ZoneSet=true, AvailabilityZone=nil unpins the zone (microversion 2.91).
// An empty Host is omitted, matching Python's truthiness check.
type UnshelveServerOpts struct {
	Host             string  `json:"host"`
	AvailabilityZone *string `json:"availability_zone"`
	ZoneSet          bool    `json:"-"`
}

// UnshelveServerOption is one option of UnshelveWithOptions:
// WithUnshelveAvailabilityZone, WithUnshelveUnpinnedAvailabilityZone,
// WithUnshelveDefaultAvailabilityZone, request.Header, request.Field or
// request.Custom over *request.Config[UnshelveServerOpts] (for example to set
// Host). Other shared calls fail with resource.ErrInvalidOption before HTTP.
type UnshelveServerOption = request.WriteCall

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

// FloatingIPServerOption is one option of AddFloatingIP: request.Header,
// request.Field or request.Custom over *request.Config[FloatingIPServerOpts]
// (for example to set FixedAddress, or to clear FixedAddress). Other shared
// calls fail with resource.ErrInvalidOption before HTTP.
type FloatingIPServerOption = request.WriteCall

package network

import (
	"fmt"
	"net/netip"

	floatingipapi "gophercloudsdk/network/v2/extensions/layer3/floatingips"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// CreateFloatingIPRequest selects the external allocation network. A name is
// resolved exactly among external networks; an explicit ID is passed directly
// to Neutron, which validates that the network is external.
type CreateFloatingIPRequest struct{ Network resource.Ref }

type createFloatingIPOptions struct {
	server, port, destination *resource.Ref
	fixedAddress              string
	base                      floatingipapi.CreateOpts
	fields                    []floatingipapi.CreateOption
	wait                      bool
	waitOptions               []resource.WaitOption
}

type CreateFloatingIPOption func(*createFloatingIPOptions) error

// WithServer attaches the new floating IP to a server's uniquely selected
// fixed IPv4 address. Names use the configured Compute dependency resolver.
func WithServer(ref resource.Ref) CreateFloatingIPOption {
	return func(o *createFloatingIPOptions) error {
		if err := ref.Validate(); err != nil {
			return fmt.Errorf("server: %w", err)
		}
		o.server = &ref
		return nil
	}
}

// WithPort selects a particular Neutron port instead of listing server ports.
// It may be combined with WithServer to verify that the port belongs to that
// server. Fixed IPv4 ambiguity still requires WithFixedAddress.
func WithPort(ref resource.Ref) CreateFloatingIPOption {
	return func(o *createFloatingIPOptions) error {
		if err := ref.Validate(); err != nil {
			return fmt.Errorf("port: %w", err)
		}
		o.port = &ref
		return nil
	}
}

// WithNATDestination limits the destination port to a private network. It
// requires WithServer or WithPort and supports exact network names or IDs.
func WithNATDestination(ref resource.Ref) CreateFloatingIPOption {
	return func(o *createFloatingIPOptions) error {
		if err := ref.Validate(); err != nil {
			return fmt.Errorf("NAT destination: %w", err)
		}
		o.destination = &ref
		return nil
	}
}

// WithFixedAddress chooses an exact fixed IPv4 address on the destination port.
// IPv6 addresses and malformed strings are rejected before any API call.
func WithFixedAddress(address string) CreateFloatingIPOption {
	return func(o *createFloatingIPOptions) error {
		parsed, err := netip.ParseAddr(address)
		if err != nil || !parsed.Is4() {
			return floatingIPInvalid("fixed address must be a valid IPv4 address")
		}
		o.fixedAddress = parsed.String()
		return nil
	}
}

// WithDescription supplies the new floating IP's description.
func WithDescription(description string) CreateFloatingIPOption {
	return func(o *createFloatingIPOptions) error { o.base.Description = description; return nil }
}

// WithFloatingIPAddress requests a particular IPv4 address from the external
// network. Neutron validates subnet membership and allocation availability.
func WithFloatingIPAddress(address string) CreateFloatingIPOption {
	return func(o *createFloatingIPOptions) error {
		parsed, err := netip.ParseAddr(address)
		if err != nil || !parsed.Is4() {
			return floatingIPInvalid("floating address must be a valid IPv4 address")
		}
		o.base.FloatingIP = parsed.String()
		return nil
	}
}

// WithFloatingIPField adds an extension inside the floatingip object. Values
// are snapshotted, and core input fields cannot overwrite workflow selections.
func WithFloatingIPField(key string, value any) CreateFloatingIPOption {
	field := floatingipapi.WithCreateField(key, value)
	return func(o *createFloatingIPOptions) error {
		o.fields = append(o.fields, field)
		return nil
	}
}

// WithWait waits for ACTIVE after successful creation and association. Wait
// options are validated before any API call. A post-creation failure returns
// the created resource alongside the error and never deletes it automatically.
func WithWait(options ...resource.WaitOption) CreateFloatingIPOption {
	options = append([]resource.WaitOption(nil), options...)
	return func(o *createFloatingIPOptions) error {
		if err := resource.ValidateWaitOptionsFor[floatingipapi.FloatingIP](options...); err != nil {
			return err
		}
		o.wait, o.waitOptions = true, options
		return nil
	}
}

func floatingIPInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

func (o *createFloatingIPOptions) validateFields() error {
	configuration, err := request.Apply(o.base, o.fields...)
	if err != nil {
		return err
	}
	_, err = request.MergeFieldsFor(map[string]any{}, configuration.Fields, o.base)
	return err
}

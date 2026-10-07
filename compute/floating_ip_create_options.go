package compute

import (
	"context"
	"errors"
	"slices"
	"time"

	"gophercloudsdk/internal/cloudread"
	"gophercloudsdk/resource"
)

// A nil Network chooses the default network/pool. An explicit empty string
// chooses Neutron's default network but is a literal empty Nova pool.
type CreateFloatingIPRequest struct{ Network *string }

// Wait defaults to false. WaitTimeout defaults to60 seconds after allocation;
// zero/negative timeout immediately expires a consumed Neutron wait. An absent
// port or Nova backend ignores server/fixed/NAT/wait. Timeout is an optional
// whole-operation cap, independent of the source's post-allocation wait budget.
type FloatingIPCreateOpts struct {
	Server                               resource.Ref
	PortID, FixedAddress, NATDestination string
	Wait                                 bool
	WaitTimeout, WaitInterval            time.Duration
	Source                               *FloatingIPSource
	Location                             *resource.CloudLocation
	Strict, DirectGet                    bool
	Timeout                              time.Duration
}
type FloatingIPCreateOption func(*FloatingIPCreateOpts) error

func WithFloatingIPCreateOptions(value FloatingIPCreateOpts) FloatingIPCreateOption {
	owned := cloneFloatingIPCreateOptions(value)
	return func(o *FloatingIPCreateOpts) error { *o = cloneFloatingIPCreateOptions(owned); return nil }
}
func WithFloatingIPCreateServer(value resource.Ref) FloatingIPCreateOption {
	return func(o *FloatingIPCreateOpts) error { o.Server = value; return nil }
}
func WithFloatingIPCreatePort(value string) FloatingIPCreateOption {
	return func(o *FloatingIPCreateOpts) error { o.PortID = value; return nil }
}
func WithFloatingIPCreateFixedAddress(value string) FloatingIPCreateOption {
	return func(o *FloatingIPCreateOpts) error { o.FixedAddress = value; return nil }
}
func WithFloatingIPCreateNATDestination(value string) FloatingIPCreateOption {
	return func(o *FloatingIPCreateOpts) error { o.NATDestination = value; return nil }
}
func WithFloatingIPCreateWait(value bool) FloatingIPCreateOption {
	return func(o *FloatingIPCreateOpts) error { o.Wait = value; return nil }
}
func WithFloatingIPCreateWaitTimeout(value time.Duration) FloatingIPCreateOption {
	return func(o *FloatingIPCreateOpts) error { o.WaitTimeout = value; return nil }
}
func WithFloatingIPCreateWaitInterval(value time.Duration) FloatingIPCreateOption {
	return func(o *FloatingIPCreateOpts) error {
		if value <= 0 {
			return invalid("floating IP create wait interval must be positive")
		}
		o.WaitInterval = value
		return nil
	}
}
func WithFloatingIPCreateSource(value FloatingIPSource) FloatingIPCreateOption {
	return func(o *FloatingIPCreateOpts) error { copy := value; o.Source = &copy; return nil }
}
func WithFloatingIPCreateLocation(value resource.CloudLocation) FloatingIPCreateOption {
	owned := value.Clone()
	return func(o *FloatingIPCreateOpts) error { copy := owned.Clone(); o.Location = &copy; return nil }
}
func WithFloatingIPCreateStrict(value bool) FloatingIPCreateOption {
	return func(o *FloatingIPCreateOpts) error { o.Strict = value; return nil }
}
func WithFloatingIPCreateDirectGet(value bool) FloatingIPCreateOption {
	return func(o *FloatingIPCreateOpts) error { o.DirectGet = value; return nil }
}
func WithFloatingIPCreateTimeout(value time.Duration) FloatingIPCreateOption {
	return func(o *FloatingIPCreateOpts) error {
		if value <= 0 {
			return invalid("floating IP create timeout must be positive")
		}
		o.Timeout = value
		return nil
	}
}
func WithUnlimitedFloatingIPCreateTimeout() FloatingIPCreateOption {
	return func(o *FloatingIPCreateOpts) error { o.Timeout = 0; return nil }
}
func PrepareFloatingIPCreateOptions(ctx context.Context, options ...FloatingIPCreateOption) (FloatingIPCreateOpts, error) {
	return prepareFloatingIPCreateOptions(options, func() error { return cloudread.Context(ctx) })
}
func prepareFloatingIPCreateOptions(options []FloatingIPCreateOption, guard func() error) (FloatingIPCreateOpts, error) {
	value := FloatingIPCreateOpts{WaitTimeout: time.Minute}
	for _, option := range slices.Clone(options) {
		if err := guard(); err != nil {
			return FloatingIPCreateOpts{}, err
		}
		if option == nil {
			return FloatingIPCreateOpts{}, invalid("nil floating IP create option")
		}
		next := cloneFloatingIPCreateOptions(value)
		err := option(&next)
		value = cloneFloatingIPCreateOptions(next)
		if err = errors.Join(err, guard()); err != nil {
			return FloatingIPCreateOpts{}, err
		}
	}
	if value.WaitInterval < 0 {
		return FloatingIPCreateOpts{}, invalid("negative floating IP create wait interval")
	}
	if _, err := prepareFloatingIPQueryOptions([]FloatingIPQueryOption{WithFloatingIPQueryOptions(value.query())}, guard); err != nil {
		return FloatingIPCreateOpts{}, err
	}
	return cloneFloatingIPCreateOptions(value), guard()
}
func (value FloatingIPCreateOpts) query() FloatingIPQueryOpts {
	return FloatingIPQueryOpts{Source: value.Source, Location: value.Location, Strict: value.Strict, DirectGet: value.DirectGet, Timeout: value.Timeout}
}
func cloneFloatingIPCreateOptions(value FloatingIPCreateOpts) FloatingIPCreateOpts {
	copy := value
	if value.Source != nil {
		source := *value.Source
		copy.Source = &source
	}
	if value.Location != nil {
		location := value.Location.Clone()
		copy.Location = &location
	}
	return copy
}

package network

import (
	"context"
	"errors"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// AttachFloatingIPRequest selects an existing IP and a server. IP uses an
// explicit resource ID or Name(floating IPv4 address); there is no ID guessing.
type AttachFloatingIPRequest struct {
	Server resource.Ref
	IP     resource.Ref
}

type attachFloatingIPOptions struct{ destination createFloatingIPOptions }
type AttachFloatingIPOption func(*attachFloatingIPOptions) error

// AttachFloatingIPPolicy owns reusable destination and readiness options.
// Existing-IP attachment has no allocation, reuse or current-project policy.
type AttachFloatingIPPolicy struct {
	options  attachFloatingIPOptions
	prepared bool
}

func PrepareAttachFloatingIPOptions(ctx context.Context, options ...AttachFloatingIPOption) (AttachFloatingIPPolicy, error) {
	var policy AttachFloatingIPPolicy
	if ctx == nil {
		return policy, floatingIPInvalid("context is required")
	}
	if err := errors.Join(ctx.Err(), context.Cause(ctx)); err != nil {
		return policy, err
	}
	for _, option := range append([]AttachFloatingIPOption(nil), options...) {
		if option == nil {
			return policy, floatingIPInvalid("nil attach floating IP option")
		}
		err := option(&policy.options)
		if err = errors.Join(err, ctx.Err(), context.Cause(ctx)); err != nil {
			return policy, err
		}
	}
	policy.options.destination = cloneAttachDestination(policy.options.destination)
	policy.prepared = true
	return policy, nil
}

func cloneAttachDestination(source createFloatingIPOptions) createFloatingIPOptions {
	var result createFloatingIPOptions
	if source.port != nil {
		ref := *source.port
		result.port = &ref
	}
	if source.destination != nil {
		ref := *source.destination
		result.destination = &ref
	}
	result.fixedAddress, result.wait = source.fixedAddress, source.wait
	result.waitOptions = append([]resource.WaitOption(nil), source.waitOptions...)
	return result
}

func WithAttachFloatingIPPolicy(policy AttachFloatingIPPolicy) AttachFloatingIPOption {
	return func(o *attachFloatingIPOptions) error {
		if !policy.prepared {
			return floatingIPInvalid("attach floating IP policy must be prepared")
		}
		o.destination = cloneAttachDestination(policy.options.destination)
		return nil
	}
}

// WithAttachDestinationPolicy shares the destination/wait policy of Ensure
// without applying callbacks again or carrying its allocation/project options.
func WithAttachDestinationPolicy(policy EnsureFloatingIPPolicy) AttachFloatingIPOption {
	return func(o *attachFloatingIPOptions) error {
		if !policy.prepared {
			return floatingIPInvalid("ensure floating IP policy must be prepared")
		}
		o.destination = cloneAttachDestination(policy.options.destination)
		return nil
	}
}

func attachDestinationOption(option CreateFloatingIPOption) AttachFloatingIPOption {
	return func(o *attachFloatingIPOptions) error { return option(&o.destination) }
}

func WithAttachPort(ref resource.Ref) AttachFloatingIPOption {
	return attachDestinationOption(WithPort(ref))
}

func WithAttachNATDestination(ref resource.Ref) AttachFloatingIPOption {
	return attachDestinationOption(WithNATDestination(ref))
}

func WithAttachFixedAddress(address string) AttachFloatingIPOption {
	return attachDestinationOption(WithFixedAddress(address))
}

func WithAttachWait(options ...resource.WaitOption) AttachFloatingIPOption {
	return attachDestinationOption(WithWait(options...))
}

func WithAttachActive() AttachFloatingIPOption {
	return func(o *attachFloatingIPOptions) error { o.destination.wait = true; return nil }
}

func WithAttachNoWait() AttachFloatingIPOption {
	return func(o *attachFloatingIPOptions) error { o.destination.wait = false; return nil }
}

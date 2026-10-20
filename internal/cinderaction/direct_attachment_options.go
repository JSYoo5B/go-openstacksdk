package cinderaction

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// DirectAttachOptions preserves selector presence. Instance wins over HostName,
// including an explicitly empty instance. Only the selected value is consumed.
type DirectAttachOptions struct{ Instance, HostName *string }
type DirectAttachOption func(*DirectAttachOptions) error

func cloneDirectAttach(value DirectAttachOptions) DirectAttachOptions {
	value.Instance = clonePointer(value.Instance)
	value.HostName = clonePointer(value.HostName)
	return value
}

func WithDirectAttachOptions(value DirectAttachOptions) DirectAttachOption {
	owned := cloneDirectAttach(value)
	return func(target *DirectAttachOptions) error { *target = cloneDirectAttach(owned); return nil }
}

func WithDirectAttachInstance(value string) DirectAttachOption {
	return func(target *DirectAttachOptions) error { target.Instance = clonePointer(&value); return nil }
}

func WithDirectAttachHostName(value string) DirectAttachOption {
	return func(target *DirectAttachOptions) error { target.HostName = clonePointer(&value); return nil }
}

func PrepareDirectAttach(ctx context.Context, options ...DirectAttachOption) (DirectAttachOptions, error) {
	return prepareDirectAttach(options, func() error { return cloudread.Context(ctx) })
}

func prepareDirectAttach(options []DirectAttachOption, guard func() error) (DirectAttachOptions, error) {
	policy, err := prepareOptions(options, cloneDirectAttach, func(*DirectAttachOptions) {}, guard)
	if err != nil {
		return DirectAttachOptions{}, err
	}
	selected := policy.Instance
	if selected == nil {
		selected = policy.HostName
	}
	if selected == nil {
		return DirectAttachOptions{}, fmt.Errorf("%w: instance or host name is required", resource.ErrInvalidOption)
	}
	if err := validateBodyStrings(selected); err != nil {
		return DirectAttachOptions{}, err
	}
	return policy, nil
}

// DirectDetachOptions defaults Force to false. Connector is owned JSON data;
// only a nonempty connector in the force branch is validated and transmitted.
type DirectDetachOptions struct {
	Force     *bool
	Connector map[string]json.RawMessage
}
type DirectDetachOption func(*DirectDetachOptions) error

func cloneConnector(value map[string]json.RawMessage) map[string]json.RawMessage {
	if value == nil {
		return nil
	}
	owned := make(map[string]json.RawMessage, len(value))
	for key, member := range value {
		owned[key] = bytes.Clone(member)
	}
	return owned
}

func cloneDirectDetach(value DirectDetachOptions) DirectDetachOptions {
	value.Force = clonePointer(value.Force)
	value.Connector = cloneConnector(value.Connector)
	return value
}

func WithDirectDetachOptions(value DirectDetachOptions) DirectDetachOption {
	owned := cloneDirectDetach(value)
	return func(target *DirectDetachOptions) error { *target = cloneDirectDetach(owned); return nil }
}

func WithDirectDetachForce(value bool) DirectDetachOption {
	return func(target *DirectDetachOptions) error { target.Force = clonePointer(&value); return nil }
}

func WithDirectDetachConnector(value map[string]json.RawMessage) DirectDetachOption {
	owned := cloneConnector(value)
	return func(target *DirectDetachOptions) error { target.Connector = cloneConnector(owned); return nil }
}

func PrepareDirectDetach(ctx context.Context, options ...DirectDetachOption) (DirectDetachOptions, error) {
	return prepareDirectDetach(options, func() error { return cloudread.Context(ctx) })
}

func prepareDirectDetach(options []DirectDetachOption, guard func() error) (DirectDetachOptions, error) {
	policy, err := prepareOptions(options, cloneDirectDetach, func(value *DirectDetachOptions) {
		if value.Force == nil {
			value.Force = new(bool)
		}
	}, guard)
	if err != nil {
		return DirectDetachOptions{}, err
	}
	if *policy.Force {
		for key, member := range policy.Connector {
			if !utf8.ValidString(key) || !utf8.Valid(member) || !json.Valid(member) {
				return DirectDetachOptions{}, fmt.Errorf("%w: force detach connector requires UTF-8 keys and JSON values", resource.ErrInvalidOption)
			}
		}
	}
	return policy, nil
}

package cinderaction

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"gophercloudsdk/internal/cloudread"
	"gophercloudsdk/resource"
)

// ReadonlyOptions owns the optional flag. Nil selects the Proxy default true.
type ReadonlyOptions struct{ Readonly *bool }
type ReadonlyOption func(*ReadonlyOptions) error

func cloneReadonly(value ReadonlyOptions) ReadonlyOptions {
	if value.Readonly != nil {
		owned := *value.Readonly
		value.Readonly = &owned
	}
	return value
}

func WithReadonlyOptions(value ReadonlyOptions) ReadonlyOption {
	owned := cloneReadonly(value)
	return func(target *ReadonlyOptions) error { *target = cloneReadonly(owned); return nil }
}
func WithReadonly(value bool) ReadonlyOption {
	return func(target *ReadonlyOptions) error { owned := value; target.Readonly = &owned; return nil }
}

// PrepareReadonly runs originals once without selecting a service or doing HTTP.
func PrepareReadonly(ctx context.Context, options ...ReadonlyOption) (ReadonlyOptions, error) {
	return prepareReadonly(options, func() error { return cloudread.Context(ctx) })
}

func prepareReadonly(options []ReadonlyOption, guard func() error) (ReadonlyOptions, error) {
	options = slices.Clone(options)
	value := ReadonlyOptions{}
	if err := guard(); err != nil {
		return value, err
	}
	for _, option := range options {
		if err := guard(); err != nil {
			return ReadonlyOptions{}, err
		}
		if option == nil {
			return ReadonlyOptions{}, fmt.Errorf("%w: nil volume readonly option", resource.ErrInvalidOption)
		}
		next := cloneReadonly(value)
		err := option(&next)
		value = cloneReadonly(next)
		if err = errors.Join(err, guard()); err != nil {
			return ReadonlyOptions{}, err
		}
	}
	if value.Readonly == nil {
		yes := true
		value.Readonly = &yes
	}
	return cloneReadonly(value), guard()
}

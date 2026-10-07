package cinderaction

import (
	"context"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudread"
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
	return prepareOptions(options, cloneReadonly, func(value *ReadonlyOptions) {
		if value.Readonly == nil {
			yes := true
			value.Readonly = &yes
		}
	}, guard)
}

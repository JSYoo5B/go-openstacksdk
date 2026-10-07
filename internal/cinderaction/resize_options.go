package cinderaction

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudread"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// RetypeOptions distinguishes the default never from an explicit empty policy.
type RetypeOptions struct{ MigrationPolicy *string }
type RetypeOption func(*RetypeOptions) error

func cloneRetype(value RetypeOptions) RetypeOptions {
	if value.MigrationPolicy != nil {
		owned := *value.MigrationPolicy
		value.MigrationPolicy = &owned
	}
	return value
}
func WithRetypeOptions(value RetypeOptions) RetypeOption {
	owned := cloneRetype(value)
	return func(target *RetypeOptions) error { *target = cloneRetype(owned); return nil }
}
func WithRetypeMigrationPolicy(value string) RetypeOption {
	return func(target *RetypeOptions) error { owned := value; target.MigrationPolicy = &owned; return nil }
}
func PrepareRetype(ctx context.Context, options ...RetypeOption) (RetypeOptions, error) {
	return prepareRetype(options, func() error { return cloudread.Context(ctx) })
}
func prepareRetype(options []RetypeOption, guard func() error) (RetypeOptions, error) {
	policy, err := prepareOptions(options, cloneRetype, func(value *RetypeOptions) {
		if value.MigrationPolicy == nil {
			never := "never"
			value.MigrationPolicy = &never
		}
	}, guard)
	if err != nil {
		return RetypeOptions{}, err
	}
	if !utf8.ValidString(*policy.MigrationPolicy) {
		return RetypeOptions{}, fmt.Errorf("%w: volume migration policy must be UTF-8", resource.ErrInvalidOption)
	}
	return policy, nil
}

// ExtendCompletionOptions defaults Error to false and always transmits it.
type ExtendCompletionOptions struct{ Error *bool }
type ExtendCompletionOption func(*ExtendCompletionOptions) error

func cloneExtendCompletion(value ExtendCompletionOptions) ExtendCompletionOptions {
	if value.Error != nil {
		owned := *value.Error
		value.Error = &owned
	}
	return value
}
func WithExtendCompletionOptions(value ExtendCompletionOptions) ExtendCompletionOption {
	owned := cloneExtendCompletion(value)
	return func(target *ExtendCompletionOptions) error { *target = cloneExtendCompletion(owned); return nil }
}
func WithExtendCompletionError(value bool) ExtendCompletionOption {
	return func(target *ExtendCompletionOptions) error { owned := value; target.Error = &owned; return nil }
}
func PrepareExtendCompletion(ctx context.Context, options ...ExtendCompletionOption) (ExtendCompletionOptions, error) {
	return prepareExtendCompletion(options, func() error { return cloudread.Context(ctx) })
}
func prepareExtendCompletion(options []ExtendCompletionOption, guard func() error) (ExtendCompletionOptions, error) {
	return prepareOptions(options, cloneExtendCompletion, func(value *ExtendCompletionOptions) {
		if value.Error == nil {
			no := false
			value.Error = &no
		}
	}, guard)
}

package tokens

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ValidateTokenOpts selects optional Python validate_token query arguments.
// False omits the corresponding parameter; later options replace earlier ones.
type ValidateTokenOpts struct {
	NoCatalog    bool
	AllowExpired bool
}

type ValidateTokenOption func(*ValidateTokenOpts) error

// WithValidateTokenOpts captures and replaces the complete policy.
func WithValidateTokenOpts(value ValidateTokenOpts) ValidateTokenOption {
	return func(config *ValidateTokenOpts) error { *config = value; return nil }
}

func WithValidateTokenNoCatalog(value bool) ValidateTokenOption {
	return func(config *ValidateTokenOpts) error { config.NoCatalog = value; return nil }
}

func WithValidateTokenAllowExpired(value bool) ValidateTokenOption {
	return func(config *ValidateTokenOpts) error { config.AllowExpired = value; return nil }
}

// CheckTokenOpts selects the optional Python check_token query argument.
type CheckTokenOpts struct{ AllowExpired bool }
type CheckTokenOption func(*CheckTokenOpts) error

func WithCheckTokenOpts(value CheckTokenOpts) CheckTokenOption {
	return func(config *CheckTokenOpts) error { *config = value; return nil }
}

func WithCheckTokenAllowExpired(value bool) CheckTokenOption {
	return func(config *CheckTokenOpts) error { config.AllowExpired = value; return nil }
}

func prepareTokenOptions[T any](ctx context.Context, check func(context.Context) error, options []func(*T) error) (T, error) {
	var config T
	for _, apply := range slices.Clone(options) {
		if err := check(ctx); err != nil {
			return config, err
		}
		if apply == nil {
			return config, fmt.Errorf("%w: nil token option", resource.ErrInvalidOption)
		}
		// Do not retain the pointer handed to a callback. Concrete options have
		// value fields, so this copy also owns every policy value after it returns.
		next := config
		callbackErr := apply(&next)
		if err := errors.Join(callbackErr, check(ctx)); err != nil {
			return config, err
		}
		config = next
	}
	return config, check(ctx)
}

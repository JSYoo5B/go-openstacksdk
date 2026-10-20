package compute

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type DeleteFloatingIPRequest struct{ ID string }

// Retries defaults to one additional delete. Zero disables verification;
// negative values perform one delete with verification, following cloud.
// Timeout defaults to no additional SDK cap and covers all attempts/queries.
type FloatingIPDeleteOpts struct {
	Retries   int
	Source    *FloatingIPSource
	Location  *resource.CloudLocation
	Strict    bool
	DirectGet bool
	Timeout   time.Duration
}
type FloatingIPDeleteOption func(*FloatingIPDeleteOpts) error

func WithFloatingIPDeleteOptions(value FloatingIPDeleteOpts) FloatingIPDeleteOption {
	owned := cloneFloatingIPDeleteOptions(value)
	return func(o *FloatingIPDeleteOpts) error { *o = cloneFloatingIPDeleteOptions(owned); return nil }
}
func WithFloatingIPDeleteRetries(value int) FloatingIPDeleteOption {
	return func(o *FloatingIPDeleteOpts) error { o.Retries = value; return nil }
}
func WithFloatingIPDeleteSource(value FloatingIPSource) FloatingIPDeleteOption {
	return func(o *FloatingIPDeleteOpts) error { copy := value; o.Source = &copy; return nil }
}
func WithFloatingIPDeleteLocation(value resource.CloudLocation) FloatingIPDeleteOption {
	owned := value.Clone()
	return func(o *FloatingIPDeleteOpts) error { copy := owned.Clone(); o.Location = &copy; return nil }
}
func WithFloatingIPDeleteStrict(value bool) FloatingIPDeleteOption {
	return func(o *FloatingIPDeleteOpts) error { o.Strict = value; return nil }
}
func WithFloatingIPDeleteDirectGet(value bool) FloatingIPDeleteOption {
	return func(o *FloatingIPDeleteOpts) error { o.DirectGet = value; return nil }
}
func WithFloatingIPDeleteTimeout(value time.Duration) FloatingIPDeleteOption {
	return func(o *FloatingIPDeleteOpts) error {
		if value <= 0 {
			return invalid("floating IP delete timeout must be positive")
		}
		o.Timeout = value
		return nil
	}
}
func WithUnlimitedFloatingIPDeleteTimeout() FloatingIPDeleteOption {
	return func(o *FloatingIPDeleteOpts) error { o.Timeout = 0; return nil }
}
func PrepareFloatingIPDeleteOptions(ctx context.Context, options ...FloatingIPDeleteOption) (FloatingIPDeleteOpts, error) {
	return prepareFloatingIPDeleteOptions(options, func() error { return cloudread.Context(ctx) })
}
func prepareFloatingIPDeleteOptions(options []FloatingIPDeleteOption, guard func() error) (FloatingIPDeleteOpts, error) {
	value := FloatingIPDeleteOpts{Retries: 1}
	for _, option := range slices.Clone(options) {
		if err := guard(); err != nil {
			return FloatingIPDeleteOpts{}, err
		}
		if option == nil {
			return FloatingIPDeleteOpts{}, invalid("nil floating IP delete option")
		}
		next := cloneFloatingIPDeleteOptions(value)
		err := option(&next)
		value = cloneFloatingIPDeleteOptions(next)
		if err = errors.Join(err, guard()); err != nil {
			return FloatingIPDeleteOpts{}, err
		}
	}
	if _, err := prepareFloatingIPQueryOptions([]FloatingIPQueryOption{WithFloatingIPQueryOptions(value.query())}, guard); err != nil {
		return FloatingIPDeleteOpts{}, err
	}
	return cloneFloatingIPDeleteOptions(value), guard()
}
func (value FloatingIPDeleteOpts) query() FloatingIPQueryOpts {
	return FloatingIPQueryOpts{Source: value.Source, Location: value.Location, Strict: value.Strict, DirectGet: value.DirectGet, Timeout: value.Timeout}
}
func cloneFloatingIPDeleteOptions(value FloatingIPDeleteOpts) FloatingIPDeleteOpts {
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

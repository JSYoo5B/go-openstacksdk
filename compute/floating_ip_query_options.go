package compute

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudread"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

type SearchFloatingIPsRequest struct{ ID string }
type GetFloatingIPByIDRequest struct{ ID string }
type GetFloatingIPRequest struct {
	ID string
	// Existing preserves the supplied object, following cloud entity lookup.
	Existing *FloatingIPRecord
}
type SearchFloatingIPPoolsRequest struct{ Name string }

// FloatingIPQueryOpts owns JSON filters and normalization facts. List consumes
// dictionary filters as Neutron parameters; search preserves the source's
// dictionary-versus-local-expression dispatch. Defaults add no SDK deadline.
type FloatingIPQueryOpts struct {
	Source    *FloatingIPSource
	Filters   *json.RawMessage
	Location  *resource.CloudLocation
	Strict    bool
	DirectGet bool
	Timeout   time.Duration
}
type FloatingIPQueryOption func(*FloatingIPQueryOpts) error

func WithFloatingIPQueryOptions(value FloatingIPQueryOpts) FloatingIPQueryOption {
	owned := cloneFloatingIPQueryOptions(value)
	return func(o *FloatingIPQueryOpts) error { *o = cloneFloatingIPQueryOptions(owned); return nil }
}
func WithFloatingIPQuerySource(source FloatingIPSource) FloatingIPQueryOption {
	return func(o *FloatingIPQueryOpts) error { copy := source; o.Source = &copy; return nil }
}
func WithFloatingIPQueryFilters(value json.RawMessage) FloatingIPQueryOption {
	owned := bytes.Clone(value)
	return func(o *FloatingIPQueryOpts) error {
		copy := json.RawMessage(bytes.Clone(owned))
		o.Filters = &copy
		return nil
	}
}
func WithFloatingIPQueryExpression(expression string) FloatingIPQueryOption {
	raw, _ := json.Marshal(expression)
	return WithFloatingIPQueryFilters(raw)
}
func WithFloatingIPQueryLocation(value resource.CloudLocation) FloatingIPQueryOption {
	owned := value.Clone()
	return func(o *FloatingIPQueryOpts) error { copy := owned.Clone(); o.Location = &copy; return nil }
}
func WithFloatingIPQueryStrict(strict bool) FloatingIPQueryOption {
	return func(o *FloatingIPQueryOpts) error { o.Strict = strict; return nil }
}
func WithFloatingIPQueryDirectGet(enabled bool) FloatingIPQueryOption {
	return func(o *FloatingIPQueryOpts) error { o.DirectGet = enabled; return nil }
}
func WithFloatingIPQueryTimeout(timeout time.Duration) FloatingIPQueryOption {
	return func(o *FloatingIPQueryOpts) error {
		if timeout <= 0 {
			return invalid("floating IP query timeout must be positive")
		}
		o.Timeout = timeout
		return nil
	}
}
func WithUnlimitedFloatingIPQueryTimeout() FloatingIPQueryOption {
	return func(o *FloatingIPQueryOpts) error { o.Timeout = 0; return nil }
}

func PrepareFloatingIPQueryOptions(ctx context.Context, options ...FloatingIPQueryOption) (FloatingIPQueryOpts, error) {
	return prepareFloatingIPQueryOptions(options, func() error { return cloudread.Context(ctx) })
}
func prepareFloatingIPQueryOptions(options []FloatingIPQueryOption, guard func() error) (FloatingIPQueryOpts, error) {
	var value FloatingIPQueryOpts
	for _, option := range slices.Clone(options) {
		if err := guard(); err != nil {
			return FloatingIPQueryOpts{}, err
		}
		if option == nil {
			return FloatingIPQueryOpts{}, invalid("nil floating IP query option")
		}
		next := cloneFloatingIPQueryOptions(value)
		err := option(&next)
		value = cloneFloatingIPQueryOptions(next)
		if err = errors.Join(err, guard()); err != nil {
			return FloatingIPQueryOpts{}, err
		}
	}
	if value.Source != nil {
		if _, err := PrepareServerAddressPolicy(WithFloatingIPSource(*value.Source)); err != nil {
			return FloatingIPQueryOpts{}, err
		}
	}
	if value.Timeout < 0 {
		return FloatingIPQueryOpts{}, invalid("negative floating IP query timeout")
	}
	return cloneFloatingIPQueryOptions(value), guard()
}
func cloneFloatingIPQueryOptions(value FloatingIPQueryOpts) FloatingIPQueryOpts {
	copy := value
	if value.Source != nil {
		source := *value.Source
		copy.Source = &source
	}
	if value.Filters != nil {
		raw := json.RawMessage(bytes.Clone(*value.Filters))
		copy.Filters = &raw
	}
	if value.Location != nil {
		location := value.Location.Clone()
		copy.Location = &location
	}
	return copy
}

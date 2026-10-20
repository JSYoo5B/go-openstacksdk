// Package cloudlimits implements the library-owned Cinder limits workflow.
package cloudlimits

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type Input struct{ NameOrID string }
type Options struct{ Location *resource.CloudLocation }
type Option func(*Options) error

type Page struct {
	Body       json.RawMessage
	Header     http.Header
	StatusCode int
}

type ProjectResult struct {
	Project  *resource.RawResource
	ID       json.RawMessage
	SeededID bool
	Observed *Page
	Pages    []*Page
}

type Result struct {
	Project            *ProjectResult
	RequestedProjectID json.RawMessage
	Observed           *Page
	Limits             *resource.RawResource
	Value              json.RawMessage
}

func cloneOptions(value Options) Options {
	if value.Location != nil {
		copy := value.Location.Clone()
		value.Location = &copy
	}
	return value
}

func WithOptions(value Options) Option {
	owned := cloneOptions(value)
	return func(target *Options) error {
		if target == nil {
			return fmt.Errorf("%w: limits options are required", resource.ErrInvalidOption)
		}
		*target = cloneOptions(owned)
		return nil
	}
}

func WithLocation(value resource.CloudLocation) Option {
	owned := value.Clone()
	return func(target *Options) error {
		if target == nil {
			return fmt.Errorf("%w: limits options are required", resource.ErrInvalidOption)
		}
		copy := owned.Clone()
		target.Location = &copy
		return nil
	}
}

func Prepare(ctx context.Context, options []Option, guard func(context.Context) error) (Options, error) {
	if err := cloudread.Context(ctx); err != nil {
		return Options{}, err
	}
	check := func() error {
		if guard != nil {
			return guard(ctx)
		}
		return cloudread.Context(ctx)
	}
	var value Options
	for _, apply := range slices.Clone(options) {
		if err := check(); err != nil {
			return Options{}, err
		}
		if apply == nil {
			return Options{}, fmt.Errorf("%w: nil limits option", resource.ErrInvalidOption)
		}
		err := apply(&value)
		value = cloneOptions(value)
		if err = errors.Join(err, check()); err != nil {
			return Options{}, cloudread.ContextError(ctx, err)
		}
	}
	return cloneOptions(value), check()
}

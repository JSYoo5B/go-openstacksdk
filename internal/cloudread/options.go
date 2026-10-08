package cloudread

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"slices"

	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// OwnReadConfig snapshots the mutable request carriers around an option
// callback. Bindings still own their typed options and argument payloads;
// arbitrary application values cannot be deep-copied by the common reader.
func OwnReadConfig[T any](config *request.Config[T]) {
	query := make(url.Values, len(config.Query))
	for key, values := range config.Query {
		query[key] = slices.Clone(values)
	}
	config.Query = query
	config.Headers = maps.Clone(config.Headers)
	if config.Headers == nil {
		config.Headers = make(map[string]string)
	}
	fields := make(map[string]json.RawMessage, len(config.Fields))
	for key, raw := range config.Fields {
		fields[key] = bytes.Clone(raw)
	}
	config.Fields = fields
	config.Arguments = maps.Clone(config.Arguments)
	if config.Arguments == nil {
		config.Arguments = make(map[string]any)
	}
}

// ApplyReadOptions checks an operation around each callback and owns its
// mutable inputs at both boundaries. The binding supplies typed ownership;
// service capabilities remain a separate binding decision.
func ApplyReadOptions[T any](ctx context.Context, base T, options []request.Option[T], own func(*request.Config[T]), guard func(context.Context) error) (request.Config[T], error) {
	check := func() error {
		if err := Context(ctx); err != nil {
			return err
		}
		if guard != nil {
			return guard(ctx)
		}
		return nil
	}
	guarded := make([]request.Option[T], len(options))
	for index, option := range slices.Clone(options) {
		apply := option
		guarded[index] = func(config *request.Config[T]) error {
			if err := check(); err != nil {
				return err
			}
			if apply == nil {
				return fmt.Errorf("%w: nil read option", resource.ErrInvalidOption)
			}
			own(config)
			err := apply(config)
			own(config)
			return errors.Join(err, check())
		}
	}
	config, err := request.Apply(base, guarded...)
	return config, errors.Join(err, check())
}

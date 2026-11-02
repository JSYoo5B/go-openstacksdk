package cloudsnapshot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type ReadOptions struct{ Location *resource.CloudLocation }
type SearchOptions struct {
	Filters  *json.RawMessage
	Location *resource.CloudLocation
}

// ListOptions separates list controls from server/local raw dictionary filters.
// Nil Detailed and Paginated select true. Explicit typed controls override
// their counterparts in Filters without giving callers route/decoder ownership.
type ListOptions struct {
	Detailed           *bool
	Filters            *json.RawMessage
	Location           *resource.CloudLocation
	Paginated          *bool
	MaxItems           *int
	Microversion       *string
	Headers            map[string]string
	Expression         *string
	AllowUnknownParams *bool
	ConflictingAttrs   *json.RawMessage
}

type Option[T any] func(*T) error
type ReadOption = Option[ReadOptions]
type SearchOption = Option[SearchOptions]

// ListOption is one option of the snapshot and backup lists: WithListOptions,
// WithListLocation, WithListFilters, WithListDetailed, WithListExpression,
// WithListAllowUnknownParams, WithListConflictingAttrs, request.Custom over
// *ListOptions, request.Paginated, request.MaxItems, request.Microversion or
// request.Headers (replaces the headers). Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type ListOption = request.ListCall

var listCalls = calls.Spec[ListOptions]{
	Headers:      calls.Value(func(o *ListOptions) *map[string]string { return &o.Headers }),
	Microversion: calls.Pointer(func(o *ListOptions) **string { return &o.Microversion }),
	Paging: calls.Paging[ListOptions]{
		MaxItems:  calls.Pointer(func(o *ListOptions) **int { return &o.MaxItems }),
		Paginated: calls.Pointer(func(o *ListOptions) **bool { return &o.Paginated }),
	},
}

func WithReadOptions(value ReadOptions) ReadOption       { return withValue(value, cloneRead) }
func WithSearchOptions(value SearchOptions) SearchOption { return withValue(value, cloneSearch) }
func WithListOptions(value ListOptions) ListOption {
	return request.Custom(withValue(value, cloneList))
}

func WithReadLocation(value resource.CloudLocation) ReadOption {
	owned := value.Clone()
	return func(target *ReadOptions) error { copy := owned.Clone(); target.Location = &copy; return nil }
}
func WithSearchLocation(value resource.CloudLocation) SearchOption {
	owned := value.Clone()
	return func(target *SearchOptions) error { copy := owned.Clone(); target.Location = &copy; return nil }
}
func WithListLocation(value resource.CloudLocation) ListOption {
	owned := value.Clone()
	return request.Custom(func(target *ListOptions) error { copy := owned.Clone(); target.Location = &copy; return nil })
}
func WithSearchFilters(value json.RawMessage) SearchOption {
	owned := json.RawMessage(bytes.Clone(value))
	return func(target *SearchOptions) error { target.Filters = cloneRaw(&owned); return nil }
}
func WithSearchExpression(value string) SearchOption {
	raw, _ := json.Marshal(value)
	return WithSearchFilters(raw)
}
func WithListFilters(value json.RawMessage) ListOption {
	owned := json.RawMessage(bytes.Clone(value))
	return request.Custom(func(target *ListOptions) error { target.Filters = cloneRaw(&owned); return nil })
}
func WithListDetailed(value bool) ListOption {
	return request.Custom(func(target *ListOptions) error { target.Detailed = clonePointer(&value); return nil })
}
func WithListExpression(value string) ListOption {
	return request.Custom(func(target *ListOptions) error { target.Expression = clonePointer(&value); return nil })
}
func WithListAllowUnknownParams(value bool) ListOption {
	return request.Custom(func(target *ListOptions) error { target.AllowUnknownParams = clonePointer(&value); return nil })
}
func WithListConflictingAttrs(value json.RawMessage) ListOption {
	owned := json.RawMessage(bytes.Clone(value))
	return request.Custom(func(target *ListOptions) error { target.ConflictingAttrs = cloneRaw(&owned); return nil })
}

func PrepareRead(ctx context.Context, options ...ReadOption) (ReadOptions, error) {
	return prepare(ctx, options, cloneRead, nil)
}
func PrepareSearch(ctx context.Context, options ...SearchOption) (SearchOptions, error) {
	return prepare(ctx, options, cloneSearch, nil)
}
func PrepareList(ctx context.Context, options ...ListOption) (ListOptions, error) {
	return prepare(ctx, listOptions(options), cloneList, nil)
}

// listOptions translates shared list calls in caller order; a nil call stays
// nil so prepare reports it with the Cinder nil-option error.
func listOptions(options []ListOption) []Option[ListOptions] {
	translated := calls.Translate(options, listCalls)
	out := make([]Option[ListOptions], len(translated))
	for i, apply := range translated {
		if apply != nil {
			out[i] = apply
		}
	}
	return out
}

func withValue[T any](value T, clone func(T) T) Option[T] {
	owned := clone(value)
	return func(target *T) error { *target = clone(owned); return nil }
}
func prepare[T any](ctx context.Context, options []Option[T], clone func(T) T, guard func() error) (T, error) {
	var value T
	if err := cloudread.Context(ctx); err != nil {
		return value, err
	}
	check := func() error {
		if guard != nil {
			return cloudread.ContextError(ctx, guard())
		}
		return cloudread.Context(ctx)
	}
	options = slices.Clone(options)
	if err := check(); err != nil {
		return value, err
	}
	for _, option := range options {
		if err := check(); err != nil {
			var zero T
			return zero, err
		}
		if option == nil {
			var zero T
			return zero, fmt.Errorf("%w: nil Cinder resource option", resource.ErrInvalidOption)
		}
		next := clone(value)
		err := option(&next)
		value = clone(next)
		if err = errors.Join(err, check()); err != nil {
			var zero T
			return zero, err
		}
	}
	return clone(value), check()
}
func cloneRead(value ReadOptions) ReadOptions {
	value.Location = cloneLocation(value.Location)
	return value
}
func cloneSearch(value SearchOptions) SearchOptions {
	value.Filters = cloneRaw(value.Filters)
	value.Location = cloneLocation(value.Location)
	return value
}
func cloneList(value ListOptions) ListOptions {
	value.Detailed = clonePointer(value.Detailed)
	value.Filters = cloneRaw(value.Filters)
	value.Location = cloneLocation(value.Location)
	value.Paginated = clonePointer(value.Paginated)
	value.MaxItems = clonePointer(value.MaxItems)
	value.Microversion = clonePointer(value.Microversion)
	value.Headers = maps.Clone(value.Headers)
	value.Expression = clonePointer(value.Expression)
	value.AllowUnknownParams = clonePointer(value.AllowUnknownParams)
	value.ConflictingAttrs = cloneRaw(value.ConflictingAttrs)
	return value
}
func cloneLocation(value *resource.CloudLocation) *resource.CloudLocation {
	if value == nil {
		return nil
	}
	copy := value.Clone()
	return &copy
}
func cloneRaw(value *json.RawMessage) *json.RawMessage {
	if value == nil {
		return nil
	}
	copy := json.RawMessage(bytes.Clone(*value))
	return &copy
}
func clonePointer[T any](value *T) *T {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

package cloudsnapshot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
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
type ListOption = Option[ListOptions]

func WithReadOptions(value ReadOptions) ReadOption       { return withValue(value, cloneRead) }
func WithSearchOptions(value SearchOptions) SearchOption { return withValue(value, cloneSearch) }
func WithListOptions(value ListOptions) ListOption       { return withValue(value, cloneList) }

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
	return func(target *ListOptions) error { copy := owned.Clone(); target.Location = &copy; return nil }
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
	return func(target *ListOptions) error { target.Filters = cloneRaw(&owned); return nil }
}
func WithListDetailed(value bool) ListOption {
	return func(target *ListOptions) error { target.Detailed = clonePointer(&value); return nil }
}
func WithListPagination(value bool) ListOption {
	return func(target *ListOptions) error { target.Paginated = clonePointer(&value); return nil }
}
func WithListMaxItems(value int) ListOption {
	return func(target *ListOptions) error { target.MaxItems = clonePointer(&value); return nil }
}
func WithListMicroversion(value string) ListOption {
	return func(target *ListOptions) error { target.Microversion = clonePointer(&value); return nil }
}
func WithListHeaders(value map[string]string) ListOption {
	owned := maps.Clone(value)
	return func(target *ListOptions) error { target.Headers = maps.Clone(owned); return nil }
}
func WithListExpression(value string) ListOption {
	return func(target *ListOptions) error { target.Expression = clonePointer(&value); return nil }
}
func WithListAllowUnknownParams(value bool) ListOption {
	return func(target *ListOptions) error { target.AllowUnknownParams = clonePointer(&value); return nil }
}
func WithListConflictingAttrs(value json.RawMessage) ListOption {
	owned := json.RawMessage(bytes.Clone(value))
	return func(target *ListOptions) error { target.ConflictingAttrs = cloneRaw(&owned); return nil }
}

func PrepareRead(ctx context.Context, options ...ReadOption) (ReadOptions, error) {
	return prepare(ctx, options, cloneRead, nil)
}
func PrepareSearch(ctx context.Context, options ...SearchOption) (SearchOptions, error) {
	return prepare(ctx, options, cloneSearch, nil)
}
func PrepareList(ctx context.Context, options ...ListOption) (ListOptions, error) {
	return prepare(ctx, options, cloneList, nil)
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

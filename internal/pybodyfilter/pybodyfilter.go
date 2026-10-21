// Package pybodyfilter builds list Collections that apply pinned Python
// Resource.list client-side Body-attribute filters to native Gophercloud pages.
package pybodyfilter

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

// Spec describes one native list. Aliases maps each accepted Python attribute
// name to its canonical response field; every canonical field must map to
// itself. Rows are matched on their original JSON, so null and omission are
// equivalent and numbers keep their spelling.
type Spec[T any] struct {
	Kind     string
	Envelope string
	Pager    func() pagination.Pager
	Extract  func(pagination.Page) ([]T, error)
	ID, Name func(*T) string
	Get      func(context.Context, string) (*T, error)
	Aliases  map[string]string
}

// Collection returns a Collection whose List/All accept resource.WithBodyFilter
// and resource.WithBodyFilters. Without a Body filter it streams the same
// native pager as the ordinary API. Server query extensions are unsupported.
func Collection[T any](spec Spec[T]) *resource.Collection[T] {
	reject := func(query url.Values) bool { return len(query) != 0 }
	plain := func(ctx context.Context, query url.Values, control resource.ListControl) iter.Seq2[*T, error] {
		if reject(query) {
			return func(yield func(*T, error) bool) { yield(nil, resource.ErrUnsupported) }
		}
		return resource.StreamWithControl(ctx, spec.Pager(), spec.Extract, control)
	}
	return resource.NewCollection(resource.Adapter[T]{
		Kind: spec.Kind, ID: spec.ID, Name: spec.Name, Get: spec.Get,
		IterateControlled: plain,
		IterateBodyControlled: func(ctx context.Context, query url.Values, control resource.ListControl) iter.Seq2[*resource.BodyRecord[T], error] {
			if reject(query) {
				return func(yield func(*resource.BodyRecord[T], error) bool) { yield(nil, resource.ErrUnsupported) }
			}
			return resource.BodyStreamWithControl(ctx, spec.Pager(), spec.Extract, spec.Envelope, control)
		},
		BodyFilterFields: spec.Aliases,
		BodyFilterRecordValue: func(record *resource.BodyRecord[T], field string) (json.RawMessage, error) {
			if record == nil || record.Fields == nil {
				return nil, fmt.Errorf("%w: list row is not an object", resource.ErrInvalidOption)
			}
			return resource.BodyRecordField(record.Fields, field, resource.BodyFieldJSON)
		},
	})
}

// Aliases adds the identity mapping for every canonical field in renames
// (Python attribute name -> wire name) and in plain (same name on both sides).
func Aliases(plain []string, renames map[string]string) map[string]string {
	out := make(map[string]string, len(plain)+2*len(renames))
	for _, name := range plain {
		out[name] = name
	}
	for alias, wire := range renames {
		out[alias] = wire
		out[wire] = wire
	}
	return out
}

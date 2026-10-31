// Package calls translates the shared call options of package request into
// callbacks over the state of one hand-written (SDK-owned) operation.
//
// An operation declares its option type as an alias of one request category,
// for example
//
//	type DeleteOption = request.DeleteCall
//
// and describes in a Spec how each shared control changes its own state S:
// either its exported ...Opts struct or a request.Config of that struct. A nil
// Spec member means the operation does not implement that control, so a call
// the category admits but the operation never supported fails with
// request.UnsupportedCall when it is applied, before HTTP.
//
// Translate keeps the operation's own apply loop: it returns one callback per
// call, in caller order, so later calls win, a nil call stays nil for the
// operation to report with its own error, and errors surface at the call's
// position. ApplyConfig is the loop for operations whose state is a
// request.Config; it reports failures like request.Apply.
//
// request.Custom over S always runs as given; Custom over any other state
// type is unsupported. Operation-specific options of a package are built with
// request.Custom.
package calls

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"reflect"
	"slices"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Spec says how one operation state S interprets shared calls. Each member
// handles one control; a nil member rejects that control with
// request.UnsupportedCall. Callbacks receive owned copies of the call's
// inputs.
type Spec[S any] struct {
	// Header handles request.Header: one additional header.
	Header func(s *S, key, value string) error
	// Headers handles request.Headers: the map replaces the collected
	// headers. It is nil when the caller passed nil.
	Headers func(s *S, values map[string]string) error
	// Microversion handles request.Microversion.
	Microversion func(s *S, version string) error
	// BasePath handles request.BasePath.
	BasePath func(s *S, path string) error
	// IgnoreMissing handles request.IgnoreMissing.
	IgnoreMissing func(s *S, ignore bool) error
	// Query handles request.Query. The entry carries Key, Form, Text,
	// Values, JSON and JSONErr; see request.CallEntry.
	Query func(s *S, entry request.CallEntry) error
	// Field handles request.Field. The entry carries Key, JSON and JSONErr.
	Field func(s *S, entry request.CallEntry) error
	Paging[S]
}

// Paging holds the handlers of the shared paging controls.
type Paging[S any] struct {
	Limit     func(s *S, n int) error
	Marker    func(s *S, marker string) error
	MaxItems  func(s *S, n int) error
	Paginated func(s *S, paginated bool) error
}

// Translate turns calls into callbacks over S, one per call and in order. A
// nil call, or request.Custom with a nil function, yields a nil callback so
// the operation reports its own nil-option error at that position. A call
// whose control spec does not handle yields a callback that returns
// request.UnsupportedCall.
func Translate[S any, C request.Call](calls []C, spec Spec[S]) []func(*S) error {
	out := make([]func(*S) error, len(calls))
	for i, call := range calls {
		if any(call) == nil {
			continue
		}
		entry, err := request.Entry(call)
		if err != nil {
			out[i] = func(*S) error { return err }
			continue
		}
		out[i] = translate(entry, spec)
	}
	return out
}

func cloneEntry(e request.CallEntry) request.CallEntry {
	e.Headers = maps.Clone(e.Headers)
	e.Values = slices.Clone(e.Values)
	if e.JSON != nil {
		e.JSON = bytes.Clone(e.JSON)
	}
	return e
}

func translate[S any](entry request.CallEntry, spec Spec[S]) func(*S) error {
	unsupported := func(*S) error { return request.UnsupportedCall(entry) }
	switch entry.Kind {
	case request.CallCustom:
		fn, ok := request.CustomFunc[S](entry)
		if !ok {
			return unsupported
		}
		if fn == nil {
			return nil
		}
		return fn
	case request.CallHeader:
		if spec.Header == nil {
			return unsupported
		}
		return func(s *S) error { return spec.Header(s, entry.Key, entry.Text) }
	case request.CallHeaders:
		if spec.Headers == nil {
			return unsupported
		}
		return func(s *S) error { return spec.Headers(s, maps.Clone(entry.Headers)) }
	case request.CallMicroversion:
		if spec.Microversion == nil {
			return unsupported
		}
		return func(s *S) error { return spec.Microversion(s, entry.Text) }
	case request.CallBasePath:
		if spec.BasePath == nil {
			return unsupported
		}
		return func(s *S) error { return spec.BasePath(s, entry.Text) }
	case request.CallIgnoreMissing:
		if spec.IgnoreMissing == nil {
			return unsupported
		}
		return func(s *S) error { return spec.IgnoreMissing(s, entry.Bool) }
	case request.CallQuery:
		if spec.Query == nil {
			return unsupported
		}
		return func(s *S) error { return spec.Query(s, cloneEntry(entry)) }
	case request.CallField:
		if spec.Field == nil {
			return unsupported
		}
		return func(s *S) error { return spec.Field(s, cloneEntry(entry)) }
	case request.CallLimit:
		if spec.Limit == nil {
			return unsupported
		}
		return func(s *S) error { return spec.Limit(s, entry.Int) }
	case request.CallMarker:
		if spec.Marker == nil {
			return unsupported
		}
		return func(s *S) error { return spec.Marker(s, entry.Text) }
	case request.CallMaxItems:
		if spec.MaxItems == nil {
			return unsupported
		}
		return func(s *S) error { return spec.MaxItems(s, entry.Int) }
	case request.CallPaginated:
		if spec.Paginated == nil {
			return unsupported
		}
		return func(s *S) error { return spec.Paginated(s, entry.Bool) }
	}
	return unsupported
}

// ApplyConfig builds a request.Config from base and calls, in caller order,
// like request.Apply: a nil call or a Custom with a nil function fails with
// "nil request option" and a nil typed options pointer fails after all calls.
// Capability checks stay with the operation.
func ApplyConfig[T any, C request.Call](base T, spec Spec[request.Config[T]], calls ...C) (request.Config[T], error) {
	c := request.Config[T]{Options: base, Fields: make(map[string]json.RawMessage), Query: make(url.Values), Arguments: make(map[string]any), Headers: make(map[string]string)}
	for _, apply := range Translate(calls, spec) {
		if apply == nil {
			return c, fmt.Errorf("%w: nil request option", resource.ErrInvalidOption)
		}
		if err := apply(&c); err != nil {
			return c, err
		}
	}
	v := reflect.ValueOf(c.Options)
	if v.IsValid() && v.Kind() == reflect.Pointer && v.IsNil() {
		return c, fmt.Errorf("%w: nil typed request options", resource.ErrInvalidOption)
	}
	return c, nil
}

// ConfigHeader handles request.Header like request.WithHeader.
func ConfigHeader[T any](c *request.Config[T], key, value string) error {
	return request.WithHeader[T](key, value)(c)
}

// ConfigField handles request.Field like request.WithField: an empty key is
// reported before the value's marshal error.
func ConfigField[T any](c *request.Config[T], entry request.CallEntry) error {
	if strings.TrimSpace(entry.Key) == "" {
		return fmt.Errorf("%w: empty extension key", resource.ErrInvalidOption)
	}
	if entry.JSONErr != nil {
		return fmt.Errorf("%w: extension %q: %v", resource.ErrInvalidOption, entry.Key, entry.JSONErr)
	}
	c.Fields[entry.Key] = bytes.Clone(entry.JSON)
	return nil
}

// ConfigQuery handles request.Query with a string value like
// request.WithQuery and with a []string value like request.WithQueryValues.
// Any other value type fails with resource.ErrInvalidOption, because these
// operations serialize query values as text.
func ConfigQuery[T any](c *request.Config[T], entry request.CallEntry) error {
	switch entry.Form {
	case request.QueryText:
		return request.WithQuery[T](entry.Key, entry.Text)(c)
	case request.QueryList:
		return request.WithQueryValues[T](entry.Key, entry.Values...)(c)
	}
	if strings.TrimSpace(entry.Key) == "" {
		return fmt.Errorf("%w: empty query key", resource.ErrInvalidOption)
	}
	return fmt.Errorf("%w: query %q needs a string or []string value for this operation", resource.ErrInvalidOption, entry.Key)
}

// HeaderMap returns Header and Headers handlers over the additional header
// map that target points at. Header adds one key to a copy of the map;
// Headers replaces the map with the given copy.
func HeaderMap[S any](target func(*S) *map[string]string) (header func(*S, string, string) error, headers func(*S, map[string]string) error) {
	header = func(s *S, key, value string) error {
		p := target(s)
		next := maps.Clone(*p)
		if next == nil {
			next = make(map[string]string)
		}
		next[key] = value
		*p = next
		return nil
	}
	headers = func(s *S, values map[string]string) error {
		*target(s) = values
		return nil
	}
	return header, headers
}

// MergeHeader returns a Header handler for an operation whose own headers
// option merges a map into the headers collected so far (a request.Custom
// over S, such as an object storage With<Op>Headers). One shared Header call
// is applied as that option over a one-entry map, so it is validated and
// merged exactly like the operation's own map form. Such an operation keeps
// its merging map option and leaves Spec.Headers nil, because request.Headers
// replaces rather than merges.
func MergeHeader[S any, C request.Call](headers func(map[string]string) C) func(*S, string, string) error {
	return func(s *S, key, value string) error {
		entry, err := request.Entry(headers(map[string]string{key: value}))
		if err != nil {
			return err
		}
		fn, ok := request.CustomFunc[S](entry)
		if !ok || fn == nil {
			return request.UnsupportedCall(entry)
		}
		return fn(s)
	}
}

// Pointer returns a handler that stores a copy of the value through the
// pointer field that target points at.
func Pointer[S, V any](target func(*S) **V) func(*S, V) error {
	return func(s *S, v V) error {
		*target(s) = &v
		return nil
	}
}

// Value returns a handler that stores the value in the field that target
// points at.
func Value[S, V any](target func(*S) *V) func(*S, V) error {
	return func(s *S, v V) error {
		*target(s) = v
		return nil
	}
}

// KeywordFunc applies one Python keyword argument given as JSON: the value
// marshaled when the call was constructed and its marshal error.
type KeywordFunc[S any] func(s *S, key string, value json.RawMessage, err error) error

// JSONQuery returns a Query handler that passes the call's key and JSON value
// to kw, for operations that take JSON keyword arguments.
func JSONQuery[S any](kw KeywordFunc[S]) func(*S, request.CallEntry) error {
	return func(s *S, entry request.CallEntry) error { return kw(s, entry.Key, entry.JSON, entry.JSONErr) }
}

// JSONPaging returns paging handlers that pass each control to kw as the
// Python keyword argument limit, marker, max_items or paginated with its
// JSON value.
func JSONPaging[S any](kw KeywordFunc[S]) Paging[S] {
	call := func(s *S, key string, value any) error {
		raw, err := json.Marshal(value)
		return kw(s, key, raw, err)
	}
	return Paging[S]{
		Limit:     func(s *S, n int) error { return call(s, "limit", n) },
		Marker:    func(s *S, marker string) error { return call(s, "marker", marker) },
		MaxItems:  func(s *S, n int) error { return call(s, "max_items", n) },
		Paginated: func(s *S, paginated bool) error { return call(s, "paginated", paginated) },
	}
}

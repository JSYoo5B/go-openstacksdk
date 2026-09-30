// Package request implements SDK-owned request configuration and extension
// serialization for generated APIs. Callers supply concrete options, never builders.
package request

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/url"
	"reflect"
	"strings"

	"gophercloudsdk/resource"
)

// Download retains a streaming response body and its parsed metadata. The caller
// must close it after reading, just as with an HTTP response body.
type Download[T any] struct {
	Body   io.ReadCloser
	Header T
}

func (d *Download[T]) Read(p []byte) (int, error) { return d.Body.Read(p) }
func (d *Download[T]) Close() error               { return d.Body.Close() }

// OpenDownload closes the response on metadata errors, preventing leaks before
// a caller receives ownership of the stream.
func OpenDownload[T any](body io.ReadCloser, header T, err error) (*Download[T], error) {
	if err != nil {
		if body != nil {
			_ = body.Close()
		}
		return nil, err
	}
	return &Download[T]{Body: body, Header: header}, nil
}

// Config holds an operation's typed options and optional extension inputs.
type Config[T any] struct {
	Options   T
	Fields    map[string]json.RawMessage
	Query     url.Values
	Arguments map[string]any
	Headers   map[string]string
}

type Option[T any] func(*Config[T]) error

func Apply[T any](base T, options ...Option[T]) (Config[T], error) {
	c := Config[T]{Options: base, Fields: make(map[string]json.RawMessage), Query: make(url.Values), Arguments: make(map[string]any), Headers: make(map[string]string)}
	for _, apply := range options {
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

// MergeFieldsFor also protects omitted fields declared by the concrete input.
// Use typed options for core fields and WithField for additional API extensions.
func MergeFieldsFor[T any](body map[string]any, fields map[string]json.RawMessage, options T) (map[string]any, error) {
	reserved := map[string]bool{}
	var visit func(reflect.Type)
	visit = func(t reflect.Type) {
		if t == nil {
			return
		}
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			return
		}
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if field.Anonymous {
				visit(field.Type)
			}
			key := strings.Split(field.Tag.Get("json"), ",")[0]
			if key != "" && key != "-" {
				reserved[key] = true
			}
		}
	}
	visit(reflect.TypeOf(options))
	for key := range fields {
		if reserved[key] {
			return nil, fmt.Errorf("%w: extension %q is a core input field", resource.ErrInvalidOption, key)
		}
	}
	return MergeFields(body, fields)
}

func WithOptions[T any](options T) Option[T] {
	return func(c *Config[T]) error { c.Options = options; return nil }
}

// WithField snapshots an extension JSON value. Core input fields cannot be
// overwritten by extension fields. The API validates the extension's schema.
func WithField[T any](key string, value any) Option[T] {
	encoded, err := json.Marshal(value)
	return func(c *Config[T]) error {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("%w: empty extension key", resource.ErrInvalidOption)
		}
		if err != nil {
			return fmt.Errorf("%w: extension %q: %v", resource.ErrInvalidOption, key, err)
		}
		c.Fields[key] = append(json.RawMessage(nil), encoded...)
		return nil
	}
}

func WithQuery[T any](key, value string) Option[T] {
	return func(c *Config[T]) error {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("%w: empty query key", resource.ErrInvalidOption)
		}
		c.Query.Set(key, value)
		return nil
	}
}

// WithArgument is used by generated typed helpers for optional secondary inputs.
func WithArgument[T any](key string, value any) Option[T] {
	return func(c *Config[T]) error { c.Arguments[key] = value; return nil }
}

// MergeFields places extensions inside a request's single resource envelope,
// or at the root for unwrapped request bodies. Existing fields are protected.
func MergeFields(body map[string]any, fields map[string]json.RawMessage) (map[string]any, error) {
	if len(fields) == 0 {
		return body, nil
	}
	body = maps.Clone(body)
	if body == nil {
		body = make(map[string]any)
	}
	target := body
	if len(body) == 1 {
		for envelope, value := range body {
			if nested, ok := value.(map[string]any); ok {
				target = maps.Clone(nested)
				body[envelope] = target
			}
		}
	}
	for key, value := range fields {
		if _, exists := target[key]; exists {
			return nil, fmt.Errorf("%w: extension %q conflicts with input", resource.ErrInvalidOption, key)
		}
		target[key] = append(json.RawMessage(nil), value...)
	}
	return body, nil
}

// ExtendQuery retains the upstream serialization before applying extra filters.
func ExtendQuery(query string, extra url.Values) (string, error) {
	values, err := url.ParseQuery(strings.TrimPrefix(query, "?"))
	if err != nil {
		return "", err
	}
	for key, value := range extra {
		values[key] = append([]string(nil), value...)
	}
	if len(values) == 0 {
		return "", nil
	}
	return "?" + values.Encode(), nil
}

func Wrap(operation, kind string, err error) error {
	if err == nil {
		return nil
	}
	return &resource.OperationError{Operation: operation, Resource: kind, Cause: err}
}

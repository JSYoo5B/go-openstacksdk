package request

import (
	"fmt"
	"maps"
	"net/http"
	"reflect"
	"slices"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// WithHeader configures an additional API header without a header builder.
func WithHeader[T any](key, value string) Option[T] {
	return func(c *Config[T]) error {
		if strings.TrimSpace(key) == "" || strings.ContainsAny(key, " \t\r\n:") || strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("%w: invalid extension header", resource.ErrInvalidOption)
		}
		c.Headers[http.CanonicalHeaderKey(key)] = value
		return nil
	}
}

// MergeHeadersFor prevents extension headers from overwriting serialized or
// omitted concrete input headers. Header names are compared without case.
func MergeHeadersFor[T any](headers, extra map[string]string, options T) (map[string]string, error) {
	result := maps.Clone(headers)
	if result == nil {
		result = map[string]string{}
	}
	reserved := map[string]bool{}
	for key := range headers {
		reserved[strings.ToLower(key)] = true
	}
	var visit func(reflect.Type)
	seen := map[reflect.Type]bool{}
	visit = func(t reflect.Type) {
		if t == nil || seen[t] {
			return
		}
		seen[t] = true
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
			key := strings.Split(field.Tag.Get("h"), ",")[0]
			if key != "" && key != "-" {
				reserved[strings.ToLower(key)] = true
			}
		}
	}
	visit(reflect.TypeOf(options))
	for key, value := range extra {
		if reserved[strings.ToLower(key)] {
			return nil, fmt.Errorf("%w: header %q conflicts with input", resource.ErrInvalidOption, key)
		}
		result[key] = value
	}
	return result, nil
}

// Argument retrieves a library-owned optional secondary input without panics.
func Argument[V any, T any](c Config[T], key string) (V, bool, error) {
	var zero V
	value, exists := c.Arguments[key]
	if !exists {
		return zero, false, nil
	}
	typed, ok := value.(V)
	if !ok {
		return zero, true, fmt.Errorf("%w: incompatible argument %q", resource.ErrInvalidOption, key)
	}
	v := reflect.ValueOf(typed)
	if v.IsValid() && v.Kind() == reflect.Pointer && v.IsNil() {
		return zero, true, fmt.Errorf("%w: nil argument %q", resource.ErrInvalidOption, key)
	}
	return typed, true, nil
}

// ValidateCapabilities rejects unsupported extension mechanisms, including
// options assembled through the generic request helpers.
func ValidateCapabilities[T any](c Config[T], body, query, headers bool, arguments ...string) error {
	if !body && len(c.Fields) > 0 {
		return fmt.Errorf("%w: JSON field extensions are unsupported for this operation", resource.ErrInvalidOption)
	}
	if !query && len(c.Query) > 0 {
		return fmt.Errorf("%w: query extensions are unsupported for this operation", resource.ErrInvalidOption)
	}
	if !headers && len(c.Headers) > 0 {
		return fmt.Errorf("%w: header extensions are unsupported for this operation", resource.ErrInvalidOption)
	}
	for key := range c.Arguments {
		if !slices.Contains(arguments, key) {
			return fmt.Errorf("%w: unsupported argument %q", resource.ErrInvalidOption, key)
		}
	}
	return nil
}

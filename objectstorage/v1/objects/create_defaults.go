package objects

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// WithCreateBytes snapshots a native object PUT body when the option is built.
// Every application starts a fresh reader, so the option can be reused. A nil
// slice, like a nonnil empty slice, creates an empty object. File uploads and
// replay evidence belong to CreateObject's separate owned-source workflow.
func WithCreateBytes(value []byte) CreateOption {
	owned := bytes.Clone(value)
	return func(cfg *request.Config[CreateOpts]) error {
		cfg.Options.Content = bytes.NewReader(owned)
		return nil
	}
}

// normalizeCreateDefaults runs after all typed request options have been applied.
// Upstream's ETag builder reads Content even when it is nil. An absent reader is
// the SDK's empty-object default; an interface containing a nil reader is an
// invalid supplied source rather than permission to invoke methods on nil.
func normalizeCreateDefaults(opts CreateOpts) (CreateOpts, error) {
	if opts.Content == nil {
		opts.Content = bytes.NewReader(nil)
		return opts, nil
	}
	value := reflect.ValueOf(opts.Content)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		if value.IsNil() {
			return opts, fmt.Errorf("%w: nil object create reader", resource.ErrInvalidOption)
		}
	}
	return opts, nil
}

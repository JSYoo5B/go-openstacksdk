package request

import (
	"fmt"
	"slices"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// WithQueryValues sets a query extension key to every value in order, so the
// key repeats on the wire like a Python list passed to requests params. A
// later option for the same key replaces these values, and no values remove
// an earlier extension for the key. Bindings that extend a typed query with
// ExtendQuery replace the typed serialization of that key.
func WithQueryValues[T any](key string, values ...string) Option[T] {
	owned := slices.Clone(values)
	return func(c *Config[T]) error {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("%w: empty query key", resource.ErrInvalidOption)
		}
		if len(owned) == 0 {
			c.Query.Del(key)
			return nil
		}
		c.Query[key] = slices.Clone(owned)
		return nil
	}
}

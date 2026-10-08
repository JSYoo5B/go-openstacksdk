package request

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// NumericID adapts shared string references to APIs with native integer IDs.
// Invalid or overflowing values fail before the HTTP request.
func NumericID[T integer](id string) (T, error) {
	var zero T
	t := reflect.TypeOf(zero)
	if t.Kind() >= reflect.Uint && t.Kind() <= reflect.Uint64 {
		value, err := strconv.ParseUint(id, 10, t.Bits())
		if err != nil {
			return zero, fmt.Errorf("%w: invalid numeric ID %q: %v", resource.ErrInvalidOption, id, err)
		}
		return T(value), nil
	}
	value, err := strconv.ParseInt(id, 10, t.Bits())
	if err != nil {
		return zero, fmt.Errorf("%w: invalid numeric ID %q: %v", resource.ErrInvalidOption, id, err)
	}
	return T(value), nil
}

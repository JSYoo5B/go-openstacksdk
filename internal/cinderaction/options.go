package cinderaction

import (
	"errors"
	"fmt"
	"slices"

	"gophercloudsdk/resource"
)

// prepareOptions owns callback order, intermediate values and final defaults.
// Only library supplied clone/default/guard functions select the request policy.
func prepareOptions[T any, O ~func(*T) error](options []O, clone func(T) T, defaults func(*T), guard func() error) (T, error) {
	var zero, value T
	options = slices.Clone(options)
	if err := guard(); err != nil {
		return zero, err
	}
	for _, option := range options {
		if err := guard(); err != nil {
			return zero, err
		}
		if option == nil {
			return zero, fmt.Errorf("%w: nil volume action option", resource.ErrInvalidOption)
		}
		next := clone(value)
		err := option(&next)
		value = clone(next)
		if err = errors.Join(err, guard()); err != nil {
			return zero, err
		}
	}
	defaults(&value)
	if err := guard(); err != nil {
		return zero, err
	}
	return clone(value), nil
}

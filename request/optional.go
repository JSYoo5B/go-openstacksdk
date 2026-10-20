package request

import (
	"bytes"
	"encoding/json"
)

// Optional distinguishes an omitted field, JSON null and a supplied value.
// Use the json tag ",omitzero" on concrete request fields. The zero value is
// omitted, while Present preserves false, zero and empty strings.
type Optional[T any] struct {
	value T
	set   bool
	null  bool
}

func Present[T any](value T) Optional[T] { return Optional[T]{value: value, set: true} }
func Null[T any]() Optional[T]           { return Optional[T]{set: true, null: true} }

func (o Optional[T]) IsZero() bool { return !o.set }
func (o Optional[T]) IsSet() bool  { return o.set }

// IsNull reports whether Null or an explicit JSON null selected the field.
func (o Optional[T]) IsNull() bool { return o.set && o.null }

// Get returns a supplied non-null value. The boolean is false for omitted and
// explicit-null fields; use IsSet and IsNull when those cases matter.
func (o Optional[T]) Get() (T, bool) {
	if o.set && !o.null {
		return o.value, true
	}
	var zero T
	return zero, false
}

func (o Optional[T]) MarshalJSON() ([]byte, error) {
	if !o.set || o.null {
		return []byte("null"), nil
	}
	return json.Marshal(o.value)
}

func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		*o = Null[T]()
		return nil
	}
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*o = Present(value)
	return nil
}

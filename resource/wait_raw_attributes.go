package resource

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
)

// waitFields resolves the status and progress selectors for this binding. A
// binding with WaitAttributes reads a caller-selected status attribute from
// the record's declared Resource view instead of an exported struct field;
// progress selection still uses the record struct.
func (c *Collection[T]) waitFields(o waitOptions) (status, progress *reflect.StructField, err error) {
	if !c.rawWaitAttribute(o) {
		return waitFields[T](o)
	}
	structural := o
	structural.statusAttribute = ""
	_, progress, err = waitFields[T](structural)
	return nil, progress, err
}

func (c *Collection[T]) rawWaitAttribute(o waitOptions) bool {
	return c.binding.WaitAttributes != nil && o.statusAttribute != ""
}

// waitStatus reads the selected status attribute. Raw view values follow the
// typed field rules: a JSON string is compared, while an attribute that is
// absent from the view, null or not a JSON string fails with ErrUnsupported.
func (c *Collection[T]) waitStatus(value *T, o waitOptions, field *reflect.StructField) (string, error) {
	if !c.rawWaitAttribute(o) {
		return waitStatus(value, field, c.binding.Status)
	}
	var view map[string]json.RawMessage
	if value != nil {
		view = c.binding.WaitAttributes(value)
	}
	raw, exists := view[o.statusAttribute]
	if !exists {
		return "", fmt.Errorf("%w: status attribute %q is not an attribute of the %s record", ErrUnsupported, o.statusAttribute, c.binding.Kind)
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return "", fmt.Errorf("%w: status attribute %q is nil", ErrUnsupported, o.statusAttribute)
	}
	if trimmed[0] != '"' {
		return "", fmt.Errorf("%w: status attribute %q must be a JSON string", ErrUnsupported, o.statusAttribute)
	}
	var text string
	if err := json.Unmarshal(trimmed, &text); err != nil {
		return "", fmt.Errorf("%w: status attribute %q: %w", ErrUnsupported, o.statusAttribute, err)
	}
	return text, nil
}

// RawResourceView returns the fields of a record's declared Resource view, or
// nil when the record carries none. SDK bindings use it for WaitAttributes.
func RawResourceView(value *RawResource) map[string]json.RawMessage {
	if value == nil {
		return nil
	}
	return value.Body
}

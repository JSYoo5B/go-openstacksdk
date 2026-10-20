package resource

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"unicode/utf8"
)

// RawResource owns a resource's complete JSON fields and response metadata.
// It does not impose a service model's known-field types. Decode provides an
// explicit atomic projection into an existing typed model when one is needed.
type RawResource struct{ Metadata }

func (r *RawResource) UnmarshalJSON(data []byte) error {
	if r == nil {
		return fmt.Errorf("%w: resource receiver is required", ErrInvalidOption)
	}
	if !utf8.Valid(data) || len(bytes.TrimSpace(data)) == 0 || bytes.TrimSpace(data)[0] != '{' {
		return fmt.Errorf("%w: resource must be a nonnull UTF-8 JSON object", ErrInvalidOption)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return fmt.Errorf("%w: resource JSON: %w", ErrInvalidOption, err)
	}
	metadata := Metadata{Body: fields, Header: r.Header.Clone(), StatusCode: r.StatusCode}
	*r = RawResource{Metadata: metadata}
	return nil
}

// MarshalJSON emits current resource fields, without adding metadata keys.
func (r RawResource) MarshalJSON() ([]byte, error) {
	if r.Body == nil {
		return nil, fmt.Errorf("%w: resource fields are required", ErrInvalidOption)
	}
	return json.Marshal(r.Body)
}

// Clone makes an independent copy, including raw field bytes and headers.
func (r *RawResource) Clone() *RawResource {
	if r == nil {
		return nil
	}
	return &RawResource{Metadata: cloneRawMetadata(r.Metadata)}
}

// Decode atomically projects current fields into a nonnil pointer target.
// SDK models with an exported Metadata field also receive owned response
// headers and status. Their existing JSON decoders still enforce their schemas.
func (r *RawResource) Decode(target any) error {
	wrap := func(cause error) error {
		return &OperationError{Operation: "RawResource.Decode", Resource: "resource", Cause: fmt.Errorf("%w: %w", ErrInvalidOption, cause)}
	}
	if r == nil || r.Body == nil {
		return wrap(fmt.Errorf("resource fields are required"))
	}
	value := reflect.ValueOf(target)
	if !value.IsValid() || value.Kind() != reflect.Pointer || value.IsNil() {
		return wrap(fmt.Errorf("target must be a nonnil pointer"))
	}
	data, err := json.Marshal(r.Body)
	if err != nil {
		return wrap(err)
	}
	if !utf8.Valid(data) {
		return wrap(fmt.Errorf("resource fields must contain UTF-8 JSON"))
	}
	decoded := reflect.New(value.Elem().Type())
	if decoded.Elem().Kind() == reflect.Struct {
		field, ok := decoded.Elem().Type().FieldByName("Metadata")
		if ok && len(field.Index) == 1 && field.Type == reflect.TypeOf(Metadata{}) {
			metadata := decoded.Elem().Field(field.Index[0])
			if metadata.CanSet() {
				metadata.Set(reflect.ValueOf(cloneRawMetadata(r.Metadata)))
			}
		}
	}
	if err := json.Unmarshal(data, decoded.Interface()); err != nil {
		return wrap(err)
	}
	value.Elem().Set(decoded.Elem())
	return nil
}

func cloneRawMetadata(value Metadata) Metadata {
	copy := value
	copy.Header = value.Header.Clone()
	copy.Links = slices.Clone(value.Links)
	if value.CreatedAt != nil {
		text := *value.CreatedAt
		copy.CreatedAt = &text
	}
	if value.UpdatedAt != nil {
		text := *value.UpdatedAt
		copy.UpdatedAt = &text
	}
	if value.Body != nil {
		copy.Body = make(map[string]json.RawMessage, len(value.Body))
		for key, raw := range value.Body {
			copy.Body[key] = bytes.Clone(raw)
		}
	}
	return copy
}

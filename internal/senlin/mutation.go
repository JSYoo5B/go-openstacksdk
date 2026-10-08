package senlin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Snapshot retains JSON presence and numbers while taking ownership of mutable
// input. RawMessage distinguishes an omitted value, null and an empty object.
func Snapshot[T any](value T) request.Option[T] {
	encoded, err := json.Marshal(value)
	return func(config *request.Config[T]) error {
		if err != nil {
			return fmt.Errorf("%w: Senlin options: %v", resource.ErrInvalidOption, err)
		}
		var copied T
		if err := json.Unmarshal(encoded, &copied); err != nil {
			return fmt.Errorf("%w: Senlin options: %v", resource.ErrInvalidOption, err)
		}
		config.Options = copied
		return nil
	}
}

func Required(values ...string) error {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%w: required Senlin strings must not be empty", resource.ErrInvalidOption)
		}
	}
	return nil
}

// Object validates an explicitly supplied, non-null object. The service owns
// plugin-specific schemas; JSON scalars and arrays are never object inputs.
func Object(raw json.RawMessage, field string) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return fmt.Errorf("%w: Senlin %s must be a JSON object", resource.ErrInvalidOption, field)
	}
	return nil
}

// OptionalObject also admits absence and explicit null. Server permissions and
// field schemas decide whether null has a meaning for the individual request.
func OptionalObject(raw json.RawMessage, field string) error {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil
	}
	return Object(raw, field)
}

// Headers validates additional headers for requests with or without a body.
// Authentication, version and transport headers remain owned by the SDK.
func Headers(headers map[string]string) error {
	for key, value := range headers {
		if strings.TrimSpace(key) == "" || strings.ContainsAny(key, " \t\r\n:") || strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("%w: invalid Senlin extension header", resource.ErrInvalidOption)
		}
		switch strings.ToLower(key) {
		case "openstack-api-version", "x-auth-token", "x-service-token", "authorization", "host", "cookie", "content-type", "content-length":
			return fmt.Errorf("%w: extension header %q is owned by the SDK", resource.ErrInvalidOption, key)
		}
	}
	return nil
}

// Body snapshots a single resource envelope. Extensions cannot replace concrete
// inputs, response identities or SDK-owned auth/version/transport headers.
// Stateless updates need an explicit field; cached dirty-state no-ops belong to
// a resource lifecycle, not this request serializer.
func Body[T any](config request.Config[T], envelope string, forbidden ...string) (json.RawMessage, error) {
	return body(config, envelope, false, forbidden...)
}

// CommandBody retains an empty parameter object for commands such as check.
// Concrete fields and caller extensions follow the ordinary mutation policy.
func CommandBody[T any](config request.Config[T], command string, forbidden ...string) (json.RawMessage, error) {
	return body(config, command, true, forbidden...)
}

// FlatBody snapshots an unwrapped attribute object. Concrete inputs and
// SDK-owned headers retain the same protection as wrapped mutation bodies.
func FlatBody[T any](config request.Config[T], forbidden ...string) (json.RawMessage, error) {
	fields, err := inputBody(config, false, forbidden...)
	if err != nil {
		return nil, err
	}
	return json.Marshal(fields)
}

func body[T any](config request.Config[T], envelope string, allowEmpty bool, forbidden ...string) (json.RawMessage, error) {
	fields, err := inputBody(config, allowEmpty, forbidden...)
	if err != nil {
		return nil, err
	}
	if envelope == "" {
		return nil, fmt.Errorf("%w: Senlin resource envelope is required", resource.ErrInvalidOption)
	}
	return json.Marshal(map[string]any{envelope: fields})
}

func inputBody[T any](config request.Config[T], allowEmpty bool, forbidden ...string) (map[string]any, error) {
	if err := request.ValidateCapabilities(config, true, false, true); err != nil {
		return nil, err
	}
	if err := Headers(config.Headers); err != nil {
		return nil, err
	}
	for key := range config.Fields {
		for _, protected := range forbidden {
			if key == protected {
				return nil, fmt.Errorf("%w: extension field %q is owned by the SDK", resource.ErrInvalidOption, key)
			}
		}
	}
	encoded, err := json.Marshal(config.Options)
	if err != nil {
		return nil, fmt.Errorf("%w: Senlin input: %v", resource.ErrInvalidOption, err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil || fields == nil {
		return nil, fmt.Errorf("%w: Senlin request options must describe an object", resource.ErrInvalidOption)
	}
	body := make(map[string]any, len(fields))
	for key, value := range fields {
		body[key] = value
	}
	body, err = request.MergeFieldsFor(body, config.Fields, config.Options)
	if err != nil {
		return nil, err
	}
	if len(body) == 0 && !allowEmpty {
		return nil, fmt.Errorf("%w: Senlin mutation requires at least one field", resource.ErrInvalidOption)
	}
	return body, nil
}

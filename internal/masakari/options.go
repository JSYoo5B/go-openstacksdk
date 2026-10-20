package masakari

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Snapshot captures pointers, JSON payloads and nested collections at option
// construction, and recreates a fresh value whenever the option is reused.
func Snapshot[T any](value T) request.Option[T] {
	encoded, err := json.Marshal(value)
	return func(config *request.Config[T]) error {
		if err != nil {
			return fmt.Errorf("%w: Masakari options: %v", resource.ErrInvalidOption, err)
		}
		var copied T
		if err := json.Unmarshal(encoded, &copied); err != nil {
			return fmt.Errorf("%w: Masakari options: %v", resource.ErrInvalidOption, err)
		}
		config.Options = copied
		return nil
	}
}

func Required(values ...string) error {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%w: required Masakari strings must not be empty", resource.ErrInvalidOption)
		}
	}
	return nil
}

func Headers(headers map[string]string) error {
	for key := range headers {
		switch strings.ToLower(key) {
		case "openstack-api-version", "x-auth-token", "x-service-token", "authorization", "host", "cookie", "content-type", "content-length":
			return fmt.Errorf("%w: extension header %q is owned by the SDK", resource.ErrInvalidOption, key)
		}
	}
	return nil
}

// Body protects concrete fields, response-only identities and parent selectors.
func Body[T any](config request.Config[T], envelope string, forbidden ...string) (json.RawMessage, error) {
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
		return nil, fmt.Errorf("%w: Masakari input: %v", resource.ErrInvalidOption, err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return nil, err
	}
	body := make(map[string]any, len(fields))
	for key, value := range fields {
		body[key] = value
	}
	body, err = request.MergeFieldsFor(body, config.Fields, config.Options)
	if err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("%w: Masakari update requires at least one field", resource.ErrInvalidOption)
	}
	return json.Marshal(map[string]any{envelope: body})
}

// Query protects even omitted concrete filters from extension replacement.
func Query[T any](config request.Config[T], core url.Values, reserved ...string) (url.Values, error) {
	if err := request.ValidateCapabilities(config, false, true, false); err != nil {
		return nil, err
	}
	for key, values := range config.Query {
		for _, protected := range reserved {
			if key == protected {
				return nil, fmt.Errorf("%w: extension query %q is a concrete input", resource.ErrInvalidOption, key)
			}
		}
		core[key] = append([]string(nil), values...)
	}
	return core, nil
}

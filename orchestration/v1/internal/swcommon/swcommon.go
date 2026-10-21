// Package swcommon holds helpers shared by the Heat software configuration and
// software deployment packages: option snapshots, protected-field checks and
// unwrapped JSON body construction.
package swcommon

import (
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Snapshot captures pointers, JSON values and nested data at option
// construction and recreates a fresh copy whenever the option is applied.
func Snapshot[T any](value T) request.Option[T] {
	encoded, err := json.Marshal(value)
	return func(config *request.Config[T]) error {
		if err != nil {
			return fmt.Errorf("%w: Heat software options: %v", resource.ErrInvalidOption, err)
		}
		var copied T
		if err := json.Unmarshal(encoded, &copied); err != nil {
			return fmt.Errorf("%w: Heat software options: %v", resource.ErrInvalidOption, err)
		}
		config.Options = copied
		return nil
	}
}

// RawJSON encodes a Go value into an independent JSON snapshot.
func RawJSON(what string, value any) (json.RawMessage, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", resource.ErrInvalidOption, what, err)
	}
	return encoded, nil
}

// Headers rejects extension headers that the SDK owns.
func Headers(headers map[string]string) error {
	for key := range headers {
		switch strings.ToLower(key) {
		case "x-auth-token", "x-service-token", "authorization", "host", "cookie", "content-type", "content-length", "accept":
			return fmt.Errorf("%w: extension header %q is owned by the SDK", resource.ErrInvalidOption, key)
		}
	}
	return nil
}

// Body builds an unwrapped JSON object from concrete options. Heat accepts
// software configuration and deployment fields at the root. Extension fields
// cannot replace a declared field, even an omitted one, nor a protected
// response-only member.
func Body[T any](config request.Config[T], protected ...string) (json.RawMessage, error) {
	if err := request.ValidateCapabilities(config, true, false, true); err != nil {
		return nil, err
	}
	if err := Headers(config.Headers); err != nil {
		return nil, err
	}
	for key := range config.Fields {
		if slices.Contains(protected, key) {
			return nil, fmt.Errorf("%w: extension field %q is owned by the SDK", resource.ErrInvalidOption, key)
		}
	}
	encoded, err := json.Marshal(config.Options)
	if err != nil {
		return nil, fmt.Errorf("%w: Heat software input: %v", resource.ErrInvalidOption, err)
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
	return json.Marshal(body)
}

// ListQuery builds a list query. Extension keys cannot replace concrete ones.
func ListQuery[T any](config request.Config[T], core url.Values, reserved ...string) (url.Values, error) {
	if err := request.ValidateCapabilities(config, false, true, false); err != nil {
		return nil, err
	}
	for key, values := range config.Query {
		if slices.Contains(reserved, key) {
			return nil, fmt.Errorf("%w: extension query %q is a concrete input", resource.ErrInvalidOption, key)
		}
		core[key] = slices.Clone(values)
	}
	return core, nil
}

// Paging adds Heat's limit and marker. A limit below one is rejected.
func Paging(values url.Values, limit *int, marker *string) error {
	if limit != nil {
		if *limit < 1 {
			return fmt.Errorf("%w: list limit must be positive", resource.ErrInvalidOption)
		}
		values.Set("limit", strconv.Itoa(*limit))
	}
	if marker != nil {
		if *marker == "" {
			return fmt.Errorf("%w: list marker must not be empty", resource.ErrInvalidOption)
		}
		values.Set("marker", *marker)
	}
	return nil
}

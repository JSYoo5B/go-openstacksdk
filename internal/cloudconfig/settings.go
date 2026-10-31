package cloudconfig

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// invalid matches the root package's configuration error messages.
func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

// configuredServiceSettings reads has_<service> flags and secgroup_source from
// the merged clouds.yaml settings of one cloud.
func configuredServiceSettings(settings map[string]any) (map[string]bool, *string, error) {
	var flags map[string]bool
	for key, value := range settings {
		if !strings.HasPrefix(key, "has_") || len(key) == len("has_") {
			continue
		}
		enabled, ok := value.(bool)
		if !ok {
			// CloudRegion.has_service asserts a bool value.
			return nil, nil, invalid("%s must be a boolean, got %T", key, value)
		}
		if flags == nil {
			flags = make(map[string]bool)
		}
		// Python looks the flag up by its exact normalized key only.
		flags[key[len("has_"):]] = enabled
	}
	value, present := settings["secgroup_source"]
	if !present {
		value, present = settings["secgroup-source"]
	}
	if !present {
		return flags, nil, nil
	}
	if value == nil {
		empty := ""
		return flags, &empty, nil
	}
	text, ok := value.(string)
	if !ok {
		return nil, nil, invalid("secgroup_source must be a string or null, got %T", value)
	}
	return flags, &text, nil
}

func configuredImageCreatePolicy(name string, settings map[string]any) (image.ImageCreatePolicy, error) {
	policy := image.ImageCreatePolicy{CloudName: name}
	for key, target := range map[string]*json.RawMessage{"image_format": &policy.ImageFormat, "image_api_use_tasks": &policy.UseTasks} {
		if value, present := settings[key]; present {
			normalized, err := imageCreateConfigJSON(value)
			if err != nil {
				return policy, err
			}
			raw, err := json.Marshal(normalized)
			if err != nil {
				return policy, fmt.Errorf("%s: %w", key, err)
			}
			*target = raw
		}
	}
	if value, present := settings["disable_vendor_agent"]; present {
		normalized, err := imageCreateConfigJSON(value)
		if err != nil {
			return policy, err
		}
		raw, err := json.Marshal(normalized)
		if err != nil {
			return policy, err
		}
		policy.RawVendorAgent = raw
	}
	if value, present := settings["has_object_store"]; present {
		normalized, err := imageCreateConfigJSON(value)
		if err != nil {
			return policy, err
		}
		raw, err := json.Marshal(normalized)
		if err != nil {
			return policy, err
		}
		policy.RawObjectStoreEnabled = raw
	}
	return image.PrepareImageCreatePolicy(image.WithImageCreatePolicyOpts(policy))
}

// YAML permits maps with interface keys; JSON-domain SDK configuration requires
// string keys while preserving explicit nulls, arrays, numbers and booleans.
func imageCreateConfigJSON(value any) (any, error) {
	switch value := value.(type) {
	case map[any]any:
		result := make(map[string]any, len(value))
		for key, child := range value {
			name, ok := key.(string)
			if !ok {
				return nil, invalid("image configuration keys must be strings")
			}
			converted, err := imageCreateConfigJSON(child)
			if err != nil {
				return nil, err
			}
			result[name] = converted
		}
		return result, nil
	case map[string]any:
		result := make(map[string]any, len(value))
		for key, child := range value {
			converted, err := imageCreateConfigJSON(child)
			if err != nil {
				return nil, err
			}
			result[key] = converted
		}
		return result, nil
	case []any:
		result := make([]any, len(value))
		for i, child := range value {
			converted, err := imageCreateConfigJSON(child)
			if err != nil {
				return nil, err
			}
			result[i] = converted
		}
		return result, nil
	default:
		return value, nil
	}
}

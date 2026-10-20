package openstack

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	objectapi "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1"
)

// WithImageCreatePolicy overrides merged cloud image settings with concrete SDK
// options. Defaults, service construction and extension handling remain owned.
func WithImageCreatePolicy(options ...image.ImageCreatePolicyOption) ConnectionOption {
	captured := append([]image.ImageCreatePolicyOption(nil), options...)
	return func(config *connectionOptions) error {
		policy, err := image.PrepareImageCreatePolicy(captured...)
		if err == nil {
			config.imageCreatePolicy, config.imageCreatePolicySet = policy, true
		}
		return err
	}
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

// Only the configured Task branch resolves Swift. Source-only registration lets
// the Glance operation observe later Swift binding changes without recursion.
func (c *Connection) imageCreateSwiftService(ctx context.Context) (*objectapi.Service, error) {
	service, err := c.ObjectStorageV1(ctx)
	if err != nil {
		return nil, err
	}
	if service == nil || service.RawClient() == nil || service.Containers == nil || service.Objects == nil {
		return nil, invalid("image task Swift service is required")
	}
	client, containers, objects, provider := service.RawClient(), service.Containers, service.Objects, c.provider
	rest.RegisterOperationSource(ctx, func(context.Context) error {
		if c.provider != provider || service.RawClient() != client || service.Containers != containers || service.Objects != objects || containers.RawClient() != client || objects.RawClient() != client {
			return invalid("image task Swift service source changed")
		}
		return nil
	})
	return service, nil
}

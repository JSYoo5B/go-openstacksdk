package stackresources

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// WithNestedDepth includes nested resources in the read-only list. Each result
// retains its actual owner; use ResourceView.Identity before binding that owner.
func WithNestedDepth(depth int) resource.ListOption {
	return resource.WithQuery("nested_depth", strconv.Itoa(depth))
}

func WithDetails(enabled bool) resource.ListOption {
	return resource.WithQuery("with_detail", strconv.FormatBool(enabled))
}

// HealthOption configures the optional reason for a health change.
type HealthOption func(*healthOptions) error
type healthOptions struct{ reason string }

func WithHealthReason(reason string) HealthOption {
	return func(options *healthOptions) error { options.reason = reason; return nil }
}

func healthConfig(options []HealthOption) (healthOptions, error) {
	var result healthOptions
	for _, apply := range options {
		if apply == nil {
			return result, fmt.Errorf("%w: nil health option", resource.ErrInvalidOption)
		}
		if err := apply(&result); err != nil {
			return result, err
		}
	}
	return result, nil
}

func validateListQuery(query url.Values) error {
	for key := range query {
		switch key {
		case "type", "status", "name", "action", "id", "physical_resource_id":
		case "nested_depth":
			depth, err := strconv.Atoi(query.Get(key))
			if err != nil || depth < 0 {
				return fmt.Errorf("%w: nested_depth must be a non-negative integer", resource.ErrInvalidOption)
			}
		case "with_detail":
			if _, err := strconv.ParseBool(query.Get(key)); err != nil {
				return fmt.Errorf("%w: with_detail must be a boolean", resource.ErrInvalidOption)
			}
		default:
			return fmt.Errorf("%w: Heat resource lists do not support query %q", resource.ErrUnsupported, key)
		}
	}
	return nil
}

func failedStatus(status string) bool {
	return strings.EqualFold(status, "ERROR") || strings.HasSuffix(strings.ToUpper(status), "_FAILED")
}

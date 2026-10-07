package nodes

import (
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

const localFiltersKey = "nodes.local_filters"

func filterField(key string) (string, error) {
	switch key {
	case "project_id":
		return "project", nil
	case "domain_id":
		return "domain", nil
	case "user_id":
		return "user", nil
	case "id", "name", "physical_id", "cluster_id", "profile_id", "profile_name", "project", "domain", "user", "index", "role", "init_at", "created_at", "updated_at", "status", "status_reason", "metadata", "data", "details", "dependents", "tainted":
		return key, nil
	default:
		return "", fmt.Errorf("%w: unknown node Body filter %q", resource.ErrInvalidOption, key)
	}
}

// WithListFilter applies a response Body filter without adding a wire query.
// Nested objects match subsets; arrays and numbers are compared exactly.
func WithListFilter(key string, value any) ListOption {
	field, fieldErr := filterField(key)
	raw, encodeErr := json.Marshal(value)
	return func(config *request.Config[ListOpts]) error {
		if fieldErr != nil {
			return fieldErr
		}
		if encodeErr != nil {
			return fmt.Errorf("%w: node Body filter: %v", resource.ErrInvalidOption, encodeErr)
		}
		values, _, err := request.Argument[map[string]json.RawMessage](*config, localFiltersKey)
		if err != nil {
			return err
		}
		if values == nil {
			values = make(map[string]json.RawMessage)
		}
		values[field] = append(json.RawMessage(nil), raw...)
		config.Arguments[localFiltersKey] = values
		return nil
	}
}

func prepareFilters(config request.Config[ListOpts]) (map[string]json.RawMessage, error) {
	values, _, err := request.Argument[map[string]json.RawMessage](config, localFiltersKey)
	if err != nil {
		return nil, err
	}
	filters := make(map[string]json.RawMessage, len(values))
	for key, raw := range values {
		field, err := filterField(key)
		if err != nil {
			return nil, err
		}
		if !json.Valid(raw) {
			return nil, fmt.Errorf("%w: invalid node filter JSON", resource.ErrInvalidOption)
		}
		filters[field] = append(json.RawMessage(nil), raw...)
	}
	return filters, nil
}

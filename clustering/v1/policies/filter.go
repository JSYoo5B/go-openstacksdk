package policies

import (
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

const localFiltersKey = "policies.local_filters"

func filterField(key string) (string, error) {
	switch key {
	case "project_id":
		return "project", nil
	case "domain_id":
		return "domain", nil
	case "user_id":
		return "user", nil
	case "id", "name", "type", "project", "domain", "user", "spec", "data", "created_at", "updated_at":
		return key, nil
	default:
		return "", fmt.Errorf("%w: unknown policy Body filter %q", resource.ErrInvalidOption, key)
	}
}

// WithListFilter applies a response Body filter locally. Object values match
// recursive subsets, numbers compare exactly, and arrays compare in order.
func WithListFilter(key string, value any) ListOption {
	field, fieldErr := filterField(key)
	encoded, encodeErr := json.Marshal(value)
	return func(config *request.Config[ListOpts]) error {
		if fieldErr != nil {
			return fieldErr
		}
		if encodeErr != nil {
			return fmt.Errorf("%w: policy filter JSON: %v", resource.ErrInvalidOption, encodeErr)
		}
		filters, _, err := request.Argument[map[string]json.RawMessage](*config, localFiltersKey)
		if err != nil {
			return err
		}
		if filters == nil {
			filters = make(map[string]json.RawMessage)
		}
		filters[field] = append(json.RawMessage(nil), encoded...)
		config.Arguments[localFiltersKey] = filters
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
			return nil, fmt.Errorf("%w: invalid policy filter JSON", resource.ErrInvalidOption)
		}
		filters[field] = append(json.RawMessage(nil), raw...)
	}
	return filters, nil
}

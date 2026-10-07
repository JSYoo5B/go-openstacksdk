package profiles

import (
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

const localFiltersKey = "profiles.local_filters"

func filterField(key string) (string, error) {
	switch key {
	case "project_id":
		return "project", nil
	case "domain_id":
		return "domain", nil
	case "user_id":
		return "user", nil
	case "id", "name", "type", "project", "domain", "user", "spec", "metadata", "created_at", "updated_at":
		return key, nil
	default:
		return "", fmt.Errorf("%w: unknown profile Body filter %q", resource.ErrInvalidOption, key)
	}
}

// WithListFilter selects a known response field locally. Object filters match
// recursively as subsets; arrays remain ordered, complete values. Numbers are
// compared exactly rather than being converted through float64.
func WithListFilter(key string, value any) ListOption {
	field, fieldErr := filterField(key)
	encoded, encodeErr := json.Marshal(value)
	return func(config *request.Config[ListOpts]) error {
		if fieldErr != nil {
			return fieldErr
		}
		if encodeErr != nil {
			return fmt.Errorf("%w: profile filter JSON: %v", resource.ErrInvalidOption, encodeErr)
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
			return nil, fmt.Errorf("%w: invalid profile filter JSON for %q", resource.ErrInvalidOption, key)
		}
		filters[field] = append(json.RawMessage(nil), raw...)
	}
	return filters, nil
}

package clusters

import (
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

const localFiltersKey = "clusters.local_filters"

func filterField(key string) (string, error) {
	switch key {
	case "project_id":
		return "project", nil
	case "domain_id":
		return "domain", nil
	case "user_id":
		return "user", nil
	case "node_ids":
		return "nodes", nil
	case "is_profile_only":
		return "profile_only", nil
	case "id", "name", "profile_id", "profile_name", "project", "domain", "user", "init_at", "created_at", "updated_at", "min_size", "max_size", "desired_capacity", "timeout", "status", "status_reason", "config", "metadata", "data", "nodes", "dependents", "profile_only":
		return key, nil
	default:
		return "", fmt.Errorf("%w: unknown cluster Body filter %q", resource.ErrInvalidOption, key)
	}
}

// WithListFilter filters a known response field locally. Object filters match
// recursively as subsets, and numeric comparisons retain exact decimal values.
func WithListFilter(key string, value any) ListOption {
	field, fieldErr := filterField(key)
	encoded, encodeErr := json.Marshal(value)
	return func(config *request.Config[ListOpts]) error {
		if fieldErr != nil {
			return fieldErr
		}
		if encodeErr != nil {
			return fmt.Errorf("%w: cluster filter JSON: %v", resource.ErrInvalidOption, encodeErr)
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
			return nil, fmt.Errorf("%w: invalid cluster filter JSON for %q", resource.ErrInvalidOption, key)
		}
		filters[field] = append(json.RawMessage(nil), raw...)
	}
	return filters, nil
}

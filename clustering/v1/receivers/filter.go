package receivers

import (
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

const localFiltersKey = "receivers.local_filters"

func filterField(key string) (string, error) {
	switch key {
	case "project_id":
		return "project", nil
	case "domain_id":
		return "domain", nil
	case "user_id":
		return "user", nil
	case "id", "name", "type", "cluster_id", "action", "user", "project", "domain", "actor", "params", "channel", "created_at", "updated_at":
		return key, nil
	default:
		return "", fmt.Errorf("%w: unknown receiver Body filter %q", resource.ErrInvalidOption, key)
	}
}

// WithListFilter filters response Body values locally without a wire query.
// Objects match subsets; arrays and JSON numbers are compared exactly.
func WithListFilter(key string, value any) ListOption {
	field, fieldErr := filterField(key)
	raw, encodeErr := json.Marshal(value)
	return func(config *request.Config[ListOpts]) error {
		if fieldErr != nil {
			return fieldErr
		}
		if encodeErr != nil {
			return fmt.Errorf("%w: receiver Body filter: %v", resource.ErrInvalidOption, encodeErr)
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
			return nil, fmt.Errorf("%w: invalid receiver filter JSON", resource.ErrInvalidOption)
		}
		filters[field] = append(json.RawMessage(nil), raw...)
	}
	return filters, nil
}

package serviceinfo

import (
	"encoding/json"
	"fmt"

	"gophercloudsdk/resource"
)

// UsageInfo retains the complete flat quota usage document. Usage is a nonnil
// map on success, including when the deployment returns an empty usage object.
type UsageInfo struct {
	resource.Metadata
	Usage map[string]*UsageResource `json:"usage"`
}

// UsageResource preserves signed integer metrics and their raw fields. Nil
// metrics distinguish missing or null values from explicit zero. Resource names
// and unknown metrics are passive data and do not select routes or quota policy.
type UsageResource struct {
	Limit *int64                     `json:"limit"`
	Usage *int64                     `json:"usage"`
	Body  map[string]json.RawMessage `json:"-"`
}

func (value *UsageInfo) UnmarshalJSON(data []byte) error {
	fields, err := objectFields(data)
	if err != nil {
		return err
	}
	raw, exists := fieldPresent(fields, "usage")
	if !exists {
		return fmt.Errorf("discovery field usage must be a nonnull object")
	}
	entries, err := objectFields(raw)
	if err != nil {
		return fmt.Errorf("discovery field usage: %w", err)
	}
	decoded := UsageInfo{
		Metadata: resource.Metadata{Body: fields, Header: value.Header, StatusCode: value.StatusCode},
		Usage:    make(map[string]*UsageResource, len(entries)),
	}
	for name, raw := range entries {
		var entry UsageResource
		if err := json.Unmarshal(raw, &entry); err != nil {
			return fmt.Errorf("usage resource %q: %w", name, err)
		}
		decoded.Usage[name] = &entry
	}
	*value = decoded
	return nil
}

func (value *UsageResource) UnmarshalJSON(data []byte) error {
	fields, err := objectFields(data)
	if err != nil {
		return err
	}
	decoded := UsageResource{Body: fields}
	if decoded.Limit, err = usageInteger(fields, "limit"); err != nil {
		return err
	}
	if decoded.Usage, err = usageInteger(fields, "usage"); err != nil {
		return err
	}
	*value = decoded
	return nil
}

func usageInteger(fields map[string]json.RawMessage, key string) (*int64, error) {
	raw, exists := fieldPresent(fields, key)
	if !exists {
		return nil, nil
	}
	var value int64
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("usage field %q: %w", key, err)
	}
	return &value, nil
}

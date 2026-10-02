// Package secretconsumers manages associations between one Barbican secret
// and its consumers. Response references are evidence, never request targets.
package secretconsumers

import (
	"encoding/json"
	"fmt"

	"gophercloudsdk/resource"
)

// Consumer has no synthesized resource ID. Its three association fields are
// strings; Body retains null, omission and exact unknown JSON independently.
type Consumer struct {
	resource.Metadata
	Service      string  `json:"service"`
	ResourceType string  `json:"resource_type"`
	ResourceID   string  `json:"resource_id"`
	Status       string  `json:"status"`
	CreatedAt    *string `json:"created"`
	UpdatedAt    *string `json:"updated"`
}

func (value *Consumer) UnmarshalJSON(data []byte) error {
	type plain Consumer
	var decoded plain
	if err := decodeCanonical(data, &decoded, &decoded.Metadata, "service", "resource_type", "resource_id", "status", "created", "updated"); err != nil {
		return err
	}
	*value = Consumer(decoded)
	return nil
}

// SecretResponse is the actual root secret object returned by POST or DELETE.
// It is not a Consumer populated from the request. SecretRef is never followed.
type SecretResponse struct {
	resource.Metadata
	Name      string      `json:"name"`
	Status    string      `json:"status"`
	SecretRef string      `json:"secret_ref"`
	Consumers []*Consumer `json:"consumers"`
	CreatedAt *string     `json:"created"`
	UpdatedAt *string     `json:"updated"`
}

func (value *SecretResponse) UnmarshalJSON(data []byte) error {
	type plain SecretResponse
	var decoded plain
	if err := decodeCanonical(data, &decoded, &decoded.Metadata, "name", "status", "secret_ref", "consumers", "created", "updated"); err != nil {
		return err
	}
	// An association row must be an object; null elements must not fabricate
	// zero-valued consumers in an otherwise accepted mutation response.
	for _, consumer := range decoded.Consumers {
		if consumer == nil {
			return fmt.Errorf("consumer row must be a JSON object")
		}
	}
	*value = SecretResponse(decoded)
	return nil
}

func decodeCanonical(data []byte, target any, metadata *resource.Metadata, keys ...string) error {
	var raw resource.Metadata
	if err := resource.DecodeObject(data, &struct{}{}, &raw); err != nil {
		return err
	}
	projection := make(map[string]json.RawMessage, len(keys))
	for _, key := range keys {
		if value, exists := raw.Body[key]; exists {
			projection[key] = value
		}
	}
	encoded, err := json.Marshal(projection)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(encoded, target); err != nil {
		return err
	}
	*metadata = raw
	return nil
}

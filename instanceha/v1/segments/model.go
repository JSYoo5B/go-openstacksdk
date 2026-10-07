// Package segments manages Masakari failover segments.
package segments

import (
	"encoding/json"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// Segment.UUID is the endpoint identity. ID retains the distinct database ID,
// which may be encoded as a JSON integer or string.
type Segment struct {
	resource.Metadata
	ID             json.RawMessage `json:"id"`
	UUID           string          `json:"uuid"`
	Name           string          `json:"name"`
	Description    *string         `json:"description"`
	RecoveryMethod string          `json:"recovery_method"`
	ServiceType    string          `json:"service_type"`
	Enabled        *bool           `json:"enabled"`
}

func (s *Segment) UnmarshalJSON(data []byte) error {
	type plain Segment
	var value plain
	if err := resource.DecodeObject(data, &value, &value.Metadata); err != nil {
		return err
	}
	*s = Segment(value)
	return nil
}

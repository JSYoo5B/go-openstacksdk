// Package profiles manages Senlin profile specifications and user metadata.
package profiles

import (
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Profile retains both typed fields and their original JSON representation.
// Validation results may have no ID; no identity is synthesized from the request.
type Profile struct {
	resource.Metadata
	ID           string                     `json:"id"`
	Name         string                     `json:"name"`
	Type         string                     `json:"type"`
	ProjectID    string                     `json:"project"`
	DomainID     *string                    `json:"domain"`
	UserID       string                     `json:"user"`
	Spec         map[string]json.RawMessage `json:"spec"`
	UserMetadata map[string]json.RawMessage `json:"metadata"`
}

func (value *Profile) UnmarshalJSON(data []byte) error {
	type plain Profile
	var decoded plain
	if err := resource.DecodeObject(data, &decoded, &decoded.Metadata); err != nil {
		return err
	}
	*value = Profile(decoded)
	return nil
}

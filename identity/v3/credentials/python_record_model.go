package credentials

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// CredentialRecord owns the declared Credential view and original HTTP evidence.
// Blob and other untyped Python Body fields retain their JSON types and numbers.
type CredentialRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

func (v *CredentialRecord) UnmarshalJSON(data []byte) error {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	return json.Unmarshal(data, v.Wire)
}
func recordMetadata(v *CredentialRecord) *resource.Metadata {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	return &v.Wire.Metadata
}

var recordBodyFields = []string{"id", "name", "blob", "project_id", "type", "user_id"}

func recordKnown(key string) bool {
	for _, field := range recordBodyFields {
		if key == field {
			return true
		}
	}
	return false
}
func recordInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func cloneRecordFields(values map[string]json.RawMessage) map[string]json.RawMessage {
	result := make(map[string]json.RawMessage, len(values))
	for key, raw := range values {
		result[key] = bytes.Clone(raw)
	}
	return result
}
func projectRecord(v *CredentialRecord, seed map[string]json.RawMessage) {
	body := cloneRecordFields(seed)
	if v.Wire != nil {
		for _, key := range recordBodyFields {
			if raw, ok := v.Wire.Body[key]; ok {
				body[key] = bytes.Clone(raw)
			}
		}
	}
	for _, key := range recordBodyFields {
		if _, ok := body[key]; !ok {
			body[key] = json.RawMessage("null")
		}
	}
	body["location"] = json.RawMessage("null")
	v.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: body, Header: v.Header.Clone(), StatusCode: v.StatusCode}}
}
func recordFromResponse(r *rest.Response, seed map[string]json.RawMessage) (*CredentialRecord, error) {
	if r == nil {
		return nil, nil
	}
	v := &CredentialRecord{Envelope: bytes.Clone(r.Body), Header: r.Header.Clone(), StatusCode: r.StatusCode}
	if !json.Valid(r.Body) {
		projectRecord(v, seed)
		return v, nil
	}
	if !utf8.Valid(r.Body) {
		return v, recordInvalid("record response must be UTF-8")
	}
	raw := bytes.TrimSpace(r.Body)
	if len(raw) == 0 || raw[0] != '{' {
		return v, recordInvalid("record response must be an object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return v, err
	}
	if nested, ok := fields["credential"]; ok {
		raw = nested
	}
	var wire resource.RawResource
	if err := json.Unmarshal(raw, &wire); err != nil {
		return v, err
	}
	wire.Header, wire.StatusCode = r.Header.Clone(), r.StatusCode
	v.Wire = wire.Clone()
	projectRecord(v, seed)
	return v, nil
}
func recordText(v *CredentialRecord, key string) string {
	if v == nil || v.Resource == nil {
		return ""
	}
	var text string
	_ = json.Unmarshal(v.Resource.Body[key], &text)
	return text
}
func recordMarker(v *CredentialRecord) (string, error) {
	if v == nil || v.Wire == nil {
		return "", recordInvalid("record marker missing")
	}
	var id string
	if err := json.Unmarshal(v.Wire.Body["id"], &id); err != nil || id == "" {
		return "", recordInvalid("record marker must be a nonempty string")
	}
	return id, nil
}

package trusts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// TrustRecord owns a declared Python descriptor view and separate raw HTTP
// evidence. Untyped roles, expiry and counters retain their original JSON types.
type TrustRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

func (v *TrustRecord) UnmarshalJSON(data []byte) error {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	return json.Unmarshal(data, v.Wire)
}
func recordMetadata(v *TrustRecord) *resource.Metadata {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	return &v.Wire.Metadata
}

var recordBodyFields = []string{"id", "name", "allow_redelegation", "expires_at", "is_impersonation", "links", "project_id", "role_links", "roles", "redelegated_trust_id", "redelegation_count", "remaining_uses", "trustee_user_id", "trustor_user_id"}

// recordKnown names canonical Body descriptors, as required for local filters.
func recordKnown(key string) bool {
	for _, field := range recordBodyFields {
		if key == field {
			return true
		}
	}
	return false
}
func recordFieldWire(key string) (string, bool) {
	if key == "impersonation" || key == "is_impersonation" {
		return "impersonation", true
	}
	return key, recordKnown(key)
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
func prepareRecordCreate(values map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	result := make(map[string]json.RawMessage)
	for key, raw := range values {
		wire, known := recordFieldWire(key)
		if !known {
			continue
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, recordInvalid("attribute %q must be UTF-8 JSON", key)
		}
		if _, exists := result[wire]; exists {
			return nil, recordInvalid("attribute aliases conflict for %q", wire)
		}
		result[wire] = bytes.Clone(raw)
	}
	return result, nil
}

func projectRecord(v *TrustRecord, seed map[string]json.RawMessage) error {
	merged := cloneRecordFields(seed)
	if v.Wire != nil {
		responseFields, err := prepareRecordCreate(v.Wire.Body)
		if err != nil {
			return err
		}
		for wire, raw := range responseFields {
			merged[wire] = bytes.Clone(raw)
		}
	}
	body := make(map[string]json.RawMessage, len(recordBodyFields)+1)
	for _, key := range recordBodyFields {
		wire, _ := recordFieldWire(key)
		raw, exists := merged[wire]
		if !exists {
			raw = json.RawMessage("null")
		}
		if (key == "allow_redelegation" || key == "is_impersonation") && !bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			truth, err := cloudfilter.PythonTruthy(raw)
			if err != nil {
				return err
			}
			raw, _ = json.Marshal(truth)
		}
		body[key] = bytes.Clone(raw)
	}
	body["location"] = json.RawMessage("null")
	v.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: body, Header: v.Header.Clone(), StatusCode: v.StatusCode}}
	return nil
}
func recordFromResponse(r *rest.Response, seed map[string]json.RawMessage) (*TrustRecord, error) {
	if r == nil {
		return nil, nil
	}
	value := &TrustRecord{Envelope: bytes.Clone(r.Body), Header: r.Header.Clone(), StatusCode: r.StatusCode}
	if !json.Valid(r.Body) {
		return value, projectRecord(value, seed)
	}
	if !utf8.Valid(r.Body) {
		return value, recordInvalid("record response must be UTF-8")
	}
	raw := bytes.TrimSpace(r.Body)
	if len(raw) == 0 || raw[0] != '{' {
		return value, recordInvalid("record response must be an object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return value, err
	}
	if nested, ok := fields["trust"]; ok {
		raw = nested
	}
	var wire resource.RawResource
	if err := json.Unmarshal(raw, &wire); err != nil {
		return value, err
	}
	wire.Header, wire.StatusCode = r.Header.Clone(), r.StatusCode
	value.Wire = wire.Clone()
	return value, projectRecord(value, seed)
}
func recordText(value *TrustRecord, key string) string {
	if value == nil || value.Resource == nil {
		return ""
	}
	var text string
	_ = json.Unmarshal(value.Resource.Body[key], &text)
	return text
}
func recordMarker(value *TrustRecord) (string, error) {
	if value == nil || value.Wire == nil {
		return "", recordInvalid("record marker missing")
	}
	var id string
	if err := json.Unmarshal(value.Wire.Body["id"], &id); err != nil || id == "" {
		return "", recordInvalid("record marker must be a nonempty string")
	}
	return id, nil
}

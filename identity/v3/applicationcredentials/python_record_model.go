package applicationcredentials

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ApplicationCredentialRecord keeps a declared Resource view and the original
// response independently. Dates and roles/access_rules remain raw JSON values.
type ApplicationCredentialRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

// AccessRuleRecord uses the same owned container with AccessRule descriptors.
type AccessRuleRecord = ApplicationCredentialRecord

func (v *ApplicationCredentialRecord) UnmarshalJSON(data []byte) error {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	return json.Unmarshal(data, v.Wire)
}
func recordMetadata(v *ApplicationCredentialRecord) *resource.Metadata {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	return &v.Wire.Metadata
}

var credentialFields = []string{"id", "name", "user", "links", "secret", "description", "expires_at", "roles", "unrestricted", "project_id", "access_rules"}
var ruleFields = []string{"id", "name", "links", "method", "path", "service"}

func recordFields(credential bool) []string {
	if credential {
		return credentialFields
	}
	return ruleFields
}
func recordKnown(key string, credential bool) bool {
	for _, field := range recordFields(credential) {
		if key == field {
			return true
		}
	}
	return false
}
func recordInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func copyRecordFields(fields map[string]json.RawMessage) map[string]json.RawMessage {
	result := make(map[string]json.RawMessage, len(fields))
	for k, v := range fields {
		result[k] = bytes.Clone(v)
	}
	return result
}
func projectRecord(v *ApplicationCredentialRecord, userID string, credential bool, seed map[string]json.RawMessage) error {
	view := copyRecordFields(seed)
	if v.Wire != nil {
		for _, key := range recordFields(credential) {
			if raw, ok := v.Wire.Body[key]; ok {
				view[key] = bytes.Clone(raw)
			}
		}
	}
	for _, key := range recordFields(credential) {
		if _, ok := view[key]; !ok {
			view[key] = json.RawMessage("null")
		}
	}
	if credential {
		normalized, err := resource.BodyRecordField(view, "unrestricted", resource.BodyFieldBoolean)
		if err != nil {
			return err
		}
		view["unrestricted"] = normalized
	}
	view["user_id"], _ = json.Marshal(userID)
	view["location"] = json.RawMessage("null")
	v.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: view, Header: v.Header.Clone(), StatusCode: v.StatusCode}}
	return nil
}
func recordFromResponse(response *rest.Response, userID string, credential bool, seed map[string]json.RawMessage) (*ApplicationCredentialRecord, error) {
	if response == nil {
		return nil, nil
	}
	v := &ApplicationCredentialRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	if !json.Valid(response.Body) {
		return v, projectRecord(v, userID, credential, seed)
	}
	if !utf8.Valid(response.Body) {
		return v, recordInvalid("record response must be UTF-8")
	}
	var body json.RawMessage = bytes.Clone(response.Body)
	if raw := bytes.TrimSpace(body); len(raw) == 0 || raw[0] != '{' {
		return v, recordInvalid("record response must be an object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return v, err
	}
	key := "access_rule"
	if credential {
		key = "application_credential"
	}
	if nested, ok := fields[key]; ok {
		body = nested
	}
	var wire resource.RawResource
	if err := json.Unmarshal(body, &wire); err != nil {
		return v, err
	}
	wire.Header, wire.StatusCode = response.Header.Clone(), response.StatusCode
	v.Wire = wire.Clone()
	return v, projectRecord(v, userID, credential, seed)
}
func recordText(v *ApplicationCredentialRecord, key string) string {
	if v == nil || v.Resource == nil {
		return ""
	}
	var text string
	_ = json.Unmarshal(v.Resource.Body[key], &text)
	return text
}
func recordMarker(v *ApplicationCredentialRecord) (string, error) {
	if v == nil || v.Wire == nil {
		return "", recordInvalid("record marker missing")
	}
	var id string
	if err := json.Unmarshal(v.Wire.Body["id"], &id); err != nil || id == "" {
		return "", recordInvalid("record marker must be a nonempty string")
	}
	return id, nil
}

package backups

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// BackupResourceRecord owns the declared Python-shaped view and literal HTTP
// response independently. BackupRecord remains the native export-record model.
type BackupResourceRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	Discovery      []FindDiscoveryObservation
	fields         map[string]json.RawMessage
}

// FindDiscoveryObservation is actual accepted Cinder discovery evidence;
// it never supplies backup fields or replaces the backup response status.
type FindDiscoveryObservation struct {
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
}

var findBodyFields = []string{
	"id", "name", "availability_zone", "container", "created_at", "data_timestamp",
	"description", "encryption_key_id", "fail_reason", "force", "has_dependent_backups",
	"is_incremental", "links", "metadata", "object_count", "project_id", "size",
	"snapshot_id", "status", "updated_at", "user_id", "volume_id", "volume_name",
}

func findBodyKnown(key string) bool { return slices.Contains(findBodyFields, key) }
func findCanonical(key string) (string, bool) {
	if key == "os-backup-project-attr:project_id" {
		return "project_id", true
	}
	return key, findBodyKnown(key)
}
func findInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func findFields(fields map[string]json.RawMessage) map[string]json.RawMessage {
	owned := make(map[string]json.RawMessage, len(fields))
	for key, raw := range fields {
		owned[key] = bytes.Clone(raw)
	}
	return owned
}
func findSegment(value string) error {
	if strings.TrimSpace(value) == "" || value == "." || value == ".." || !utf8.ValidString(value) {
		return findInvalid("nonblank UTF-8 identity segment is required")
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return findInvalid("identity has controls")
		}
	}
	return nil
}
func findIncoming(raw json.RawMessage) (map[string]json.RawMessage, error) {
	members, err := cloudfilter.ObjectMembers(raw)
	if err != nil {
		return nil, err
	}
	fields := make(map[string]json.RawMessage)
	for _, member := range members {
		if key, known := findCanonical(member.Key); known {
			fields[key] = bytes.Clone(member.Value)
		}
	}
	return fields, nil
}
func (v *BackupResourceRecord) UnmarshalJSON(data []byte) error {
	var wire resource.RawResource
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	fields, err := findIncoming(data)
	if err != nil {
		return err
	}
	v.Wire = wire.Clone()
	v.fields = fields
	return nil
}
func findMetadata(v *BackupResourceRecord) *resource.Metadata {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	return &v.Wire.Metadata
}
func findProject(value *BackupResourceRecord, seed map[string]json.RawMessage) error {
	fields := findFields(seed)
	for key, raw := range value.fields {
		fields[key] = bytes.Clone(raw)
	}
	body := make(map[string]json.RawMessage, len(findBodyFields)+1)
	for _, key := range findBodyFields {
		raw, err := resource.BodyRecordField(fields, key, resource.BodyFieldJSON)
		if err != nil {
			return err
		}
		switch key {
		case "force", "has_dependent_backups", "is_incremental":
			raw, err = jsonfilter.BooleanJSON(raw)
		case "size", "object_count":
			raw, err = jsonfilter.DescriptorIntegerJSON(raw)
		case "links":
			trimmed := bytes.TrimSpace(raw)
			if len(trimmed) > 0 && trimmed[0] != '[' && !bytes.Equal(trimmed, []byte("null")) {
				raw = append(append(json.RawMessage{'['}, raw...), ']')
			}
		case "metadata":
			trimmed := bytes.TrimSpace(raw)
			if len(trimmed) > 0 && trimmed[0] != '{' && !bytes.Equal(trimmed, []byte("null")) {
				raw = json.RawMessage("{}")
			}
		}
		if err != nil {
			return fmt.Errorf("%w: descriptor %q: %w", resource.ErrInvalidOption, key, err)
		}
		body[key] = bytes.Clone(raw)
	}
	body["location"] = json.RawMessage("null")
	value.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: body, Header: value.Header.Clone(), StatusCode: value.StatusCode}}
	return nil
}
func findResponse(response *rest.Response, seed map[string]json.RawMessage) (*BackupResourceRecord, error) {
	if response == nil {
		return nil, nil
	}
	value := &BackupResourceRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	if !utf8.Valid(response.Body) {
		return value, findInvalid("backup response must be UTF-8")
	}
	// Resource._translate_response tolerates an accepted non-JSON body.
	if !json.Valid(response.Body) {
		return value, findProject(value, seed)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(response.Body, &envelope); err != nil || envelope == nil {
		return value, findInvalid("backup response must be a nonnull object")
	}
	raw := response.Body
	if nested, exists := envelope["backup"]; exists {
		raw = nested
	}
	if err := value.UnmarshalJSON(raw); err != nil {
		return value, err
	}
	value.Wire.Header, value.Wire.StatusCode = response.Header.Clone(), response.StatusCode
	return value, findProject(value, seed)
}
func findText(value *BackupResourceRecord, key string) string {
	if value == nil || value.Resource == nil {
		return ""
	}
	var text string
	_ = json.Unmarshal(value.Resource.Body[key], &text)
	return text
}
func findMarker(value *BackupResourceRecord) (string, error) {
	// Shared paging re-decodes the retained last row without projecting a
	// public Resource. Read its original declared fields, never a consumer's
	// mutable view. Source obtains value.id before Requests query encoding.
	if value == nil || value.Wire == nil {
		return "", findInvalid("backup marker is missing")
	}
	raw := bytes.TrimSpace(value.fields["id"])
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return "", findInvalid("backup marker is missing")
	}
	// Body(id) is untyped. Marker transport stringifies that JSON-domain ID,
	// while lookup equality never equates a numeric ID with a string identity.
	marker, err := cloudfilter.PythonString(raw)
	if err != nil || marker == "" {
		return "", findInvalid("backup marker must be nonempty")
	}
	return marker, nil
}
func findDiscovery(values []FindDiscoveryObservation) []FindDiscoveryObservation {
	owned := make([]FindDiscoveryObservation, len(values))
	for i, v := range values {
		owned[i] = FindDiscoveryObservation{Envelope: bytes.Clone(v.Envelope), Header: v.Header.Clone(), StatusCode: v.StatusCode}
	}
	return owned
}

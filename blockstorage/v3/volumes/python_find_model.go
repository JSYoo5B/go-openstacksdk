package volumes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cindervolume"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// VolumeResourceRecord owns the declared Python-shaped view and literal HTTP
// response independently. The native Volume model remains available through existing typed operations.
type VolumeResourceRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	Discovery      []FindDiscoveryObservation
	fields         map[string]json.RawMessage
}

// FindDiscoveryObservation is actual accepted Cinder discovery evidence;
// it never supplies volume fields or replaces the volume response status.
type FindDiscoveryObservation struct {
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
}

var findDescriptors = cindervolume.Descriptors()

func findBodyKnown(key string) bool {
	for _, descriptor := range findDescriptors {
		if key == descriptor.Attribute {
			return true
		}
	}
	return false
}
func findCanonical(key string) (string, bool) {
	for _, descriptor := range findDescriptors {
		if key == descriptor.Attribute || key == descriptor.Wire {
			return descriptor.Attribute, true
		}
	}
	return key, false
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
func (v *VolumeResourceRecord) UnmarshalJSON(data []byte) error {
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
func findMetadata(v *VolumeResourceRecord) *resource.Metadata {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	return &v.Wire.Metadata
}

// findListConstructor checks keys passed as explicit arguments by Source list.
// Resource.existing also binds cls and supplies _synchronized to __init__.
// This is list-only: fetch merges a member response into an existing object.
// Source removes self before construction; unknown self stays only in Wire.
func findListConstructor(value *VolumeResourceRecord) error {
	for _, key := range []string{"connection", "microversion", "_synchronized", "cls"} {
		if _, present := value.Wire.Body[key]; present {
			return findInvalid("volume list row %q conflicts with the Source Resource constructor", key)
		}
	}
	return nil
}

// findProject reuses the owned Volume descriptor and location translators.
// Incoming aliases have already been consumed in their original member order.
func findProject(value *VolumeResourceRecord, seed map[string]json.RawMessage, location resource.CloudLocation) error {
	fields := findFields(seed)
	for key, raw := range value.fields {
		fields[key] = bytes.Clone(raw)
	}
	raw, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	computed, err := location.ForResource(fields["project_id"], fields["availability_zone"])
	if err != nil {
		return err
	}
	view, err := cindervolume.View(raw, computed)
	if err != nil {
		return fmt.Errorf("%w: %w", resource.ErrInvalidOption, err)
	}
	var projected resource.RawResource
	if err := json.Unmarshal(view, &projected); err != nil {
		return err
	}
	projected.Header, projected.StatusCode = value.Header.Clone(), value.StatusCode
	value.Resource = projected.Clone()
	return nil
}
func findResponse(response *rest.Response, seed map[string]json.RawMessage, location resource.CloudLocation) (*VolumeResourceRecord, error) {
	if response == nil {
		return nil, nil
	}
	value := &VolumeResourceRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	if !utf8.Valid(response.Body) {
		return value, findInvalid("volume response must be UTF-8")
	}
	// Resource._translate_response tolerates an accepted non-JSON body.
	if !json.Valid(response.Body) {
		return value, findProject(value, seed, location)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(response.Body, &envelope); err != nil || envelope == nil {
		return value, findInvalid("volume response must be a nonnull object")
	}
	raw := response.Body
	if nested, exists := envelope["volume"]; exists {
		raw = nested
	}
	if err := value.UnmarshalJSON(raw); err != nil {
		return value, err
	}
	value.Wire.Header, value.Wire.StatusCode = response.Header.Clone(), response.StatusCode
	return value, findProject(value, seed, location)
}
func findText(value *VolumeResourceRecord, key string) string {
	if value == nil || value.Resource == nil {
		return ""
	}
	var text string
	_ = json.Unmarshal(value.Resource.Body[key], &text)
	return text
}
func findMarker(value *VolumeResourceRecord) (string, error) {
	// Shared paging re-decodes the retained last row without projecting a
	// public Resource. Read its original declared fields, never a consumer's
	// mutable view. Source obtains value.id before Requests query encoding.
	if value == nil || value.Wire == nil {
		return "", findInvalid("volume marker is missing")
	}
	raw := bytes.TrimSpace(value.fields["id"])
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return "", findInvalid("volume marker is missing")
	}
	// Body(id) is untyped. Marker transport stringifies that JSON-domain ID,
	// while lookup equality never equates a numeric ID with a string identity.
	marker, err := cloudfilter.PythonString(raw)
	if err != nil || marker == "" {
		return "", findInvalid("volume marker must be nonempty")
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

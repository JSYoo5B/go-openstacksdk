package attachments

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

// AttachmentRecord separates the declared Python-shaped view from literal
// HTTP fields. Resource includes nullable defaults; Wire never gains defaults.
type AttachmentRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	fields         map[string]json.RawMessage
}

var pythonAttachmentFields = []struct{ attribute, wire string }{
	{"id", "id"}, {"name", "name"}, {"status", "status"},
	{"instance", "instance"}, {"volume_id", "volume_id"},
	{"attached_at", "attach_time"}, {"detached_at", "detach_time"},
	{"attach_mode", "mode"}, {"connection_info", "connection_info"}, {"connector", "connector"},
}

func pythonAttachmentField(key string) (string, bool) {
	for _, field := range pythonAttachmentFields {
		if key == field.attribute || key == field.wire {
			return field.wire, true
		}
	}
	return "", false
}
func pythonAttachmentInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func pythonAttachmentClone(fields map[string]json.RawMessage) map[string]json.RawMessage {
	result := make(map[string]json.RawMessage, len(fields))
	for key, raw := range fields {
		result[key] = bytes.Clone(raw)
	}
	return result
}
func (v *AttachmentRecord) UnmarshalJSON(data []byte) error {
	var wire resource.RawResource
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	fields, err := pythonAttachmentResponseFields(data)
	if err != nil {
		return err
	}
	v.Wire = wire.Clone()
	v.fields = fields
	return nil
}
func pythonAttachmentMetadata(v *AttachmentRecord) *resource.Metadata {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	return &v.Wire.Metadata
}

// Incoming aliases are transposed in JSON member order, matching _consume_attrs.
func pythonAttachmentResponseFields(raw json.RawMessage) (map[string]json.RawMessage, error) {
	members, err := cloudfilter.ObjectMembers(raw)
	if err != nil {
		return nil, err
	}
	fields := make(map[string]json.RawMessage)
	for _, member := range members {
		if key, ok := pythonAttachmentField(member.Key); ok {
			fields[key] = bytes.Clone(member.Value)
		}
	}
	return fields, nil
}
func pythonAttachmentProject(v *AttachmentRecord, fields map[string]json.RawMessage) {
	body := make(map[string]json.RawMessage, len(pythonAttachmentFields)+1)
	for _, field := range pythonAttachmentFields {
		raw, ok := fields[field.wire]
		if !ok {
			raw = json.RawMessage("null")
		}
		// fields._convert_type(type=dict) returns {} for nonnull nonobjects.
		if field.wire == "connection_info" && !bytes.Equal(bytes.TrimSpace(raw), []byte("null")) && !bytes.HasPrefix(bytes.TrimSpace(raw), []byte("{")) {
			raw = json.RawMessage("{}")
		}
		body[field.attribute] = bytes.Clone(raw)
	}
	body["location"] = json.RawMessage("null")
	v.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: body, Header: v.Header.Clone(), StatusCode: v.StatusCode}}
}
func pythonAttachmentFromResponse(r *rest.Response, seed map[string]json.RawMessage) (*AttachmentRecord, error) {
	if r == nil {
		return nil, nil
	}
	v := &AttachmentRecord{Envelope: bytes.Clone(r.Body), Header: r.Header.Clone(), StatusCode: r.StatusCode}
	fields := pythonAttachmentClone(seed)
	if !utf8.Valid(r.Body) {
		return v, pythonAttachmentInvalid("attachment response must be UTF-8")
	}
	// Resource._translate_response ignores an unparseable accepted body.
	if !json.Valid(r.Body) {
		pythonAttachmentProject(v, fields)
		return v, nil
	}
	raw := bytes.TrimSpace(r.Body)
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil || envelope == nil {
		return v, pythonAttachmentInvalid("attachment response must be a nonnull object")
	}
	if nested, ok := envelope["attachment"]; ok {
		raw = nested
	}
	var wire resource.RawResource
	if err := json.Unmarshal(raw, &wire); err != nil {
		return v, err
	}
	wire.Header, wire.StatusCode = r.Header.Clone(), r.StatusCode
	v.Wire = wire.Clone()
	incoming, err := pythonAttachmentResponseFields(raw)
	if err != nil {
		return v, err
	}
	for key, value := range incoming {
		fields[key] = bytes.Clone(value)
	}
	pythonAttachmentProject(v, fields)
	return v, nil
}
func pythonAttachmentMarker(v *AttachmentRecord) (string, error) {
	if v == nil || v.Wire == nil {
		return "", pythonAttachmentInvalid("attachment marker is missing")
	}
	var id string
	if err := json.Unmarshal(v.Wire.Body["id"], &id); err != nil || id == "" {
		return "", pythonAttachmentInvalid("attachment marker must be a nonempty string")
	}
	return id, nil
}

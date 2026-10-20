package snapshots

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// RecordObservation owns a capability response, separately from snapshot ACK.
type RecordObservation struct {
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
}

// SnapshotRecord separates Source descriptor values from literal wire evidence.
type SnapshotRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	Discovery      []RecordObservation
	fields         map[string]json.RawMessage
}

var findDescriptors = []struct {
	attribute, wire string
	conversion      byte
}{
	{"consumes_quota", "consumes_quota", 0}, {"created_at", "created_at", 0},
	{"description", "description", 0}, {"group_snapshot_id", "group_snapshot_id", 0},
	{"is_forced", "force", 'b'}, {"progress", "os-extended-snapshot-attributes:progress", 0},
	{"project_id", "os-extended-snapshot-attributes:project_id", 0}, {"size", "size", 'i'},
	{"status", "status", 0}, {"updated_at", "updated_at", 0}, {"user_id", "user_id", 0},
	{"volume_id", "volume_id", 0}, {"id", "id", 0}, {"name", "name", 0}, {"metadata", "metadata", 'd'},
}

func findInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func findClone(v map[string]json.RawMessage) map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, len(v))
	for key, raw := range v {
		out[key] = bytes.Clone(raw)
	}
	return out
}
func findBodyKnown(key string) bool {
	for _, field := range findDescriptors {
		if key == field.attribute {
			return true
		}
	}
	return false
}
func findCollect(raw json.RawMessage, list bool) (map[string]json.RawMessage, error) {
	members, err := cloudfilter.ObjectMembers(raw)
	if err != nil {
		return nil, err
	}
	fields := make(map[string]json.RawMessage)
	for _, member := range members {
		if list && (member.Key == "connection" || member.Key == "microversion" || member.Key == "_synchronized") {
			return nil, findInvalid("snapshot list constructor collides with %q", member.Key)
		}
		for _, field := range findDescriptors {
			if member.Key == field.attribute || member.Key == field.wire {
				fields[field.attribute] = bytes.Clone(member.Value)
			}
		}
	}
	return fields, nil
}
func (v *SnapshotRecord) UnmarshalJSON(raw []byte) error {
	var wire resource.RawResource
	if err := json.Unmarshal(raw, &wire); err != nil {
		return err
	}
	fields, err := findCollect(raw, true)
	if err != nil {
		return err
	}
	v.Wire = wire.Clone()
	v.fields = fields
	return nil
}
func findMetadata(v *SnapshotRecord) *resource.Metadata {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	return &v.Wire.Metadata
}
func findProject(v *SnapshotRecord, fields map[string]json.RawMessage) error {
	body := findClone(fields)
	for _, field := range findDescriptors {
		raw := bytes.TrimSpace(body[field.attribute])
		if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
			body[field.attribute] = json.RawMessage("null")
			continue
		}
		switch field.conversion {
		case 'i':
			converted, err := jsonfilter.DescriptorIntegerJSON(raw)
			if err != nil {
				return fmt.Errorf("snapshot field %s: %w", field.attribute, err)
			}
			body[field.attribute] = converted
		case 'd':
			if raw[0] != '{' {
				body[field.attribute] = json.RawMessage("{}")
			}
		case 'b':
			if bytes.Equal(raw, []byte("true")) || bytes.Equal(raw, []byte("false")) {
				continue
			}
			var text string
			if json.Unmarshal(raw, &text) == nil {
				switch strings.ToLower(text) {
				case "true":
					body[field.attribute] = json.RawMessage("true")
					continue
				case "false":
					body[field.attribute] = json.RawMessage("false")
					continue
				}
			}
			return findInvalid("snapshot is_forced expects a BoolStr true/false value")
		}
	}
	body["location"] = json.RawMessage("null")
	v.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: body, Header: v.Header.Clone(), StatusCode: v.StatusCode}}
	return nil
}
func findResponse(r *rest.Response, id string) (*SnapshotRecord, error) {
	if r == nil {
		return nil, nil
	}
	v := &SnapshotRecord{Envelope: bytes.Clone(r.Body), Header: r.Header.Clone(), StatusCode: r.StatusCode}
	seed, _ := json.Marshal(id)
	fields := map[string]json.RawMessage{"id": seed}
	if !utf8.Valid(r.Body) {
		return v, findInvalid("snapshot response must be UTF-8")
	}
	if !json.Valid(r.Body) {
		return v, findProject(v, fields)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(r.Body, &envelope); err != nil || envelope == nil {
		return v, findInvalid("snapshot response must be a nonnull object")
	}
	raw := r.Body
	if nested, ok := envelope["snapshot"]; ok {
		raw = nested
	}
	var wire resource.RawResource
	if err := json.Unmarshal(raw, &wire); err != nil {
		return v, err
	}
	wire.Header, wire.StatusCode = r.Header.Clone(), r.StatusCode
	v.Wire = wire.Clone()
	incoming, err := findCollect(raw, false)
	if err != nil {
		return v, err
	}
	for key, value := range incoming {
		fields[key] = bytes.Clone(value)
	}
	return v, findProject(v, fields)
}
func findText(v *SnapshotRecord, key string) string {
	if v == nil || v.Resource == nil {
		return ""
	}
	var text string
	_ = json.Unmarshal(v.Resource.Body[key], &text)
	return text
}
func findMarker(v *SnapshotRecord) (string, error) {
	if v == nil || v.Wire == nil {
		return "", findInvalid("snapshot marker missing")
	}
	var id string
	if json.Unmarshal(v.Wire.Body["id"], &id) != nil || id == "" {
		return "", findInvalid("snapshot marker must be a nonempty wire string")
	}
	return id, nil
}

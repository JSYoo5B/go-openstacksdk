// Package consistencygroups provides owned records for the legacy Cinder v3 API.
package consistencygroups

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cindervolume"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ConsistencyGroupRecord separates the declared Source ConsistencyGroup view from complete wire and
// HTTP evidence. Records own their bytes; changing one cannot change another.
type ConsistencyGroupRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	fields         map[string]json.RawMessage
}

// ConsistencyGroup declares eight Body descriptors plus inherited id.
// Only volume_types has a list descriptor; location is connection-derived.
var recordFields = []string{
	"availability_zone", "created_at", "description", "name", "status",
	"volume_types", "add_volumes", "remove_volumes", "id",
}

func recordInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func recordClone(fields map[string]json.RawMessage) map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, len(fields))
	for key, raw := range fields {
		out[key] = bytes.Clone(raw)
	}
	return out
}
func recordKnown(key string) bool {
	for _, field := range recordFields {
		if field == key {
			return true
		}
	}
	return false
}
func recordReserved(key string) bool {
	switch key {
	case "self", "cls", "connection", "_synchronized", "microversion", "base_path", "resource_type", "requires_id", "error_message", "skip_cache", "resource_response_key", "resource_request_key", "params", "prepend_key":
		return true
	}
	return false
}
func recordCollect(raw json.RawMessage, list bool) (map[string]json.RawMessage, error) {
	members, err := cloudfilter.ObjectMembers(raw)
	if err != nil {
		return nil, err
	}
	fields := make(map[string]json.RawMessage)
	for _, member := range members {
		if list && (member.Key == "cls" || member.Key == "connection" || member.Key == "microversion" || member.Key == "_synchronized") {
			return nil, recordInvalid("consistency group list constructor collides with %q", member.Key)
		}
		if recordKnown(member.Key) {
			fields[member.Key] = bytes.Clone(member.Value)
		}
	}
	return fields, nil
}
func (v *ConsistencyGroupRecord) UnmarshalJSON(raw []byte) error {
	var wire resource.RawResource
	if err := json.Unmarshal(raw, &wire); err != nil {
		return err
	}
	fields, err := recordCollect(raw, true)
	if err != nil {
		return err
	}
	v.Wire, v.fields = wire.Clone(), fields
	return nil
}
func recordMetadata(v *ConsistencyGroupRecord) *resource.Metadata {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	return &v.Wire.Metadata
}
func recordProject(v *ConsistencyGroupRecord, fields map[string]json.RawMessage) error {
	body := make(map[string]json.RawMessage, len(recordFields)+1)
	for _, key := range recordFields {
		raw := bytes.Clone(fields[key])
		if len(raw) == 0 {
			raw = json.RawMessage("null")
		}
		kind := cindervolume.Unchanged
		switch key {
		case "volume_types":
			kind = cindervolume.List
		}
		converted, err := cindervolume.Convert(raw, kind)
		if err != nil {
			return fmt.Errorf("consistency group field %s: %w", key, err)
		}
		body[key] = converted
	}
	// Connection-derived location is unavailable to this standalone leaf.
	body["location"] = json.RawMessage("null")
	v.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: body, Header: v.Header.Clone(), StatusCode: v.StatusCode}}
	return nil
}
func recordReceipt(r *rest.Response) *ConsistencyGroupRecord {
	if r == nil {
		return nil
	}
	return &ConsistencyGroupRecord{Envelope: bytes.Clone(r.Body), Header: r.Header.Clone(), StatusCode: r.StatusCode}
}
func recordResponse(r *rest.Response, seed map[string]json.RawMessage) (*ConsistencyGroupRecord, error) {
	v := recordReceipt(r)
	if v == nil {
		return nil, nil
	}
	fields := recordClone(seed)
	if !utf8.Valid(r.Body) {
		return v, recordInvalid("consistency group response must be UTF-8")
	}
	// Source _translate_response ignores JSON ValueError on accepted responses.
	if !json.Valid(r.Body) {
		return v, recordProject(v, fields)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(r.Body, &envelope); err != nil || envelope == nil {
		return v, recordInvalid("consistency group response must be a nonnull JSON object")
	}
	raw := r.Body
	if nested, exists := envelope["consistencygroup"]; exists {
		raw = nested
	}
	var wire resource.RawResource
	if err := json.Unmarshal(raw, &wire); err != nil {
		return v, err
	}
	wire.Header, wire.StatusCode = r.Header.Clone(), r.StatusCode
	v.Wire = wire.Clone()
	incoming, err := recordCollect(raw, false)
	if err != nil {
		return v, err
	}
	for key, value := range incoming {
		fields[key] = bytes.Clone(value)
	}
	return v, recordProject(v, fields)
}
func recordMarker(v *ConsistencyGroupRecord) (string, error) {
	if v == nil || v.Wire == nil {
		return "", recordInvalid("consistency group marker requires a retained wire ID")
	}
	values, err := cloudfilter.RequestQueryValues(v.Wire.Body["id"])
	if err != nil || len(values) != 1 || values[0] == "" {
		return "", recordInvalid("consistency group marker requires one nonempty wire value")
	}
	return values[0], nil
}

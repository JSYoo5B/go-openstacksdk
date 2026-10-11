// Package groupsnapshots provides Cinder generic group-snapshot user operations.
package groupsnapshots

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"net/http"
	"unicode/utf8"
)

type DiscoveryObservation struct {
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
}

// GroupSnapshotRecord separates declared Source values from actual HTTP proof.
type GroupSnapshotRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	Discovery      []DiscoveryObservation
	fields         map[string]json.RawMessage
}

// DeleteRecordResult is an actual DELETE receipt, including an ignored clean404.
// It is not proof that the resource never existed or remains absent.
type DeleteRecordResult struct {
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	IgnoredMissing bool
	Discovery      []DiscoveryObservation
}

var recordFields = []string{"created_at", "description", "group_id", "group_type_id", "id", "name", "project_id", "status"}

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
		if key == field {
			return true
		}
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
			return nil, recordInvalid("group snapshot list constructor collides with %q", member.Key)
		}
		if recordKnown(member.Key) {
			fields[member.Key] = bytes.Clone(member.Value)
		}
	}
	return fields, nil
}
func (v *GroupSnapshotRecord) UnmarshalJSON(raw []byte) error {
	var wire resource.RawResource
	if err := json.Unmarshal(raw, &wire); err != nil {
		return err
	}
	fields, err := recordCollect(raw, true)
	if err != nil {
		return err
	}
	v.Wire = wire.Clone()
	v.fields = fields
	return nil
}
func recordMetadata(v *GroupSnapshotRecord) *resource.Metadata {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	return &v.Wire.Metadata
}
func recordProject(v *GroupSnapshotRecord, seed map[string]json.RawMessage, location resource.CloudLocation) error {
	fields := recordClone(seed)
	for key, raw := range v.fields {
		fields[key] = bytes.Clone(raw)
	}
	body := make(map[string]json.RawMessage, len(recordFields)+1)
	for _, key := range recordFields {
		raw, exists := fields[key]
		if !exists {
			raw = json.RawMessage("null")
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return recordInvalid("group snapshot field %q requires UTF-8 JSON", key)
		}
		body[key] = bytes.Clone(raw)
	}
	computed, err := location.ForResource(body["project_id"], nil)
	if err != nil {
		return err
	}
	body["location"] = computed
	v.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: body, Header: v.Header.Clone(), StatusCode: v.StatusCode}}
	return nil
}
func recordReceipt(r *rest.Response) *GroupSnapshotRecord {
	if r == nil {
		return nil
	}
	return &GroupSnapshotRecord{Envelope: bytes.Clone(r.Body), Header: r.Header.Clone(), StatusCode: r.StatusCode}
}
func recordResponse(r *rest.Response, seed map[string]json.RawMessage, location resource.CloudLocation) (*GroupSnapshotRecord, error) {
	value := recordReceipt(r)
	if value == nil {
		return nil, nil
	}
	if !utf8.Valid(r.Body) {
		return value, recordInvalid("group snapshot response must be UTF-8")
	}
	if !json.Valid(r.Body) {
		return value, recordProject(value, seed, location)
	}
	var envelope resource.RawResource
	if err := json.Unmarshal(r.Body, &envelope); err != nil {
		return value, err
	}
	raw := r.Body
	if nested, ok := envelope.Body["group_snapshot"]; ok {
		raw = nested
	}
	var wire resource.RawResource
	if err := json.Unmarshal(raw, &wire); err != nil {
		return value, err
	}
	fields, err := recordCollect(raw, false)
	if err != nil {
		return value, err
	}
	wire.Header, wire.StatusCode = r.Header.Clone(), r.StatusCode
	value.Wire = wire.Clone()
	value.fields = fields
	return value, recordProject(value, seed, location)
}
func recordMarker(value *GroupSnapshotRecord) (string, error) {
	if value == nil || value.Wire == nil {
		return "", recordInvalid("group snapshot marker requires retained row")
	}
	raw, exists := value.fields["id"]
	if !exists || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return "", recordInvalid("group snapshot marker requires ID")
	}
	values, err := cloudfilter.RequestQueryValues(raw)
	if err != nil || len(values) != 1 || values[0] == "" {
		return "", recordInvalid("group snapshot marker requires one nonempty scalar ID")
	}
	return values[0], nil
}
func recordDiscovery(values []DiscoveryObservation) []DiscoveryObservation {
	out := make([]DiscoveryObservation, len(values))
	for i, v := range values {
		out[i] = DiscoveryObservation{Envelope: bytes.Clone(v.Envelope), Header: v.Header.Clone(), StatusCode: v.StatusCode}
	}
	return out
}

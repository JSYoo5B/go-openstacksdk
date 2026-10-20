// Package messages provides Cinder user-message lookup, listing and deletion.
package messages

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

// MessageRecord separates declared Source properties from complete HTTP proof.
type MessageRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	fields         map[string]json.RawMessage
}

var recordStrings = []string{"created_at", "event_id", "guaranteed_until", "message_level", "request_id", "resource_type", "resource_uuid", "user_message"}

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
	if key == "id" || key == "name" {
		return true
	}
	for _, field := range recordStrings {
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
		if list && (member.Key == "connection" || member.Key == "microversion" || member.Key == "_synchronized") {
			return nil, recordInvalid("message list constructor collides with %q", member.Key)
		}
		if recordKnown(member.Key) {
			fields[member.Key] = bytes.Clone(member.Value)
		}
	}
	return fields, nil
}
func (v *MessageRecord) UnmarshalJSON(raw []byte) error {
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
func recordMetadata(v *MessageRecord) *resource.Metadata {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	return &v.Wire.Metadata
}
func recordProject(v *MessageRecord, fields map[string]json.RawMessage) error {
	body := recordClone(fields)
	for _, key := range append([]string{"id", "name"}, recordStrings...) {
		raw := bytes.TrimSpace(body[key])
		if len(raw) == 0 {
			body[key] = json.RawMessage("null")
			continue
		}
		if bytes.Equal(raw, []byte("null")) || key == "id" || key == "name" {
			continue
		}
		text, err := cloudfilter.PythonString(raw)
		if err != nil {
			return fmt.Errorf("message field %s: %w", key, err)
		}
		body[key], _ = json.Marshal(text)
	}
	body["location"] = json.RawMessage("null")
	v.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: body, Header: v.Header.Clone(), StatusCode: v.StatusCode}}
	return nil
}
func recordReceipt(r *rest.Response) *MessageRecord {
	if r == nil {
		return nil
	}
	return &MessageRecord{Envelope: bytes.Clone(r.Body), Header: r.Header.Clone(), StatusCode: r.StatusCode}
}
func recordResponse(r *rest.Response, id string, sourceEnvelope bool) (*MessageRecord, error) {
	v := recordReceipt(r)
	if v == nil {
		return nil, nil
	}
	seed, _ := json.Marshal(id)
	fields := map[string]json.RawMessage{"id": seed}
	if !utf8.Valid(r.Body) {
		return v, recordInvalid("message response must be UTF-8")
	}
	if !json.Valid(r.Body) {
		return v, recordProject(v, fields)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(r.Body, &envelope); err != nil || envelope == nil {
		return v, recordInvalid("message response must be a nonnull object")
	}
	raw := r.Body
	if nested, exists := envelope["messages"]; exists {
		raw = nested
	} else if !sourceEnvelope {
		if nested, exists := envelope["message"]; exists {
			raw = nested
		}
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
func recordMarker(v *MessageRecord) (string, error) {
	if v == nil || v.Wire == nil {
		return "", recordInvalid("message marker requires retained wire ID")
	}
	values, err := cloudfilter.RequestQueryValues(v.Wire.Body["id"])
	if err != nil || len(values) != 1 || values[0] == "" {
		return "", recordInvalid("message marker requires one nonempty wire value")
	}
	return values[0], nil
}

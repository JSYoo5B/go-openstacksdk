package rules

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// SecurityGroupRuleRecord owns a Source descriptor view separately from the literal response row.
// Envelope/Header/StatusCode refer only to an actual HTTP observation. A seeded
// member view does not supply fields to Wire or claim a backend update.
type SecurityGroupRuleRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	fields         map[string]json.RawMessage
	match          json.RawMessage
}

func findInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func findClone(values map[string]json.RawMessage) map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, len(values))
	for key, raw := range values {
		out[key] = bytes.Clone(raw)
	}
	return out
}
func findCanonical(key string) (string, bool) {
	if _, ok := neutronBodyFieldNames[key]; ok {
		return key, true
	}
	for canonical, wire := range neutronBodyFieldNames {
		if wire == key {
			return canonical, true
		}
	}
	return key, false
}
func findList(raw json.RawMessage) (json.RawMessage, error) {
	if !utf8.Valid(raw) || !json.Valid(raw) {
		return nil, findInvalid("list descriptor must contain UTF-8 JSON")
	}
	v := bytes.TrimSpace(raw)
	if bytes.Equal(v, []byte("null")) || v[0] == '[' {
		return bytes.Clone(raw), nil
	}
	return json.RawMessage(append(append([]byte{'['}, raw...), ']')), nil
}
func findDescriptor(key string, raw json.RawMessage) (json.RawMessage, error) {
	if raw == nil {
		raw = json.RawMessage("null")
	}
	if !utf8.Valid(raw) || !json.Valid(raw) {
		return nil, findInvalid("descriptor %q must contain UTF-8 JSON", key)
	}
	token := bytes.TrimSpace(raw)
	if bytes.Equal(token, []byte("null")) {
		return bytes.Clone(raw), nil
	}
	switch key {
	case "revision_number", "port_range_min", "port_range_max":
		return jsonfilter.DescriptorIntegerJSON(raw)
	case "tags":
		return findList(raw)

	}
	return bytes.Clone(raw), nil
}
func findTruthy(raw json.RawMessage) (bool, error) {
	token := bytes.TrimSpace(raw)
	if len(token) > 0 && (token[0] == '-' || token[0] >= '0' && token[0] <= '9') && strings.ContainsAny(string(token), ".eE") {
		floating, err := strconv.ParseFloat(string(token), 64)
		if err != nil && !errors.Is(err, strconv.ErrRange) {
			return false, err
		}
		return floating != 0, nil
	}
	return cloudfilter.PythonTruthy(raw)
}

// Header mapping is case-insensitive for the wire hyphen name; its canonical
// underscore attribute is an exact kwargs name. IF_MATCH remains unknown Body.
func findHeaderKey(key string) bool {
	return key == "if_match" || strings.EqualFold(key, "if-match")
}
func findHeader(header http.Header) json.RawMessage {
	for key, values := range header {
		if findHeaderKey(key) {
			if len(values) > 0 {
				raw, _ := json.Marshal([]string{strings.Join(values, ", ")})
				return raw
			}
		}
	}
	return nil
}
func findCollect(raw json.RawMessage, list bool) (map[string]json.RawMessage, json.RawMessage, error) {
	members, err := cloudfilter.ObjectMembers(raw)
	if err != nil {
		return nil, nil, err
	}
	fields := make(map[string]json.RawMessage)
	var match json.RawMessage
	for _, member := range members {
		if member.Key == "self" {
			continue
		}
		if list && (member.Key == "cls" || member.Key == "connection" || member.Key == "microversion" || member.Key == "_synchronized") {
			return nil, nil, findInvalid("list constructor collides with %q", member.Key)
		}
		if list && findHeaderKey(member.Key) {
			match, err = findList(member.Value)
			if err != nil {
				return nil, nil, err
			}
			continue
		}
		canonical, _ := findCanonical(member.Key)
		fields[canonical] = bytes.Clone(member.Value)
	}
	return fields, match, nil
}
func (v *SecurityGroupRuleRecord) UnmarshalJSON(raw []byte) error {
	var wire resource.RawResource
	if err := json.Unmarshal(raw, &wire); err != nil {
		return err
	}
	fields, match, err := findCollect(raw, true)
	if err != nil {
		return err
	}
	v.Wire = wire.Clone()
	v.fields, v.match = fields, bytes.Clone(match)
	return nil
}
func findMetadata(v *SecurityGroupRuleRecord) *resource.Metadata {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	return &v.Wire.Metadata
}
func findProject(v *SecurityGroupRuleRecord, fields map[string]json.RawMessage, match json.RawMessage) error {
	body := findClone(fields)
	if _, exists := body["project_id"]; !exists {
		if tenant, exists := body["tenant_id"]; exists {
			body["project_id"] = bytes.Clone(tenant)
		}
	}
	for canonical := range neutronBodyFieldNames {
		raw, present := body[canonical]
		if !present {
			raw = json.RawMessage("null")
			if canonical == "tags" {
				raw = json.RawMessage(`[]`)
			}
		}
		converted, err := findDescriptor(canonical, raw)
		if err != nil {
			return findInvalid("response %s descriptor: %v", canonical, err)
		}
		body[canonical] = converted
	}
	if match == nil {
		match = json.RawMessage("null")
	}
	body["if_match"] = bytes.Clone(match)
	body["location"] = json.RawMessage("null")
	v.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: body, Header: v.Header.Clone(), StatusCode: v.StatusCode}}
	return nil
}
func findReceipt(wire *rest.Response) *SecurityGroupRuleRecord {
	if wire == nil {
		return nil
	}
	return &SecurityGroupRuleRecord{Envelope: bytes.Clone(wire.Body), Header: wire.Header.Clone(), StatusCode: wire.StatusCode}
}
func findResponse(wire *rest.Response, seed map[string]json.RawMessage, seedMatch json.RawMessage) (*SecurityGroupRuleRecord, error) {
	v := findReceipt(wire)
	fields := findClone(seed)
	match := bytes.Clone(seedMatch)
	if !utf8.Valid(wire.Body) {
		return v, findInvalid("member response must be UTF-8")
	}
	if json.Valid(wire.Body) {
		members, err := cloudfilter.ObjectMembers(wire.Body)
		if err != nil {
			return v, findInvalid("member response must be an object: %v", err)
		}
		raw := wire.Body
		for _, member := range members {
			if member.Key == "security_group_rule" {
				raw = member.Value
			}
		}
		var actual resource.RawResource
		if err := json.Unmarshal(raw, &actual); err != nil {
			return v, err
		}
		actual.Header, actual.StatusCode = wire.Header.Clone(), wire.StatusCode
		v.Wire = actual.Clone()
		incoming, _, err := findCollect(raw, false)
		if err != nil {
			return v, err
		}
		for key, raw := range incoming {
			fields[key] = bytes.Clone(raw)
		}
	}
	if received := findHeader(wire.Header); received != nil {
		match = received
	}
	return v, findProject(v, fields, match)
}
func findText(v *SecurityGroupRuleRecord, key string) string {
	if v == nil || v.Resource == nil {
		return ""
	}
	var text string
	_ = json.Unmarshal(v.Resource.Body[key], &text)
	return text
}
func findMarker(v *SecurityGroupRuleRecord) (string, error) {
	if v == nil || v.Wire == nil {
		return "", findInvalid("marker row required")
	}
	var id string
	if json.Unmarshal(v.Wire.Body["id"], &id) != nil || id == "" {
		return "", findInvalid("marker must be a nonempty wire string")
	}
	return id, nil
}

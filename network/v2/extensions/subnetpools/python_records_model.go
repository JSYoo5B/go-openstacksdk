package subnetpools

import (
	"bytes"
	"encoding/json"
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

// SubnetPoolRecord separates the Source descriptor view from the literal row
// and actual HTTP observation. This class has tag Body defaults but no
// NetworkResource Header descriptors. A no-op update has no Wire or HTTP ACK.
type SubnetPoolRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	fields         map[string]json.RawMessage
}

func poolRecordInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func poolRecordClone(fields map[string]json.RawMessage) map[string]json.RawMessage {
	result := make(map[string]json.RawMessage, len(fields))
	for key, raw := range fields {
		result[key] = bytes.Clone(raw)
	}
	return result
}
func poolRecordCanonical(key string) string {
	if _, known := neutronBodyFieldNames[key]; known {
		return key
	}
	for canonical, wire := range neutronBodyFieldNames {
		if key == wire {
			return canonical
		}
	}
	return key
}
func poolRecordKnown(key string) bool { _, known := neutronBodyFieldNames[key]; return known }
func poolRecordJSON(raw json.RawMessage) error {
	if !utf8.Valid(raw) || !json.Valid(raw) {
		return poolRecordInvalid("complete UTF-8 JSON required")
	}
	return nil
}
func poolRecordCollect(raw json.RawMessage, list bool) (map[string]json.RawMessage, error) {
	members, err := cloudfilter.ObjectMembers(raw)
	if err != nil {
		return nil, err
	}
	fields := make(map[string]json.RawMessage, len(members))
	for _, member := range members {
		if member.Key == "self" {
			continue
		}
		if list {
			switch member.Key {
			case "cls", "connection", "microversion", "_synchronized":
				return nil, poolRecordInvalid("list constructor collides with %q", member.Key)
			}
		}
		fields[poolRecordCanonical(member.Key)] = bytes.Clone(member.Value)
	}
	return fields, nil
}
func (v *SubnetPoolRecord) UnmarshalJSON(raw []byte) error {
	var wire resource.RawResource
	if err := json.Unmarshal(raw, &wire); err != nil {
		return err
	}
	fields, err := poolRecordCollect(raw, true)
	if err != nil {
		return err
	}
	v.Wire = wire.Clone()
	v.fields = fields
	return nil
}
func poolRecordMetadata(v *SubnetPoolRecord) *resource.Metadata {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	return &v.Wire.Metadata
}

// Python response.json decodes decimal/exponent numbers to binary64. The
// shared exact-decimal helper intentionally has a different numeric policy.
func poolRecordBoolean(raw json.RawMessage) (json.RawMessage, error) {
	token := bytes.TrimSpace(raw)
	if len(token) > 0 && (token[0] == '-' || token[0] >= '0' && token[0] <= '9') && strings.ContainsAny(string(token), ".eE") {
		number, err := strconv.ParseFloat(string(token), 64)
		if err != nil {
			if numeric, ok := err.(*strconv.NumError); !ok || numeric.Err != strconv.ErrRange {
				return nil, err
			}
		}
		if number == 0 {
			return json.RawMessage("false"), nil
		}
		return json.RawMessage("true"), nil
	}
	return jsonfilter.BooleanJSON(raw)
}
func poolRecordDescriptor(key string, raw json.RawMessage) (json.RawMessage, error) {
	if err := poolRecordJSON(raw); err != nil {
		return nil, err
	}
	token := bytes.TrimSpace(raw)
	if bytes.Equal(token, []byte("null")) {
		return bytes.Clone(raw), nil
	}
	switch key {
	case "default_prefix_length", "default_quota", "ip_version", "maximum_prefix_length", "minimum_prefix_length", "revision_number":
		return jsonfilter.DescriptorIntegerJSON(raw)
	case "is_default", "is_shared":
		return poolRecordBoolean(raw)
	case "prefixes", "tags":
		if token[0] != '[' {
			return json.RawMessage(append(append([]byte{'['}, raw...), ']')), nil
		}
	}
	return bytes.Clone(raw), nil
}
func poolRecordProject(v *SubnetPoolRecord, fields map[string]json.RawMessage) error {
	body := poolRecordClone(fields)
	if _, present := body["project_id"]; !present {
		if tenant, present := body["tenant_id"]; present {
			body["project_id"] = bytes.Clone(tenant)
		}
	}
	for canonical := range neutronBodyFieldNames {
		raw, present := body[canonical]
		if !present {
			raw = json.RawMessage("null")
			if canonical == "tags" {
				raw = json.RawMessage("[]")
			}
		}
		converted, err := poolRecordDescriptor(canonical, raw)
		if err != nil {
			return poolRecordInvalid("%s descriptor: %v", canonical, err)
		}
		body[canonical] = converted
	}
	delete(body, "self")
	body["location"] = json.RawMessage("null")
	v.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: body, Header: v.Header.Clone(), StatusCode: v.StatusCode}}
	return nil
}
func poolRecordReceipt(response *rest.Response) *SubnetPoolRecord {
	if response == nil {
		return nil
	}
	return &SubnetPoolRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
}
func poolRecordResponse(response *rest.Response, seed map[string]json.RawMessage) (*SubnetPoolRecord, error) {
	value := poolRecordReceipt(response)
	fields := poolRecordClone(seed)
	if !utf8.Valid(response.Body) {
		return value, poolRecordInvalid("response must be UTF-8")
	}
	if json.Valid(response.Body) {
		members, err := cloudfilter.ObjectMembers(response.Body)
		if err != nil {
			return value, poolRecordInvalid("member response must be an object: %v", err)
		}
		raw := response.Body
		for _, member := range members {
			if member.Key == "subnetpool" {
				raw = member.Value
			}
		}
		var actual resource.RawResource
		if err := json.Unmarshal(raw, &actual); err != nil {
			return value, err
		}
		actual.Header, actual.StatusCode = response.Header.Clone(), response.StatusCode
		value.Wire = actual.Clone()
		incoming, err := poolRecordCollect(raw, false)
		if err != nil {
			return value, err
		}
		for key, raw := range incoming {
			fields[key] = bytes.Clone(raw)
		}
	}
	return value, poolRecordProject(value, fields)
}
func poolRecordText(value *SubnetPoolRecord, key string) string {
	if value == nil || value.Resource == nil {
		return ""
	}
	var text string
	_ = json.Unmarshal(value.Resource.Body[key], &text)
	return text
}
func poolRecordMarker(value *SubnetPoolRecord) (string, error) {
	if value == nil || value.Wire == nil {
		return "", poolRecordInvalid("actual marker row required")
	}
	var id string
	if json.Unmarshal(value.Wire.Body["id"], &id) != nil || id == "" {
		return "", poolRecordInvalid("marker must be a nonempty wire string")
	}
	return id, nil
}

package portforwarding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// PortForwardingRecord separates the pinned Python descriptor view from the
// original wire row and HTTP observation. ParentID is the fixed URI component;
// it is not a Body attribute. Unknown Body extensions remain in both views.
type PortForwardingRecord struct {
	Resource, Wire *resource.RawResource
	ParentID       string
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

func (v *PortForwardingRecord) UnmarshalJSON(data []byte) error {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	return json.Unmarshal(data, v.Wire)
}
func forwardingMetadata(v *PortForwardingRecord) *resource.Metadata {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	return &v.Wire.Metadata
}

var forwardingBodyFields = []string{"id", "name", "internal_port_id", "internal_ip_address", "internal_port", "external_port", "protocol", "description"}

func forwardingKnown(key string) bool {
	for _, field := range forwardingBodyFields {
		if key == field {
			return true
		}
	}
	return false
}
func forwardingInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func forwardingClone(values map[string]json.RawMessage) map[string]json.RawMessage {
	result := make(map[string]json.RawMessage, len(values))
	for key, raw := range values {
		result[key] = bytes.Clone(raw)
	}
	return result
}
func projectForwarding(v *PortForwardingRecord, seed map[string]json.RawMessage) error {
	body := forwardingClone(seed)
	if v.Wire != nil {
		for key, raw := range v.Wire.Body {
			if key != "self" {
				body[key] = bytes.Clone(raw)
			}
		}
	}
	for _, key := range forwardingBodyFields {
		if _, ok := body[key]; !ok {
			body[key] = json.RawMessage("null")
		}
	}
	for _, key := range []string{"internal_port", "external_port"} {
		raw, err := jsonfilter.DescriptorIntegerJSON(body[key])
		if err != nil {
			return forwardingInvalid("%s descriptor: %v", key, err)
		}
		body[key] = raw
	}
	body["location"] = json.RawMessage("null")
	// Resource.to_dict omits URI components. If the server returns the URI name
	// as an unknown Body key, attribute access still selects the fixed parent.
	if _, ok := body["floatingip_id"]; ok {
		body["floatingip_id"], _ = json.Marshal(v.ParentID)
	}
	v.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: body, Header: v.Header.Clone(), StatusCode: v.StatusCode}}
	return nil
}
func forwardingFromResponse(r *rest.Response, parent string, seed map[string]json.RawMessage) (*PortForwardingRecord, error) {
	if r == nil {
		return nil, nil
	}
	v := &PortForwardingRecord{ParentID: parent, Envelope: bytes.Clone(r.Body), Header: r.Header.Clone(), StatusCode: r.StatusCode}
	if !json.Valid(r.Body) {
		return v, projectForwarding(v, seed)
	}
	if !utf8.Valid(r.Body) {
		return v, forwardingInvalid("response must be UTF-8 JSON")
	}
	raw := bytes.TrimSpace(r.Body)
	if len(raw) == 0 || raw[0] != '{' {
		return v, forwardingInvalid("response must be an object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return v, err
	}
	if nested, ok := fields["port_forwarding"]; ok {
		raw = nested
	}
	var wire resource.RawResource
	if err := json.Unmarshal(raw, &wire); err != nil {
		return v, err
	}
	wire.Header, wire.StatusCode = r.Header.Clone(), r.StatusCode
	v.Wire = wire.Clone()
	return v, projectForwarding(v, seed)
}
func forwardingText(v *PortForwardingRecord, key string) string {
	if v == nil || v.Resource == nil {
		return ""
	}
	var text string
	_ = json.Unmarshal(v.Resource.Body[key], &text)
	return text
}
func forwardingMarker(v *PortForwardingRecord) (string, error) {
	if v == nil || v.Wire == nil {
		return "", forwardingInvalid("pagination marker is missing")
	}
	var id string
	if err := json.Unmarshal(v.Wire.Body["id"], &id); err != nil || id == "" {
		return "", forwardingInvalid("pagination marker must be a nonempty string ID")
	}
	return id, nil
}

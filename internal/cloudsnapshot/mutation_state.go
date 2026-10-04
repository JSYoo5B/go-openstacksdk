package cloudsnapshot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"gophercloudsdk/internal/cloudfilter"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/resource"
)

// mutationState contains known logical inputs, never an HTTP response record.
// Overlay preserves prior fields and resolves wire/attribute aliases in parsed
// dictionary order. Descriptor conversion happens only when a view is built.
type mutationState [len(descriptors)]json.RawMessage

func (s *mutationState) overlay(raw json.RawMessage) error {
	members, err := cloudfilter.ObjectMembers(raw)
	if err != nil {
		return err
	}
	for _, member := range members {
		for i, field := range descriptors {
			if member.Key == field.attribute || member.Key == field.wire {
				s[i] = bytes.Clone(member.Value)
			}
		}
	}
	return nil
}

func (s *mutationState) object() json.RawMessage {
	var out bytes.Buffer
	out.WriteByte('{')
	first := true
	for i, field := range descriptors {
		if s[i] == nil {
			continue
		}
		if !first {
			out.WriteByte(',')
		}
		first = false
		key, _ := json.Marshal(field.attribute)
		out.Write(key)
		out.WriteByte(':')
		out.Write(s[i])
	}
	out.WriteByte('}')
	return bytes.Clone(out.Bytes())
}

func (s *mutationState) view(location resource.CloudLocation) (json.RawMessage, error) {
	view, _, err := normalize(s.object(), nil, false, location)
	return view, err
}

func (s *mutationState) id() json.RawMessage {
	if s[12] == nil {
		return json.RawMessage("null")
	}
	return bytes.Clone(s[12])
}

// Route validation is reached only when a physical member request is needed.
// No-wait and cached-ready results can expose an arbitrary nullable source ID.
func (s *mutationState) routeID() (string, error) {
	raw := bytes.TrimSpace(s.id())
	if len(raw) == 0 || raw[0] != '"' {
		return "", invalid("snapshot route ID must be a nonempty string")
	}
	var id string
	if err := json.Unmarshal(raw, &id); err != nil {
		return "", invalid("snapshot route ID: %v", err)
	}
	if err := ValidateID(id); err != nil {
		return "", err
	}
	return id, nil
}

func (s *mutationState) status(nullable bool) (string, error) {
	raw := bytes.TrimSpace(s[8])
	if nullable && (len(raw) == 0 || bytes.Equal(raw, []byte("null"))) {
		return "", nil
	}
	if len(raw) == 0 || raw[0] != '"' {
		return "", invalid("snapshot status must be a string at this wait phase")
	}
	var status string
	if err := json.Unmarshal(raw, &status); err != nil {
		return "", invalid("snapshot status: %v", err)
	}
	return status, nil
}

// An empty or malformed UTF-8 JSON reply leaves the Resource unchanged. It
// does not manufacture an empty actual object. Valid wrong-shaped JSON fails.
func mutationObject(wire *rest.Response) (json.RawMessage, *resource.RawResource, error) {
	if !utf8.Valid(wire.Body) {
		return nil, nil, wire.Fail(fmt.Errorf("snapshot response must be UTF-8 JSON"))
	}
	if !json.Valid(wire.Body) {
		return nil, nil, nil
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(wire.Body, &envelope); err != nil {
		return nil, nil, wire.Fail(err)
	}
	if envelope == nil {
		return nil, nil, wire.Fail(fmt.Errorf("snapshot response must be a nonnull JSON object"))
	}
	raw, present := envelope["snapshot"]
	if !present {
		raw = wire.Body
	}
	var actual resource.RawResource
	if err := json.Unmarshal(raw, &actual); err != nil {
		return nil, nil, wire.Fail(err)
	}
	actual.Header, actual.StatusCode = wire.Header.Clone(), wire.StatusCode
	return bytes.Clone(raw), &actual, nil
}

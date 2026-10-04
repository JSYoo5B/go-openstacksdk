package cloudsnapshot

import (
	"bytes"
	"encoding/json"

	"gophercloudsdk/internal/cloudfilter"
)

// Only the two library-owned Cinder descriptor tables use these helpers.
// Resource-specific arrays retain their own dimensions and field indices.
func overlayMutationFields(values []json.RawMessage, fields []descriptor, raw json.RawMessage) error {
	members, err := cloudfilter.ObjectMembers(raw)
	if err != nil {
		return err
	}
	for _, member := range members {
		for i, field := range fields {
			if member.Key == field.attribute || member.Key == field.wire {
				values[i] = bytes.Clone(member.Value)
			}
		}
	}
	return nil
}

func mutationFieldsObject(values []json.RawMessage, fields []descriptor) json.RawMessage {
	var out bytes.Buffer
	out.WriteByte('{')
	first := true
	for i, field := range fields {
		if values[i] == nil {
			continue
		}
		if !first {
			out.WriteByte(',')
		}
		first = false
		key, _ := json.Marshal(field.attribute)
		out.Write(key)
		out.WriteByte(':')
		out.Write(values[i])
	}
	out.WriteByte('}')
	return bytes.Clone(out.Bytes())
}

func mutationField(raw json.RawMessage) json.RawMessage {
	if raw == nil {
		return json.RawMessage("null")
	}
	return bytes.Clone(raw)
}

func mutationRouteID(raw json.RawMessage, kind string) (string, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '"' {
		return "", invalid("%s route ID must be a nonempty string", kind)
	}
	var id string
	if err := json.Unmarshal(raw, &id); err != nil {
		return "", invalid("%s route ID: %v", kind, err)
	}
	if err := validateIDFor(id, kind); err != nil {
		return "", err
	}
	return id, nil
}

func mutationStatus(raw json.RawMessage, kind string, nullable bool) (string, error) {
	raw = bytes.TrimSpace(raw)
	if nullable && (len(raw) == 0 || bytes.Equal(raw, []byte("null"))) {
		return "", nil
	}
	if len(raw) == 0 || raw[0] != '"' {
		return "", invalid("%s status must be a string at this wait phase", kind)
	}
	var status string
	if err := json.Unmarshal(raw, &status); err != nil {
		return "", invalid("%s status: %v", kind, err)
	}
	return status, nil
}

package cindervolume

import (
	"bytes"
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

// Fields canonicalizes recognized aliases in JSON member order while retaining unknown inputs.
func Fields(raw json.RawMessage) (map[string]json.RawMessage, error) {
	if !utf8.Valid(raw) || !json.Valid(raw) || len(bytes.TrimSpace(raw)) == 0 || bytes.TrimSpace(raw)[0] != '{' {
		return nil, fmt.Errorf("volume mutation fields must be a UTF-8 JSON object")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	_, _ = decoder.Token()
	fields := make(map[string]json.RawMessage)
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		name := key.(string)
		for _, descriptor := range descriptors {
			if name == descriptor.Attribute || name == descriptor.Wire {
				name = descriptor.Wire
				break
			}
		}
		fields[name] = bytes.Clone(value)
	}
	return fields, nil
}

func MergeResponse(fields map[string]json.RawMessage, body []byte) error {
	if !utf8.Valid(body) {
		return fmt.Errorf("volume update response must be valid UTF-8")
	}
	if !json.Valid(body) {
		return nil
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil || envelope == nil {
		return fmt.Errorf("volume update response must be a nonnull JSON object")
	}
	raw := json.RawMessage(body)
	if selected, present := envelope["volume"]; present {
		raw = selected
	}
	server, err := Fields(raw)
	if err != nil {
		return err
	}
	for _, descriptor := range descriptors {
		if value, exists := server[descriptor.Wire]; exists {
			fields[descriptor.Wire] = bytes.Clone(value)
		}
	}
	return nil
}

package secrets

import (
	"bytes"
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

var findRecordDescriptors = map[string]string{
	"algorithm": "algorithm", "bit_length": "bit_length", "content_types": "content_types",
	"expires_at": "expiration", "created_at": "created", "updated_at": "updated", "mode": "mode",
	"name": "name", "secret_ref": "secret_ref", "secret_type": "secret_type", "status": "status",
	"payload": "payload", "payload_content_type": "payload_content_type", "payload_content_encoding": "payload_content_encoding",
}

func findRecordFields(fields map[string]json.RawMessage) map[string]json.RawMessage {
	result := make(map[string]json.RawMessage, len(fields))
	for key, raw := range fields {
		result[key] = bytes.Clone(raw)
	}
	// Source consumes attribute aliases and canonical response fields. Canonical
	// wire spelling wins deterministically if both are present (Go policy).
	for attribute, canonical := range findRecordDescriptors {
		if _, exists := result[canonical]; !exists {
			if raw, present := fields[attribute]; present {
				result[canonical] = bytes.Clone(raw)
			}
		}
	}
	if _, exists := result["secret_ref"]; !exists {
		if raw, present := fields["secret_id"]; present {
			result["secret_ref"] = bytes.Clone(raw)
		}
	}
	return result
}

func projectFindRecord(wire *resource.RawResource, fields map[string]json.RawMessage, requestID *string) (*resource.RawResource, error) {
	view := wire.Clone()
	view.Body = make(map[string]json.RawMessage)
	for attribute, canonical := range findRecordDescriptors {
		raw, err := resource.BodyRecordField(fields, canonical, resource.BodyFieldJSON)
		if err != nil {
			return nil, err
		}
		view.Body[attribute] = raw
	}
	if raw, exists := fields["id"]; exists {
		view.Body["id"] = bytes.Clone(raw)
	} else if requestID != nil {
		view.Body["id"], _ = json.Marshal(*requestID)
	} else {
		view.Body["id"], _ = resource.BodyRecordField(fields, "secret_ref", resource.BodyFieldJSON)
	}
	formatted, err := secretBodyReferenceID(fields)
	if err != nil {
		return nil, err
	}
	view.Body["secret_id"] = formatted
	view.Body["location"] = json.RawMessage("null")
	raw := bytes.TrimSpace(view.Body["content_types"])
	if !bytes.Equal(raw, []byte("null")) && len(raw) > 0 && raw[0] != '{' {
		view.Body["content_types"] = json.RawMessage("{}")
	}
	return view, nil
}

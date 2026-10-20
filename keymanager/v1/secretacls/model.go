// Package secretacls implements the Python key_manager secret ACL proxy
// operations over one fixed secret. Response references are evidence only.
package secretacls

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// SecretACL is the Python SecretACL Resource view. SecretID is the URI field
// from the request scope. Read and ACLRef are the declared Body fields: JSON
// null when absent, a request value when sent, overlaid by a JSON object
// response. Metadata keeps the actual response body, header and status.
type SecretACL struct {
	resource.Metadata
	SecretID string
	Read     json.RawMessage
	ACLRef   json.RawMessage
}

// ACLInput carries the Python **attrs that SecretACL declares. A nil field is
// not sent; Read must be a JSON object because the declared type is dict.
type ACLInput struct {
	Read   json.RawMessage
	ACLRef json.RawMessage
}

func (input ACLInput) body() (map[string]json.RawMessage, error) {
	body := make(map[string]json.RawMessage)
	if input.Read != nil {
		if first := bytes.TrimSpace(input.Read); len(first) == 0 || first[0] != '{' || !json.Valid(first) {
			return nil, fmt.Errorf("%w: secret ACL read must be a JSON object", resource.ErrInvalidOption)
		}
		body["read"] = bytes.Clone(input.Read)
	}
	if input.ACLRef != nil {
		if !json.Valid(input.ACLRef) {
			return nil, fmt.Errorf("%w: secret ACL acl_ref must be complete JSON", resource.ErrInvalidOption)
		}
		body["acl_ref"] = bytes.Clone(input.ACLRef)
	}
	return body, nil
}

// overlay follows Python _translate_response: empty or invalid JSON is
// ignored, a valid non-object fails, and only declared keys of an object
// replace the seeded values. A dict-typed read must stay an object or null.
func (value *SecretACL) overlay(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || !json.Valid(trimmed) {
		return nil
	}
	if trimmed[0] != '{' {
		return fmt.Errorf("%w: secret ACL response must be a JSON object", resource.ErrInvalidOption)
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &body); err != nil {
		return fmt.Errorf("%w: secret ACL response must be a JSON object", resource.ErrInvalidOption)
	}
	if read, present := body["read"]; present {
		if first := bytes.TrimSpace(read); first[0] != '{' && !bytes.Equal(first, []byte("null")) {
			return fmt.Errorf("%w: secret ACL read must be a JSON object or null", resource.ErrInvalidOption)
		}
		value.Read = bytes.Clone(read)
	}
	if ref, present := body["acl_ref"]; present {
		value.ACLRef = bytes.Clone(ref)
	}
	value.Body = body
	return nil
}

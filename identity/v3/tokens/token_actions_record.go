package tokens

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// TokenRecord owns the Python Token descriptor view and the original JSON
// response independently. expires_at and issued_at remain raw descriptor values;
// timestamps, unknown fields and large numbers are never forced through Token.
// SubjectToken comes from the response header, not the submitted token or id.
// Resource, Wire, Envelope and Header can be changed independently by callers.
type TokenRecord struct {
	Resource, Wire *resource.RawResource
	SubjectToken   *string
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

func tokenRecordReceipt(response *rest.Response) *TokenRecord {
	if response == nil {
		return nil
	}
	return &TokenRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
}

// Token has no connected constructor here: Source ret = cls() means location
// is null. id/name are inherited Body fields, and id is not a subject-token alias.
var tokenRecordFields = [...]string{
	"id", "name", "audit_ids", "catalog", "expires_at", "issued_at", "methods",
	"user", "project", "domain", "is_domain", "system", "roles",
}

func translateTokenRecord(response *rest.Response) (*TokenRecord, error) {
	value := tokenRecordReceipt(response)
	view := make(map[string]json.RawMessage, len(tokenRecordFields)+2)
	for _, key := range tokenRecordFields {
		view[key] = json.RawMessage("null")
	}
	view["location"], view["subject_token"] = json.RawMessage("null"), json.RawMessage("null")
	for key, values := range response.Header {
		if (strings.EqualFold(key, "X-Subject-Token") || strings.EqualFold(key, "subject_token")) && len(values) != 0 {
			subject := strings.Join(values, ", ")
			value.SubjectToken = &subject
			view["subject_token"], _ = json.Marshal(subject)
			break
		}
	}
	value.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: view, Header: response.Header.Clone(), StatusCode: response.StatusCode}}
	// Source ignores response.json() ValueError, including an empty204 body.
	// Valid nonobject JSON is different: the later pop/consume operation fails.
	if !json.Valid(response.Body) {
		return value, nil
	}
	if !utf8.Valid(response.Body) {
		return value, tokenRecordInvalid("token response must be UTF-8")
	}
	var body json.RawMessage = bytes.Clone(response.Body)
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return value, tokenRecordInvalid("token response must be a JSON object")
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return value, err
	}
	if nested, present := envelope["token"]; present {
		body = nested
	}
	var wire resource.RawResource
	if err := json.Unmarshal(body, &wire); err != nil {
		return value, err
	}
	wire.Header, wire.StatusCode = response.Header.Clone(), response.StatusCode
	value.Wire = wire.Clone()
	for _, key := range tokenRecordFields {
		raw := wire.Body[key]
		if raw == nil {
			continue
		}
		var projected json.RawMessage
		var err error
		switch key {
		case "audit_ids", "methods":
			projected, err = tokenRecordList(raw, false)
		case "catalog", "roles":
			projected, err = tokenRecordList(raw, true)
		case "user", "project", "domain", "system":
			projected = tokenRecordDictionary(raw)
		case "is_domain":
			projected, err = resource.BodyRecordField(wire.Body, key, resource.BodyFieldBoolean)
		default:
			projected = bytes.Clone(raw)
		}
		if err != nil {
			return value, err
		}
		view[key] = projected
	}
	return value, nil
}

func tokenRecordDictionary(raw json.RawMessage) json.RawMessage {
	trimmed := bytes.TrimSpace(raw)
	if bytes.Equal(trimmed, []byte("null")) || len(trimmed) != 0 && trimmed[0] == '{' {
		return bytes.Clone(raw)
	}
	return json.RawMessage("{}")
}

func tokenRecordList(raw json.RawMessage, dictionaries bool) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	if bytes.Equal(trimmed, []byte("null")) {
		return bytes.Clone(raw), nil
	}
	var values []json.RawMessage
	if len(trimmed) != 0 && trimmed[0] == '[' {
		if err := json.Unmarshal(raw, &values); err != nil {
			return nil, err
		}
	} else {
		values = []json.RawMessage{bytes.Clone(raw)}
	}
	if dictionaries {
		for i, item := range values {
			// _convert_type(item, dict) maps even a null list element to {}.
			if item = bytes.TrimSpace(item); len(item) == 0 || item[0] != '{' {
				values[i] = json.RawMessage("{}")
			}
		}
	}
	return json.Marshal(values)
}

func tokenRecordInvalid(message string) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, message)
}

package ports

import (
	"bytes"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
)

// BulkPortInput owns one set of raw Source attributes in insertion order.
// Descriptor conversion applies to response views, never to request Body values.
type BulkPortInput struct {
	Attributes map[string]json.RawMessage
	raw        *json.RawMessage
	order      []string
	err        error
}
type BulkPortAttribute func(*BulkPortInput) error

func cloneBulkPortInput(value BulkPortInput) BulkPortInput {
	fields := make(map[string]json.RawMessage, len(value.Attributes))
	for key, raw := range value.Attributes {
		fields[key] = bytes.Clone(raw)
	}
	value.Attributes = fields
	value.order = slices.Clone(value.order)
	if value.raw != nil {
		raw := json.RawMessage(bytes.Clone(*value.raw))
		value.raw = &raw
	}
	return value
}
func BulkPort(options ...BulkPortAttribute) BulkPortInput {
	input := BulkPortInput{Attributes: make(map[string]json.RawMessage)}
	for _, apply := range slices.Clone(options) {
		if apply == nil {
			input.err = findInvalid("nil bulk port attribute option")
			break
		}
		next := cloneBulkPortInput(input)
		if err := apply(&next); err != nil {
			input.err = err
			break
		}
		input = cloneBulkPortInput(next)
	}
	return input
}
func BulkPortJSON(value json.RawMessage) BulkPortInput {
	raw := json.RawMessage(bytes.Clone(value))
	return BulkPortInput{raw: &raw}
}
func BulkPortFields(fields map[string]json.RawMessage) BulkPortInput {
	return cloneBulkPortInput(BulkPortInput{Attributes: fields})
}
func WithBulkPortAttribute(key string, value any) BulkPortAttribute {
	if err := bulkConstructorKey(key); err != nil {
		return func(*BulkPortInput) error { return err }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *BulkPortInput) error {
		if err != nil {
			return errors.Join(findInvalid("bulk attribute encoding"), err)
		}
		if c.raw != nil {
			return findInvalid("raw bulk JSON cannot be mixed with attribute options")
		}
		if c.Attributes == nil {
			c.Attributes = make(map[string]json.RawMessage)
		}
		c.Attributes[key] = bytes.Clone(owned)
		if !slices.Contains(c.order, key) {
			c.order = append(c.order, key)
		}
		return nil
	}
}
func bulkConstructorKey(key string) error {
	if !utf8.ValidString(key) {
		return findInvalid("bulk attribute name must be UTF-8")
	}
	switch key {
	case "cls", "self", "connection", "_synchronized":
		return findInvalid("%q duplicates a Source bulk constructor argument", key)
	}
	return nil
}

type BulkPortOpts struct {
	Headers      map[string]string
	Microversion *string
}
type BulkPortOption func(*BulkPortOpts) error

func cloneBulkPortOpts(value BulkPortOpts) BulkPortOpts {
	value.Headers = maps.Clone(value.Headers)
	if value.Headers == nil {
		value.Headers = make(map[string]string)
	}
	if value.Microversion != nil {
		v := *value.Microversion
		value.Microversion = &v
	}
	return value
}
func WithBulkPortOpts(value BulkPortOpts) BulkPortOption {
	owned := cloneBulkPortOpts(value)
	return func(c *BulkPortOpts) error { *c = cloneBulkPortOpts(owned); return nil }
}
func WithBulkPortHeaders(value map[string]string) BulkPortOption {
	owned := maps.Clone(value)
	return func(c *BulkPortOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithBulkPortMicroversion(value string) BulkPortOption {
	return func(c *BulkPortOpts) error { v := value; c.Microversion = &v; return nil }
}

func bulkPortMembers(input BulkPortInput) ([]cloudfilter.JSONMember, error) {
	if input.err != nil {
		return nil, input.err
	}
	if input.raw != nil {
		return cloudfilter.ObjectMembers(*input.raw)
	}
	keys := slices.Clone(input.order)
	missing := make([]string, 0)
	for key := range input.Attributes {
		if !slices.Contains(keys, key) {
			missing = append(missing, key)
		}
	}
	slices.Sort(missing)
	keys = append(keys, missing...)
	members := make([]cloudfilter.JSONMember, 0, len(keys))
	for _, key := range keys {
		raw, present := input.Attributes[key]
		if !present {
			continue
		}
		if raw == nil {
			raw = json.RawMessage("null")
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, findInvalid("bulk attribute %q must contain complete UTF-8 JSON", key)
		}
		members = append(members, cloudfilter.JSONMember{Key: key, Value: bytes.Clone(raw)})
	}
	return members, nil
}

func prepareBulkPort(input BulkPortInput) (map[string]json.RawMessage, map[string]string, error) {
	members, err := bulkPortMembers(input)
	if err != nil {
		return nil, nil, errors.Join(findInvalid("bulk port must be an attribute object"), err)
	}
	body := make(map[string]json.RawMessage)
	headers := make(map[string]string)
	for _, member := range members {
		key := member.Key
		if err := bulkConstructorKey(key); err != nil {
			return nil, nil, err
		}
		if key == "microversion" {
			continue
		}
		if key == "if_match" || strings.EqualFold(key, "if-match") {
			value, err := bulkHeaderValue(member.Value)
			if err != nil {
				return nil, nil, err
			}
			headers["If-Match"] = value
			continue
		}
		for _, field := range recordFindProfile.Body {
			if key == field.Name || key == field.Wire {
				key = field.Wire
				break
			}
		}
		body[key] = bytes.Clone(member.Value)
	}
	return body, headers, nil
}
func bulkHeaderValue(raw json.RawMessage) (string, error) {
	token := bytes.TrimSpace(raw)
	value := ""
	if len(token) > 0 && token[0] == '[' {
		var elements []json.RawMessage
		if err := json.Unmarshal(token, &elements); err != nil {
			return "", errors.Join(findInvalid("If-Match list requires strings"), err)
		}
		values := make([]string, len(elements))
		for i, element := range elements {
			if len(bytes.TrimSpace(element)) == 0 || bytes.TrimSpace(element)[0] != '"' {
				return "", findInvalid("If-Match list requires only strings")
			}
			if err := json.Unmarshal(element, &values[i]); err != nil {
				return "", err
			}
		}
		value = strings.Join(values, ", ")
	} else {
		var err error
		value, err = cloudfilter.PythonString(raw)
		if err != nil {
			return "", err
		}
	}
	if strings.ContainsAny(value, "\r\n") {
		return "", findInvalid("If-Match contains invalid header bytes")
	}
	return value, nil
}

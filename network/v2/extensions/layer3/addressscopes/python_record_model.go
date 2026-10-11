package addressscopes

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// AddressScopeRecord separates the pinned descriptor view from the original
// wire row and HTTP receipt. Source AddressScope inherits Resource directly;
// NetworkResource's revision_number/if_match and tag defaults do not apply.
type AddressScopeRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	row            json.RawMessage
}

func (v *AddressScopeRecord) UnmarshalJSON(data []byte) error {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	v.row = bytes.Clone(data)
	return json.Unmarshal(data, v.Wire)
}

var scopeRecordProfile = neutronfind.Profile{
	Kind: "addressscopes", Path: "address-scopes", Singular: "address_scope", Plural: "address_scopes",
	IDKey: "id", NameKey: "name", PreserveUnknownBody: true, NameQueryHint: true, VersionedPath: "/v2.0/address-scopes",
	Body: []neutronfind.Descriptor{
		{Name: "id", Wire: "id", Missing: "null"},
		{Name: "name", Wire: "name", Missing: "null"},
		{Name: "project_id", Wire: "project_id", Missing: "null", Alias: "tenant_id"},
		{Name: "tenant_id", Wire: "tenant_id", Missing: "null"},
		{Name: "ip_version", Wire: "ip_version", Kind: neutronfind.Integer, Missing: "null"},
		{Name: "is_shared", Wire: "shared", Kind: neutronfind.Boolean, Missing: "null"},
	},
	Query: []neutronfind.QueryPair{
		{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"},
		{Name: "fields", Wire: "fields"}, {Name: "name", Wire: "name"},
		{Name: "ip_version", Wire: "ip_version"}, {Name: "project_id", Wire: "project_id"},
		{Name: "sort_key", Wire: "sort_key"}, {Name: "sort_dir", Wire: "sort_dir"},
		{Name: "is_shared", Wire: "shared"},
	},
}

func scopeRecordInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func scopeRecordFields(values map[string]json.RawMessage) map[string]json.RawMessage {
	result := make(map[string]json.RawMessage, len(values))
	for key, raw := range values {
		result[key] = bytes.Clone(raw)
	}
	return result
}
func scopeRecordID(id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." || !utf8.ValidString(id) {
		return scopeRecordInvalid("nonblank UTF-8 identity segment is required")
	}
	for _, char := range id {
		if unicode.IsControl(char) {
			return scopeRecordInvalid("identity contains controls")
		}
	}
	return nil
}
func scopeRecordKnown(key string) bool {
	for _, field := range scopeRecordProfile.Body {
		if key == field.Name {
			return true
		}
	}
	return false
}
func scopeRecordProject(value *AddressScopeRecord, seed map[string]json.RawMessage) error {
	body := scopeRecordFields(seed)
	// Bulk constructor seeds are unordered; canonical values own simultaneous
	// wire aliases. HTTP response members retain Source's input-order merge.
	for _, field := range scopeRecordProfile.Body {
		if field.Name != field.Wire {
			if raw, supplied := body[field.Wire]; supplied {
				if _, canonical := body[field.Name]; !canonical {
					body[field.Name] = bytes.Clone(raw)
				}
				delete(body, field.Wire)
			}
		}
	}
	delete(body, "self")
	if value.Wire != nil {
		members, err := cloudfilter.ObjectMembers(value.row)
		if err != nil {
			return err
		}
		for _, member := range members {
			if member.Key == "self" {
				continue
			}
			consumed := false
			for _, field := range scopeRecordProfile.Body {
				if member.Key == field.Name || member.Key == field.Wire {
					body[field.Name] = bytes.Clone(member.Value)
					consumed = true
				}
			}
			if !consumed {
				body[member.Key] = bytes.Clone(member.Value)
			}
		}
	}
	for _, field := range scopeRecordProfile.Body {
		raw, supplied := body[field.Name]
		if !supplied && field.Alias != "" {
			raw, supplied = body[field.Alias]
		}
		if !supplied {
			body[field.Name] = json.RawMessage(field.Missing)
			continue
		}
		raw = bytes.Clone(raw)
		var err error
		if !bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			switch field.Kind {
			case neutronfind.Boolean:
				var truth bool
				truth, err = scopeRecordTruthy(raw)
				if err == nil {
					if truth {
						raw = json.RawMessage("true")
					} else {
						raw = json.RawMessage("false")
					}
				}
			case neutronfind.Integer:
				raw, err = jsonfilter.DescriptorIntegerJSON(raw)
			}
		}
		if err != nil {
			return fmt.Errorf("%w: %s descriptor: %w", resource.ErrInvalidOption, field.Name, err)
		}
		body[field.Name] = raw
	}
	if _, supplied := body["location"]; !supplied {
		body["location"] = json.RawMessage("null")
	}
	value.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: body, Header: value.Header.Clone(), StatusCode: value.StatusCode}}
	return nil
}
func scopeRecordResponse(response *rest.Response, seed map[string]json.RawMessage) (*AddressScopeRecord, error) {
	if response == nil {
		return nil, nil
	}
	value := &AddressScopeRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	// Source fetch ignores JSON ValueError, keeping constructor state. It does
	// not invent a wire object for malformed JSON or a bodyless response.
	if !json.Valid(response.Body) {
		return value, scopeRecordProject(value, seed)
	}
	if !utf8.Valid(response.Body) {
		return value, scopeRecordInvalid("response must contain UTF-8 JSON")
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(response.Body, &envelope); err != nil {
		return value, err
	}
	raw := response.Body
	if nested, supplied := envelope[scopeRecordProfile.Singular]; supplied {
		raw = nested
	}
	if err := value.UnmarshalJSON(raw); err != nil {
		return value, err
	}
	value.Wire.Header, value.Wire.StatusCode = response.Header.Clone(), response.StatusCode
	return value, scopeRecordProject(value, seed)
}
func scopeRecordMetadata(value *AddressScopeRecord) *resource.Metadata {
	if value.Wire == nil {
		value.Wire = &resource.RawResource{}
	}
	return &value.Wire.Metadata
}
func scopeRecordText(value *AddressScopeRecord, key string) string {
	if value == nil || value.Resource == nil {
		return ""
	}
	var result string
	_ = json.Unmarshal(value.Resource.Body[key], &result)
	return result
}
func scopeRecordMarker(value *AddressScopeRecord) (string, error) {
	if value == nil || value.Wire == nil {
		return "", scopeRecordInvalid("pagination marker requires actual response ID")
	}
	values, err := cloudfilter.RequestQueryValues(value.Wire.Body["id"])
	if err != nil || len(values) != 1 || strings.TrimSpace(values[0]) == "" {
		return "", errors.Join(scopeRecordInvalid("pagination marker requires one nonempty query value"), err)
	}
	return values[0], nil
}

// JSON fraction/exponent tokens become Python binary64. This affects boolean
// descriptors and paging truthiness but does not round actual receipt bytes.
func scopeRecordTruthy(raw json.RawMessage) (bool, error) {
	if !utf8.Valid(raw) || !json.Valid(raw) {
		return false, scopeRecordInvalid("truthiness requires complete UTF-8 JSON")
	}
	token := bytes.TrimSpace(raw)
	if (token[0] == '-' || token[0] >= '0' && token[0] <= '9') && strings.ContainsAny(string(token), ".eE") {
		number, err := strconv.ParseFloat(string(token), 64)
		if err != nil && !errors.Is(err, strconv.ErrRange) {
			return false, err
		}
		return number != 0, nil
	}
	return cloudfilter.PythonTruthy(raw)
}
func scopeRecordLocalMatch(actual, filter json.RawMessage) (bool, error) {
	if len(actual) == 0 {
		actual = json.RawMessage("null")
	}
	trim := bytes.TrimSpace(filter)
	if len(trim) > 0 && trim[0] == '{' {
		members, err := cloudfilter.ObjectMembers(trim)
		if err != nil {
			return false, err
		}
		truthy, err := scopeRecordTruthy(actual)
		if err != nil || !truthy {
			return false, err
		}
		if len(members) == 0 {
			return true, nil
		}
		var observed map[string]json.RawMessage
		if err := json.Unmarshal(actual, &observed); err != nil || observed == nil {
			return false, scopeRecordInvalid("local dictionary filter requires actual mapping")
		}
		for _, member := range members {
			matched, err := scopeRecordLocalMatch(observed[member.Key], member.Value)
			if err != nil || !matched {
				return matched, err
			}
		}
		return true, nil
	}
	return jsonfilter.EqualPythonJSON(actual, filter)
}

package containers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ContainerRecord separates a declared Source-shaped view, request seeds and
// complete physical PUT evidence. Resource includes unobserved seed attributes;
// it does not assert that a successful PUT returned refreshed container state.
type ContainerRecord struct {
	Resource, Wire        *resource.RawResource
	SeedBody, SeedHeaders map[string]json.RawMessage
	Metadata              map[string]string
	Envelope              json.RawMessage
	Header                http.Header
	StatusCode            int
}

type recordHeaderField struct{ field, wire, kind string }

var recordHeaderFields = []recordHeaderField{
	{"object_count", "x-container-object-count", "int"}, {"bytes_used", "x-container-bytes-used", "int"},
	{"timestamp", "x-timestamp", ""}, {"is_newest", "x-newest", "bool"},
	{"read_ACL", "x-container-read", ""}, {"write_ACL", "x-container-write", ""},
	{"sync_to", "x-container-sync-to", ""}, {"sync_key", "x-container-sync-key", ""},
	{"versions_location", "x-versions-location", ""}, {"history_location", "x-history-location", ""},
	{"content_type", "content-type", ""}, {"is_content_type_detected", "x-detect-content-type", "bool"},
	{"storage_policy", "x-storage-policy", ""}, {"if_none_match", "if-none-match", ""},
	{"meta_temp_url_key", "x-container-meta-temp-url-key", ""}, {"meta_temp_url_key_2", "x-container-meta-temp-url-key-2", ""},
}

func recordHeader(key string) (recordHeaderField, bool) {
	for _, decl := range recordHeaderFields {
		if key == decl.field || strings.EqualFold(key, decl.wire) {
			return decl, true
		}
	}
	return recordHeaderField{}, false
}
func cloneRecordFields(values map[string]json.RawMessage) map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, len(values))
	for key, raw := range values {
		out[key] = bytes.Clone(raw)
	}
	return out
}
func recordHeaderString(raw json.RawMessage) (string, error) {
	trim := bytes.TrimSpace(raw)
	if len(trim) > 0 && trim[0] == '[' {
		var list []json.RawMessage
		if err := json.Unmarshal(trim, &list); err != nil {
			return "", err
		}
		stringsOnly := make([]string, len(list))
		for index, item := range list {
			if len(item) == 0 || item[0] != '"' {
				return "", metadataInvalid("container Header lists require strings")
			}
			if err := json.Unmarshal(item, &stringsOnly[index]); err != nil {
				return "", err
			}
		}
		return strings.Join(stringsOnly, ", "), nil
	}
	return cloudfilter.PythonString(raw)
}

// recordDescriptors validates the same typed seed descriptors Source accesses
// during Resource construction. Callers retain raw seeds and request spellings.
func recordDescriptors(body, headers map[string]json.RawMessage) (map[string]json.RawMessage, map[string]json.RawMessage, error) {
	convertedBody := cloneRecordFields(body)
	for _, key := range []string{"count", "bytes"} {
		if raw, present := body[key]; present {
			value, err := jsonfilter.DescriptorIntegerJSON(raw)
			if err != nil {
				return nil, nil, err
			}
			convertedBody[key] = value
		}
	}
	convertedHeaders := cloneRecordFields(headers)
	for _, decl := range recordHeaderFields {
		raw, present := headers[decl.field]
		if !present {
			continue
		}
		var err error
		switch decl.kind {
		case "int":
			raw, err = jsonfilter.DescriptorIntegerJSON(raw)
		case "bool":
			raw, err = jsonfilter.BooleanJSON(raw)
		}
		if err != nil {
			return nil, nil, err
		}
		convertedHeaders[decl.field] = raw
	}
	return convertedBody, convertedHeaders, nil
}

func recordProjection(response *rest.Response, bodySeed, headerSeed map[string]json.RawMessage) (*ContainerRecord, error) {
	if response == nil {
		return nil, nil
	}
	record := &ContainerRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode, SeedBody: cloneRecordFields(bodySeed), SeedHeaders: cloneRecordFields(headerSeed)}
	record.Wire = &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{}, Header: response.Header.Clone(), StatusCode: response.StatusCode}}
	body := cloneRecordFields(bodySeed)
	headers := cloneRecordFields(headerSeed)
	// Source ignores a body which fails JSON parsing, but a parsed scalar/list
	// cannot satisfy Resource's body-pop/mapping translation and is an error.
	if json.Valid(response.Body) {
		if !utf8.Valid(response.Body) {
			return record, fmt.Errorf("container response must be UTF-8")
		}
		if err := json.Unmarshal(response.Body, record.Wire); err != nil {
			return record, err
		}
		for _, key := range []string{"id", "name", "count", "bytes"} {
			if raw, present := record.Wire.Body[key]; present {
				body[key] = bytes.Clone(raw)
			}
		}
	}
	record.Metadata = make(map[string]string)
	seen := make(map[string]bool)
	for key, values := range response.Header {
		decl, known := recordHeader(key)
		custom := strings.HasPrefix(strings.ToLower(key), "x-container-meta-")
		if !known && !custom {
			continue
		}
		canonical := http.CanonicalHeaderKey(key)
		if seen[canonical] || len(values) != 1 {
			return record, fmt.Errorf("container response header %q is ambiguous", key)
		}
		seen[canonical] = true
		if known {
			raw, _ := json.Marshal(values[0])
			headers[decl.field] = raw
		} else {
			record.Metadata[strings.ToLower(key[len("x-container-meta-"):])] = values[0]
		}
	}
	convertedBody, convertedHeaders, err := recordDescriptors(body, headers)
	if err != nil {
		return record, err
	}
	view := make(map[string]json.RawMessage)
	for _, key := range []string{"name", "id", "count", "bytes"} {
		view[key] = json.RawMessage("null")
		if value, present := convertedBody[key]; present {
			view[key] = bytes.Clone(value)
		}
	}
	if _, present := body["id"]; !present {
		view["id"] = bytes.Clone(view["name"])
	}
	for _, decl := range recordHeaderFields {
		view[decl.field] = json.RawMessage("null")
		if value, present := convertedHeaders[decl.field]; present {
			view[decl.field] = bytes.Clone(value)
		}
	}
	for _, pair := range [][2]string{{"count", "object_count"}, {"bytes", "bytes_used"}} {
		_, bodyPresent := body[pair[0]]
		_, headerPresent := headers[pair[1]]
		if !bodyPresent && headerPresent {
			view[pair[0]] = bytes.Clone(view[pair[1]])
		}
		if !headerPresent && bodyPresent {
			view[pair[1]] = bytes.Clone(view[pair[0]])
		}
	}
	view["location"] = json.RawMessage("null")
	record.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: view, Header: response.Header.Clone(), StatusCode: response.StatusCode}}
	return record, nil
}

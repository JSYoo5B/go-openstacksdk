// Package sfcrecord owns the Neutron networking-sfc record workflow shared by
// the SDK-owned flow classifier, port pair, port pair group, port chain and
// service graph packages. It reproduces pinned openstacksdk Resource
// create/fetch/commit/delete/list and find behavior over raw JSON so that no
// declared value is coerced beyond the Python Body descriptor types.
package sfcrecord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Descriptor types of Python resource.Body(type=...).
const (
	TypeInt  = "int"
	TypeBool = "bool"
	TypeList = "list"
	TypeDict = "dict"
)

// Spec describes one Python Resource class. Path is the collection path
// below the Neutron v2.0 endpoint. Fields lists every declared Body key and
// Types the descriptor type of the typed ones. Attrs maps every accepted
// Python attribute name and Body key to its Body key. Query maps the
// QueryParameters names (including limit and marker) to wire names. Filters
// maps the Python Body attribute names that Resource.list filters locally
// (Body attributes outside the query mapping) to Body keys.
type Spec struct {
	Kind         string
	Path         string
	ResourceKey  string
	ResourcesKey string
	Fields       []string
	Types        map[string]string
	Attrs        map[string]string
	Query        map[string]string
	Filters      map[string]string
}

// Record keeps the Python Resource view and the actual HTTP evidence.
// View holds every declared Body key (JSON null when unknown) after the
// descriptor type conversion and the project_id to tenant_id alias. Wire is
// the unwrapped response object, nil when the response had no JSON object.
type Record struct {
	View       map[string]json.RawMessage
	Wire       map[string]json.RawMessage
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
}

// Text returns a declared string value, or "" when it is not a JSON string.
func (r *Record) Text(key string) string {
	if r == nil {
		return ""
	}
	var text string
	if err := json.Unmarshal(r.View[key], &text); err != nil {
		return ""
	}
	return text
}

// Field decodes one declared value. JSON null, absence and a JSON type
// mismatch return nil; the raw value stays available in View.
func Field[T any](r *Record, key string) *T {
	if r == nil {
		return nil
	}
	raw, ok := r.View[key]
	if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil
	}
	value := new(T)
	if err := json.Unmarshal(raw, value); err != nil {
		return nil
	}
	return value
}

// Value decodes one declared value like Field but returns the zero value
// instead of nil.
func Value[T any](r *Record, key string) T {
	var zero T
	if value := Field[T](r, key); value != nil {
		return *value
	}
	return zero
}

func Invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

// ValidateID accepts one nonblank UTF-8 segment without control characters.
// The segment is URL-escaped exactly once when it becomes part of a route.
func ValidateID(label, id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." || !utf8.ValidString(id) {
		return Invalid("%s must be a nonblank UTF-8 segment", label)
	}
	for _, c := range id {
		if unicode.IsControl(c) {
			return Invalid("%s has control characters", label)
		}
	}
	return nil
}

func cloneFields(values map[string]json.RawMessage) map[string]json.RawMessage {
	if values == nil {
		return nil
	}
	result := make(map[string]json.RawMessage, len(values))
	for key, raw := range values {
		result[key] = bytes.Clone(raw)
	}
	return result
}

// Clone returns an independent copy of the record.
func (r *Record) Clone() *Record {
	if r == nil {
		return nil
	}
	return &Record{View: cloneFields(r.View), Wire: cloneFields(r.Wire), Envelope: bytes.Clone(r.Envelope), Header: r.Header.Clone(), StatusCode: r.StatusCode}
}

// convert follows fields._convert_type for one non-null declared value.
func convert(kind string, raw json.RawMessage) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return json.RawMessage("null"), nil
	}
	switch kind {
	case TypeInt:
		return jsonfilter.DescriptorIntegerJSON(trimmed)
	case TypeBool:
		return jsonfilter.BooleanJSON(trimmed)
	case TypeList:
		if trimmed[0] != '[' {
			return json.RawMessage("[" + string(trimmed) + "]"), nil
		}
	case TypeDict:
		if trimmed[0] != '{' {
			return json.RawMessage("{}"), nil
		}
	}
	return bytes.Clone(trimmed), nil
}

// project builds the Python attribute view: seeded values, overlaid by the
// actual response object, with project_id falling back to tenant_id when the
// resource never received project_id (Body alias semantics).
func (s *Spec) project(r *Record, seed map[string]json.RawMessage) error {
	present := make(map[string]json.RawMessage, len(s.Fields))
	for key, raw := range seed {
		present[key] = raw
	}
	for key, raw := range r.Wire {
		if s.declared(key) {
			present[key] = raw
		}
	}
	if _, ok := present["project_id"]; !ok && s.declared("project_id") {
		if tenant, ok := present["tenant_id"]; ok {
			present["project_id"] = tenant
		}
	}
	view := make(map[string]json.RawMessage, len(s.Fields))
	for _, key := range s.Fields {
		raw, ok := present[key]
		if !ok {
			view[key] = json.RawMessage("null")
			continue
		}
		converted, err := convert(s.Types[key], raw)
		if err != nil {
			return Invalid("%s field %q: %v", s.Kind, key, err)
		}
		view[key] = converted
	}
	r.View = view
	return nil
}

// declared reports whether key is a declared Body key.
func (s *Spec) declared(key string) bool {
	for _, field := range s.Fields {
		if field == key {
			return true
		}
	}
	return false
}

// fromResponse follows Resource._translate_response: invalid or empty JSON keeps
// the seeded values, an object (or its resource_key member) overlays declared
// keys, and any other JSON value fails with the accepted response as proof.
func (s *Spec) fromResponse(r *rest.Response, seed map[string]json.RawMessage) (*Record, error) {
	value := &Record{Envelope: bytes.Clone(r.Body), Header: r.Header.Clone(), StatusCode: r.StatusCode}
	raw := bytes.TrimSpace(r.Body)
	if len(raw) == 0 || !json.Valid(raw) {
		return value, s.project(value, seed)
	}
	if !utf8.Valid(raw) || raw[0] != '{' {
		return value, Invalid("%s response must be a JSON object", s.Kind)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return value, err
	}
	if nested, ok := fields[s.ResourceKey]; ok {
		trimmed := bytes.TrimSpace(nested)
		if len(trimmed) == 0 || trimmed[0] != '{' {
			return value, Invalid("%s response %q must be a JSON object", s.Kind, s.ResourceKey)
		}
		fields = nil
		if err := json.Unmarshal(trimmed, &fields); err != nil {
			return value, err
		}
	}
	delete(fields, "self")
	value.Wire = cloneFields(fields)
	if value.Wire == nil {
		value.Wire = map[string]json.RawMessage{}
	}
	return value, s.project(value, seed)
}

// fields resolves Python attribute names and Body keys to Body keys. Unknown
// names are dropped, as Resource.new drops them; aliases of one Body key that
// carry different values are rejected because Python keeps an arbitrary one.
func (s *Spec) fields(values map[string]json.RawMessage, skipID bool) (map[string]json.RawMessage, error) {
	result := make(map[string]json.RawMessage)
	for key, raw := range values {
		wire, known := s.Attrs[key]
		if !known || skipID && wire == "id" {
			continue
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, Invalid("attribute %q must be UTF-8 JSON", key)
		}
		if previous, exists := result[wire]; exists && !bytes.Equal(previous, raw) {
			return nil, Invalid("attribute aliases conflict for %q", wire)
		}
		result[wire] = bytes.Clone(raw)
	}
	return result, nil
}

// Headers copies caller headers for one operation.
func Headers(values map[string]string) map[string]string { return maps.Clone(values) }

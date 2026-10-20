// Package cinderowned owns the Cinder v3 record workflow shared by the
// SDK-owned group type, default type and cluster packages and by the owned
// service actions. It reproduces pinned openstacksdk Resource
// create/fetch/commit/delete/list/find and action behavior over raw JSON, so
// that no declared value is coerced, and it selects the Cinder microversion
// the way the Python Resource does.
package cinderowned

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Spec describes one Python Resource class. Fields lists every declared Body
// key. Attrs maps every accepted Python attribute name and Body key to its
// Body key. Defaults holds Body defaults (JSON) used when a key is unknown.
// Query maps accepted server query names other than limit and marker to wire
// names; Filters maps Python attribute names that Resource.list filters
// locally to Body keys. IDKey is the Body key Python uses as the id (the
// alternate id when the class declares one).
type Spec struct {
	Kind         string
	ResourceKey  string
	ResourcesKey string
	Fields       []string
	Attrs        map[string]string
	Defaults     map[string]json.RawMessage
	Query        map[string]string
	Filters      map[string]string
	IDKey        string
}

// Record keeps the Python Resource view and the actual HTTP evidence.
// View holds every declared Body key (its default or JSON null when
// unknown). Wire is the unwrapped response object, nil when the response had
// no JSON object. Discovery holds actual version discovery responses.
type Record struct {
	View       map[string]json.RawMessage
	Wire       map[string]json.RawMessage
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
	Discovery  []Observation
}

// Observation is one actual HTTP response kept as evidence.
type Observation struct {
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
	return Decode[T](r.View[key])
}

// Decode decodes raw JSON into T, or returns nil for null, absence or a
// type mismatch.
func Decode[T any](raw json.RawMessage) *T {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil
	}
	value := new(T)
	if err := json.Unmarshal(raw, value); err != nil {
		return nil
	}
	return value
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

// CloneFields returns an independent copy of raw JSON fields.
func CloneFields(values map[string]json.RawMessage) map[string]json.RawMessage {
	if values == nil {
		return nil
	}
	result := make(map[string]json.RawMessage, len(values))
	for key, raw := range values {
		result[key] = bytes.Clone(raw)
	}
	return result
}

func cloneObservations(values []Observation) []Observation {
	if values == nil {
		return nil
	}
	result := make([]Observation, len(values))
	for i, value := range values {
		result[i] = Observation{Envelope: bytes.Clone(value.Envelope), Header: value.Header.Clone(), StatusCode: value.StatusCode}
	}
	return result
}

// Clone returns an independent copy of the record.
func (r *Record) Clone() *Record {
	if r == nil {
		return nil
	}
	return &Record{View: CloneFields(r.View), Wire: CloneFields(r.Wire), Envelope: bytes.Clone(r.Envelope), Header: r.Header.Clone(), StatusCode: r.StatusCode, Discovery: cloneObservations(r.Discovery)}
}

// Project builds the declared view: default or null, then seed, then wire.
func (s *Spec) Project(r *Record, seed map[string]json.RawMessage) {
	view := make(map[string]json.RawMessage, len(s.Fields))
	for _, key := range s.Fields {
		view[key] = json.RawMessage("null")
		if raw, ok := s.Defaults[key]; ok {
			view[key] = bytes.Clone(raw)
		}
		if raw, ok := seed[key]; ok {
			view[key] = bytes.Clone(raw)
		}
		if raw, ok := r.Wire[key]; ok {
			view[key] = bytes.Clone(raw)
		}
	}
	r.View = view
}

// FromResponse follows Resource._translate_response: invalid or empty JSON
// keeps the seeded values, an object (or its resource_key member) overlays
// declared keys, and any other JSON value fails with the accepted response.
func (s *Spec) FromResponse(r *rest.Response, seed map[string]json.RawMessage) (*Record, error) {
	value := &Record{Envelope: bytes.Clone(r.Body), Header: r.Header.Clone(), StatusCode: r.StatusCode}
	raw := bytes.TrimSpace(r.Body)
	if len(raw) == 0 || !json.Valid(raw) {
		s.Project(value, seed)
		return value, nil
	}
	if !utf8.Valid(raw) || raw[0] != '{' {
		return value, Invalid("%s response must be a JSON object", s.Kind)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return value, err
	}
	if nested, ok := fields[s.ResourceKey]; ok && s.ResourceKey != "" {
		trimmed := bytes.TrimSpace(nested)
		if len(trimmed) == 0 || trimmed[0] != '{' {
			return value, Invalid("%s response %q must be a JSON object", s.Kind, s.ResourceKey)
		}
		fields = nil
		if err := json.Unmarshal(trimmed, &fields); err != nil {
			return value, err
		}
	}
	value.Wire = CloneFields(fields)
	if value.Wire == nil {
		value.Wire = map[string]json.RawMessage{}
	}
	s.Project(value, seed)
	return value, nil
}

// BodyFields resolves Python attribute names and Body keys to Body keys. Unknown
// names are dropped, as Resource.new drops them; aliases of one Body key that
// carry different values are rejected because Python keeps an arbitrary one.
func (s *Spec) BodyFields(values map[string]json.RawMessage, skipID bool) (map[string]json.RawMessage, error) {
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

// PythonID follows Resource.id: the id Body value when it is present and not
// null, otherwise the alternate id (IDKey) value.
func (s *Spec) PythonID(body map[string]json.RawMessage) json.RawMessage {
	if raw, ok := body["id"]; ok && !bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return raw
	}
	return body[s.idKey()]
}

// StringValue returns a JSON string value, or "" for any other value.
func StringValue(raw json.RawMessage) string {
	if text := Decode[string](raw); text != nil {
		return *text
	}
	return ""
}

func (s *Spec) idKey() string {
	if s.IDKey == "" {
		return "id"
	}
	return s.IDKey
}

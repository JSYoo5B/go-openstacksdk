package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"unicode/utf8"

	"gophercloudsdk/resource"
)

// Info retains Swift's complete capability object and five common sections.
// Missing or null sections are nil; an empty object is a nonnil empty map.
// Dates and links remain passive raw fields in Metadata.Body.
type Info struct {
	resource.Metadata
	Swift      map[string]json.RawMessage `json:"swift"`
	SLO        map[string]json.RawMessage `json:"slo"`
	BulkDelete map[string]json.RawMessage `json:"bulk_delete"`
	StaticWeb  map[string]json.RawMessage `json:"staticweb"`
	TempURL    map[string]json.RawMessage `json:"tempurl"`
}

// UnmarshalJSON commits only a complete UTF-8 object with nullable object
// sections. Section values own their bytes independently of Metadata.Body.
func (i *Info) UnmarshalJSON(data []byte) error {
	if i == nil {
		return fmt.Errorf("Info decoder requires a destination")
	}
	if !utf8.Valid(data) {
		return fmt.Errorf("Swift info must be UTF-8 JSON")
	}
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return fmt.Errorf("Swift info must be a JSON object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	value := Info{Metadata: resource.Metadata{Body: fields}}
	for _, section := range []struct {
		name   string
		target *map[string]json.RawMessage
	}{
		{"swift", &value.Swift}, {"slo", &value.SLO},
		{"bulk_delete", &value.BulkDelete}, {"staticweb", &value.StaticWeb},
		{"tempurl", &value.TempURL},
	} {
		raw, present := fields[section.name]
		if !present || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			continue
		}
		if err := json.Unmarshal(raw, section.target); err != nil {
			return fmt.Errorf("Swift info section %q must be an object: %w", section.name, err)
		}
	}
	*i = value
	return nil
}

// ObjectSegmentSizeResult retains the actual capabilities response, including
// a clean 404 or 412 used for fallback. On an accepted-response error Size is
// zero and Info is present only if the complete capability object decoded.
type ObjectSegmentSizeResult struct {
	RequestedSize  int64
	Size           int64
	MaxFileSize    int64
	MinSegmentSize int64
	UsedFallback   bool
	Info           *Info
	Header         http.Header
	StatusCode     int
	Body           []byte
}

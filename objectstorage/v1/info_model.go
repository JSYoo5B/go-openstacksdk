package v1

import (
	"encoding/json"
	"fmt"
	"net/http"

	"gophercloudsdk/internal/swiftinfo"
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
	sections, err := swiftinfo.Decode(data)
	if err != nil {
		return err
	}
	value := Info{
		Metadata: resource.Metadata{Body: sections.Body},
		Swift:    sections.Swift, SLO: sections.SLO, BulkDelete: sections.BulkDelete,
		StaticWeb: sections.StaticWeb, TempURL: sections.TempURL,
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

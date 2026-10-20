// Package resourcefilters owns the Cinder v3 allowed-filter catalogue.
package resourcefilters

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/cindervolume"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ResourceFilterRecord retains the declared descriptor view independently of
// literal server fields and whole-page HTTP evidence.
type ResourceFilterRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	Microversion   string
	Discovery      []DiscoveryObservation
}
type DiscoveryObservation struct {
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func (v *ResourceFilterRecord) UnmarshalJSON(raw []byte) error {
	var actual resource.RawResource
	if err := json.Unmarshal(raw, &actual); err != nil {
		return err
	}
	for _, key := range []string{"cls", "connection", "microversion", "_synchronized"} {
		if _, exists := actual.Body[key]; exists {
			return fmt.Errorf("resource-filter list constructor collides with %q", key)
		}
	}
	v.Wire = actual.Clone()
	return nil
}
func metadata(v *ResourceFilterRecord) *resource.Metadata {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	return &v.Wire.Metadata
}
func project(v *ResourceFilterRecord) error {
	body := make(map[string]json.RawMessage, 5)
	for _, key := range []string{"id", "name", "filters", "resource"} {
		raw := bytes.Clone(v.Wire.Body[key])
		if len(raw) == 0 {
			raw = json.RawMessage("null")
		}
		var err error
		if key == "filters" {
			raw, err = cindervolume.Convert(raw, cindervolume.List)
		}
		if key == "resource" && !bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			var text string
			text, err = cloudfilter.PythonString(raw)
			if err == nil {
				raw, err = json.Marshal(text)
			}
		}
		if err != nil {
			return fmt.Errorf("resource-filter field %s: %w", key, err)
		}
		body[key] = raw
	}
	// This standalone leaf has no Connection-derived location. Server location
	// remains literal Wire evidence and cannot select a service or token scope.
	body["location"] = json.RawMessage("null")
	v.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: body, Header: v.Header.Clone(), StatusCode: v.StatusCode}}
	return nil
}
func marker(v *ResourceFilterRecord) (string, error) {
	if v == nil || v.Wire == nil {
		return "", invalid("resource-filter marker needs actual wire ID")
	}
	values, err := cloudfilter.RequestQueryValues(v.Wire.Body["id"])
	if err != nil || len(values) != 1 || values[0] == "" {
		return "", invalid("resource-filter marker needs one nonempty wire ID")
	}
	return values[0], nil
}
func observation(r *rest.Response) DiscoveryObservation {
	return DiscoveryObservation{bytes.Clone(r.Body), r.Header.Clone(), r.StatusCode}
}
func cloneDiscovery(values []DiscoveryObservation) []DiscoveryObservation {
	out := make([]DiscoveryObservation, len(values))
	for i, v := range values {
		out[i] = DiscoveryObservation{bytes.Clone(v.Envelope), v.Header.Clone(), v.StatusCode}
	}
	return out
}

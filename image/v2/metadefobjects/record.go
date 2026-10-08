package metadefobjects

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Record separates the declared metadata object view from its original row.
// Namespace is fixed request provenance. Every channel independently owns its
// bytes and response headers; links and unknown fields remain passive in Wire.
type Record struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	Namespace      *string
}

type recordRow struct{ Record }

func (value *recordRow) UnmarshalJSON(data []byte) error {
	if value == nil {
		return invalid("record receiver is required")
	}
	if value.Wire == nil {
		value.Wire = &resource.RawResource{}
	}
	return json.Unmarshal(data, value.Wire)
}
func recordMetadata(value *recordRow) *resource.Metadata {
	if value.Wire == nil {
		value.Wire = &resource.RawResource{}
	}
	return &value.Wire.Metadata
}
func prepareRecord(value *recordRow, namespace *string, location json.RawMessage, envelope []byte) error {
	if value == nil || value.Wire == nil || namespace == nil {
		return invalid("record wire fields are required")
	}
	// Source supplies these constructor arguments explicitly for every row.
	// Their presence in **raw_resource raises duplicate binding errors there.
	for _, key := range []string{"connection", "_synchronized", "microversion"} {
		if _, present := value.Wire.Body[key]; present {
			return invalid("object row %q collides with a source constructor argument", key)
		}
	}
	view := value.Wire.Clone()
	view.Body = make(map[string]json.RawMessage)
	keys := []string{"id", "name", "created_at", "updated_at", "description", "properties", "required", "namespace_name", "location"}
	for _, key := range keys {
		raw, err := resource.BodyRecordField(value.Wire.Body, key, resource.BodyFieldJSON)
		if err != nil {
			return err
		}
		view.Body[key] = raw
	}
	if _, present := value.Wire.Body["id"]; !present {
		view.Body["id"] = bytes.Clone(view.Body["name"])
	}
	// The fixed URI component is consumed by the pinned collector, so it
	// recomputes location even for an otherwise empty metadata object row.
	view.Body["location"] = bytes.Clone(location)
	view.Body["namespace_name"], _ = json.Marshal(*namespace)
	value.Namespace = copyPointer(namespace)
	value.Resource = view
	value.Envelope = bytes.Clone(envelope)
	value.Header, value.StatusCode = value.Wire.Header.Clone(), value.Wire.StatusCode
	return nil
}
func recordMarker(value *recordRow) (string, error) {
	if value == nil || value.Wire == nil {
		return "", invalid("marker wire fields are required")
	}
	raw, exists := value.Wire.Body["id"]
	if !exists {
		raw = value.Wire.Body["name"]
	}
	var marker string
	if err := json.Unmarshal(raw, &marker); err != nil {
		return "", invalid("wire marker must be a string identity: %v", err)
	}
	if strings.TrimSpace(marker) == "" {
		return "", invalid("wire marker must be nonempty")
	}
	if err := queryText(marker); err != nil {
		return "", err
	}
	return marker, nil
}

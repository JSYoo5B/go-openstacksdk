package metadefresourcetypes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Record separates a declared Resource view from the actual response or row.
// CreateRecord leaves Wire nil when an accepted response cannot be parsed as JSON.
// Namespace is fixed request provenance, nil for a global resource type.
// Every channel owns its bytes and response headers.
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
	if value == nil || value.Wire == nil {
		return invalid("record wire fields are required")
	}
	view := value.Wire.Clone()
	view.Body = make(map[string]json.RawMessage)
	keys := []string{"id", "name", "created_at", "updated_at", "location"}
	if namespace != nil {
		keys = append(keys, "prefix", "properties_target", "namespace_name")
	}
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
	// The pinned collector recomputes attributes whenever a declared Body or
	// URI component was consumed. These classes compute no response fields,
	// so a row location survives only in a global row with no declared Body.
	consumed := namespace != nil
	for _, key := range []string{"id", "name", "created_at", "updated_at"} {
		if _, present := value.Wire.Body[key]; present {
			consumed = true
		}
	}
	if _, present := value.Wire.Body["location"]; !present || consumed {
		view.Body["location"] = bytes.Clone(location)
	}
	if namespace != nil {
		view.Body["namespace_name"], _ = json.Marshal(*namespace)
		value.Namespace = copyPointer(namespace)
	}
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

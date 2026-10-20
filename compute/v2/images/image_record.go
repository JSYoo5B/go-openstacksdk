// Package images provides the version-bounded legacy Nova image proxy.
// New applications should use the Image (Glance) service instead.
package images

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ImageRecord separates the declared Python view from actual HTTP evidence.
// Dates and untyped fields retain their JSON values rather than native coercion.
type ImageRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	row            json.RawMessage
}

func (v *ImageRecord) UnmarshalJSON(data []byte) error {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	if err := json.Unmarshal(data, v.Wire); err != nil {
		return err
	}
	v.row = bytes.Clone(data)
	return nil
}
func imageMetadata(v *ImageRecord) *resource.Metadata {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	return &v.Wire.Metadata
}

var imageFields = []string{"id", "name", "links", "created_at", "min_disk", "min_ram", "progress", "status", "updated_at", "size", "metadata"}

func imageCanonical(key string) (string, bool) {
	switch key {
	case "created":
		return "created_at", true
	case "minDisk":
		return "min_disk", true
	case "minRam":
		return "min_ram", true
	case "updated":
		return "updated_at", true
	case "OS-EXT-IMG-SIZE:size":
		return "size", true
	}
	for _, field := range imageFields {
		if key == field {
			return key, true
		}
	}
	return "", false
}
func imageInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func cloneImageFields(fields map[string]json.RawMessage) map[string]json.RawMessage {
	owned := make(map[string]json.RawMessage, len(fields))
	for key, raw := range fields {
		owned[key] = bytes.Clone(raw)
	}
	return owned
}

// Ordered object members preserve Python's last alias in response dictionary order.
func normalizedImageFields(raw json.RawMessage) (map[string]json.RawMessage, error) {
	members, err := cloudfilter.ObjectMembers(raw)
	if err != nil {
		return nil, err
	}
	fields := make(map[string]json.RawMessage)
	for _, member := range members {
		if canonical, ok := imageCanonical(member.Key); ok {
			fields[canonical] = bytes.Clone(member.Value)
		}
	}
	return fields, nil
}
func projectImage(v *ImageRecord, fields map[string]json.RawMessage) error {
	body := make(map[string]json.RawMessage, len(imageFields)+1)
	for _, key := range imageFields {
		raw, err := resource.BodyRecordField(fields, key, resource.BodyFieldJSON)
		if err != nil {
			return err
		}
		switch key {
		case "min_disk", "min_ram", "progress", "size":
			raw, err = jsonfilter.DescriptorIntegerJSON(raw)
		case "metadata":
			trimmed := bytes.TrimSpace(raw)
			if len(trimmed) != 0 && trimmed[0] != '{' && !bytes.Equal(trimmed, []byte("null")) {
				raw = json.RawMessage(`{}`)
			}
		}
		if err != nil {
			return imageInvalid("descriptor %q: %v", key, err)
		}
		body[key] = bytes.Clone(raw)
	}
	body["location"] = json.RawMessage("null")
	v.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: body, Header: v.Header.Clone(), StatusCode: v.StatusCode}}
	return nil
}
func imageReceipt(r *rest.Response) *ImageRecord {
	if r == nil {
		return nil
	}
	return &ImageRecord{Envelope: bytes.Clone(r.Body), Header: r.Header.Clone(), StatusCode: r.StatusCode}
}
func imageFromResponse(r *rest.Response, id string) (*ImageRecord, error) {
	v := imageReceipt(r)
	if v == nil {
		return nil, nil
	}
	rawID, _ := json.Marshal(id)
	fields := map[string]json.RawMessage{"id": rawID}
	if !json.Valid(r.Body) {
		return v, projectImage(v, fields)
	}
	if !utf8.Valid(r.Body) {
		return v, imageInvalid("image response must be UTF-8 JSON")
	}
	var root resource.RawResource
	if err := json.Unmarshal(r.Body, &root); err != nil {
		return v, err
	}
	raw := r.Body
	if nested, ok := root.Body["image"]; ok {
		raw = nested
	}
	var wire resource.RawResource
	if err := json.Unmarshal(raw, &wire); err != nil {
		return v, err
	}
	wire.Header, wire.StatusCode = r.Header.Clone(), r.StatusCode
	v.Wire = wire.Clone()
	responseFields, err := normalizedImageFields(raw)
	if err != nil {
		return v, err
	}
	for key, value := range responseFields {
		fields[key] = value
	}
	return v, projectImage(v, fields)
}
func imageText(v *ImageRecord, key string) string {
	if v == nil || v.Resource == nil {
		return ""
	}
	var value string
	_ = json.Unmarshal(v.Resource.Body[key], &value)
	return value
}
func imageMarker(v *ImageRecord) (string, error) {
	if v == nil || v.Wire == nil {
		return "", imageInvalid("image marker missing")
	}
	var value string
	if err := json.Unmarshal(v.Wire.Body["id"], &value); err != nil || value == "" {
		return "", imageInvalid("image marker must be a nonempty string")
	}
	return value, nil
}

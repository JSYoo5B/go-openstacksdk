package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ImageMemberRecord separates the declared member view from its actual response.
// ImageID is fixed request provenance. Resource, Wire, Envelope and Header own
// their bytes independently. Wire is nil when a fetch tolerated invalid JSON.
type ImageMemberRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	ImageID        *string
}

type imageMemberRecordField struct{ canonical, wire string }

// Seven Body descriptors, including inherited id/name. image_id is URI and
// location is Computed; neither may be replaced by response Body attributes.
var imageMemberRecordFields = [...]imageMemberRecordField{
	{"id", "id"}, {"name", "name"}, {"member_id", "member"},
	{"created_at", "created_at"}, {"status", "status"},
	{"schema", "schema"}, {"updated_at", "updated_at"},
}

func normalizeImageMemberRecord(fields map[string]json.RawMessage, envelope json.RawMessage) (map[string]json.RawMessage, error) {
	normalized := make(map[string]json.RawMessage)
	put := func(field imageMemberRecordField, raw json.RawMessage) error {
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return uploadInvalid("member attribute %q must be complete UTF-8 JSON", field.canonical)
		}
		normalized[field.canonical] = bytes.Clone(raw)
		return nil
	}
	if len(envelope) != 0 {
		if !utf8.Valid(envelope) {
			return nil, uploadInvalid("member envelope must be UTF-8")
		}
		members, err := cloudfilter.ObjectMembers(envelope)
		if err != nil {
			return nil, err
		}
		// Parsed dictionary insertion order, including first-position/last-value
		// duplicate keys, determines which canonical/remote alias wins.
		for _, member := range members {
			for _, field := range imageMemberRecordFields {
				if member.Key == field.canonical || member.Key == field.wire {
					if err := put(field, member.Value); err != nil {
						return nil, err
					}
					break
				}
			}
		}
		return normalized, nil
	}
	for _, field := range imageMemberRecordFields {
		raw, present := fields[field.canonical]
		if !present && field.wire != field.canonical {
			raw, present = fields[field.wire]
		}
		if present {
			if err := put(field, raw); err != nil {
				return nil, err
			}
		}
	}
	return normalized, nil
}

func projectImageMemberRecord(fields map[string]json.RawMessage, imageID string, location json.RawMessage, metadata resource.Metadata) (*resource.RawResource, error) {
	normalized, err := normalizeImageMemberRecord(fields, nil)
	if err != nil {
		return nil, err
	}
	if !utf8.ValidString(imageID) {
		return nil, uploadInvalid("member image ID must be UTF-8")
	}
	if location == nil {
		location = json.RawMessage("null")
	}
	if !utf8.Valid(location) || !json.Valid(location) {
		return nil, uploadInvalid("member location must be complete UTF-8 JSON")
	}
	projected := make(map[string]json.RawMessage, len(imageMemberRecordFields)+2)
	for _, field := range imageMemberRecordFields {
		raw, present := normalized[field.canonical]
		if !present {
			raw = json.RawMessage("null")
		}
		value, err := resource.BodyRecordField(map[string]json.RawMessage{field.canonical: raw}, field.canonical, resource.BodyFieldJSON)
		if err != nil {
			return nil, err
		}
		projected[field.canonical] = value
	}
	// Presence is authoritative even when the explicit id is JSON null.
	if _, present := normalized["id"]; !present {
		projected["id"] = bytes.Clone(projected["member_id"])
	}
	projected["image_id"], err = json.Marshal(imageID)
	if err != nil {
		return nil, err
	}
	projected["location"] = bytes.Clone(location)
	return &resource.RawResource{Metadata: resource.Metadata{Body: projected, Header: metadata.Header.Clone(), StatusCode: metadata.StatusCode}}, nil
}

func imageMemberRecordFromResponse(ctx context.Context, check func(context.Context) error, seed map[string]json.RawMessage, imageID string, location json.RawMessage, response *rest.Response) (*ImageMemberRecord, error) {
	if !utf8.Valid(response.Body) {
		return nil, response.Fail(uploadInvalid("member response must be UTF-8"))
	}
	parent := imageID
	record := &ImageMemberRecord{ImageID: &parent, Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	fields, err := normalizeImageMemberRecord(seed, nil)
	if err != nil {
		return nil, response.Fail(err)
	}
	if json.Valid(response.Body) {
		wire := &resource.RawResource{Metadata: resource.Metadata{Header: response.Header.Clone(), StatusCode: response.StatusCode}}
		if err := json.Unmarshal(response.Body, wire); err != nil {
			return nil, response.Fail(err)
		}
		responseFields, err := normalizeImageMemberRecord(wire.Body, response.Body)
		if err != nil {
			return nil, response.Fail(err)
		}
		for key, raw := range responseFields {
			fields[key] = bytes.Clone(raw)
		}
		record.Wire = wire
	}
	record.Resource, err = projectImageMemberRecord(fields, imageID, location, resource.Metadata{Header: response.Header.Clone(), StatusCode: response.StatusCode})
	if err = errors.Join(err, check(ctx)); err != nil {
		return nil, response.Fail(err)
	}
	return record, nil
}

type imageMemberRecordRow struct {
	ImageMemberRecord
	rowJSON json.RawMessage
}

func (value *imageMemberRecordRow) UnmarshalJSON(data []byte) error {
	if value == nil {
		return uploadInvalid("member record receiver is required")
	}
	if value.Wire == nil {
		value.Wire = &resource.RawResource{}
	}
	if err := json.Unmarshal(data, value.Wire); err != nil {
		return err
	}
	value.rowJSON = bytes.Clone(data)
	return nil
}
func imageMemberRecordMetadata(value *imageMemberRecordRow) *resource.Metadata {
	if value.Wire == nil {
		value.Wire = &resource.RawResource{}
	}
	return &value.Wire.Metadata
}
func prepareImageMemberRecordRow(value *imageMemberRecordRow, imageID string, location json.RawMessage, envelope []byte) error {
	if value == nil || value.Wire == nil {
		return uploadInvalid("member record wire fields are required")
	}
	// Source supplies these bindings to every cls.existing(**row) constructor.
	for _, key := range []string{"connection", "_synchronized", "microversion"} {
		if _, present := value.Wire.Body[key]; present {
			return uploadInvalid("member row %q collides with a source constructor argument", key)
		}
	}
	fields, err := normalizeImageMemberRecord(value.Wire.Body, value.rowJSON)
	if err != nil {
		return err
	}
	value.Resource, err = projectImageMemberRecord(fields, imageID, location, value.Wire.Metadata)
	if err != nil {
		return err
	}
	parent := imageID
	value.ImageID = &parent
	value.Envelope = bytes.Clone(envelope)
	value.Header, value.StatusCode = value.Wire.Header.Clone(), value.Wire.StatusCode
	return nil
}
func imageMemberRecordMarker(value *imageMemberRecordRow) (string, error) {
	if value == nil || value.Wire == nil {
		return "", uploadInvalid("member marker wire fields are required")
	}
	// Use the original row rather than mutable yielded Resource attributes.
	fields, err := normalizeImageMemberRecord(value.Wire.Body, value.rowJSON)
	if err != nil {
		return "", err
	}
	raw, present := fields["id"]
	if !present {
		raw = fields["member_id"]
	}
	var marker string
	if err := json.Unmarshal(raw, &marker); err != nil {
		return "", uploadInvalid("member wire marker must be a string identity: %v", err)
	}
	if strings.TrimSpace(marker) == "" {
		return "", uploadInvalid("member wire marker must be nonempty")
	}
	if err := taskQueryText(marker); err != nil {
		return "", err
	}
	return marker, nil
}

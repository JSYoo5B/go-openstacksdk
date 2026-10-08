package flavors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// FlavorRecord keeps the declared flavor view separate from the actual
// response. Resource applies response descriptors and defaults; Wire and the
// receipt preserve actual values. Enrichment records a separate extra-specs
// response when a list or find operation requests conditional enrichment.
type FlavorRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	Enrichment     *FlavorExtraSpecsRecord
}

// List rows are flat. Unlike keypairs, a nested flavor member is an unknown
// response field here and must not replace the outer row.
type flavorListRecord struct{ FlavorRecord }

func (value *flavorListRecord) UnmarshalJSON(data []byte) error {
	if value == nil {
		return fmt.Errorf("%w: flavor record receiver is required", resource.ErrInvalidOption)
	}
	if value.Wire == nil {
		value.Wire = &resource.RawResource{}
	}
	// Keep the Wire address stable for the REST decoder's metadata pointer.
	if err := json.Unmarshal(data, value.Wire); err != nil {
		return err
	}
	value.Envelope = bytes.Clone(data)
	value.Resource = nil
	value.Enrichment = nil
	return nil
}

func flavorRecordMetadata(value *FlavorRecord) *resource.Metadata {
	if value == nil {
		return nil
	}
	if value.Wire == nil {
		value.Wire = &resource.RawResource{}
	}
	return &value.Wire.Metadata
}

func flavorRecordCanonicalField(key string) (string, bool) {
	switch key {
	case "id", "name", "original_name", "description", "disk", "ram", "vcpus", "swap", "ephemeral", "is_public", "is_disabled", "rxtx_factor", "extra_specs":
		return key, true
	case "os-flavor-access:is_public":
		return "is_public", true
	case "OS-FLV-EXT-DATA:ephemeral":
		return "ephemeral", true
	case "OS-FLV-DISABLED:disabled":
		return "is_disabled", true
	default:
		return "", false
	}
}

// The selected raw object supplies response alias order. A canonical seed map
// without an envelope uses canonical spelling when both spellings are present;
// the response then overlays it in parsed JSON dictionary order.
func normalizedFlavorRecordFields(fields map[string]json.RawMessage, envelope json.RawMessage) (map[string]json.RawMessage, error) {
	normalized := make(map[string]json.RawMessage, 13)
	for key, raw := range fields {
		canonical, recognized := flavorRecordCanonicalField(key)
		if recognized && canonical == key {
			normalized[canonical] = bytes.Clone(raw)
		}
	}
	for key, raw := range fields {
		canonical, recognized := flavorRecordCanonicalField(key)
		if recognized && canonical != key {
			if _, present := normalized[canonical]; !present {
				normalized[canonical] = bytes.Clone(raw)
			}
		}
	}
	if len(bytes.TrimSpace(envelope)) == 0 {
		return normalized, nil
	}
	members, err := cloudfilter.ObjectMembers(envelope)
	if err != nil {
		return nil, fmt.Errorf("%w: flavor response object: %w", resource.ErrInvalidOption, err)
	}
	for _, member := range members {
		if canonical, recognized := flavorRecordCanonicalField(member.Key); recognized {
			normalized[canonical] = bytes.Clone(member.Value)
		}
	}
	return normalized, nil
}

func projectFlavorRecord(fields map[string]json.RawMessage, metadata resource.Metadata) (*resource.RawResource, error) {
	view := (&resource.RawResource{Metadata: metadata}).Clone()
	view.Body = make(map[string]json.RawMessage, 14)
	read := func(key string) (json.RawMessage, error) {
		if raw, present := fields[key]; present && !utf8.Valid(raw) {
			return nil, fmt.Errorf("%w: flavor field %q must contain UTF-8 JSON", resource.ErrInvalidOption, key)
		}
		return resource.BodyRecordField(fields, key, resource.BodyFieldJSON)
	}
	for _, key := range []string{"name", "original_name", "description"} {
		raw, err := read(key)
		if err != nil {
			return nil, err
		}
		if key == "name" {
			if _, present := fields[key]; !present {
				raw, err = read("original_name")
				if err != nil {
					return nil, err
				}
			}
		}
		view.Body[key] = raw
	}
	view.Body["id"] = json.RawMessage("null")
	for _, key := range []string{"id", "name", "original_name"} {
		raw, err := read(key)
		if err != nil {
			return nil, err
		}
		truthy, err := cloudfilter.PythonTruthy(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: flavor identity field %q: %w", resource.ErrInvalidOption, key, err)
		}
		if truthy {
			view.Body["id"] = raw
			break
		}
	}
	for _, key := range []string{"disk", "ram", "vcpus", "swap", "ephemeral"} {
		raw, err := read(key)
		if err != nil {
			return nil, err
		}
		if _, present := fields[key]; !present {
			raw = json.RawMessage("0")
		}
		view.Body[key], err = jsonfilter.DescriptorIntegerJSON(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: flavor integer field %q: %w", resource.ErrInvalidOption, key, err)
		}
	}
	for _, key := range []string{"is_public", "is_disabled"} {
		if _, err := read(key); err != nil {
			return nil, err
		}
		raw, err := resource.BodyRecordField(fields, key, resource.BodyFieldBoolean)
		if err != nil {
			return nil, err
		}
		if key == "is_public" {
			if _, present := fields[key]; !present {
				raw = json.RawMessage("true")
			}
		}
		view.Body[key] = raw
	}
	raw, err := read("rxtx_factor")
	if err != nil {
		return nil, err
	}
	view.Body["rxtx_factor"], err = jsonfilter.DescriptorFloatJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: flavor float field %q: %w", resource.ErrInvalidOption, "rxtx_factor", err)
	}
	raw, err = read("extra_specs")
	if err != nil {
		return nil, err
	}
	if _, present := fields["extra_specs"]; !present {
		raw = nil
	}
	view.Body["extra_specs"] = flavorExtraSpecsView(raw)
	view.Body["location"] = json.RawMessage("null")
	return view, nil
}

func prepareFlavorListRecord(value *FlavorRecord) error {
	if value == nil || value.Wire == nil || value.Wire.Body == nil {
		return fmt.Errorf("%w: flavor response fields are required", resource.ErrInvalidOption)
	}
	fields, err := normalizedFlavorRecordFields(value.Wire.Body, value.Envelope)
	if err != nil {
		return err
	}
	value.Header, value.StatusCode = value.Wire.Header.Clone(), value.Wire.StatusCode
	value.Resource, err = projectFlavorRecord(fields, value.Wire.Metadata)
	return err
}

func flavorRecordMarker(value *FlavorRecord) (string, error) {
	if value == nil {
		return "", fmt.Errorf("%w: flavor marker record is required", resource.ErrInvalidOption)
	}
	var raw json.RawMessage
	if value.Wire != nil {
		// The shared pager decodes its own last consumed row for marker
		// fallback. That snapshot has no projected view and cannot depend on
		// the view a consumer has already changed after yield.
		fields, err := normalizedFlavorRecordFields(value.Wire.Body, value.Envelope)
		if err != nil {
			return "", err
		}
		for _, key := range []string{"id", "name", "original_name"} {
			candidate, err := resource.BodyRecordField(fields, key, resource.BodyFieldJSON)
			if err != nil {
				return "", err
			}
			truthy, err := cloudfilter.PythonTruthy(candidate)
			if err != nil {
				return "", err
			}
			if truthy {
				raw = candidate
				break
			}
		}
	} else if value.Resource != nil {
		var err error
		raw, err = resource.BodyRecordField(value.Resource.Body, "id", resource.BodyFieldJSON)
		if err != nil {
			return "", err
		}
	} else {
		return "", fmt.Errorf("%w: flavor marker fields are required", resource.ErrInvalidOption)
	}
	var marker string
	if err := json.Unmarshal(raw, &marker); err != nil {
		return "", fmt.Errorf("%w: flavor marker must be a string identity: %w", resource.ErrInvalidOption, err)
	}
	if strings.TrimSpace(marker) == "" {
		return "", fmt.Errorf("%w: flavor marker must be nonempty", resource.ErrInvalidOption)
	}
	return marker, nil
}

func flavorRecordFilterValue(value *FlavorRecord, field string) (json.RawMessage, error) {
	if value == nil || value.Resource == nil {
		return nil, fmt.Errorf("%w: flavor Resource is required", resource.ErrInvalidOption)
	}
	return resource.BodyRecordField(value.Resource.Body, field, resource.BodyFieldJSON)
}

func flavorRecordIdentity(value *FlavorRecord) string {
	return flavorRecordString(value, "id")
}

func flavorRecordName(value *FlavorRecord) string {
	return flavorRecordString(value, "name")
}

// Identity comparison is passive: a numeric or container-valued descriptor
// does not become a string name or a validated request path.
func flavorRecordString(value *FlavorRecord, field string) string {
	if value == nil || value.Resource == nil {
		return ""
	}
	var text string
	if err := json.Unmarshal(value.Resource.Body[field], &text); err != nil {
		return ""
	}
	return text
}

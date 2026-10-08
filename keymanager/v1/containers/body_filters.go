package containers

import (
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// containerBodyFilterValue projects a pinned Python attribute from the original
// row after native whole-page decoding. Response references are passive values.
func containerBodyFilterValue(record *resource.BodyRecord[Container], key string) (json.RawMessage, error) {
	if record == nil {
		return nil, fmt.Errorf("%w: container Body record is required", resource.ErrInvalidOption)
	}
	if record.Fields == nil {
		return nil, fmt.Errorf("%w: null container list row is not an object", resource.ErrInvalidOption)
	}
	field := key
	switch key {
	case "id":
		if _, exists := record.Fields["id"]; !exists {
			field = "container_ref"
		}
	case "container_id":
		raw, err := resource.BodyRecordField(record.Fields, "container_ref", resource.BodyFieldJSON)
		if err != nil {
			return nil, err
		}
		value, err := jsonfilter.ReferenceLastComponent(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: container_ref formatter: %w", resource.ErrInvalidOption, err)
		}
		return value, nil
	case "created_at":
		field = "created"
	case "updated_at":
		field = "updated"
	case "name", "container_ref", "secret_refs", "consumers", "status", "type":
	default:
		return nil, fmt.Errorf("%w: unsupported container Body filter %q", resource.ErrInvalidOption, key)
	}
	return resource.BodyRecordField(record.Fields, field, resource.BodyFieldJSON)
}

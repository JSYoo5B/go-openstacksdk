package orders

import (
	"encoding/json"
	"fmt"

	"gophercloudsdk/internal/jsonfilter"
	"gophercloudsdk/resource"
)

// orderBodyFilterValue projects original response attributes after native
// whole-page decoding. References and raw IDs are passive matching values.
func orderBodyFilterValue(record *resource.BodyRecord[Order], key string) (json.RawMessage, error) {
	if record == nil {
		return nil, fmt.Errorf("%w: order Body record is required", resource.ErrInvalidOption)
	}
	if record.Fields == nil {
		return nil, fmt.Errorf("%w: null order list row is not an object", resource.ErrInvalidOption)
	}
	field := key
	switch key {
	case "id":
		if _, exists := record.Fields["id"]; !exists {
			field = "order_ref"
		}
	case "order_id", "secret_id":
		field = "order_ref"
		if key == "secret_id" {
			field = "secret_ref"
		}
		raw, err := resource.BodyRecordField(record.Fields, field, resource.BodyFieldJSON)
		if err != nil {
			return nil, err
		}
		value, err := jsonfilter.ReferenceLastComponent(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: %s formatter: %w", resource.ErrInvalidOption, field, err)
		}
		return value, nil
	case "created_at":
		field = "created"
	case "updated_at":
		field = "updated"
	case "name", "creator_id", "meta", "order_ref", "secret_ref", "status", "sub_status", "sub_status_message", "type":
	default:
		return nil, fmt.Errorf("%w: unsupported order Body filter %q", resource.ErrInvalidOption, key)
	}
	return resource.BodyRecordField(record.Fields, field, resource.BodyFieldJSON)
}

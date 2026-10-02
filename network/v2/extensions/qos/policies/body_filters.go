package policies

import (
	"encoding/json"
	"fmt"

	"gophercloudsdk/resource"
)

// qosPolicyBodyFilterValue compares original fields only after native
// whole-page decoding. Local tenant_id does not fall back to project_id.
func qosPolicyBodyFilterValue(record *resource.BodyRecord[Policy], key string) (json.RawMessage, error) {
	if record == nil {
		return nil, fmt.Errorf("%w: QoS policy Body record is required", resource.ErrInvalidOption)
	}
	if record.Fields == nil {
		return nil, fmt.Errorf("%w: null QoS policy list row is not an object", resource.ErrInvalidOption)
	}
	switch key {
	case "rules", "tenant_id":
		return resource.BodyRecordField(record.Fields, key, resource.BodyFieldJSON)
	default:
		return nil, fmt.Errorf("%w: unsupported QoS policy Body filter %q", resource.ErrInvalidOption, key)
	}
}

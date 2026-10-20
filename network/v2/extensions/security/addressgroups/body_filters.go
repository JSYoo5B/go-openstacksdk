package addressgroups

import (
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// addressGroupBodyFilterValue compares original list fields after the native
// whole-page decoder. The deprecated tenant_id stays separate from project_id.
func addressGroupBodyFilterValue(record *resource.BodyRecord[AddressGroup], key string) (json.RawMessage, error) {
	if record == nil {
		return nil, fmt.Errorf("%w: address group Body record is required", resource.ErrInvalidOption)
	}
	if record.Fields == nil {
		return nil, fmt.Errorf("%w: null address group list row is not an object", resource.ErrInvalidOption)
	}
	switch key {
	case "id", "tenant_id", "addresses":
		return resource.BodyRecordField(record.Fields, key, resource.BodyFieldJSON)
	default:
		return nil, fmt.Errorf("%w: unsupported address group Body filter %q", resource.ErrInvalidOption, key)
	}
}

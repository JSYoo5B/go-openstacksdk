package groups

import (
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// securityGroupBodyFilterValue compares the three audited non-query Body
// attributes against original response fields after native whole-page decoding.
func securityGroupBodyFilterValue(record *resource.BodyRecord[SecGroup], key string) (json.RawMessage, error) {
	if record == nil {
		return nil, fmt.Errorf("%w: security group Body record is required", resource.ErrInvalidOption)
	}
	if record.Fields == nil {
		return nil, fmt.Errorf("%w: null security group list row is not an object", resource.ErrInvalidOption)
	}
	switch key {
	case "created_at", "updated_at", "security_group_rules":
		return resource.BodyRecordField(record.Fields, key, resource.BodyFieldJSON)
	default:
		return nil, fmt.Errorf("%w: unsupported security group Body filter %q", resource.ErrInvalidOption, key)
	}
}

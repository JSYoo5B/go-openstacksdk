package subnetpools

import (
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// subnetPoolBodyFilterValue reads the original row after native whole-page
// decoding. Integer response descriptors use the SDK's exact-integral policy;
// tenant_id never falls back to project_id and timestamps retain their text.
func subnetPoolBodyFilterValue(record *resource.BodyRecord[SubnetPool], key string) (json.RawMessage, error) {
	if record == nil {
		return nil, fmt.Errorf("%w: subnet pool Body record is required", resource.ErrInvalidOption)
	}
	if record.Fields == nil {
		return nil, fmt.Errorf("%w: null subnet pool list row is not an object", resource.ErrInvalidOption)
	}
	switch key {
	case "default_prefixlen", "default_quota", "max_prefixlen", "min_prefixlen", "revision_number":
		return resource.BodyRecordField(record.Fields, key, resource.BodyFieldInteger)
	case "id", "created_at", "prefixes", "tenant_id", "updated_at":
		return resource.BodyRecordField(record.Fields, key, resource.BodyFieldJSON)
	default:
		return nil, fmt.Errorf("%w: unsupported subnet pool Body filter %q", resource.ErrInvalidOption, key)
	}
}

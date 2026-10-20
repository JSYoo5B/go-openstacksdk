package networks

import (
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// networkBodyFilterValue reads original lowercase response fields after native
// whole-page decoding. Only audited response bool/int descriptors normalize;
// arrays, timestamp text and unknown JSON values retain their original form.
func networkBodyFilterValue(record *resource.BodyRecord[Network], key string) (json.RawMessage, error) {
	if record == nil {
		return nil, fmt.Errorf("%w: network Body record is required", resource.ErrInvalidOption)
	}
	if record.Fields == nil {
		return nil, fmt.Errorf("%w: null network list row is not an object", resource.ErrInvalidOption)
	}
	switch key {
	case "is_default", "pvlan", "vlan_qinq", "vlan_transparent":
		return resource.BodyRecordField(record.Fields, key, resource.BodyFieldBoolean)
	case "mtu", "revision_number":
		return resource.BodyRecordField(record.Fields, key, resource.BodyFieldInteger)
	case "availability_zone_hints", "availability_zones", "created_at", "dns_domain", "qos_policy_id", "segments", "subnets", "updated_at":
		return resource.BodyRecordField(record.Fields, key, resource.BodyFieldJSON)
	default:
		return nil, fmt.Errorf("%w: unsupported network Body filter %q", resource.ErrInvalidOption, key)
	}
}

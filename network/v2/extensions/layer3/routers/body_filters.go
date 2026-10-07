package routers

import (
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// routerBodyFilterValue reads original lowercase response fields after native
// whole-page decoding. The Python revision_number descriptor reads revision;
// it is independent of the native model's revision_number field.
func routerBodyFilterValue(record *resource.BodyRecord[Router], key string) (json.RawMessage, error) {
	if record == nil {
		return nil, fmt.Errorf("%w: router Body record is required", resource.ErrInvalidOption)
	}
	if record.Fields == nil {
		return nil, fmt.Errorf("%w: null router list row is not an object", resource.ErrInvalidOption)
	}
	switch key {
	case "enable_ndp_proxy":
		return resource.BodyRecordField(record.Fields, key, resource.BodyFieldBoolean)
	case "evpn_vni", "revision":
		return resource.BodyRecordField(record.Fields, key, resource.BodyFieldInteger)
	case "availability_zone_hints", "availability_zones", "created_at", "external_gateway_info", "routes", "tenant_id", "updated_at":
		return resource.BodyRecordField(record.Fields, key, resource.BodyFieldJSON)
	default:
		return nil, fmt.Errorf("%w: unsupported router Body filter %q", resource.ErrInvalidOption, key)
	}
}

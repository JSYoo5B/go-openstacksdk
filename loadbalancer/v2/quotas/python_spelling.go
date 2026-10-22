package quotas

import (
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/request"
)

const legacyKeysArgument = "quota_legacy_keys"

// WithLegacyQuotaKeys selects the request spelling used for the load balancer
// and health monitor limits. By default Update sends the Octavia-documented
// keys loadbalancer and healthmonitor. When enabled is true it sends the
// deprecated keys load_balancer and health_monitor instead, which is the
// spelling pinned Python openstacksdk Quota uses. Octavia's QuotaBase request
// type declares both spellings (the underscore names are marked deprecated
// since Rocky) and stores either under the same quota column. Only the key
// names change: values, explicit null inheritance from WithDefaultLimit and
// the other limits are serialized identically. The last option wins.
func WithLegacyQuotaKeys(enabled bool) UpdateOption {
	return request.WithArgument[UpdateOpts](legacyKeysArgument, enabled)
}

// applyLegacyQuotaKeys renames the canonical core keys in place when the
// caller selected the deprecated Python spelling.
func applyLegacyQuotaKeys(config request.Config[UpdateOpts], core map[string]json.RawMessage) error {
	enabled, _, err := request.Argument[bool](config, legacyKeysArgument)
	if err != nil || !enabled {
		return err
	}
	for canonical, legacy := range map[string]string{"loadbalancer": "load_balancer", "healthmonitor": "health_monitor"} {
		if value, exists := core[canonical]; exists {
			delete(core, canonical)
			core[legacy] = value
		}
	}
	return nil
}

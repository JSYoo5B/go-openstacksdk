package network

import (
	"bytes"
	"encoding/json"
	"maps"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
)

// firewallFindOptions maps cloud firewall filters to the pinned Python
// Proxy find_firewall_*(name_or_id, ignore_missing=False, **filters) call,
// which forwards every filter to Resource.find:
//
//   - name_or_id and ignore_missing duplicate the explicit arguments
//     (Python TypeError) and are rejected before HTTP;
//   - microversion is Resource.find's keyword: it versions only the initial
//     member GET, while the fallback LIST and the later DELETE/PUT return to
//     the adapter default. It cannot be combined with the Go transport
//     Microversion extra;
//   - list_base_path is Resource.find's fallback LIST route;
//   - requires_id, base_path, skip_cache, error_message,
//     resource_response_key and headers are consumed by the shared Source
//     fetch-control profile (headers is an ordinary GET query value and a
//     formal fallback LIST header mapping; base_path routes the GET and the
//     LIST; skip_cache has no Go response cache to bypass);
//   - max_items is a raw Source cap compared before each row, so a cap equal
//     to a whole page still performs Python's next marker GET.
//
// Every other filter is an ordinary member GET query, a constructor seed and
// a fallback LIST query or local Body filter.
func firewallFindOptions(filters json.RawMessage, headers map[string]string, microversion *string) (neutronfind.Options, error) {
	strict := false
	policy := neutronfind.Options{IgnoreMissing: &strict, Query: map[string]json.RawMessage{}, Headers: maps.Clone(headers)}
	if microversion != nil {
		value := *microversion
		policy.Microversion = &value
	}
	members, err := qosPolicyCloudPairs(filters, false)
	if err != nil {
		return policy, err
	}
	for _, member := range members {
		switch member.Key {
		case "name_or_id", "ignore_missing":
			return policy, qosReadInvalid("%q duplicates cloud Find argument", member.Key)
		case "microversion", "list_base_path":
			formal := member.Key
			if formal == "list_base_path" {
				formal = "base_path"
			}
			normalized, _, cause := neutronfind.NormalizeProxyListOptions(neutronfind.Options{Query: map[string]json.RawMessage{formal: bytes.Clone(member.Value)}})
			if cause != nil {
				return policy, cause
			}
			if member.Key == "list_base_path" {
				policy.ListBasePath = normalized.BasePath
				continue
			}
			if microversion != nil {
				return policy, qosReadInvalid("filters microversion cannot be combined with the Go transport Microversion option")
			}
			policy.Microversion = normalized.Microversion
			continue
		}
		policy.Query[member.Key] = bytes.Clone(member.Value)
		policy.Order = append(policy.Order, member.Key)
	}
	return policy, nil
}

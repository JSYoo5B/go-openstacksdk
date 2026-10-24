package rules

import (
	"context"
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// MinimumPacketRateResources returns a Collection over the minimum packet rate
// rules of one fixed QoS policy, bound to direct GET
// qos/policies/{policy_id}/minimum_packet_rate_rules/{id} for the shared wait
// engine, such as network.WaitForDelete. Only resource.ID references are
// supported. The record view keeps attributes as raw JSON keyed by Python
// attribute name; resource.WithStatusAttribute selects one of them by exact
// name and requires a JSON string value.
func (a *API) MinimumPacketRateResources(policyID string) *resource.Collection[MinimumPacketRateRecord] {
	return resource.NewCollection(resource.Adapter[MinimumPacketRateRecord]{
		WaitAttributes: func(v *MinimumPacketRateRecord) map[string]json.RawMessage {
			return resource.RawResourceView(v.Resource)
		},
		Kind: "rules",
		Get: func(ctx context.Context, id string) (*MinimumPacketRateRecord, error) {
			return a.GetMinimumPacketRateRecord(ctx, policyID, id)
		},
		ID: func(v *MinimumPacketRateRecord) string { return rawText(v.Resource, "id") },
	})
}

// PacketRateLimitResources returns a Collection over the packet rate limit
// rules of one fixed QoS policy, bound to direct GET
// qos/policies/{policy_id}/packet_rate_limit_rules/{id} for the shared wait
// engine, such as network.WaitForDelete. Only resource.ID references are
// supported. The record view keeps attributes as raw JSON keyed by Python
// attribute name; resource.WithStatusAttribute selects one of them by exact
// name and requires a JSON string value.
func (a *API) PacketRateLimitResources(policyID string) *resource.Collection[PacketRateLimitRecord] {
	return resource.NewCollection(resource.Adapter[PacketRateLimitRecord]{
		WaitAttributes: func(v *PacketRateLimitRecord) map[string]json.RawMessage { return resource.RawResourceView(v.Resource) },
		Kind:           "rules",
		Get: func(ctx context.Context, id string) (*PacketRateLimitRecord, error) {
			return a.GetPacketRateLimitRecord(ctx, policyID, id)
		},
		ID: func(v *PacketRateLimitRecord) string { return rawText(v.Resource, "id") },
	})
}

func rawText(value *resource.RawResource, key string) string {
	if value == nil {
		return ""
	}
	var text string
	_ = json.Unmarshal(value.Body[key], &text)
	return text
}

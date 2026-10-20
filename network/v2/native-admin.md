# Native Admin

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:networking/v2/extensions/agents.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.Get`

- Get GETs agents/{id}, accepts only 200, decodes space-separated zone-less created_at/started_at/heartbeat_timestamp (empty string is zero, RFC3339 fails), and returns a nil agent without error when the agent envelope is missing or null.
  Tests: `network/v2/extensions/agents/native_admin_test.go:TestNativeAgentRoutesBodiesPagingAndDecode`, `network/v2/extensions/agents/native_admin_test.go:TestNativeAgentStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/agents call only; Python agent resource models, finders and admin helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/agents.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.WithUpdateField`

- Update PUTs {"agent":{...}} with pointer description/admin_state_up (empty and false sent explicitly) and extensions inside the envelope to agents/{id}, rejects a core-key extension and nil option before HTTP, and accepts only 200.
  Tests: `network/v2/extensions/agents/native_admin_test.go:TestNativeAgentRoutesBodiesPagingAndDecode`, `network/v2/extensions/agents/native_admin_test.go:TestNativeAgentStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/agents call only; Python agent resource models, finders and admin helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/agents.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.Delete`

- Delete sends DELETE agents/{id} without a body and accepts only 202/204.
  Tests: `network/v2/extensions/agents/native_admin_test.go:TestNativeAgentRoutesBodiesPagingAndDecode`, `network/v2/extensions/agents/native_admin_test.go:TestNativeAgentStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/agents call only; Python agent resource models, finders and admin helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/agents.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.WithListQuery`

- List GETs agents with ListOpts (pointer alive including false) and extension query values, follows agents_links rel=next hrefs while ignoring a links.next string, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `network/v2/extensions/agents/native_admin_test.go:TestNativeAgentRoutesBodiesPagingAndDecode`, `network/v2/extensions/agents/native_admin_test.go:TestNativeAgentStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/agents call only; Python agent resource models, finders and admin helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/agents.ListDHCPNetworks`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.ListDHCPNetworks`

- ListDHCPNetworks GETs agents/{id}/dhcp-networks as one request, accepts only 200, and decodes the networks key into []networks.Network with a missing key returning nil without error.
  Tests: `network/v2/extensions/agents/native_admin_test.go:TestNativeAgentRoutesBodiesPagingAndDecode`, `network/v2/extensions/agents/native_admin_test.go:TestNativeAgentStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/agents call only; Python agent resource models, finders and admin helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/agents.ScheduleDHCPNetwork`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.ScheduleDHCPNetwork`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.WithScheduleDHCPNetworkOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.WithScheduleDHCPNetworkField`

- ScheduleDHCPNetwork POSTs an unenveloped {"network_id":...} body with top-level extensions to agents/{id}/dhcp-networks, rejects an empty network_id, a core-key extension and nil option before HTTP, ignores the response body, and accepts only 201.
  Tests: `network/v2/extensions/agents/native_admin_test.go:TestNativeAgentRoutesBodiesPagingAndDecode`, `network/v2/extensions/agents/native_admin_test.go:TestNativeAgentStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/agents call only; Python agent resource models, finders and admin helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/agents.RemoveDHCPNetwork`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.RemoveDHCPNetwork`

- RemoveDHCPNetwork sends DELETE agents/{id}/dhcp-networks/{networkID} and accepts only 202/204.
  Tests: `network/v2/extensions/agents/native_admin_test.go:TestNativeAgentRoutesBodiesPagingAndDecode`, `network/v2/extensions/agents/native_admin_test.go:TestNativeAgentStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/agents call only; Python agent resource models, finders and admin helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/agents.ListL3Routers`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.ListL3Routers`

- ListL3Routers GETs agents/{id}/l3-routers as one request, accepts only 200, and decodes the routers key into []routers.Router with a missing key returning nil without error.
  Tests: `network/v2/extensions/agents/native_admin_test.go:TestNativeAgentRoutesBodiesPagingAndDecode`, `network/v2/extensions/agents/native_admin_test.go:TestNativeAgentStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/agents call only; Python agent resource models, finders and admin helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/agents.ScheduleL3Router`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.ScheduleL3Router`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.WithScheduleL3RouterOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.WithScheduleL3RouterField`

- ScheduleL3Router POSTs an unenveloped {"router_id":...} body with top-level extensions to agents/{id}/l3-routers, rejects an empty router_id, a core-key extension and nil option before HTTP, ignores the response body, and accepts only 201.
  Tests: `network/v2/extensions/agents/native_admin_test.go:TestNativeAgentRoutesBodiesPagingAndDecode`, `network/v2/extensions/agents/native_admin_test.go:TestNativeAgentStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/agents call only; Python agent resource models, finders and admin helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/agents.RemoveL3Router`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.RemoveL3Router`

- RemoveL3Router sends DELETE agents/{id}/l3-routers/{routerID} and accepts only 202/204.
  Tests: `network/v2/extensions/agents/native_admin_test.go:TestNativeAgentRoutesBodiesPagingAndDecode`, `network/v2/extensions/agents/native_admin_test.go:TestNativeAgentStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/agents call only; Python agent resource models, finders and admin helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/agents.ListBGPSpeakers`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.ListBGPSpeakers`

- ListBGPSpeakers GETs agents/{agentID}/bgp-drinstances as a single page that follows no links, decodes bgp_speakers into speakers.BGPSpeaker, yields nothing for an empty page, and reports io.EOF for a bodyless 204 and native page status errors without operation context.
  Tests: `network/v2/extensions/agents/native_admin_test.go:TestNativeAgentRoutesBodiesPagingAndDecode`, `network/v2/extensions/agents/native_admin_test.go:TestNativeAgentStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/agents call only; Python agent resource models, finders and admin helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/agents.ScheduleBGPSpeaker`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.ScheduleBGPSpeaker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.WithScheduleBGPSpeakerOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.WithScheduleBGPSpeakerField`

- ScheduleBGPSpeaker POSTs an unenveloped {"bgp_speaker_id":...} body with top-level extensions to agents/{agentID}/bgp-drinstances, rejects an empty speaker ID, a core-key extension and nil option before HTTP, ignores the response body, and accepts only 201.
  Tests: `network/v2/extensions/agents/native_admin_test.go:TestNativeAgentRoutesBodiesPagingAndDecode`, `network/v2/extensions/agents/native_admin_test.go:TestNativeAgentStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/agents call only; Python agent resource models, finders and admin helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/agents.RemoveBGPSpeaker`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.RemoveBGPSpeaker`

- RemoveBGPSpeaker sends DELETE agents/{agentID}/bgp-drinstances/{speakerID} and accepts only 202/204.
  Tests: `network/v2/extensions/agents/native_admin_test.go:TestNativeAgentRoutesBodiesPagingAndDecode`, `network/v2/extensions/agents/native_admin_test.go:TestNativeAgentStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/agents call only; Python agent resource models, finders and admin helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/agents.ListDRAgentHostingBGPSpeakers`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.ListDRAgentHostingBGPSpeakers`

- ListDRAgentHostingBGPSpeakers GETs bgp-speakers/{bgpSpeakerID}/bgp-dragents, follows agents_links rel=next hrefs while ignoring a links.next string, yields nothing for an empty page, and reports io.EOF for a bodyless 204 and native page status errors without operation context.
  Tests: `network/v2/extensions/agents/native_admin_test.go:TestNativeAgentRoutesBodiesPagingAndDecode`, `network/v2/extensions/agents/native_admin_test.go:TestNativeAgentStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/agents call only; Python agent resource models, finders and admin helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/segments.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/segments.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/segments.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/segments.WithCreateField`

- Create POSTs {"segment":{...}} to segments with extensions inside the envelope, omits empty optional fields including segmentation_id 0, rejects empty network_id/network_type, a core-key extension and nil option before HTTP, and accepts only 201/202.
  Tests: `network/v2/extensions/segments/native_admin_test.go:TestNativeSegmentRoutesBodiesPagingAndDecode`, `network/v2/extensions/segments/native_admin_test.go:TestNativeSegmentStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/segments call only; Python segment resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/segments.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/segments.API.Get`

- Get GETs segments/{id}, accepts only 200, decodes RFC3339 timestamps (zone-less forms fail) and null segmentation_id as 0, returns a zero segment for {} or a null envelope, and fails for other keys or a non-object envelope.
  Tests: `network/v2/extensions/segments/native_admin_test.go:TestNativeSegmentRoutesBodiesPagingAndDecode`, `network/v2/extensions/segments/native_admin_test.go:TestNativeSegmentStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/segments call only; Python segment resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/segments.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/segments.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/segments.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/segments.WithUpdateField`

- Update PUTs {"segment":{...}} with pointer name/description/segmentation_id (empty and 0 sent explicitly) and extensions inside the envelope to segments/{id}, rejects a core-key extension and nil option before HTTP, and accepts only 200.
  Tests: `network/v2/extensions/segments/native_admin_test.go:TestNativeSegmentRoutesBodiesPagingAndDecode`, `network/v2/extensions/segments/native_admin_test.go:TestNativeSegmentStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/segments call only; Python segment resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/segments.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/segments.API.Delete`

- Delete sends DELETE segments/{id} without a body and accepts only 202/204.
  Tests: `network/v2/extensions/segments/native_admin_test.go:TestNativeSegmentRoutesBodiesPagingAndDecode`, `network/v2/extensions/segments/native_admin_test.go:TestNativeSegmentStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/segments call only; Python segment resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/segments.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/segments.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/segments.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/segments.WithListQuery`

- List GETs segments with ListOpts (integer filters omitted at 0) and extension query values, ignores the Neutron segments_links array and follows only a links.next string because SegmentPage defines no NextPageURL, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `network/v2/extensions/segments/native_admin_test.go:TestNativeSegmentRoutesBodiesPagingAndDecode`, `network/v2/extensions/segments/native_admin_test.go:TestNativeSegmentStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/segments call only; Python segment resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/networkipavailabilities.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.API.Get`

- Get GETs network-ip-availabilities/{id}, accepts only 200, decodes total_ips/used_ips (network and every subnet, including exponent notation) into exact decimal strings, fails when a count is missing or null, and returns nil without error when the envelope is missing or null.
  Tests: `network/v2/extensions/networkipavailabilities/native_admin_test.go:TestNativeIPAvailabilityRoutesSinglePageAndDecode`, `network/v2/extensions/networkipavailabilities/native_admin_test.go:TestNativeIPAvailabilityStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/networkipavailabilities call only; Python network IP availability resource models and finders are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/networkipavailabilities.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithListQuery`

- List GETs network-ip-availabilities with ListOpts and extension query values as a single page that follows no links, decodes counts into exact decimal strings with a native page error for a missing count, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `network/v2/extensions/networkipavailabilities/native_admin_test.go:TestNativeIPAvailabilityRoutesSinglePageAndDecode`, `network/v2/extensions/networkipavailabilities/native_admin_test.go:TestNativeIPAvailabilityStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/networkipavailabilities call only; Python network IP availability resource models and finders are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/layer3/routers.ListL3Agents`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.ListL3Agents`

- ListL3Agents GETs routers/{id}/l3-agents as a single page that follows neither agents_links nor links.next, decodes agents into routers.L3Agent with space-separated zone-less timestamps (RFC3339 is a page decode error), yields nothing for an empty or keyless page, and reports io.EOF for a bodyless 204 and native page status errors without operation context.
  Tests: `network/v2/extensions/layer3/routers/native_l3agents_test.go:TestNativeRouterL3AgentsSinglePageDecodeAndStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/layer3/routers call only; Python router L3 agent helpers and agent resource models are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

## References

- [agents](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/extensions/agents/requests.go)
- [segments](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/extensions/segments/requests.go)
- [networkipavailabilities](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/extensions/networkipavailabilities/requests.go)
- [routers](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/extensions/layer3/routers/requests.go)

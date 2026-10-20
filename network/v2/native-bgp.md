# Native Bgp

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:networking/v2/extensions/bgp/peers.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.WithCreateField`

- Create POSTs {"bgp_peer":{...}} to bgp-peers with no required field, always sending auth_type, name, peer_ip and remote_as and omitting only an empty password, merges extensions inside the envelope, rejects core-key extensions and nil options before HTTP, decodes the bgp_peer envelope, and accepts only 201/202.
  Tests: `network/v2/extensions/bgp/peers/native_bgp_peers_test.go:TestNativeBGPPeerRoutesBodiesListAndDecode`, `network/v2/extensions/bgp/peers/native_bgp_peers_test.go:TestNativeBGPPeerStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgp/peers call only; Python BGP peer resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgp/peers.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.API.Delete`

- Delete sends DELETE bgp-peers/{id} without a body and accepts only 202/204.
  Tests: `network/v2/extensions/bgp/peers/native_bgp_peers_test.go:TestNativeBGPPeerRoutesBodiesListAndDecode`, `network/v2/extensions/bgp/peers/native_bgp_peers_test.go:TestNativeBGPPeerStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgp/peers call only; Python BGP peer resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgp/peers.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.API.Get`

- Get GETs bgp-peers/{id}, decodes the bgp_peer envelope (an empty object or null envelope yields a zero value, while other keys, a non-object envelope or a string remote_as fail with operation context), and accepts only 200.
  Tests: `network/v2/extensions/bgp/peers/native_bgp_peers_test.go:TestNativeBGPPeerRoutesBodiesListAndDecode`, `network/v2/extensions/bgp/peers/native_bgp_peers_test.go:TestNativeBGPPeerStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgp/peers call only; Python BGP peer resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgp/peers.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.API.List`

- List GETs bgp-peers once through a single-page pager that ignores bgp_peers_links and links.next and takes no query options, yields nothing for an empty page, and reports io.EOF for a bodyless 204 and native page status errors without operation context.
  Tests: `network/v2/extensions/bgp/peers/native_bgp_peers_test.go:TestNativeBGPPeerRoutesBodiesListAndDecode`, `network/v2/extensions/bgp/peers/native_bgp_peers_test.go:TestNativeBGPPeerStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgp/peers call only; Python BGP peer resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgp/peers.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.WithUpdateField`

- Update PUTs {"bgp_peer":{...}} to bgp-peers/{id} omitting an empty name and password (so empty options send {"bgp_peer":{}}), merges extensions inside the envelope, rejects core-key extensions and nil options before HTTP, and accepts only 200.
  Tests: `network/v2/extensions/bgp/peers/native_bgp_peers_test.go:TestNativeBGPPeerRoutesBodiesListAndDecode`, `network/v2/extensions/bgp/peers/native_bgp_peers_test.go:TestNativeBGPPeerStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgp/peers call only; Python BGP peer resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgp/speakers.AddBGPPeer`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.API.AddBGPPeer`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.WithAddBGPPeerOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.WithAddBGPPeerField`

- AddBGPPeer PUTs the envelope-free body {"bgp_peer_id":...} (an empty ID is sent) to bgp-speakers/{id}/add_bgp_peer with extensions at the top level, rejects core-key extensions before HTTP, decodes the whole response without an envelope into AddBGPPeerOpts (an enveloped response silently yields an empty ID, while an empty body or array fails), and accepts only 200.
  Tests: `network/v2/extensions/bgp/speakers/native_bgp_speakers_test.go:TestNativeBGPSpeakerRoutesBodiesActionsAndDecode`, `network/v2/extensions/bgp/speakers/native_bgp_speakers_test.go:TestNativeBGPSpeakerStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgp/speakers call only; Python BGP speaker resource models, finders, waits and dragent scheduling helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgp/speakers.AddGatewayNetwork`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.API.AddGatewayNetwork`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.WithAddGatewayNetworkOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.WithAddGatewayNetworkField`

- AddGatewayNetwork PUTs the envelope-free body {"network_id":...} to bgp-speakers/{id}/add_gateway_network with extensions at the top level, rejects nil options before HTTP, decodes the whole response without an envelope into AddGatewayNetworkOpts (an empty body or array fails), and accepts only 200.
  Tests: `network/v2/extensions/bgp/speakers/native_bgp_speakers_test.go:TestNativeBGPSpeakerRoutesBodiesActionsAndDecode`, `network/v2/extensions/bgp/speakers/native_bgp_speakers_test.go:TestNativeBGPSpeakerStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgp/speakers call only; Python BGP speaker resource models, finders, waits and dragent scheduling helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgp/speakers.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.WithCreateField`

- Create POSTs {"bgp_speaker":{...}} to bgp-speakers with no required field, always sending name, ip_version, local_as as a string and both advertise flags and omitting only empty networks, merges extensions inside the envelope, rejects core-key extensions and nil options before HTTP, decodes the bgp_speaker envelope with an integer local_as, and accepts only 201/202.
  Tests: `network/v2/extensions/bgp/speakers/native_bgp_speakers_test.go:TestNativeBGPSpeakerRoutesBodiesActionsAndDecode`, `network/v2/extensions/bgp/speakers/native_bgp_speakers_test.go:TestNativeBGPSpeakerStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgp/speakers call only; Python BGP speaker resource models, finders, waits and dragent scheduling helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgp/speakers.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.API.Delete`

- Delete sends DELETE bgp-speakers/{id} without a body and accepts only 202/204.
  Tests: `network/v2/extensions/bgp/speakers/native_bgp_speakers_test.go:TestNativeBGPSpeakerRoutesBodiesActionsAndDecode`, `network/v2/extensions/bgp/speakers/native_bgp_speakers_test.go:TestNativeBGPSpeakerStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgp/speakers call only; Python BGP speaker resource models, finders, waits and dragent scheduling helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgp/speakers.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.API.Get`

- Get GETs bgp-speakers/{id}, decodes the bgp_speaker envelope including networks and peers lists (an empty object or null envelope yields a zero value, while other keys, a non-object envelope or a string local_as fail with operation context), and accepts only 200.
  Tests: `network/v2/extensions/bgp/speakers/native_bgp_speakers_test.go:TestNativeBGPSpeakerRoutesBodiesActionsAndDecode`, `network/v2/extensions/bgp/speakers/native_bgp_speakers_test.go:TestNativeBGPSpeakerStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgp/speakers call only; Python BGP speaker resource models, finders, waits and dragent scheduling helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgp/speakers.GetAdvertisedRoutes`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.API.GetAdvertisedRoutes`

- GetAdvertisedRoutes GETs bgp-speakers/{id}/get_advertised_routes once through a single-page pager that ignores links.next, decodes advertised_routes destination/next_hop rows, yields nothing for an empty page, and reports io.EOF for a bodyless 204 and native page status errors without operation context.
  Tests: `network/v2/extensions/bgp/speakers/native_bgp_speakers_test.go:TestNativeBGPSpeakerRoutesBodiesActionsAndDecode`, `network/v2/extensions/bgp/speakers/native_bgp_speakers_test.go:TestNativeBGPSpeakerStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgp/speakers call only; Python BGP speaker resource models, finders, waits and dragent scheduling helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgp/speakers.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.API.List`

- List GETs bgp-speakers once through a single-page pager that ignores bgp_speakers_links and links.next and takes no query options, yields nothing for an empty page, and reports io.EOF for a bodyless 204 and native page status errors without operation context.
  Tests: `network/v2/extensions/bgp/speakers/native_bgp_speakers_test.go:TestNativeBGPSpeakerRoutesBodiesActionsAndDecode`, `network/v2/extensions/bgp/speakers/native_bgp_speakers_test.go:TestNativeBGPSpeakerStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgp/speakers call only; Python BGP speaker resource models, finders, waits and dragent scheduling helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgp/speakers.RemoveBGPPeer`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.API.RemoveBGPPeer`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.WithRemoveBGPPeerOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.WithRemoveBGPPeerField`

- RemoveBGPPeer PUTs the envelope-free body {"bgp_peer_id":...} to bgp-speakers/{id}/remove_bgp_peer with extensions at the top level, rejects core-key extensions before HTTP, never reads the response body, and accepts only 200.
  Tests: `network/v2/extensions/bgp/speakers/native_bgp_speakers_test.go:TestNativeBGPSpeakerRoutesBodiesActionsAndDecode`, `network/v2/extensions/bgp/speakers/native_bgp_speakers_test.go:TestNativeBGPSpeakerStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgp/speakers call only; Python BGP speaker resource models, finders, waits and dragent scheduling helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgp/speakers.RemoveGatewayNetwork`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.API.RemoveGatewayNetwork`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.WithRemoveGatewayNetworkOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.WithRemoveGatewayNetworkField`

- RemoveGatewayNetwork PUTs the envelope-free body {"network_id":...} (an empty ID is sent) to bgp-speakers/{id}/remove_gateway_network with extensions at the top level, rejects core-key extensions before HTTP, never reads the response body, and accepts only 200.
  Tests: `network/v2/extensions/bgp/speakers/native_bgp_speakers_test.go:TestNativeBGPSpeakerRoutesBodiesActionsAndDecode`, `network/v2/extensions/bgp/speakers/native_bgp_speakers_test.go:TestNativeBGPSpeakerStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgp/speakers call only; Python BGP speaker resource models, finders, waits and dragent scheduling helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgp/speakers.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.WithUpdateField`

- Update PUTs {"bgp_speaker":{...}} to bgp-speakers/{id} omitting an empty name but always sending both advertise flags (a name-only update also sends false for them), merges extensions inside the envelope, rejects core-key extensions before HTTP, and accepts only 200.
  Tests: `network/v2/extensions/bgp/speakers/native_bgp_speakers_test.go:TestNativeBGPSpeakerRoutesBodiesActionsAndDecode`, `network/v2/extensions/bgp/speakers/native_bgp_speakers_test.go:TestNativeBGPSpeakerStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgp/speakers call only; Python BGP speaker resource models, finders, waits and dragent scheduling helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgpvpns.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithCreateField`

- Create POSTs {"bgpvpn":{...}} to bgpvpn/bgpvpns with every field omitempty (empty options send {"bgpvpn":{}} and a zero local_pref or vni cannot be sent), merges extensions inside the envelope, rejects core-key extensions and nil options before HTTP, decodes the bgpvpn envelope with a null local_pref as nil, and accepts only 201/202.
  Tests: `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNRoutesBodiesPagingAndDecode`, `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgpvpns call only; Python BGP VPN and BGP VPN association resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgpvpns.CreateNetworkAssociation`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.CreateNetworkAssociation`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithCreateNetworkAssociationOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithCreateNetworkAssociationField`

- CreateNetworkAssociation POSTs {"network_association":{...}} to bgpvpn/bgpvpns/{id}/network_associations, requires network_id and rejects core-key extensions before HTTP, merges extensions inside the envelope, decodes the network_association envelope, and accepts only 201/202.
  Tests: `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNRoutesBodiesPagingAndDecode`, `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgpvpns call only; Python BGP VPN and BGP VPN association resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgpvpns.CreatePortAssociation`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.CreatePortAssociation`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithCreatePortAssociationOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithCreatePortAssociationField`

- CreatePortAssociation POSTs {"port_association":{...}} with routes and an explicit advertise_fixed_ips to bgpvpn/bgpvpns/{id}/port_associations, requires port_id and each route type before HTTP, merges extensions inside the envelope, decodes the port_association envelope with route local_pref pointers, and accepts only 201/202.
  Tests: `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNRoutesBodiesPagingAndDecode`, `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgpvpns call only; Python BGP VPN and BGP VPN association resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgpvpns.CreateRouterAssociation`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.CreateRouterAssociation`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithCreateRouterAssociationOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithCreateRouterAssociationField`

- CreateRouterAssociation POSTs {"router_association":{...}} with an explicit advertise_extra_routes pointer to bgpvpn/bgpvpns/{id}/router_associations, requires router_id before HTTP, decodes the router_association envelope, and accepts only 201/202.
  Tests: `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNRoutesBodiesPagingAndDecode`, `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgpvpns call only; Python BGP VPN and BGP VPN association resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgpvpns.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.Delete`

- Delete sends DELETE bgpvpn/bgpvpns/{id} without a body and accepts only 202/204.
  Tests: `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNRoutesBodiesPagingAndDecode`, `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgpvpns call only; Python BGP VPN and BGP VPN association resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgpvpns.DeleteNetworkAssociation`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.DeleteNetworkAssociation`

- DeleteNetworkAssociation sends DELETE bgpvpn/bgpvpns/{id}/network_associations/{association} without a body and accepts only 202/204.
  Tests: `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNRoutesBodiesPagingAndDecode`, `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgpvpns call only; Python BGP VPN and BGP VPN association resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgpvpns.DeletePortAssociation`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.DeletePortAssociation`

- DeletePortAssociation sends DELETE bgpvpn/bgpvpns/{id}/port_associations/{association} without a body and accepts only 202/204.
  Tests: `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNRoutesBodiesPagingAndDecode`, `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgpvpns call only; Python BGP VPN and BGP VPN association resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgpvpns.DeleteRouterAssociation`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.DeleteRouterAssociation`

- DeleteRouterAssociation sends DELETE bgpvpn/bgpvpns/{id}/router_associations/{association} without a body and accepts only 202/204.
  Tests: `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNRoutesBodiesPagingAndDecode`, `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgpvpns call only; Python BGP VPN and BGP VPN association resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgpvpns.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.Get`

- Get GETs bgpvpn/bgpvpns/{id}, decodes the bgpvpn envelope (an empty object yields a zero value, while other keys or an array body fail with operation context), and accepts only 200.
  Tests: `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNRoutesBodiesPagingAndDecode`, `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgpvpns call only; Python BGP VPN and BGP VPN association resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgpvpns.GetNetworkAssociation`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.GetNetworkAssociation`

- GetNetworkAssociation GETs bgpvpn/bgpvpns/{id}/network_associations/{association}, decodes the network_association envelope (an empty object yields a zero value, while other keys or an array body fail with operation context), and accepts only 200.
  Tests: `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNRoutesBodiesPagingAndDecode`, `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgpvpns call only; Python BGP VPN and BGP VPN association resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgpvpns.GetPortAssociation`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.GetPortAssociation`

- GetPortAssociation GETs bgpvpn/bgpvpns/{id}/port_associations/{association}, decodes the port_association envelope (an empty object yields a zero value, while other keys or an array body fail with operation context), and accepts only 200.
  Tests: `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNRoutesBodiesPagingAndDecode`, `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgpvpns call only; Python BGP VPN and BGP VPN association resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgpvpns.GetRouterAssociation`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.GetRouterAssociation`

- GetRouterAssociation GETs bgpvpn/bgpvpns/{id}/router_associations/{association}, decodes the router_association envelope (an empty object yields a zero value, while other keys or an array body fail with operation context), and accepts only 200.
  Tests: `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNRoutesBodiesPagingAndDecode`, `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgpvpns call only; Python BGP VPN and BGP VPN association resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgpvpns.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithListQuery`

- List GETs bgpvpn/bgpvpns with ListOpts (repeated fields and networks keys) and extension query values, ignores bgpvpns_links and links.next and pages only when the URL carries limit by re-requesting the current URL with marker set to the last ID until an empty page, yields nothing for an empty page, reports io.EOF for a JSON bodyless 204, ends cleanly on a 204 without content type, reports native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNRoutesBodiesPagingAndDecode`, `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgpvpns call only; Python BGP VPN and BGP VPN association resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgpvpns.ListNetworkAssociations`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.ListNetworkAssociations`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithListNetworkAssociationsOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithListNetworkAssociationsQuery`

- ListNetworkAssociations GETs bgpvpn/bgpvpns/{id}/network_associations with limit and extension query values, pages by last-ID marker only when limit is present until an empty page, yields nothing for an empty page, reports io.EOF for a JSON bodyless 204 and a decode error for a 204 without content type because its IsEmpty does not check 204, reports native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNRoutesBodiesPagingAndDecode`, `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgpvpns call only; Python BGP VPN and BGP VPN association resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgpvpns.ListPortAssociations`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.ListPortAssociations`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithListPortAssociationsOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithListPortAssociationsQuery`

- ListPortAssociations GETs bgpvpn/bgpvpns/{id}/port_associations once when no limit is given, yields nothing for an empty page, reports io.EOF for a JSON bodyless 204 and a decode error for a 204 without content type because its IsEmpty does not check 204, reports native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNRoutesBodiesPagingAndDecode`, `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgpvpns call only; Python BGP VPN and BGP VPN association resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgpvpns.ListRouterAssociations`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.ListRouterAssociations`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithListRouterAssociationsOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithListRouterAssociationsQuery`

- ListRouterAssociations GETs bgpvpn/bgpvpns/{id}/router_associations with the fields query and reads only the first page when no limit is given, yields nothing for an empty page, reports io.EOF for a JSON bodyless 204 and a decode error for a 204 without content type because its IsEmpty does not check 204, reports native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNRoutesBodiesPagingAndDecode`, `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgpvpns call only; Python BGP VPN and BGP VPN association resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgpvpns.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithUpdateField`

- Update PUTs {"bgpvpn":{...}} to bgpvpn/bgpvpns/{id} with pointer fields so an empty name, an empty route target list and a zero local_pref are sent and all-nil options send {"bgpvpn":{}}, merges extensions inside the envelope, rejects core-key extensions before HTTP, and accepts only 200.
  Tests: `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNRoutesBodiesPagingAndDecode`, `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgpvpns call only; Python BGP VPN and BGP VPN association resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgpvpns.UpdatePortAssociation`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.UpdatePortAssociation`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithUpdatePortAssociationOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithUpdatePortAssociationField`

- UpdatePortAssociation PUTs {"port_association":{...}} to bgpvpn/bgpvpns/{id}/port_associations/{association}, sends an empty routes list through the slice pointer, validates each route type before HTTP, merges extensions inside the envelope, rejects core-key extensions and nil options before HTTP, and accepts only 200.
  Tests: `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNRoutesBodiesPagingAndDecode`, `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgpvpns call only; Python BGP VPN and BGP VPN association resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/bgpvpns.UpdateRouterAssociation`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.UpdateRouterAssociation`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithUpdateRouterAssociationOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithUpdateRouterAssociationField`

- UpdateRouterAssociation PUTs {"router_association":{...}} to bgpvpn/bgpvpns/{id}/router_associations/{association} with an explicit advertise_extra_routes pointer (all-nil options send {"router_association":{}}), merges extensions inside the envelope, rejects core-key extensions before HTTP, and accepts only 200.
  Tests: `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNRoutesBodiesPagingAndDecode`, `network/v2/extensions/bgpvpns/native_bgpvpns_test.go:TestNativeBGPVPNStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/bgpvpns call only; Python BGP VPN and BGP VPN association resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

## References

- [bgp/peers](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/extensions/bgp/peers/requests.go)
- [bgp/speakers](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/extensions/bgp/speakers/requests.go)
- [bgpvpns](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/extensions/bgpvpns/requests.go)

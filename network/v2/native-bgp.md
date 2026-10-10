# Neutron native BGP·BGP VPN 호출

`service.BGPPeers`(`extensions/bgp/peers`), `service.BGPSpeakers`(`extensions/bgp/speakers`), `service.BGPVPNs`(`extensions/bgpvpns`)의 generated 메서드는 Gophercloud `v2.15.0`의 [bgp/peers](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/extensions/bgp/peers/requests.go), [bgp/speakers](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/extensions/bgp/speakers/requests.go), [bgpvpns](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/extensions/bgpvpns/requests.go) 요청을 바꾸지 않고 호출합니다. SDK는 단건 호출 오류에 `resource.OperationError{Resource: "peers"|"speakers"|"bgpvpns"}` 문맥만 더하고, 허용되지 않은 status는 native `gophercloud.ErrUnexpectedResponseCode`로 남습니다. ID는 escape 없이 경로에 이어 붙습니다.

BGP peer와 speaker 호출은 neutron-dynamic-routing의 기본 정책에서 모두 관리자 호출입니다. BGP VPN은 생성과 삭제, route target 계열 필드 수정이 기본 정책에서 관리자 호출이고, 조회·이름 수정과 association 호출은 BGP VPN을 소유한 project도 할 수 있습니다.

## BGP peer

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Create(ctx, opts, options...)` | `POST bgp-peers`, `{"bgp_peer": {...}}` | 201, 202 |
| `Get(ctx, id)` | `GET bgp-peers/{id}` | 200 |
| `Update(ctx, id, opts, options...)` | `PUT bgp-peers/{id}`, `{"bgp_peer": {...}}` | 200 |
| `List(ctx)` | `GET bgp-peers` | native pager 200, 204, 300 |
| `Delete(ctx, id)` | `DELETE bgp-peers/{id}` | 202, 204 |

`CreateOpts`에는 필수 필드가 없습니다. `Password`만 비어 있으면 생략하고, `Name`·`AuthType`·`PeerIP`·`RemoteAS`는 빈 값이어도 `""`와 `0`으로 보내므로 유효성은 Neutron이 판정합니다. `UpdateOpts`의 `Name`·`Password`는 모두 비어 있으면 생략해서 `{"bgp_peer": {}}`를 보냅니다. 응답의 `remote_as`는 숫자여야 하고 문자열이면 decode 오류입니다.

## BGP speaker

| 메서드 | 요청 | 본문·응답 | 성공 status |
|---|---|---|---|
| `Create(ctx, opts, options...)` | `POST bgp-speakers` | `{"bgp_speaker": {...}}` | 201, 202 |
| `Get(ctx, id)` | `GET bgp-speakers/{id}` | | 200 |
| `Update(ctx, id, opts, options...)` | `PUT bgp-speakers/{id}` | `{"bgp_speaker": {...}}` | 200 |
| `List(ctx)` | `GET bgp-speakers` | | native pager 200, 204, 300 |
| `Delete(ctx, id)` | `DELETE bgp-speakers/{id}` | | 202, 204 |
| `AddBGPPeer(ctx, id, opts, options...)` | `PUT bgp-speakers/{id}/add_bgp_peer` | `{"bgp_peer_id": ...}`, 응답도 같은 모양 | 200 |
| `RemoveBGPPeer(ctx, id, opts, options...)` | `PUT bgp-speakers/{id}/remove_bgp_peer` | `{"bgp_peer_id": ...}`, 응답 본문은 읽지 않음 | 200 |
| `AddGatewayNetwork(ctx, id, opts, options...)` | `PUT bgp-speakers/{id}/add_gateway_network` | `{"network_id": ...}`, 응답도 같은 모양 | 200 |
| `RemoveGatewayNetwork(ctx, id, opts, options...)` | `PUT bgp-speakers/{id}/remove_gateway_network` | `{"network_id": ...}`, 응답 본문은 읽지 않음 | 200 |
| `GetAdvertisedRoutes(ctx, id)` | `GET bgp-speakers/{id}/get_advertised_routes` | 응답 `advertised_routes` 배열 | native pager 200, 204, 300 |

`CreateOpts`도 필수 필드가 없고 `Networks`만 비어 있으면 생략합니다. `LocalAS`는 입력에서 문자열이라 `"local_as": "65000"`처럼 보내지만 응답의 `BGPSpeaker.LocalAS`는 정수입니다. 응답의 `local_as`가 문자열이면 decode 오류입니다. `UpdateOpts`의 `AdvertiseFloatingIPHostRoutes`와 `AdvertiseTenantNetworks`는 omitempty가 없어서 이름만 바꾸려 해도 두 값이 `false`로 함께 나갑니다. 기존 광고 설정을 유지하려면 현재 값을 읽어 함께 넣어야 합니다.

peer·gateway network 네 action은 본문에 envelope가 없어서 확장 필드는 `bgp_peer_id`·`network_id` 옆 최상위에 붙습니다. ID가 비어 있어도 HTTP 전에 거부하지 않고 `""`를 보냅니다. `AddBGPPeer`와 `AddGatewayNetwork`는 응답 전체를 envelope 없이 각각 `AddBGPPeerOpts`·`AddGatewayNetworkOpts`로 decode합니다. 그래서 `{"bgp_peer": {...}}`처럼 감싼 응답은 오류 없이 빈 ID가 되고, 빈 본문이나 배열은 operation 문맥과 함께 decode 오류입니다. 두 remove 호출은 응답 본문을 무시하므로 JSON이 아니어도 성공합니다.

## BGP VPN

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Create(ctx, opts, options...)` | `POST bgpvpn/bgpvpns`, `{"bgpvpn": {...}}` | 201, 202 |
| `Get(ctx, id)` | `GET bgpvpn/bgpvpns/{id}` | 200 |
| `Update(ctx, id, opts, options...)` | `PUT bgpvpn/bgpvpns/{id}`, `{"bgpvpn": {...}}` | 200 |
| `List(ctx, options...)` | `GET bgpvpn/bgpvpns` | native pager 200, 204, 300 |
| `Delete(ctx, id)` | `DELETE bgpvpn/bgpvpns/{id}` | 202, 204 |
| `CreateNetworkAssociation`·`CreateRouterAssociation`·`CreatePortAssociation(ctx, vpn, opts, options...)` | `POST bgpvpn/bgpvpns/{vpn}/network_associations` 등 | 201, 202 |
| `GetNetworkAssociation`·`GetRouterAssociation`·`GetPortAssociation(ctx, vpn, id)` | `GET .../{association}_associations/{id}` | 200 |
| `UpdateRouterAssociation`·`UpdatePortAssociation(ctx, vpn, id, opts, options...)` | `PUT .../router_associations/{id}`·`port_associations/{id}` | 200 |
| `ListNetworkAssociations`·`ListRouterAssociations`·`ListPortAssociations(ctx, vpn, options...)` | `GET .../network_associations` 등 | native pager 200, 204, 300 |
| `DeleteNetworkAssociation`·`DeleteRouterAssociation`·`DeletePortAssociation(ctx, vpn, id)` | `DELETE .../{association}_associations/{id}` | 202, 204 |

association의 envelope는 `network_association`, `router_association`, `port_association`입니다. network association은 수정 호출이 없습니다.

BGP VPN `CreateOpts`의 필드는 모두 omitempty라 아무것도 넣지 않으면 `{"bgpvpn": {}}`를 보내고, `LocalPref`·`VNI`의 0은 보낼 수 없습니다. `UpdateOpts`는 모든 필드가 pointer라서 빈 이름, 빈 route target 목록(`[]`), `local_pref: 0`을 명시해 보낼 수 있고, 모두 nil이면 `{"bgpvpn": {}}`입니다. 응답의 null `local_pref`는 nil pointer로, null 목록은 nil slice로 decode합니다.

association 생성은 `NetworkID`, `RouterID`, `PortID`가 각각 필수이고 비어 있으면 HTTP 전에 오류입니다. port association의 `Routes` 안에 있는 `PortRoutes.Type`도 필수이며, `UpdatePortAssociationOpts.Routes`처럼 slice pointer 안에 있어도 HTTP 전에 검사합니다. `AdvertiseExtraRoutes`·`AdvertiseFixedIPs`는 pointer라 false를 명시할 수 있고, 수정 옵션이 모두 nil이면 빈 envelope를 보냅니다.

## 목록과 paging

BGP peer·speaker 목록과 `GetAdvertisedRoutes`는 single page pager라서 요청을 한 번만 보냅니다. `bgp_peers_links`나 `links.next`가 있어도 따라가지 않고, query 옵션도 없습니다. 페이지가 나뉘는 배포라면 첫 페이지만 보게 됩니다.

BGP VPN과 association 목록은 marker pager입니다. 다음 페이지는 서버의 `*_links`가 아니라 현재 URL에 마지막 항목 ID를 `marker`로 넣어 만들며, 이 계산은 URL에 `limit` query가 있을 때만 합니다. 그래서 `limit` 없이 부르면 첫 페이지만 읽고, `limit`을 주면 빈 페이지를 받을 때까지 요청을 이어 갑니다. `ListOpts`의 `Fields`, `Networks`, `Routers`, `Ports`는 반복 query key가 되고, `With...Query` 확장 query도 다음 페이지 URL에 그대로 남습니다.

빈 목록은 아무 값도 내보내지 않습니다. JSON content type의 본문 없는 204는 native pager가 JSON을 먼저 읽기 때문에 `io.EOF` 오류 하나로 끝납니다. content type이 없는 204에서는 BGP VPN 목록만 빈 페이지로 끝나고, association 목록은 page의 `IsEmpty`가 204를 확인하지 않아 decode 오류를 냅니다. 목록 오류는 operation 문맥 없이 native 오류 그대로 전달됩니다.

## 공통 입력 규칙

`With...Field` 확장 필드는 envelope가 있는 본문에서는 envelope 안에, envelope가 없는 speaker action 본문에서는 최상위에 들어갑니다. 옵션 struct의 기존 JSON key와 겹치거나 nil 옵션을 주면 HTTP 전에 오류이며, 목록 호출의 nil 옵션도 요청 없이 오류 하나를 내보냅니다. 단건 응답의 `{}`나 null envelope는 빈 값으로 decode하고, 다른 key만 있거나 envelope가 객체가 아니면 operation 문맥과 함께 decode 오류입니다.

Python openstacksdk의 BGP·BGP VPN resource 모델, finder, 대기 동작은 이 문서의 native 호출 범위 밖이며 따로 관리합니다.

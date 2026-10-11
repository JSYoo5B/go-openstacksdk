# Python openstacksdk VPN·BGP 호출과 Go 대응

Python openstacksdk의 [`network/v2/_proxy.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)에 있는 VPNaaS, BGP dynamic routing, BGP VPN 메서드를 Go SDK 호출과 비교합니다. 기준 resource는 [`vpn_service.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/vpn_service.py), [`vpn_endpoint_group.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/vpn_endpoint_group.py), [`vpn_ike_policy.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/vpn_ike_policy.py), [`vpn_ipsec_policy.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/vpn_ipsec_policy.py), [`vpn_ipsec_site_connection.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/vpn_ipsec_site_connection.py), [`bgp_peer.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/bgp_peer.py), [`bgp_speaker.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/bgp_speaker.py), [`agent.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/agent.py), [`bgpvpn.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/bgpvpn.py)와 세 association 파일([network](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/bgpvpn_network_association.py), [port](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/bgpvpn_port_association.py), [router](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/bgpvpn_router_association.py))이고, 공통 동작은 [`proxy.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py)와 [`resource.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)를 따릅니다.

Go 쪽은 `extensions/vpnaas/*`, `extensions/bgp/peers`, `extensions/bgp/speakers`, `extensions/agents`, `extensions/bgpvpns`의 generated 호출입니다. 각 호출의 native 요청 형태는 [native-bgp.md](native-bgp.md)와 패키지별 native 테스트가 고정하고, 이 문서의 비교는 패키지마다 있는 `python_parity_test.go`가 검증합니다. 판정의 `대응`은 Python 인자가 낼 수 있는 효과를 공개 Go API로 모두 재현할 수 있다는 뜻이고, `미해결`은 재현할 수 없는 인자나 동작이 남아 있다는 뜻입니다.

## 메서드 대응표

| Python 메서드 | Go 호출 | 판정 |
|---|---|---|
| `create_vpn_service` | `services.API.Create` | 대응 |
| `delete_vpn_service` | `services.API.Remove(ctx, resource.ID(id))` | 대응 |
| `get_vpn_service` | `services.API.Get` | 미해결 |
| `find_vpn_service` | `Find(resource.ID)` 뒤 `Find(resource.Name)` | 미해결 |
| `vpn_services` | `services.API.List` | 대응 |
| `update_vpn_service` | `services.API.Update` | 대응 |
| `create_vpn_endpoint_group` | `endpointgroups.API.Create` | 대응 |
| `delete_vpn_endpoint_group` | `endpointgroups.API.Remove` | 대응 |
| `get_vpn_endpoint_group` | `endpointgroups.API.Get` | 미해결 |
| `find_vpn_endpoint_group` | `Find(resource.ID)` 뒤 `Find(resource.Name)` | 미해결 |
| `vpn_endpoint_groups` | `endpointgroups.API.List` | 대응 |
| `update_vpn_endpoint_group` | `endpointgroups.API.Update` | 대응 |
| `create_vpn_ike_policy` | `ikepolicies.API.Create` | 대응 |
| `delete_vpn_ike_policy` | `ikepolicies.API.Remove` | 대응 |
| `get_vpn_ike_policy` | `ikepolicies.API.Get` | 미해결 |
| `find_vpn_ike_policy` | `Find(resource.ID)` 뒤 `Find(resource.Name)` | 미해결 |
| `vpn_ike_policies` | `ikepolicies.API.List` | 대응 |
| `update_vpn_ike_policy` | `ikepolicies.API.Update` | 대응 |
| `create_vpn_ipsec_policy` | `ipsecpolicies.API.Create` | 대응 |
| `delete_vpn_ipsec_policy` | `ipsecpolicies.API.Remove` | 대응 |
| `get_vpn_ipsec_policy` | `ipsecpolicies.API.Get` | 미해결 |
| `find_vpn_ipsec_policy` | `Find(resource.ID)` 뒤 `Find(resource.Name)` | 미해결 |
| `vpn_ipsec_policies` | `ipsecpolicies.API.List` | 대응 |
| `update_vpn_ipsec_policy` | `ipsecpolicies.API.Update` | 대응 |
| `create_vpn_ipsec_site_connection` | `siteconnections.API.Create` | 대응 |
| `delete_vpn_ipsec_site_connection` | `siteconnections.API.Remove` | 대응 |
| `get_vpn_ipsec_site_connection` | `siteconnections.API.Get` | 미해결 |
| `find_vpn_ipsec_site_connection` | `Find(resource.ID)` 뒤 `Find(resource.Name)` | 미해결 |
| `vpn_ipsec_site_connections` | `siteconnections.API.List` | 대응 |
| `update_vpn_ipsec_site_connection` | `siteconnections.API.Update` | 미해결 |
| `create_bgp_peer` | `peers.API.Create` | 대응 |
| `delete_bgp_peer` | `peers.API.Remove` | 대응 |
| `get_bgp_peer` | `peers.API.Get` | 대응 |
| `find_bgp_peer` | `Find(resource.ID)` 뒤 `Find(resource.Name)` | 미해결 |
| `bgp_peers` | `peers.API.List`, `peers.API.All` | 미해결 |
| `update_bgp_peer` | `peers.API.Update` | 미해결 |
| `create_bgp_speaker` | `speakers.API.Create` | 대응 |
| `delete_bgp_speaker` | `speakers.API.Remove` | 대응 |
| `get_bgp_speaker` | `speakers.API.Get` | 대응 |
| `find_bgp_speaker` | `Find(resource.ID)` 뒤 `Find(resource.Name)` | 미해결 |
| `bgp_speakers` | `speakers.API.List`, `speakers.API.All` | 미해결 |
| `update_bgp_speaker` | `speakers.API.Update` | 미해결 |
| `add_bgp_peer_to_speaker` | `speakers.API.AddBGPPeer` | 대응 |
| `remove_bgp_peer_from_speaker` | `speakers.API.RemoveBGPPeer` | 대응 |
| `get_bgp_dragents_hosting_speaker` | `agents.API.ListDRAgentHostingBGPSpeakers` | 대응 |
| `add_bgp_speaker_to_dragent` | `agents.API.ScheduleBGPSpeaker` | 대응 |
| `get_bgp_speakers_hosted_by_dragent` | `agents.API.ListBGPSpeakers` | 대응 |
| `remove_bgp_speaker_from_dragent` | `agents.API.RemoveBGPSpeaker` | 대응 |
| `create_bgpvpn` | `bgpvpns.API.Create` | 미해결 |
| `delete_bgpvpn` | `bgpvpns.API.Remove` | 대응 |
| `get_bgpvpn` | `bgpvpns.API.Get` | 미해결 |
| `find_bgpvpn` | `Find(resource.ID)` 뒤 `Find(resource.Name)` | 미해결 |
| `bgpvpns` | `bgpvpns.API.List` | 대응 |
| `update_bgpvpn` | `bgpvpns.API.Update` | 대응 |
| `create_bgpvpn_network_association` | `bgpvpns.API.CreateNetworkAssociation` | 대응 |
| `delete_bgpvpn_network_association` | `resource.IgnoreMissing(api.DeleteNetworkAssociation(...))` | 대응 |
| `get_bgpvpn_network_association` | `bgpvpns.API.GetNetworkAssociation` | 미해결 |
| `bgpvpn_network_associations` | `bgpvpns.API.ListNetworkAssociations` | 대응 |
| `create_bgpvpn_port_association` | `bgpvpns.API.CreatePortAssociation` | 대응 |
| `delete_bgpvpn_port_association` | `resource.IgnoreMissing(api.DeletePortAssociation(...))` | 대응 |
| `get_bgpvpn_port_association` | `bgpvpns.API.GetPortAssociation` | 미해결 |
| `find_bgpvpn_port_association` | 없음 (`GetPortAssociation`, `ListPortAssociations`를 직접 조합) | 미해결 |
| `update_bgpvpn_port_association` | `bgpvpns.API.UpdatePortAssociation` | 대응 |
| `bgpvpn_port_associations` | `bgpvpns.API.ListPortAssociations` | 대응 |
| `create_bgpvpn_router_association` | `bgpvpns.API.CreateRouterAssociation` | 대응 |
| `delete_bgpvpn_router_association` | `resource.IgnoreMissing(api.DeleteRouterAssociation(...))` | 대응 |
| `get_bgpvpn_router_association` | `bgpvpns.API.GetRouterAssociation` | 미해결 |
| `update_bgpvpn_router_association` | `bgpvpns.API.UpdateRouterAssociation` | 대응 |
| `bgpvpn_router_associations` | `bgpvpns.API.ListRouterAssociations` | 대응 |

69개 중 45개가 대응이고 24개가 미해결입니다. Neutron은 microversion을 쓰지 않으므로 이 범위에서는 microversion 차이가 없습니다.

## 공통 차이

Python 메서드는 Resource 객체나 ID 문자열을 받고 수정 가능한 Resource를 돌려줍니다. Go 호출은 ID 문자열을 받고 Gophercloud 모델 포인터를 돌려주며, null이거나 빠진 필드는 0 값이 됩니다.

`delete_*`의 기본값 `ignore_missing=True`는 top-level resource에서 `API.Remove(ctx, resource.ID(id))`로 옮깁니다. `Remove`는 404를 nil로 바꾸고, `ignore_missing=False`가 필요하면 `resource.WithMissingError()`를 넘겨 `resource.ErrNotFound`를 받습니다. 409 같은 다른 오류는 그대로 돌아옵니다. generated `Delete`를 직접 부르면 404도 오류입니다. `Remove`는 `/`, `\`, `?`, `#`, `%`, 공백이 든 ID를 HTTP 전에 거부하지만 Python은 어떤 문자열이든 경로에 붙입니다. 성공 status도 다른데, Python은 모든 2xx를 받고 Go는 delete에서 202와 204, create에서 201과 202만 받습니다.

`update_*`에 속성을 하나도 주지 않으면 Python은 요청을 보내지 않습니다. Go `Update`는 빈 envelope라도 항상 PUT을 보냅니다. Python은 `_allow_unknown_attrs_in_body`가 켜진 resource에서만 모르는 keyword를 본문에 넣고, IKE·IPsec policy와 site connection에서는 버립니다. Go는 `WithCreateField`·`WithUpdateField`로 확장 key를 보낼 수 있지만 Opts struct의 JSON tag와 같은 key는 HTTP 전에 거부합니다. 그래서 omitempty 필드의 빈 값이나 0은 확장 필드로도 보낼 수 없습니다.

`find_*`는 Python에서 한 번의 호출이 ID로 GET하고, 400·403·404면 목록으로 넘어가서 ID나 이름이 같은 행을 찾습니다. Go에서는 `API.Find(ctx, resource.ID(x), resource.WithIgnoreMissing())`가 nil을 돌려줄 때 `API.Find(ctx, resource.Name(x))`를 이어 부르는 두 단계가 필요합니다. `WithIgnoreMissing`은 404만 nil로 바꾸므로 400이나 403 뒤에는 호출자가 이름 조회로 넘어가야 합니다. Go 이름 조회는 이름만 비교하고 중복이면 `resource.ErrAmbiguous`를 돌려주며, Python은 `DuplicateResource`를 냅니다. Python `find_*`는 `**query`를 ID GET과 목록 양쪽에 넘기는데 `Find`에는 query를 넘길 방법이 없습니다. 그래서 기본 인자로 부른 경우의 요청만 같고, 모든 `find_*`는 미해결로 남깁니다.

`get_*`에 `fields`가 있는 메서드는 Python이 attribute 이름을 서버 이름으로 바꿔 `fields` query를 반복해서 보냅니다. Gophercloud v2.15.0의 generated `Get`은 ID만 받고 이 resource들에는 SDK 소유의 field getter가 없어서, `fields`를 쓰는 호출은 재현할 수 없습니다. BGP peer와 speaker의 `get_*`에는 `fields`가 없으므로 대응입니다.

목록에서 Python은 resource의 `QueryParameters`에 있는 key만 서버 query로 보내고, 그 밖의 Body attribute keyword는 받아 온 행을 로컬에서 거릅니다. Go `API.List`는 `WithListOptions`와 `WithListQuery`로 준 key를 모두 서버에 보냅니다. 서버 필터와 로컬 필터의 결과가 같다면 같은 행을 얻지만, 로컬 필터를 원하면 호출자가 반복문에서 직접 걸러야 합니다.

## VPN service

`services.CreateOpts.AdminStateUp`은 omitempty가 없어서 nil이면 `"admin_state_up": null`을 보냅니다. Python은 지정하지 않은 key를 빼고 Neutron은 기본값 true를 쓰므로, Go에서는 `AdminStateUp: gophercloud.Enabled`처럼 값을 명시해야 같은 결과가 됩니다. `UpdateOpts`의 이름·설명·관리 상태는 포인터라서 빈 이름이나 false를 그대로 보낼 수 있습니다. `vpn_services(router_id=..., is_admin_state_up=True)`는 `ListOpts`의 `RouterID`, `AdminStateUp`으로 같은 query가 되고, 목록은 Python처럼 `vpnservices_links`의 next href를 따라갑니다.

## VPN endpoint group

Python `vpn_endpoint_groups(type="cidr")`는 `type`을 `endpoint_type` query로 바꿉니다. `ListOpts.Type`은 `type`으로 나가므로 Go에서는 `WithListQuery("endpoint_type", "cidr")`를 써야 합니다. `CreateOpts.Endpoints`는 omitempty가 없어서 nil이면 null을 보내니, 빈 목록이 필요하면 `[]string{}`을 넘깁니다.

## IKE policy

`UpdateOpts.Phase1NegotiationMode`와 `ListOpts.Phase1NegotiationMode`는 `phase_1_negotiation_mode`라는 key로 나갑니다. Python과 Neutron이 쓰는 key는 `phase1_negotiation_mode`이고, 이 key는 Opts에 예약되어 있지 않아서 `WithUpdateField("phase1_negotiation_mode", ...)`와 `WithListQuery("phase1_negotiation_mode", ...)`로 보낼 수 있습니다. 생성 쪽 `CreateOpts`는 올바른 key를 씁니다. lifetime은 `LifetimeCreateOpts`·`LifetimeUpdateOpts`로 보내며, 0 값과 빈 알고리즘 문자열은 빠지지만 Neutron이 어차피 거부하는 값입니다.

## IPsec policy

생성, 수정, 목록 모두 typed Opts의 key가 Python과 같습니다. lifetime과 빈 문자열의 처리는 IKE policy와 같습니다.

## IPsec site connection

`CreateOpts`는 `ikepolicy_id`, `ipsecpolicy_id`, `vpnservice_id`, `peer_address`, `peer_id`, `psk`를 항상 보내는데, 모두 Neutron이 요구하는 필드라 문제가 되지 않습니다. 목록의 `status`는 Python에서 서버 query이지만 `ListOpts`에는 필드가 없고 `Resources` collection은 status query를 지우므로, `API.List`에 `WithListQuery("status", ...)`를 넘겨야 합니다.

수정은 미해결입니다. `UpdateOpts`의 `LocalID`, `PeerCIDRs`, `PeerAddress`, `PeerID`, endpoint group ID, `PSK`, `MTU`는 포인터가 아니라서 빈 값이 빠지고, 확장 필드로도 같은 key를 보낼 수 없습니다. 그래서 `local_id=""`로 기본값을 되돌리거나 `peer_cidrs=[]`로 CIDR 목록을 비우는 Python 호출을 Go로 보낼 수 없습니다.

## BGP peer

`peers.CreateOpts`는 `name`, `auth_type`, `peer_ip`, `remote_as`를 항상 보냅니다. Python에서 `auth_type`을 생략하면 Neutron이 `none`을 쓰지만 Go는 빈 문자열을 보내므로 `AuthType: "none"`을 명시해야 합니다. `remote_as`는 Go에서 정수이고 응답의 문자열 값은 decode 오류입니다.

`UpdateOpts.Name`은 omitempty 문자열이라 `update_bgp_peer(peer, name="")`처럼 이름을 비우는 요청을 보낼 수 없어서 미해결입니다. 목록도 미해결입니다. BgpPeer의 서버 query는 `limit`과 `marker`뿐인데, generated `List`는 옵션이 없는 single page pager이고 collection에 query를 넘기면 `resource.ErrUnsupported`로 끝납니다. `bgp_peers_links`도 따라가지 않습니다. 이름 같은 Body attribute 필터는 Python도 로컬에서 거르므로 `All(ctx, resource.WithName(x))`와 결과가 같습니다. `find_bgp_peer`의 목록 단계는 BgpPeer에 `name` query가 없어서 Python도 필터 없는 `GET bgp-peers`를 보내며, Go 이름 조회도 같은 요청을 보냅니다.

## BGP speaker와 dynamic routing agent

`speakers.CreateOpts`는 두 advertise flag를 항상 보냅니다. Python은 생략하고 Neutron은 둘 다 true를 기본값으로 쓰므로, Go에서는 `AdvertiseFloatingIPHostRoutes`와 `AdvertiseTenantNetworks`를 true로 넣어야 같은 speaker가 만들어집니다. `LocalAS`는 문자열로 나가고 Neutron이 정수로 바꿉니다.

`update_bgp_speaker`는 미해결입니다. `UpdateOpts`의 두 flag는 omitempty 없는 bool이어서 이름만 바꿔도 둘 다 false로 나가고, 확장 필드로 덮을 수도 없습니다. 기존 설정을 지키려면 speaker를 먼저 읽어 같은 값을 다시 넣어야 하며 이는 Python의 요청과 다릅니다. `bgp_speakers`와 `find_bgp_speaker`는 BGP peer와 같은 이유로 미해결입니다.

`add_bgp_peer_to_speaker`는 `AddBGPPeer`로 같은 PUT을 보내지만, Python은 응답 dict 전체를 돌려주고 Go는 `bgp_peer_id`만 decode합니다. `remove_bgp_peer_from_speaker`는 `RemoveBGPPeer`와 같으며 둘 다 HTTP 오류를 돌려줍니다. Python은 모든 2xx를 받고 Go는 200만 받습니다.

`get_bgp_dragents_hosting_speaker`는 Python에서 `bgp-speakers/{id}/bgp-dragents`로 ID를 얻은 뒤 `GET agents`를 한 번 더 불러 그 행만 돌려줍니다. Go `agents.API.ListDRAgentHostingBGPSpeakers`는 첫 응답의 agent 행을 그대로 decode하고, Python과 같은 행이 필요하면 `agents.API.List`로 목록을 읽어 ID로 고르면 됩니다. `get_bgp_speakers_hosted_by_dragent`도 같은 구조이며 `agents.API.ListBGPSpeakers` 뒤에 `speakers.API.List`를 이어 부릅니다. 다만 Go speaker 목록은 첫 페이지만 읽습니다.

`add_bgp_speaker_to_dragent`와 `remove_bgp_speaker_from_dragent`는 Python이 응답을 전혀 확인하지 않아서 실패해도 None을 돌려줍니다. Go `ScheduleBGPSpeaker`는 201이 아니면, `RemoveBGPSpeaker`는 성공 status가 아니면 오류를 돌려줍니다. 이 두 호출의 Go 테스트는 `extensions/agents` 패키지 대신 `bgp/speakers/python_parity_test.go`에 있습니다.

## BGP VPN과 association

`create_bgpvpn`은 미해결입니다. `CreateOpts.LocalPref`가 omitempty 정수라서 networking-bgpvpn이 허용하는 `local_pref=0`을 생성 요청에 넣을 수 없고, 확장 필드도 거부됩니다. `UpdateOpts.LocalPref`는 포인터이므로 생성 뒤 수정으로는 0을 넣을 수 있습니다. 수정은 모든 필드가 포인터라 빈 이름, 빈 target 목록, 0을 그대로 보냅니다.

`bgpvpns(...)`의 `name`, `local_pref`, `vni`, `type`은 `ListOpts`에 없어서 `WithListQuery`로 보냅니다. `networks`, `routers`, `ports`, `fields`는 반복 query가 됩니다. Go marker pager는 `limit`이 있을 때 마지막 ID를 `marker`로 붙여 빈 페이지까지 요청을 이어 가고 `bgpvpns_links`는 보지 않습니다. Python은 link를 따라가다가 link가 없으면 같은 marker 요청으로 넘어가므로 받는 행은 같습니다. association 목록도 같은 방식이며 Python association의 서버 query는 `limit`과 `marker`뿐입니다.

`find_bgpvpn`의 목록 단계에서 Python은 `?name=`을 붙이지만, bgpvpns collection에는 name query binding이 없어서 Go 이름 조회는 필터 없는 목록을 받아 로컬에서 거릅니다. association의 생성·조회·수정·삭제는 `Create*Association`, `Get*Association`, `Update*Association`, `Delete*Association`이 같은 경로와 envelope를 씁니다. association에는 `Resources` collection이 없으므로 `delete_*_association`의 기본값은 `resource.IgnoreMissing(api.DeleteNetworkAssociation(ctx, vpnID, id))`처럼 감싸서 옮깁니다. 생성에서 `network_id`, `router_id`, `port_id`가 비었거나 port route의 `type`이 비면 Go는 HTTP 전에 거부합니다.

## 미해결 메서드

- `get_vpn_service`, `get_vpn_endpoint_group`, `get_vpn_ike_policy`, `get_vpn_ipsec_policy`, `get_vpn_ipsec_site_connection`, `get_bgpvpn`, `get_bgpvpn_network_association`, `get_bgpvpn_port_association`, `get_bgpvpn_router_association`: generated `Get`은 ID만 받으므로 `fields` query를 보내려면 SDK 소유 getter가 필요합니다.
- `find_vpn_service`, `find_vpn_endpoint_group`, `find_vpn_ike_policy`, `find_vpn_ipsec_policy`, `find_vpn_ipsec_site_connection`, `find_bgp_peer`, `find_bgp_speaker`: 기본 인자의 두 단계 조회는 같은 요청을 보내지만, `Find`가 query를 받지 않아 `**query`를 넘길 수 없습니다.
- `find_bgpvpn`: query를 넘길 수 없을 뿐 아니라 이름 조회가 `?name=` 없이 목록을 받습니다.
- `find_bgpvpn_port_association`: association용 collection이나 find helper가 없어서 ID 조회, 목록 대체, ignore_missing, 중복 검사를 호출자가 직접 만들어야 합니다.
- `bgp_peers`, `bgp_speakers`: generated 목록이 query를 받지 않는 single page pager라 `limit`과 `marker`를 보낼 수 없고 `*_links`도 따라가지 않습니다.
- `update_bgp_peer`: 이름을 빈 문자열로 바꾸는 요청을 보낼 수 없습니다.
- `update_bgp_speaker`: 두 advertise flag를 빼고 수정 요청을 보낼 수 없습니다.
- `update_vpn_ipsec_site_connection`: `local_id=""`, `peer_cidrs=[]` 같은 빈 값을 보낼 수 없습니다.
- `create_bgpvpn`: `local_pref=0`을 생성 요청에 넣을 수 없습니다.

# Neutron v2 Python proxy 대응: agent, segment, port 보조 호출

이 문서는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [network v2 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py) 가운데 agent, DHCP 스케줄링, segment, network segment range, extension, network IP availability, port 목록과 bulk 생성, port binding, local IP, auto-allocated topology, availability zone, service provider, floating IP의 port 연결, BGP speaker gateway network, trunk subport, subnet pool onboarding, 공통 wait 메서드 58개를 Go 호출과 비교합니다. Python 요청은 [Proxy 공통 helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py)와 [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py), 그리고 [Agent](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/agent.py), [Segment](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/segment.py), [Extension](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/extension.py), [NetworkIPAvailability](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/network_ip_availability.py), [Port](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/port.py), [PortBinding](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/port_binding.py), [FloatingIP](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/floating_ip.py), [BgpSpeaker](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/bgp_speaker.py), [Trunk](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/trunk.py) 파일에서 따라갔습니다. 각 package의 `python_parity_core_test.go`가 Go 쪽 요청을 고정합니다.

`go_mapping`은 모든 Python 인자의 효과를 공개 Go API로 재현할 수 있다는 뜻이고, 이때도 아래 절의 차이는 남습니다. `unresolved`는 Go에 경로, query, 본문 값이나 helper 동작이 없어서 같은 요청이나 같은 기본 동작을 만들 수 없다는 뜻입니다. 표의 Go 호출은 각 package의 `New(client)`가 돌려준 API를 기준으로 적었습니다.

| Python 메서드 | Go 호출 | 판정 |
|---|---|---|
| `activate_port_binding` | 없음 | unresolved |
| `add_dhcp_agent_to_network` | `agents.API.ScheduleDHCPNetwork(ctx, agentID, ScheduleDHCPNetworkOpts{NetworkID})` | go_mapping |
| `add_gateway_network_to_speaker` | `speakers.API.AddGatewayNetwork(ctx, id, AddGatewayNetworkOpts{NetworkID})` | go_mapping |
| `add_ip_to_port` | `floatingips.API.Update(ctx, ipID, UpdateOpts{PortID: &portID})` | go_mapping |
| `agents` | `agents.API.List(ctx, WithListOptions, WithListQuery)`, `All(ctx)` | unresolved |
| `availability_zones` | 없음 | unresolved |
| `create_local_ip` | 없음 | go_mapping |
| `create_local_ip_association` | 없음 | go_mapping |
| `create_network_segment_range` | 없음 | unresolved |
| `create_port_binding` | 없음 | unresolved |
| `create_ports` | `ports.API.Create`는 한 개씩만 생성 | unresolved |
| `create_segment` | `segments.API.Create(ctx, CreateOpts, WithCreateField)` | go_mapping |
| `delete_agent` | `agents.API.Remove(ctx, resource.ID(id))` | go_mapping |
| `delete_auto_allocated_topology` | 없음 | unresolved |
| `delete_local_ip` | 없음 | go_mapping |
| `delete_local_ip_association` | 없음 | go_mapping |
| `delete_network_segment_range` | 없음 | unresolved |
| `delete_port_binding` | 없음 | unresolved |
| `delete_segment` | `segments.API.Remove(ctx, resource.ID(id))` | go_mapping |
| `dhcp_agent_hosting_networks` | `agents.API.ListDHCPNetworks(ctx, agentID)` (query 불가) | unresolved |
| `extensions` | `extensions.API.List(ctx)` | unresolved |
| `find_available_ip` | `floatingips.API.List(ctx)`에서 첫 빈 `PortID`에서 중단 | unresolved |
| `find_extension` | `extensions.API.Get(ctx, alias)` (목록 fallback 없음) | unresolved |
| `find_local_ip` | 없음 | go_mapping |
| `find_local_ip_association` | 없음 | go_mapping |
| `find_network_ip_availability` | `networkipavailabilities.API.Get`, `List`를 직접 조합해야 함 | unresolved |
| `find_network_segment_range` | 없음 | unresolved |
| `find_segment` | `segments.API.Find`를 ID와 이름으로 따로 호출 (query 불가) | unresolved |
| `get_advertised_routes_of_speaker` | `speakers.API.GetAdvertisedRoutes(ctx, id)` | go_mapping |
| `get_agent` | `agents.API.GetWithFields(ctx, id, fields...)` | go_mapping |
| `get_auto_allocated_topology` | 없음 | unresolved |
| `get_local_ip` | 없음 | go_mapping |
| `get_local_ip_association` | 없음 | go_mapping |
| `get_network_ip_availability` | `networkipavailabilities.API.GetWithFields(ctx, networkID, fields...)` | go_mapping |
| `get_network_segment_range` | 없음 | unresolved |
| `get_segment` | `segments.API.GetWithFields(ctx, id, fields...)` | go_mapping |
| `get_subnet_ports` | `ports.API.All(ctx)` 후 `FixedIPs`의 subnet ID로 선택 | unresolved |
| `get_trunk_subports` | `trunks.API.GetSubports(ctx, id)` | go_mapping |
| `local_ip_associations` | 없음 | go_mapping |
| `local_ips` | 없음 | go_mapping |
| `network_hosting_dhcp_agents` | 없음 | unresolved |
| `network_ip_availabilities` | `networkipavailabilities.API.List(ctx, WithListOptions, WithListQuery)` | unresolved |
| `network_segment_ranges` | 없음 | unresolved |
| `onboard_network_subnets` | 없음 | unresolved |
| `port_bindings` | 없음 | unresolved |
| `ports` | `ports.API.List(ctx, WithListOptions, WithListQuery)` | unresolved |
| `remove_dhcp_agent_from_network` | `agents.API.RemoveDHCPNetwork(ctx, agentID, networkID)` | go_mapping |
| `remove_gateway_network_from_speaker` | `speakers.API.RemoveGatewayNetwork(ctx, id, RemoveGatewayNetworkOpts{NetworkID})` | go_mapping |
| `remove_ip_from_port` | `floatingips.API.Update(ctx, ipID, UpdateOpts{PortID: &empty})` | go_mapping |
| `segments` | `segments.API.List(ctx, WithListOptions, WithListQuery)` | unresolved |
| `service_providers` | 없음 | unresolved |
| `update_agent` | `agents.API.Update(ctx, id, UpdateOpts, WithUpdateField)` | go_mapping |
| `update_local_ip` | 없음 | go_mapping |
| `update_network_segment_range` | 없음 | unresolved |
| `update_segment` | `segments.API.Update(ctx, id, UpdateOpts, WithUpdateField)` | go_mapping |
| `validate_auto_allocated_topology` | 없음 | unresolved |
| `wait_for_delete` | `ports.API.WaitForDeletion` 등 일부 collection만 | unresolved |
| `wait_for_status` | `ports.API.WaitFor` 등 일부 collection만 | unresolved |

## 공통 차이

Python은 Resource 객체나 ID 문자열을 받고 mutable Resource를 돌려줍니다. Go generated 호출은 ID 문자열을 받고 Gophercloud typed model을 돌려주며, `Remove`와 `Find`는 `resource.Ref`를 받습니다. 모델에 없는 응답 field는 Go 결과에 남지 않습니다. Neutron은 microversion을 쓰지 않으므로 두 쪽 모두 microversion header를 보내지 않습니다.

Python 목록 메서드는 `QueryParameters`에 있는 인자만 서버 query로 보내고, Resource의 Body 속성이지만 query에 없는 인자는 각 page를 받은 뒤 지역에서 걸러 냅니다. 알 수 없는 인자는 버립니다. Go의 `WithListOptions`와 `WithListQuery`는 받은 key를 모두 서버로 보내므로, Python이 지역에서 거르는 조건은 호출자가 돌려받은 행에서 직접 걸러야 합니다. Python의 `is_alive`, `is_admin_state_up` 같은 속성 이름은 Go에서 `alive`, `admin_state_up` 같은 wire 이름으로 넘깁니다.

Neutron 목록의 다음 page는 `<resources>_links`의 `rel=next` 항목으로 이어집니다. Python은 다음 링크의 query를 꺼내 자신의 query에 합친 뒤 요청하고, Go는 Neutron이 준 href를 그대로 요청합니다. Neutron의 다음 링크에는 원래 filter가 들어 있으므로 실제 요청은 같습니다. agent, port, floating IP 목록은 Go도 이 링크를 따라갑니다. 다만 segment 목록의 native page는 `links.next`만 읽고, network IP availability 목록은 한 page만 읽으므로 이 둘은 첫 page에서 멈춥니다. 다음 page가 필요하면 `WithListQuery("marker", 마지막ID)`로 이어서 호출합니다.

Python delete 계열은 `ignore_missing=True`가 기본이라 404를 무시합니다. Go에서는 `Remove(ctx, resource.ID(id))`가 같은 기본값을 갖고, `resource.WithMissingError()`가 `ignore_missing=False`에 해당합니다. generated `Delete`를 직접 부르면 404가 오류로 돌아옵니다.

Python update는 바뀐 attribute가 없으면 요청을 보내지 않지만 Go `Update`는 빈 envelope라도 PUT을 보냅니다. Go option 구조체는 nil과 빈 core field를 생략하고 `WithUpdateField`로 core key를 덮을 수 없으므로, Python에서 `None`을 넘겨 JSON null을 보내는 동작은 재현하지 않습니다.

Neutron proxy의 `get_*`는 `fields` 인자를 받아 Body 속성 이름을 wire 이름으로 바꾼 뒤 `fields` query로 보냅니다. agent, segment, network IP availability package에는 sdkgen이 만든 `GetWithFields(ctx, id, fields...)`는 Python `_get_with_fields`처럼 Resource 상속 체인의 Body 속성 이름을 wire 이름으로 바꾼 뒤 `fields` query를 같은 키로 반복해서 보냅니다. 선언되지 않은 이름, 중복, 순서는 그대로 두고, 필드를 주지 않으면 query 없이 `Get`과 같은 GET을 보냅니다. 성공 상태는 Python 읽기처럼 200부터 399까지 받고, 일부 필드만 담긴 응답은 Gophercloud 모델로 decode되어 빠진 필드가 zero 값이 됩니다. 실패는 `resource.OperationError`로 감싼 native 오류로 돌아오므로 `gophercloud.ResponseCodeIs(err, 404)`로 Python의 `NotFoundException`을 구분합니다. 예를 들어 agent의 `is_alive`, `configuration`, `last_heartbeat_at`은 `alive`, `configurations`, `heartbeat_timestamp`로 나갑니다. network IP availability의 native decoder는 `total_ips`와 `used_ips`가 없으면 실패하므로, 선택 조회에서는 응답에 없는 두 값을 빈 문자열로 둡니다. 명시적인 null이나 잘못된 타입은 여전히 오류입니다.

## Agent와 DHCP 스케줄링

`agents()`는 `GET agents`입니다. Python QueryParameters의 `agent_type`, `availability_zone`, `binary`, `description`, `fields`, `host`, `topic`, `admin_state_up`, `alive`는 `agents.ListOpts`나 `WithListQuery`로 보냅니다. Python Agent의 `ha_state`와 `ha_chassis_priority`는 Go `Agent` 모델에 없지만, Neutron은 이 두 값을 router의 l3-agents 목록에서만 돌려주므로 `GET agents` 결과에는 차이가 생기지 않습니다.

`update_agent`는 `PUT agents/{id}`에 넘긴 attribute만 `agent` envelope로 보냅니다. Go `UpdateOpts`의 `Description`과 `AdminStateUp`은 포인터라서 빈 문자열과 false도 보낼 수 있고, 다른 key는 `WithUpdateField`로 붙입니다. `delete_agent`는 `Remove`가 404를 무시하며 같은 DELETE를 보냅니다.

`add_dhcp_agent_to_network`는 `POST agents/{id}/dhcp-networks`에 `{"network_id": id}`를 보내고 응답 본문을 돌려줍니다. Go `ScheduleDHCPNetwork`는 같은 요청을 보내고 오류만 돌려줍니다. `remove_dhcp_agent_from_network`는 `DELETE agents/{id}/dhcp-networks/{network_id}`인데 Python은 DELETE에도 같은 JSON 본문을 붙이고 Go는 본문 없이 보냅니다. 두 Python action은 `Proxy.request`의 기본값대로 HTTP 오류를 예외로 바꾸지 않지만 Go는 오류를 돌려줍니다.

`dhcp_agent_hosting_networks`는 `GET agents/{id}/dhcp-networks`이고 인자 없이 부르면 Go `ListDHCPNetworks`와 요청이 같습니다. 그러나 Go 호출은 query를 받지 않아 Python이 Network QueryParameters로 보내는 filter를 보낼 수 없고 `networks_links`도 따라가지 않으므로 unresolved입니다. `network_hosting_dhcp_agents`의 `GET networks/{id}/dhcp-agents`는 Gophercloud v2.15.0에 binding이 없습니다.

## Segment

`create_segment`는 `POST segments`, `update_segment`는 `PUT segments/{id}`에 넘긴 attribute를 `segment` envelope로 보냅니다. Go `CreateOpts`는 `network_id`와 `network_type`을 HTTP 전에 요구하고, 빈 문자열과 0인 `segmentation_id`는 생략합니다. Neutron은 생략한 값에 같은 기본값을 적용하므로 결과는 같습니다. `UpdateOpts`는 포인터라서 빈 설명과 `segmentation_id` 0을 보낼 수 있습니다. `delete_segment`는 `Remove`가 404를 무시합니다.

`segments()`는 `GET segments`이며 QueryParameters의 모든 key를 `ListOpts`나 `WithListQuery`로 보낼 수 있습니다. `ListOpts.SegmentationID`는 0을 생략하므로 0을 조건으로 쓰려면 `WithListQuery("segmentation_id", "0")`을 씁니다. `segmentation_id`가 null인 flat segment는 Go에서 0으로 decode되고 Python에서는 None입니다. 앞 절에서 말한 대로 Go 목록은 `segments_links`를 따라가지 않습니다.

`find_segment`는 `GET segments/{name_or_id}`가 400, 403, 404이면 `name`을 넣은 목록에서 ID나 이름이 같은 항목을 고릅니다. Go에서 `Find(ctx, resource.ID(x), resource.WithIgnoreMissing())` 다음에 `Find(ctx, resource.Name(x))`를 부르면 기본 인자의 요청과 같지만, Python이 두 단계 모두에 넘기는 `**query`를 실을 방법이 없어서 unresolved입니다.

## Extension과 network IP availability

`extensions()`는 `GET extensions`이고 Go `extensions.API.List`가 alias, 이름, 설명, `updated` 문자열을 같은 값으로 읽습니다. Python은 기본 `limit`, `marker` query를 보낼 수 있지만 Neutron의 extension 목록은 한 번에 돌아오므로 결과가 달라지지 않습니다. `find_extension`은 Go에 목록 fallback이 없고 `Get`이 404를 그대로 돌려주므로 unresolved입니다.

`network_ip_availabilities()`는 `GET network-ip-availabilities`이며 `network_id`, `network_name`, `ip_version`, `project_id`는 `ListOpts`로, `fields`, `sort_key`, `sort_dir`는 `WithListQuery`로 보냅니다. Go 모델은 `total_ips`와 `used_ips`를 큰 정수를 담을 수 있는 10진 문자열로 두고 subnet별 값을 typed 행으로 decode하며, `ip_availability_details` field는 없습니다. `find_network_ip_availability`는 이 package에 `Find`나 collection이 없어서 unresolved입니다.

## Port

`ports()`는 `GET ports`입니다. Python이 `is_admin_state_up`, `is_port_security_enabled`, `security_group_ids`, `tenant_id`를 wire 이름으로 바꾸는 부분은 Go에서 `ListOpts`의 `AdminStateUp`, `SecurityGroups`, `ProjectID`나 `WithListQuery`로 직접 지정합니다. `fixed_ips`와 `security_groups`는 두 쪽 모두 반복 query로 나갑니다. tag 조건은 Python이 목록 값을 반복 key로 보내고 Go `ListOpts`는 쉼표로 이은 한 값으로 보냅니다. Go `ports.Port` 모델에는 `binding:*`, `dns_*`, `extra_dhcp_opts`, `port_security_enabled`, `qos_*`, `resource_request`, `trunk_details` 같은 Python Body field가 없어서, 이 값들은 결과에 남지 않고 호출자가 지역에서 거를 수도 없습니다.

`get_subnet_ports(subnet)`는 query 없이 모든 port를 받은 뒤 `fixed_ips`에 그 subnet이 있는 port만 돌려줍니다. Go는 `All(ctx)`가 같은 요청을 보내고, 호출자가 `FixedIPs`의 `SubnetID`로 같은 행을 고릅니다. `create_ports`는 `{"ports": [...]}`를 한 번에 POST하는 bulk 생성인데 Go에는 한 개씩 만드는 `Create`만 있어서 unresolved입니다.

## Floating IP의 port 연결

`add_ip_to_port(port, ip)`는 FloatingIP Resource의 `port_id`를 바꾸고 commit하므로 `PUT floatingips/{id}`에 `{"floatingip": {"port_id": id}}`를 보냅니다. `remove_ip_from_port(ip)`는 `port_id`를 None으로 commit합니다. Go `Update`는 `PortID`에 port ID를 넣으면 같은 본문을 보내고, 빈 문자열을 넣으면 `null`로 바꿔 보냅니다. Python은 Resource에 쌓인 다른 변경 attribute도 함께 보내고 값이 바뀌지 않았으면 요청을 생략하지만, Go는 넘긴 field만 담아 항상 PUT을 보냅니다.

`find_available_ip()`는 query 없이 floating IP 목록을 돌면서 `port_id`가 비어 있는 첫 항목을 돌려주고 없으면 None입니다. Go `List`는 lazy iterator라서 첫 빈 `PortID`에서 `break`하면 다음 page를 요청하지 않습니다. `network.FloatingIPs.Available`은 network와 project로 거르고 새 주소를 할당할 수도 있는 다른 작업이므로 이 메서드의 대응이 아닙니다.

## BGP speaker gateway network와 trunk subport

`add_gateway_network_to_speaker`와 `remove_gateway_network_from_speaker`는 `PUT bgp-speakers/{id}/add_gateway_network`와 `remove_gateway_network`에 `{"network_id": id}`를 보냅니다. Go `AddGatewayNetwork`와 `RemoveGatewayNetwork`가 같은 요청을 보내며, 추가 응답은 `AddGatewayNetworkOpts`로 decode합니다. Python은 이 두 action에서 HTTP 오류를 예외로 바꾸지 않습니다. `get_advertised_routes_of_speaker`는 `GET bgp-speakers/{id}/get_advertised_routes`이고, Python은 응답 dictionary 전체를 돌려주며 Go는 destination과 next_hop 행을 하나씩 돌려줍니다.

`get_trunk_subports`는 `GET trunks/{id}/get_subports`이고 Go `GetSubports`가 `sub_ports`의 각 항목을 `Subport`로 decode합니다.

## Wait

`wait_for_status`는 Resource를 주기적으로 다시 읽어 지정한 attribute가 대소문자 구분 없이 목표와 같아질 때까지 기다리고, 기본 `failures=['ERROR']`, `interval=2`, `wait=None`(무제한)을 씁니다. 이미 목표 상태인 Resource를 넘기면 요청 없이 바로 돌려줍니다. Go에서는 `ports.API.WaitFor`처럼 collection이 있는 resource만 같은 polling을 할 수 있고, 기본값이 5분 timeout과 오류 접두사·접미사 판정이므로 `resource.WithUnlimitedWait()`와 `resource.WithFailureStates("ERROR")`로 Python 기본값에 맞춥니다. Go는 항상 첫 GET을 보냅니다.

`wait_for_delete`는 Neutron proxy에서 `wait=120`이 기본이고 404가 오면 원래 Resource를 돌려줍니다. Go `WaitForDeletion`은 기본 5분이므로 `resource.WithTimeout(120*time.Second)`로 맞추며 오류만 돌려줍니다. Python은 local IP, segment range, port binding, auto-allocated topology처럼 Go collection이 없는 Resource도 기다릴 수 있으므로 두 wait 메서드는 unresolved입니다.

## Unresolved 메서드

local IP와 local IP association 메서드는 SDK 소유 [local IP](extensions/localips/README.md)와 [local IP association](extensions/localipassociations/README.md) package가 추가되어 대응으로 바뀌었습니다.

다음 메서드는 Go가 같은 요청이나 기본 동작을 만들지 못합니다.

- `activate_port_binding`, `create_port_binding`, `delete_port_binding`, `port_bindings`: `ports/{port_id}/bindings` 경로와 `activate` action에 대한 binding이 Gophercloud v2.15.0에 없습니다.
- `availability_zones`, `service_providers`: `GET availability_zones`와 `GET service-providers` binding이 없습니다.
- `get_auto_allocated_topology`, `delete_auto_allocated_topology`, `validate_auto_allocated_topology`: `auto-allocated-topology/{project_id}` 경로와 `fields=dry-run` 검증 요청이 없습니다. Python은 project를 생략하면 현재 session의 project ID를 씁니다.
- `create_local_ip`, `delete_local_ip`, `find_local_ip`, `get_local_ip`, `local_ips`, `update_local_ip`, `create_local_ip_association`, `delete_local_ip_association`, `find_local_ip_association`, `get_local_ip_association`, `local_ip_associations`: `local_ips`와 `port_associations` 경로가 없습니다.
- `create_network_segment_range`, `delete_network_segment_range`, `find_network_segment_range`, `get_network_segment_range`, `network_segment_ranges`, `update_network_segment_range`: `network_segment_ranges` 경로가 없습니다.
- `network_hosting_dhcp_agents`, `onboard_network_subnets`: `networks/{id}/dhcp-agents`와 `subnetpools/{id}/onboard_network_subnets` binding이 없습니다.
- `create_ports`: bulk port 생성 호출이 없습니다.
- `dhcp_agent_hosting_networks`: Go 호출이 query를 받지 않고 `networks_links`를 따라가지 않습니다.
- `find_segment`, `find_extension`, `find_network_ip_availability`: Python의 ID 조회 뒤 목록 fallback을 `**query`와 함께 재현하는 helper가 없습니다.
- `wait_for_status`, `wait_for_delete`: Go collection이 없는 Neutron Resource에는 waiter가 없습니다.
- `agents`, `extensions`, `network_ip_availabilities`, `ports`, `segments`, `get_subnet_ports`, `find_available_ip`: Python이 Body 속성으로 행을 로컬에서 거르거나 첫 행에서 멈추는 동작, 또는 `segments_links`·`*_links` 페이지 순회를 SDK가 대신하지 않습니다. 호출자가 결과 행을 직접 걸러야만 재현되므로 다른 서비스와 같은 기준으로 미해결로 둡니다.

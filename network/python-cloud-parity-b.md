# Python cloud network helper 대응 (b)

이 문서는 openstacksdk cloud 계층의 network helper 47개를 Go SDK의 공개 API와 비교합니다. 대상은 [`openstack/cloud/_network.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py)의 36개와 [`openstack/cloud/_network_common.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py)의 11개입니다. 같은 파일의 나머지 helper는 별도 문서에서 다룹니다.

판정 기준은 공개 Go API 자체가 Python helper의 요청 순서와 결과를 재현하는지입니다. 호출자가 행을 직접 거르거나, 페이지를 손으로 따라가거나, 404를 직접 성공으로 바꾸거나, 조건에 따라 요청을 건너뛰어야 한다면 대응으로 보지 않고 미해결로 기록합니다. 계약 테스트는 [`python_parity_cloud_b_test.go`](python_parity_cloud_b_test.go)에 있습니다.

## 판정 표

| Python 메서드 | Go 호출 | 판정 |
| --- | --- | --- |
| [get_subnet_by_id](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L435) | `subnets.API.GetWithFields(ctx, id)` | 대응 |
| [get_subnetpool](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L478) | `subnetpools.API.FindIdentity(ctx, nameOrID)` | 대응 |
| [list_routers](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L168) | `routers.API.All(ctx, resource.WithFilters(filters))` | 대응 |
| [search_routers](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L77) | `routers.API.All(ctx, resource.WithFilter("name", x), resource.WithFilters(filters))` | 대응 |
| [create_security_group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1931) | `secgroups.API.CreateWithAttributes(ctx, WithAttribute(...))` | 대응 |
| [get_security_group_by_id](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1906) | `secgroups.API.GetWithFields(ctx, id)` | 대응 |
| [list_security_groups](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1845) | `secgroups.API.All(ctx, resource.WithFilters(filters))` | 대응 |
| [delete_security_group_rule](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L2219) | `secrules.API.DeleteWithOptions(ctx, id, secrules.WithDeleteIgnoreMissing(false))` | 대응 |
| [list_networks](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L151) | 없음 (`networks` 목록 자체가 미해결) | 미해결 |
| [search_networks](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L53) | 없음 | 미해결 |
| [list_subnets](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L185) | 없음 (`subnets` 목록 자체가 미해결) | 미해결 |
| [search_subnets](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L101) | 없음 | 미해결 |
| [list_ports](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L202) | 없음 | 미해결 |
| [search_ports](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L125) | 없음 | 미해결 |
| [list_router_interfaces](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2205) | 없음 | 미해결 |
| [remove_router_interface](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2174) | `routers.API.RemoveInterfaceRecord` (일부) | 미해결 |
| [update_router](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2309) | 없음 | 미해결 |
| [update_subnet](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2591) | `subnets.API.FindRecord` 후 `UpdateWithAttributes` (일부) | 미해결 |
| [update_port](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2776) | `ports.API.FindRecord` 후 `UpdateWithAttributes` (일부) | 미해결 |
| [list_qos_policies](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L333) | 없음 | 미해결 |
| [search_qos_policies](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L253) | 없음 | 미해결 |
| [update_qos_policy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1468) | 없음 | 미해결 |
| [list_qos_rule_types](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L281) | 없음 | 미해결 |
| [list_qos_bandwidth_limit_rules](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1557) | 없음 | 미해결 |
| [search_qos_bandwidth_limit_rules](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1535) | 없음 | 미해결 |
| [update_qos_bandwidth_limit_rule](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1667) | 없음 | 미해결 |
| [list_qos_dscp_marking_rules](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1786) | 없음 | 미해결 |
| [search_qos_dscp_marking_rules](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1764) | 없음 | 미해결 |
| [update_qos_dscp_marking_rule](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1877) | 없음 | 미해결 |
| [list_qos_minimum_bandwidth_rules](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1983) | 없음 | 미해결 |
| [search_qos_minimum_bandwidth_rules](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1959) | 없음 | 미해결 |
| [update_qos_minimum_bandwidth_rule](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2073) | 없음 | 미해결 |
| [list_firewall_groups](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1350) | `fwaas_v2/groups.API.All` (기본 인자만) | 미해결 |
| [list_firewall_policies](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1079) | `fwaas_v2/policies.API.All` (기본 인자만) | 미해결 |
| [list_firewall_rules](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L874) | `fwaas_v2/rules.API.All` (기본 인자만) | 미해결 |
| [update_firewall_group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1372) | 없음 | 미해결 |
| [update_firewall_policy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1110) | 없음 | 미해결 |
| [update_firewall_rule](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L913) | 없음 | 미해결 |
| [insert_rule_into_policy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1150) | 없음 | 미해결 |
| [remove_rule_from_policy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1211) | 없음 | 미해결 |
| [search_security_groups](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1834) | 없음 | 미해결 |
| [get_security_group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1878) | 없음 | 미해결 |
| [update_security_group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L2025) | 없음 | 미해결 |
| [delete_security_group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1983) | 없음 | 미해결 |
| [create_security_group_rule](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L2067) | 없음 | 미해결 |
| [add_auto_ip](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1394) | 없음 | 미해결 |
| [detach_ip_from_server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1214) | 없음 | 미해결 |

대응 8개, 미해결 39개입니다.

## Subnet과 subnet pool

`get_subnet_by_id`는 proxy `get_subnet`을 그대로 부르므로 `GET /v2.0/subnets/{id}` 한 번을 보냅니다. Go의 `subnets.API.GetWithFields`도 같은 요청을 보내고, 404는 Python의 `NotFoundException`처럼 오류로 남습니다. docstring에는 없으면 None을 돌려준다고 적혀 있지만 실제 코드는 예외를 던지므로 Go 동작과 같습니다. Python은 변경 가능한 `Subnet` Resource를 돌려주고 Go는 Gophercloud의 typed 모델을 돌려주므로, 응답의 알 수 없는 필드는 Go 모델에 남지 않습니다.

`get_subnetpool`은 `find_subnet_pool(name_or_id, ignore_missing=True)`입니다. 먼저 ID로 member GET을 보내고, 404이면 `?name=` 목록으로 다시 찾습니다. Go `subnetpools.API.FindIdentity`의 기본값도 같은 두 단계를 거치고, 찾지 못하면 `nil, nil`을 돌려줍니다. Go의 native SubnetPool은 세 prefix 길이 필드가 있어야 decode되므로, fields 투영으로 이 필드를 뺀 응답은 Python과 달리 실패합니다.

## Router

`list_routers`와 `search_routers`는 proxy `routers`에 filter를 그대로 넘깁니다. `search_routers`는 `name_or_id`를 `name` 질의로만 보내므로 ID로는 찾지 못하며, Go에서도 `resource.WithFilter("name", x)`가 서버 질의 하나로만 쓰입니다. 두 Go 호출 모두 `routers_links`를 끝까지 따라갑니다. 차이는 다음과 같습니다.

- `list_routers`는 `has_service('network')`가 거짓이면 빈 목록을 돌려줍니다. Go에서는 network 서비스가 없으면 클라이언트를 만드는 단계에서 오류가 납니다.
- Python 질의의 bool 값은 Requests가 `True`로 쓰지만 Go semantic filter는 `true`로 씁니다.
- 반환값은 Python Router Resource 대신 native `routers.Router`입니다.

`remove_router_interface`는 subnet_id와 port_id가 모두 비어 있으면 HTTP 전에 `ValueError`를 던집니다. 기본 인자로 부르면 바로 이 경우가 됩니다. Go `RemoveInterfaceRecord`는 이를 막지 않고 `{"subnet_id": null}`을 보내므로 미해결입니다. 하나 이상을 주었을 때의 본문(port 우선)은 테스트로 확인했습니다.

`update_router`는 `routes`를 줄 때 `GET /v2.0/extensions`로 `extraroute`를 확인하고, 없으면 경고만 남기고 routes를 버립니다. 이 확장 검사와 바꿀 내용이 없을 때 HTTP 없이 None을 돌려주는 동작을 수행하는 Go helper가 없습니다. `list_router_interfaces`는 `device_id`로 port를 나열한 뒤 `device_owner`로 내부/게이트웨이 port를 거르는데, 이 분류도 Go helper가 없습니다.

## Port와 subnet 갱신

`update_port`와 `update_subnet`은 `find_port`/`find_subnet`(ignore_missing=True)으로 대상을 찾고, 없으면 `SDKException`, 있으면 PUT을 보냅니다. Go에서는 `FindRecord(..., WithRecordFindIgnoreMissing(false))`와 `UpdateWithAttributes`를 이어 부르면 속성이 있을 때 같은 요청(GET, 이름이면 `?name=` 목록, PUT)이 나갑니다. `disable_gateway_ip`는 `WithAttribute("gateway_ip", nil)`로 null을 보낼 수 있습니다. 다만 기본 인자에서 차이가 있습니다. `update_port(name_or_id)`는 변경 내용이 없으면 조회만 하고 PUT을 보내지 않으며, `update_subnet(name_or_id)`는 HTTP 없이 None을 돌려줍니다. Go에서 이를 맞추려면 호출자가 PUT을 건너뛰어야 하므로 미해결로 둡니다.

## Network, subnet, port 목록

`list_networks`, `search_networks`, `list_subnets`, `search_subnets`는 proxy `networks`/`subnets` 목록에 그대로 위임합니다. 그 proxy 항목이 이미 미해결(Resource descriptor 강제 변환, Link header 연속, 반환 모델 차이)이므로 cloud helper도 미해결입니다. native Network 모델에는 `router:external`, provider, MTU, DNS, port security 같은 확장 속성이 없습니다.

`list_ports`의 native Port 모델에는 `binding:*`, `dns_*`, `port_security_enabled`, `qos_policy_id`, `extra_dhcp_opts`가 없고, 목록에서 전체 JSON을 돌려주는 port record API도 없습니다. `search_ports`는 여기에 더해 `_utils._filter_list`의 이름/ID glob 선택과 dict 또는 JMESPath 로컬 필터를 수행하므로 미해결입니다.

## QoS

QoS helper는 모두 먼저 `GET /v2.0/extensions`로 `qos` 별칭을 확인하고, 없으면 `OpenStackCloudUnavailableExtension`을 던집니다. 규칙 helper는 이어서 `find_qos_policy`로 정책을 찾고 없으면 `NotFoundException`을 던지며, update는 `qos-default`, `qos-bw-limit-direction` 확장 검사와 현재 규칙 조회까지 거칩니다. search 계열은 `_filter_list` 로컬 선택을 추가합니다. 이 순서를 수행하는 Go helper가 없으므로 13개 모두 미해결입니다.

## FWaaS

`list_firewall_*`은 기본 인자에서 `GET /v2.0/fwaas/firewall_*` 목록을 끝까지 읽고, Go `fwaas_v2/*.API.All`도 같은 요청을 보내며 Python이 선언한 필드를 모두 decode합니다. 그러나 filters에 넣은 Body 전용 키(group의 `admin_state_up`, policy의 `audited`, 공통 `id`)는 Python이 로컬로 거르는데, FWaaS collection에는 `WithFilters`/`WithBodyFilter` 분류가 없습니다. 그래서 미해결이며, 기본 인자 동작은 계약으로 남겼습니다.

`update_firewall_*`, `insert_rule_into_policy`, `remove_rule_from_policy`는 `find_firewall_*`로 이름이나 ID를 풀고, policy/port/rule 이름을 ID로 바꾸며, 이미 연결되었거나 없는 rule이면 요청 없이 현재 policy를 돌려줍니다. 이 조회와 단락 처리를 수행하는 Go helper가 없습니다.

## Security group

security group helper는 `secgroup_source` 설정을 따릅니다. 기본값은 `neutron`이고, network 서비스가 있으면 Neutron 경로를 씁니다. Go 대응은 이 기본 경로만 다룹니다. `nova` 설정이나 network 서비스가 없는 경우의 `/os-security-groups` 경로와 정규화, `secgroup_source`가 비었을 때의 `OpenStackCloudUnavailableFeature`는 Go에서 재현하지 않습니다.

- `create_security_group`은 name과 description을 항상 보내고, stateful과 project_id는 주어졌을 때만 넣습니다. project_id는 본문에서 `tenant_id` 키가 됩니다. Go에서는 `WithAttribute("tenant_id", id)`를 씁니다.
- `get_security_group_by_id`는 member GET이며 404는 오류입니다.
- `list_security_groups`는 filters를 서버로 보내고 모든 페이지를 읽습니다.
- `delete_security_group_rule`은 `ignore_missing=False`로 지웁니다. Go `DeleteWithOptions`의 기본값은 404를 무시하므로 `WithDeleteIgnoreMissing(false)`를 꼭 지정해야 합니다. Python은 True를 돌려주고 Go는 nil 오류를 돌려줍니다.

`get_security_group`, `search_security_groups`, 그리고 이를 먼저 부르는 `update_security_group`, `delete_security_group`, `create_security_group_rule`은 질의 없이 전체 목록을 받은 뒤 이름/ID 일치와 glob으로 하나를 고르고, 여러 개면 `SDKException`을 던집니다. Go의 `FindRecord`/`FindIdentity`는 member GET 후 `?name=` 목록을 보내므로 요청이 다르고, 전체 목록 위의 로컬 선택 helper도 없습니다.

## Floating IP

`add_auto_ip`은 재사용 가능한 floating IP를 찾거나 새로 만든 뒤 연결하고, 서버를 다시 읽어 `interface_ip` 문자열을 돌려줍니다. 시간 초과 때 새로 만든 IP를 지우는 처리도 있습니다. `network.FloatingIPs.Ensure`가 Neutron 재사용과 연결 부분을 다루지만 무조건 연결, 서버 갱신, 문자열 결과, Nova 경로까지 재현하지 않습니다.

`detach_ip_from_server`는 cloud `get_floating_ip`로 전체 목록에서 IP를 고르고, port가 없으면 False, 있으면 `port_id: null` PUT 후 True를 돌려줍니다. Neutron에서 404가 나면 Nova `removeFloatingIp` action으로 넘어갑니다. 이 순서와 대체 경로를 수행하는 Go helper가 없습니다.

## 미해결 목록과 부족한 부분

- list_networks, search_networks: proxy `networks` 목록이 미해결이고 native Network 모델에 확장 속성이 없습니다.
- list_subnets, search_subnets: proxy `subnets` 목록이 미해결입니다.
- list_ports, search_ports: native Port 모델에 확장 속성이 없고 port 목록 record API가 없습니다. search_ports는 로컬 `_filter_list`도 필요합니다.
- list_router_interfaces: device_owner 분류 helper가 없습니다.
- remove_router_interface: 두 인자가 모두 비었을 때 HTTP 전에 거부하지 않습니다.
- update_router: extraroute 확장 검사와 변경 없음 처리 helper가 없습니다.
- update_port, update_subnet: 변경 내용이 없을 때 PUT을 건너뛰는 helper가 없습니다.
- QoS 13개: qos 확장 검사, 정책 조회 실패 처리, 추가 확장 검사, 로컬 검색 helper가 없습니다.
- list_firewall_groups, list_firewall_policies, list_firewall_rules: Body 전용 filter 키의 로컬 필터가 없습니다.
- update_firewall_group, update_firewall_policy, update_firewall_rule, insert_rule_into_policy, remove_rule_from_policy: 이름/ID 조회, 참조 ID 변환, 단락 처리 helper가 없습니다.
- get_security_group, search_security_groups, update_security_group, delete_security_group, create_security_group_rule: 전체 목록 위의 이름/ID glob 선택 helper가 없습니다.
- add_auto_ip, detach_ip_from_server: 연결과 서버 갱신, Nova 대체 경로를 묶은 helper가 없습니다.

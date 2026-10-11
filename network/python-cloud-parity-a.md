# Python cloud network 헬퍼 대응 (A 묶음)

기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [`openstack/cloud/_network.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py)와 Gophercloud v2.15.0입니다. 이 문서는 `NetworkCloudMixin` 메서드 36개를 다룹니다. cloud 헬퍼는 network proxy 호출 앞뒤로 확장 확인, 이름 검색, 기본값 주입 같은 논리를 더하므로, Go의 공개 API가 그 요청 순서와 결과를 스스로 만들 때만 대응으로 판정했습니다.

판정 규칙은 다음과 같습니다. 호출자가 앞 호출에서 받은 ID를 다음 호출에 넘기고, `nil` 결과를 `false`로 읽는 정도의 연결은 대응으로 인정합니다. 반대로 확장 목록을 읽어 동작을 결정하거나, 인수를 검증하고 기본값을 채우거나, 404를 직접 성공으로 바꿔야 한다면 Go SDK가 그 일을 하지 않는 것이므로 `unresolved`입니다.

| Python 메서드 | Go 호출 | 판정 |
|---|---|---|
| [`get_network`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L351) | `networks.API.FindRecord` + `WithRecordFindIgnoreMissing(true)`, `WithRecordFindQuery` | go_mapping |
| [`get_network_by_id`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L377) | `networks.API.Get` | go_mapping |
| [`get_router`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L386) | `routers.API.FindRecord` + 같은 옵션 | go_mapping |
| [`get_subnet`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L413) | `subnets.API.FindRecord` + 같은 옵션 | go_mapping |
| [`get_port`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L444) | `ports.API.FindRecord` + 같은 옵션 | go_mapping |
| [`get_port_by_id`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L470) | `ports.API.Get` | go_mapping |
| [`get_network_extensions`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L734) | `extensions.API.List` | go_mapping |
| [`add_router_interface`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2151) | `routers.API.AddInterfaceRecord` + `WithRouterInterfaceSubnet`/`WithRouterInterfacePort` | go_mapping |
| [`create_port`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2690) | `ports.API.CreateWithAttributes` + `WithAttribute` | go_mapping |
| [`delete_port`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2838) | `ports.API.FindRecord` 후 `ports.API.DeleteWithOptions` | go_mapping |
| [`delete_router`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2379) | `routers.API.FindRecord` 후 `routers.API.DeleteWithOptions` | go_mapping |
| [`delete_subnet`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2569) | `subnets.API.FindRecord` 후 `subnets.API.DeleteWithOptions` | go_mapping |
| [`create_firewall_policy`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L984) | 규칙마다 `fwaas_v2/rules.API.Find(ID)` 후 `Find(Name)`, 그다음 `fwaas_v2/policies.API.Create` | go_mapping |
| [`create_firewall_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1269) | `fwaas_v2/policies.API.Find` 연결, `ports.API.FindRecord`, `fwaas_v2/groups.API.Create` | go_mapping |
| [`create_firewall_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L775) | `fwaas_v2/rules.API.Create` (protocol·action 필수) | unresolved |
| [`get_firewall_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L846) | `fwaas_v2/rules.API.Find` (filters 불가) | unresolved |
| [`get_firewall_policy`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1051) | `fwaas_v2/policies.API.Find` (filters 불가) | unresolved |
| [`get_firewall_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1331) | `fwaas_v2/groups.API.Find` (filters 불가) | unresolved |
| [`delete_firewall_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L804) | `Find` 후 `API.Delete` (404를 `False`로 바꾸는 API 없음) | unresolved |
| [`delete_firewall_policy`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1010) | 위와 같음 | unresolved |
| [`delete_firewall_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1299) | 위와 같음 | unresolved |
| [`create_router`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2250) | `routers.API.CreateWithAttributes` (기본값·AZ 확인 없음) | unresolved |
| [`create_subnet`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2401) | `networks.API.FindRecord`, `subnets.API.CreateWithAttributes` (검증·기본값 없음) | unresolved |
| [`get_qos_policy`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L221) | `extensions.API.List`, `qos/policies.API.FindRecord` | unresolved |
| [`create_qos_policy`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1435) | `qos/policies.API.CreateWithAttributes` | unresolved |
| [`delete_qos_policy`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1512) | `qos/policies.API.FindRecord`, `DeleteWithOptions` | unresolved |
| [`get_qos_rule_type_details`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L300) | `qos/ruletypes.API.GetRecord` | unresolved |
| [`get_qos_bandwidth_limit_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1594) | `qos/rules.API.GetBandwidthLimitRecord` | unresolved |
| [`create_qos_bandwidth_limit_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1621) | `qos/rules.API.CreateBandwidthLimitRule` | unresolved |
| [`delete_qos_bandwidth_limit_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1723) | `qos/rules.API.DeleteBandwidthLimitRule` | unresolved |
| [`get_qos_dscp_marking_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1819) | `qos/rules.API.GetDSCPMarkingRecord` | unresolved |
| [`create_qos_dscp_marking_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1844) | `qos/rules.API.CreateDSCPMarkingRule` | unresolved |
| [`delete_qos_dscp_marking_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1919) | `qos/rules.API.DeleteDSCPMarkingRule` | unresolved |
| [`get_qos_minimum_bandwidth_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2018) | `qos/rules.API.GetMinimumBandwidthRecord` | unresolved |
| [`create_qos_minimum_bandwidth_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2041) | `qos/rules.API.CreateMinimumBandwidthRule` | unresolved |
| [`delete_qos_minimum_bandwidth_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2115) | `qos/rules.API.DeleteMinimumBandwidthRule` | unresolved |

표의 패키지 경로는 모두 `github.com/JSYoo5B/go-openstacksdk/network/v2/` 아래입니다. `routers`는 `extensions/layer3/routers`, `qos/...`와 `fwaas_v2/...`는 `extensions/` 아래에 있습니다. 회귀 테스트는 [`python_cloud_parity_a_test.go`](python_cloud_parity_a_test.go)에 있습니다.

## Network·Router·Subnet·Port 조회

`get_network`, `get_router`, `get_subnet`, `get_port`는 `find_*`를 `ignore_missing=True`로 호출하고 `filters`를 그대로 넘깁니다. Python `Resource.find`는 먼저 `GET /{collection}/{name_or_id}`에 filters를 query로 붙여 보내고, 400·403·404이면 `name=`과 filters를 붙인 목록을 읽어 ID나 이름이 같은 항목을 고릅니다. Go의 `FindRecord`가 이 순서를 그대로 따르며 `WithRecordFindQuery`로 filters를 넘기면 두 요청 모두에 붙습니다. 찾지 못하면 Python과 같이 `nil, nil`을 돌려줍니다. 테스트는 이름 fallback, ID 직접 조회, 부재의 세 경우를 네 리소스에 대해 확인합니다.

Python docstring은 filters에 jmespath 문자열도 받는다고 적지만 실제 코드는 `**filters`로 펼치므로 dict만 동작합니다. Go도 키와 JSON 값 쌍만 받습니다. 결과는 Python Resource 객체 대신 `Resource`·`Wire` 두 속성 뷰와 원래 응답을 담은 record입니다. Router의 `FindRecord`는 router 전용 record 타입을 돌려주지만 내용 구성은 같습니다.

`get_network_by_id`와 `get_port_by_id`는 proxy `get_network`·`get_port`를 그대로 부릅니다. docstring은 없으면 None이라고 하지만 실제로는 404 예외가 납니다. Go `networks.API.Get`·`ports.API.Get`도 같은 GET을 보내고 404를 오류로 반환합니다. 결과는 Gophercloud 고유 구조체이므로 Python Resource에만 있는 일부 확장 속성은 담기지 않습니다.

`get_network_extensions`는 `GET /extensions`를 한 번 읽어 alias 집합을 돌려줍니다. Go `extensions.API.List`는 같은 요청을 보내며 각 항목의 `Alias`를 모으면 같은 정보가 됩니다. Go는 집합 대신 iterator를 반환하고, Python cloud 계층처럼 결과를 따로 캐시하지 않는 점은 같습니다.

## Port·Router·Subnet 변경

`create_port(network_id, **kwargs)`는 `network_id`를 kwargs에 넣어 proxy `create_port`에 넘길 뿐입니다. Go `ports.API.CreateWithAttributes`에 `WithAttribute("network_id", id)`와 나머지 키를 주면 같은 `POST /ports` 본문이 나갑니다. `binding:vnic_type`처럼 콜론이 있는 wire 이름과 `admin_state_up` 같은 서버 이름을 그대로 쓸 수 있습니다. Python의 `valid_kwargs` 이름 검사는 Go에 없으므로 허용 목록 밖의 키도 그대로 전송됩니다.

`add_router_interface(router, subnet_id, port_id)`는 proxy `add_interface_to_router(router, subnet=..., port=...)`로 이어집니다. truthy인 port가 있으면 `{"port_id": ...}`, 아니면 `{"subnet_id": ...}`를 보내며 둘 다 없으면 `{"subnet_id": null}`이 나갑니다. Python docstring은 둘을 함께 주면 오류라고 적지만 실제로는 port가 우선합니다. Go `AddInterfaceRecord`가 같은 규칙으로 `PUT /routers/{id}/add_router_interface`를 보내고 응답 JSON 전체를 `Value`로 돌려줍니다. Python은 Router 객체도 받지만 Go는 router ID 문자열만 받습니다.

`delete_port`, `delete_router`, `delete_subnet`은 `find_*`를 `ignore_missing=True`로 호출하고, 없으면 DELETE 없이 `False`를 돌려줍니다. 찾으면 그 ID로 proxy 삭제를 부르는데 기본값 `ignore_missing=True` 때문에 그 사이 다른 쪽이 지워 404가 나도 `True`입니다. Go에서는 `FindRecord`가 `nil`인지로 같은 bool을 얻고, 받은 ID를 `DeleteWithOptions`에 넘기면 기본으로 404를 무시합니다. 테스트는 204, 지연 404, 409, 부재를 세 리소스에 대해 확인합니다. Python은 bool을 직접 반환하지만 Go 호출자는 `FindRecord` 결과로 bool을 판단합니다. `if_revision` 조건은 cloud 헬퍼가 쓰지 않으므로 Go도 `WithDeleteRevision` 없이 호출합니다.

## Firewall (FWaaS v2)

`create_firewall_policy`는 `firewall_rules`가 있으면 각 항목을 `find_firewall_rule(ignore_missing=False)`로 ID로 바꾼 뒤 한 번 POST합니다. Go에서는 규칙마다 `rules.API.Find(ctx, resource.ID(x), resource.WithIgnoreMissing())`가 `nil`일 때 `Find(ctx, resource.Name(x))`를 이어 부르면 같은 GET과 `?name=` 목록 요청이 나가고, 끝까지 없으면 `resource.ErrNotFound`가 POST 전에 반환됩니다. 그 ID 목록을 `policies.CreateOpts.FirewallRules`에 넣습니다.

`create_firewall_group`은 `ingress_firewall_policy`·`egress_firewall_policy` 이름을 같은 방식으로 `find_firewall_policy(ignore_missing=False)`해서 `*_firewall_policy_id`로 바꾸고, `ports`의 각 항목은 `get_port`로 찾아 없으면 NotFound를 냅니다. Go에서는 policy에 위의 `Find` 연결을, port에 `ports.API.FindRecord(..., WithRecordFindIgnoreMissing(false))`를 쓰고 결과 ID를 `groups.CreateOpts`의 해당 필드에 넣습니다.

두 create 헬퍼에는 다음 차이가 있습니다. Python `Resource.find`는 ID GET이 400·403일 때도 이름 목록으로 넘어가지만 일반 `Find(resource.ID)`는 404만 부재로 봅니다. 목록에서 Python은 ID나 이름이 같은 항목을 고르지만 Go `Find(resource.Name)`은 서버가 이름으로 걸러 준 결과를 사용합니다. Gophercloud CreateOpts가 `omitempty`이므로 빈 문자열 이름, 빈 `firewall_rules`·`ports` 목록, policy 인수 `None`으로 만드는 `*_policy_id: null`은 본문에서 빠집니다. 생성 시점에는 서버 결과가 같지만 wire 본문은 다릅니다.

`create_firewall_rule()`은 kwargs만 넘기므로 인수가 없으면 `{"firewall_rule": {}}`가 나가고 Neutron이 action·protocol 기본값을 정합니다. Go `rules.API.Create`는 Gophercloud가 `protocol`과 `action`을 필수로 선언해 HTTP 전에 실패합니다. 핵심 필드는 `WithCreateField`로도 덮어쓸 수 없어서 이 본문을 보낼 방법이 없습니다.

`get_firewall_*`와 `delete_firewall_*`는 `filters`를 `find_firewall_*`에 넘깁니다. FWaaS 패키지에는 `FindRecord`나 `FindIdentity`가 없고 일반 `Find`는 query를 받지 않아서 filters를 재현할 수 없습니다. 삭제 헬퍼는 proxy 삭제를 `ignore_missing=False`로 부르고 404를 잡아 `False`로 바꾸는데, `API.Delete`는 404를 오류로 돌려주고 `Remove`나 `resource.IgnoreMissing`은 404를 성공으로 숨기므로 Python의 `False`를 SDK API가 만들지 않습니다.

## Router·Subnet 생성

`create_router`는 `admin_state_up=True`를 항상 넣고, truthy인 name과 None이 아닌 project_id를 더하며, `ext_gateway_net_id`·`enable_snat`·`ext_fixed_ips`를 `external_gateway_info`로 묶습니다. `availability_zone_hints`가 있으면 list인지 검사하고 `router_availability_zone` 확장이 없으면 예외를 냅니다. Go `routers.API.CreateWithAttributes`는 준 속성만 보내므로 인수 없이 부르면 `{"router": {}}`가 나갑니다. 호출자가 같은 본문을 직접 만들 수는 있지만 기본값, 묶음 변환, 확장 확인을 SDK가 하지 않습니다.

`create_subnet`은 `get_network`로 네트워크를 찾고(tenant_id가 있으면 filter로 사용), cidr와 subnetpool 조합을 검사하며, `ip_version=4`·`enable_dhcp=False`를 항상 넣습니다. 그 밖에도 문자열 ip_version 변환, `disable_gateway_ip`의 `gateway_ip: null` 변환, `subnetpool_name_or_id` 검색을 수행합니다. Go `subnets.API.CreateWithAttributes`는 이런 검증과 기본값 없이 준 속성만 보냅니다.

## QoS

QoS 헬퍼 13개는 모두 호출할 때마다 `GET /extensions`로 `qos` 확장이 있는지 확인하고 없으면 `OpenStackCloudUnavailableExtension`을 냅니다. `get_qos_rule_type_details`는 `qos-rule-type-details`도 확인하고, `create_qos_policy`는 `default` 인수가 있을 때 `qos-default`가 있어야 `is_default`로 보내며, `create_qos_bandwidth_limit_rule`은 `qos-bw-limit-direction`이 없으면 `direction`을 버립니다. 이 확인과 결정은 SDK API에 없고 호출자가 `extensions.API.List` 결과를 직접 검사해야 하므로 모두 `unresolved`입니다.

확장 확인 다음 요청은 Go에서 보낼 수 있고 테스트가 그 요청을 확인합니다. 정책은 `qos/policies.API.FindRecord`, 규칙 조회는 `GetBandwidthLimitRecord`·`GetDSCPMarkingRecord`·`GetMinimumBandwidthRecord`, 생성은 `Create*Rule`, rule type은 `ruletypes.API.GetRecord`가 맡습니다. 다만 규칙 헬퍼는 정책을 `ignore_missing=True`로 찾고 없으면 NotFound를 내지만, minimum bandwidth 계열은 `ignore_missing=False`로 찾는 차이가 있습니다. 규칙 삭제 헬퍼는 404를 잡아 `False`를 돌려주는데 `Delete*Rule`은 404를 오류로 반환하므로 이 부분도 SDK가 재현하지 않습니다. `delete_qos_policy`는 확장 확인 뒤 `FindRecord`와 `DeleteWithOptions`로 같은 요청을 보낼 수 있습니다.

## 미해결 목록

- `create_firewall_rule`: Gophercloud `rules.CreateOpts`가 protocol·action을 필수로 요구해 인수 없는 Python 본문을 보낼 수 없습니다.
- `get_firewall_rule`, `get_firewall_policy`, `get_firewall_group`: FWaaS 검색이 filters query와 400·403 fallback을 지원하지 않습니다.
- `delete_firewall_rule`, `delete_firewall_policy`, `delete_firewall_group`: 위 검색 제약에 더해, 삭제 404를 `False`로 바꾸는 SDK API가 없습니다.
- `create_router`: `admin_state_up=True` 기본값, gateway 인수 묶음, AZ hint 검사와 `router_availability_zone` 확장 확인을 수행하는 SDK 헬퍼가 없습니다.
- `create_subnet`: 네트워크·subnetpool 검색, cidr 조합 검증, `ip_version`·`enable_dhcp` 기본값과 gateway 변환을 수행하는 SDK 헬퍼가 없습니다.
- QoS 13개: 호출마다 `GET /extensions`로 `qos` 등 확장을 확인하는 SDK 헬퍼가 없습니다. 규칙 삭제 3개는 404를 `False`로 바꾸는 API도 없습니다.

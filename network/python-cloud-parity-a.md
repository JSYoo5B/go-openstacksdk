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
| [`get_qos_policy`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L221) | `network.Service.GetQoSPolicyRecord` ([기본 조회와 잔여](cloud-qos-policy-actions-python.md)) | unresolved · raw headers/fetch controls |
| [`create_qos_policy`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1435) | `network.Service.CreateQoSPolicyRecord` ([사용법](cloud-qos-policy-mutations-python.md)) | go_mapping · 집중 검증 PASS |
| [`delete_qos_policy`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1512) | `network.Service.DeleteQoSPolicyRecord` ([사용법](cloud-qos-policy-actions-python.md)) | go_mapping · root network race PASS |
| [`get_qos_rule_type_details`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L300) | `qos/ruletypes.API.GetRecord` | unresolved |
| [`get_qos_bandwidth_limit_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1594) | `network.Service.GetQoSBandwidthLimitRule` ([사용법](cloud-qos-bandwidth-python.md)) | go_mapping · 집중 race·독립 main build PASS |
| [`create_qos_bandwidth_limit_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1621) | `qos/rules.API.CreateBandwidthLimitRule` | unresolved |
| [`delete_qos_bandwidth_limit_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1723) | `network.Service.DeleteQoSBandwidthLimitRule` ([사용법](cloud-qos-rule-deletes-python.md)) | go_mapping · root network race PASS |
| [`get_qos_dscp_marking_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1819) | `network.Service.GetQoSDSCPMarkingRule` ([사용법](cloud-qos-dscp-python.md)) | go_mapping · 집중 race·독립 main build PASS |
| [`create_qos_dscp_marking_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1844) | `qos/rules.API.CreateDSCPMarkingRule` | unresolved |
| [`delete_qos_dscp_marking_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1919) | `network.Service.DeleteQoSDSCPMarkingRule` ([사용법](cloud-qos-rule-deletes-python.md)) | go_mapping · root network race PASS |
| [`get_qos_minimum_bandwidth_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2018) | `network.Service.GetQoSMinimumBandwidthRule` ([사용법](cloud-qos-minimum-bandwidth-python.md)) | go_mapping · 집중 race·독립 main build PASS |
| [`create_qos_minimum_bandwidth_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2041) | `qos/rules.API.CreateMinimumBandwidthRule` | unresolved |
| [`delete_qos_minimum_bandwidth_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2115) | `network.Service.DeleteQoSMinimumBandwidthRule` ([사용법](cloud-qos-rule-deletes-python.md)) | go_mapping · root network race PASS |

표의 낮은 수준 resource API는 `github.com/JSYoo5B/go-openstacksdk/network/v2/` 아래이며, `routers`는 `extensions/layer3/routers`, `qos/...`와 `fwaas_v2/...`는 `extensions/` 아래입니다. 새 QoS cloud helper는 `network.Service`에 있으며 `conn.Network(ctx)` 또는 `network.New(client)`로 접근합니다. 기존 회귀는 [`python_cloud_parity_a_test.go`](python_cloud_parity_a_test.go), 새 QoS 조회 계약과 독립 main은 각 사용 guide에서 다룹니다. 기존 QoS 사용자 조회 12행은 집중 race·독립 main build PASS입니다. Cycle4 삭제 4행은 root network race·독립 main·전체 gate와 중앙 판정도 PASS입니다. Cycle5 정책 create 1행을 더해 이 A 표는 36행 중 대응 범위22행, unresolved14행입니다. 새 정책 create의 focused race·독립 main build·중앙 판정은 PASS이며 최종204개 test package 전체 gate도 PASS입니다.

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

이전 사이클에서 A/B 묶음에 걸친 **사용자 조회 12행**을 구현했습니다. 정책 list/search 2개, rule type list 1개, bandwidth-limit·DSCP-marking·minimum-bandwidth의 list/search/get 각 3개입니다. 이 문서 A 표에는 단일 rule get 3행이 해당합니다. 새 cloud helper는 낮은 수준의 Get*Record와 달리 확장 확인과 정책 lookup까지 SDK가 소유합니다. **기존 조회 12행은 go_mapping으로 root 집중 race·독립 main build PASS입니다. 이번 Cycle4 정책/규칙 삭제 4행은 root network race PASS이며 독립 main build·204개 test package 전체 gate·중앙 판정도 PASS입니다.**

| 사용자 조회 묶음 | Go facade와 상세 guide |
|---|---|
| 정책 list/search 2행 | ListQoSPolicyRecords·SearchQoSPolicyRecords — [정책](cloud-qos-policies-python.md) |
| rule type list 1행 | ListQoSRuleTypeValues — [rule types](cloud-qos-rule-types-python.md) |
| bandwidth-limit list/search/get 3행 | ListQoSBandwidthLimitRules·SearchQoSBandwidthLimitRules·GetQoSBandwidthLimitRule — [bandwidth](cloud-qos-bandwidth-python.md) |
| DSCP-marking list/search/get 3행 | ListQoSDSCPMarkingRules·SearchQoSDSCPMarkingRules·GetQoSDSCPMarkingRule — [DSCP](cloud-qos-dscp-python.md) |
| minimum-bandwidth list/search/get 3행 | ListQoSMinimumBandwidthRules·SearchQoSMinimumBandwidthRules·GetQoSMinimumBandwidthRule — [minimum bandwidth](cloud-qos-minimum-bandwidth-python.md) |

고정 Source처럼 각 호출은 GET /extensions의 **새 전체 목록과 모든 페이지**를 소비하여 qos alias를 확인합니다. 첫 일치 뒤에도 읽고 후반 alias 오류·HTTP 실패를 숨기지 않으며 cache하지 않습니다. 확장이 없으면 typed unavailable-extension 오류로 끝나고 정책·규칙 요청으로 넘어가지 않습니다. 규칙 helper는 그 다음 정책을 GET-first ID/name lookup으로 찾으며, 없는 정책·중복 이름은 규칙 HTTP 전에 오류입니다. Bandwidth/DSCP list/search/get와 minimum-bandwidth list/search는 ignore_missing=True 뒤 missing을 오류로 바꾸고, minimum-bandwidth get만 Source의 ignore_missing=False를 따릅니다. 단건 get의 404는 규칙 목록 fallback 없이 실제 오류로 남습니다.

정책 search는 truthy 이름을 server name query로 보내고 filters가 같은 key를 덮어쓸 수 있습니다. 규칙 search는 목록을 모두 읽은 뒤 rule ID/name exact·glob와 원래 mapping 필터를 적용합니다. Source Proxy의 opaque generator/JMESPath와 literal iterable 반환, falsey filters·bound controls·실제 단계별 receipt를 각 guide에서 설명합니다. 이 대응을 전체 SDK 또는 실제 배포의 인증·권한 검증 완료로 세지 않습니다.

Cycle4에서 추가한 삭제는 [정책 delete](cloud-qos-policy-actions-python.md)와 [세 규칙 delete](cloud-qos-rule-deletes-python.md)입니다. 네 helper 모두 fresh 전체 qos extension gate와 정책 GET-first ID/name lookup을 SDK가 소유합니다. `DeleteQoSPolicyRecord`는 찾은 정책이 없으면 DELETE 없이 false이고, 찾은 뒤 proxy 기본 ignore_missing=True로 삭제하므로 그 사이 발생한 clean DELETE404도 true입니다. 세 규칙 delete는 child DELETE에 ignore_missing=False를 고정해 clean rule404만 false로 바꿉니다. missing/ambiguous policy는 false가 아닌 오류이며 bandwidth/DSCP 정책 find는 ignore_missing=True 뒤 missing을 오류로 바꾸고 minimum-bandwidth만 ignore_missing=False입니다. 규칙 GET을 삭제 전 확인으로 추가하지 않습니다. result의 `Deleted`는 오류가 없을 때 읽고, 실제 extension/policy/opaque DELETE receipt와 physical·context/native401·guard 오류를 보존합니다. 서버 정책·driver 수락이나 관리자 권한을 SDK가 대신 만들지 않습니다.

A의 QoS 13행 중 단일 rule get 3행과 삭제 4행, 정책 create 1행은 Go 대응이고 **5행은 unresolved**입니다. `GetQoSPolicyRecord`의 기본 조회·ordinary JSON kwargs·missing·clean400/403/404 fallback·GET-only microversion·fallback-only list_base_path는 구현했지만 named `get_qos_policy`의 raw `headers`와 `requires_id`, `base_path`, `error_message`, `skip_cache`, `resource_response_key`가 shared findKey에서 거부됩니다. Source raw headers는 member GET query/constructor seed로 가고 fallback에서는 LIST formal이므로 concrete HTTP Headers 옵션으로 대체할 수 없습니다. member base_path와 fallback-only list_base_path도 다른 계약입니다. [잔여 상세](cloud-qos-policy-actions-python.md#get-지원-범위와-정확한-remaining)를 확인하고 이 getter를 완료 수에 넣지 않습니다.

관리자 `get_qos_rule_type_details`와 규칙 create 3행은 유지합니다. 상세 helper의 stock QoSPlugin administrator 제한, qos/qos-rule-type-details 별도 fresh gate, bandwidth create의 qos-bw-limit-direction 및 기본값/변경 계약은 잔여입니다. 정책 create는 Cycle5의 `CreateQoSPolicyRecord`가 valid_kwargs 선검증·fresh qos/qos-default gate·빈 attrs POST와 실제 응답을 소유하며 [생성·수정 안내](cloud-qos-policy-mutations-python.md)에 독립 예제가 있습니다. Stock Neutron27의 정책 create/update 권한은 project manager 또는 admin이며, deprecated admin-only와 구분합니다.

## 미해결 목록

- `create_firewall_rule`: Gophercloud `rules.CreateOpts`가 protocol·action을 필수로 요구해 인수 없는 Python 본문을 보낼 수 없습니다.
- `get_firewall_rule`, `get_firewall_policy`, `get_firewall_group`: FWaaS 검색이 filters query와 400·403 fallback을 지원하지 않습니다.
- `delete_firewall_rule`, `delete_firewall_policy`, `delete_firewall_group`: 위 검색 제약에 더해, 삭제 404를 `False`로 바꾸는 SDK API가 없습니다.
- `create_router`: `admin_state_up=True` 기본값, gateway 인수 묶음, AZ hint 검사와 `router_availability_zone` 확장 확인을 수행하는 SDK 헬퍼가 없습니다.
- `create_subnet`: 네트워크·subnetpool 검색, cidr 조합 검증, `ip_version`·`enable_dhcp` 기본값과 gateway 변환을 수행하는 SDK 헬퍼가 없습니다.
- QoS: 단일 rule get 3행과 정책/규칙 delete 4행을 실제 cloud helper에 연결했습니다. 정책 create는 이번 cloud helper로 대응합니다. unresolved는 get_qos_policy raw headers/fetch controls, 관리자 rule-type details, 규칙 create 3행의 5개입니다. 새 delete의 root network race PASS와 독립 main build·204개 test package 전체 gate·중앙 판정 PASS을 구분합니다.

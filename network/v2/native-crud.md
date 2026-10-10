# Neutron native 생성·조회·목록·삭제

`service.Networks`, `service.Subnets`, `service.Ports`의 generated `Create/Get/List/Delete`는 Gophercloud `v2.15.0`의 [networks](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/networks/requests.go), [subnets](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/subnets/requests.go), [ports](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/ports/requests.go) 요청을 바꾸지 않고 호출합니다. SDK는 오류에 `resource.OperationError{Resource: "networks"|"subnets"|"ports"}` 문맥만 더하며, 다른 status는 native `gophercloud.ErrUnexpectedResponseCode`로 남습니다. ID는 escape 없이 경로에 이어 붙입니다. 수정 호출의 revision 조건은 [revision 조건 수정](revision-updates.md)에 따로 설명합니다.

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Create(ctx, opts, options...)` | `POST networks`·`subnets`·`ports`, `{"network"\|"subnet"\|"port": {...}}` | 201, 202 |
| `Get(ctx, id)` | `GET {collection}/{id}` | 200 |
| `List(ctx, options...)` | `GET {collection}` | native pager 200, 204, 300 |
| `Delete(ctx, id)` | `DELETE {collection}/{id}` | 202, 204 |

`With...Field` 확장 필드는 단일 resource envelope 안에 들어가며 옵션 struct의 기존 key와 겹치거나 nil 옵션이면 HTTP 전에 거부됩니다. 생성 응답은 서버가 돌려준 resource이고 SDK가 요청 값을 합성하지 않습니다. 생성·수정 시각은 `2006-01-02T15:04:05`와 RFC3339 두 형식을 모두 받습니다.

## 생성 본문

`networks.CreateOpts`에는 필수 필드가 없고 빈 값은 생략합니다. `subnets.CreateOpts`와 `ports.CreateOpts`는 `NetworkID`가 비어 있으면 HTTP 전에 필수 입력 오류입니다.

subnet의 `GatewayIP`는 nil이면 생략하고, 빈 문자열을 가리키면 `"gateway_ip": null`로 보내 gateway를 끕니다. Python의 `gateway_ip=None`과 같은 의미입니다.

port의 `ValueSpecs` map은 별도 key로 보내지 않고 `port` 객체에 펼쳐 넣습니다. `shared`와 `tenant_id`는 금지 key이고, 이미 있는 필드(예: `name`)를 덮어쓰려 하면 HTTP 전에 오류입니다. `FixedIPs`는 `[]ports.IP`처럼 JSON으로 직렬화되는 값을 그대로 보냅니다.

## 목록

각 `ListOpts`의 필드와 `WithListQuery` 확장 query를 보냅니다. ports의 `FixedIPs []ports.FixedIPOpts`는 조건마다 `fixed_ips=ip_address=...`, `fixed_ips=subnet_id=...`처럼 반복 query가 되고 `SecurityGroups`도 반복 key로 보냅니다. `*_links`의 `rel=next` href를 받은 그대로 따라가며, 호출자가 순회를 멈추면 다음 페이지를 요청하지 않습니다. 빈 목록은 아무 값도 내보내지 않고, 본문 없는 204는 native pager가 JSON을 먼저 읽기 때문에 `io.EOF` 오류 하나로 끝납니다. 목록 오류는 operation 문맥 없이 native 오류 그대로 전달됩니다.

## security group·rule·floating IP

`service.SecurityGroups`(`extensions/security/groups`), `service.SecurityRules`(`extensions/security/rules`), `service.FloatingIPs`(`extensions/layer3/floatingips`)의 generated `Create/Get/List/Delete`도 위 표와 같은 status와 목록 규칙을 따릅니다. 경로는 각각 `security-groups`, `security-group-rules`, `floatingips`이고 envelope는 `security_group`, `security_group_rule`, `floatingip`입니다.

security group의 `Name`은 필수입니다. `Stateful`은 pointer라 false를 명시해 보낼 수 있습니다. security group 목록은 Gophercloud가 builder 대신 concrete `ListOpts`만 받으므로 `WithListQuery` 확장 query가 없습니다.

rule의 `Direction`, `EtherType`, `SecGroupID`는 필수입니다. `PortRangeMin`·`PortRangeMax`는 0이면 생략하므로 ICMP type 0처럼 0을 명시해 보낼 수 없습니다. 응답의 null protocol·port range·remote group은 빈 값과 0으로 decode됩니다. `CreateBulk(ctx, []rules.CreateOpts)`는 `{"security_group_rules": [...]}` 배열을 한 번에 보내고 응답 배열을 돌려줍니다. 요소 하나라도 필수 입력이 비면 HTTP 전에 오류이며, 성공 status는 201, 202이고 확장 필드 옵션은 없습니다.

floating IP의 `FloatingNetworkID`는 필수입니다. `Description`, `FloatingIP`, `PortID`, `FixedIP`, `SubnetID`, `TenantID`, `ProjectID`는 비어 있으면 생략하고 조합의 유효성은 Neutron이 판정합니다. `WithCreateOptions`는 위치 인자로 준 옵션 전체를 교체하며, 확장 필드(예: `dns_name`)는 `floatingip` 안에 들어갑니다. 응답의 null `port_id`·`fixed_ip_address`·`router_id`는 빈 문자열이고 알 수 없는 필드는 무시합니다. 잘못된 JSON이나 `floatingip`가 객체가 아닌 응답은 operation 문맥과 함께 decode 오류입니다. 서버 연결·NAT 선택·대기를 포함한 상위 workflow는 [floating IP 보장](../floating-ip-ensure.md)을 참고합니다.

## router·extra route

`service.Routers`(`extensions/layer3/routers`)의 generated `Create/Get/List/Delete`는 위 표와 같은 status와 목록 규칙을 따르며 경로는 `routers`, envelope는 `router`입니다. `CreateOpts`에는 필수 필드가 없고, `GatewayInfo`의 `EnableSNAT`처럼 pointer인 값은 false도 명시해 보낼 수 있습니다. 응답의 `external_gateway_info`가 null이면 `GatewayInfo`는 빈 값입니다.

| 메서드 | 요청 | 본문 | 성공 status |
|---|---|---|---|
| `AddInterface(ctx, id, opts, options...)` | `PUT routers/{id}/add_router_interface` | `{"subnet_id": ...}` 또는 `{"port_id": ...}` | 200 |
| `RemoveInterface(ctx, id, opts, options...)` | `PUT routers/{id}/remove_router_interface` | `subnet_id`·`port_id` 중 하나 이상 | 200 |
| `AddExternalGateways`·`UpdateExternalGateways`·`RemoveExternalGateways` | `PUT routers/{id}/add_external_gateways` 등 | `{"router": {"external_gateways": [...]}}` | 200 |
| extraroutes `Add`·`Remove(ctx, id, opts, options...)` | `PUT routers/{id}/add_extraroutes`·`remove_extraroutes` | `{"router": {"routes": [...]}}` | 200 |

interface 본문에는 envelope가 없어서 확장 필드는 `subnet_id`·`port_id` 옆 최상위에 붙습니다. `AddInterface`는 subnet과 port 중 정확히 하나, `RemoveInterface`는 하나 이상을 요구하며 어긋나면 HTTP 전에 오류입니다. interface 응답은 `InterfaceInfo`(router·subnet·port ID)입니다. external gateway 세 호출은 `ExternalGateways`가 nil이면 HTTP 전에 필수 입력 오류이고, 빈 slice는 `[]`로 보냅니다. extra route의 `Routes`가 nil이면 `{"router": {}}`를 보냅니다. gateway·extra route 호출의 확장 필드는 `router` envelope 안에 들어가고, 응답은 갱신된 `Router`입니다. router의 L3 agent 목록(`ListL3Agents`)은 관리자 호출이라 이 절에서 다루지 않습니다.

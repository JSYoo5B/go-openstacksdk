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

## subnet pool·address scope

`service.SubnetPools`(`extensions/subnetpools`)와 `service.AddressScopes`(`extensions/layer3/addressscopes`)의 generated `Create/Get/List/Delete`도 위 목록 규칙을 따릅니다. 경로는 `subnetpools`, `address-scopes`이고 envelope는 `subnetpool`, `address_scope`입니다. 두 `Create`는 Gophercloud가 성공 status를 201 하나로 좁혀서 202도 오류입니다. Get은 200, Delete는 202와 204를 받습니다.

| 메서드 | 요청 | 본문 | 성공 status |
|---|---|---|---|
| subnetpools `AddPrefixes(ctx, id, opts, options...)` | `PUT subnetpools/{id}/add_prefixes` | `{"prefixes": [...]}` | 200 |
| subnetpools `RemovePrefixes(ctx, id, opts, options...)` | `PUT subnetpools/{id}/remove_prefixes` | `{"prefixes": [...]}` | 200 |
| addressscopes `Update(ctx, id, opts, options...)` | `PUT address-scopes/{id}` | `{"address_scope": {...}}` | 200 |

subnet pool의 `CreateOpts`에는 필수 필드가 없습니다. `Name`과 `Prefixes`는 omitempty가 없어서 빈 이름은 `""`, nil prefix 목록은 `null`로 보냅니다. `Shared`·`IsDefault`는 bool이라 false를 명시해 보낼 수 없고 생략됩니다. 응답의 `default_prefixlen`·`min_prefixlen`·`max_prefixlen`은 숫자와 숫자 문자열(`"24"`)을 모두 받습니다. 세 값 중 하나라도 없거나 null이거나 숫자로 바꿀 수 없으면 native decoder가 오류를 내며, 이때 `Get`·`Create`는 오류와 함께 앞서 decode한 부분 값을 돌려줍니다. 응답의 null `address_scope_id`는 빈 문자열입니다.

prefix 두 호출의 본문에는 envelope가 없어서 확장 필드는 `prefixes` 옆 최상위에 붙습니다. `Prefixes`가 nil이면 `{}`를 보내고, 응답은 갱신 후 pool 전체의 prefix 목록(`[]string`)입니다. subnet pool 수정의 revision 조건은 [revision 조건 수정](revision-updates.md)을 참고합니다.

address scope의 `CreateOpts`도 필수 필드가 없습니다. `Name`과 `IPVersion`은 항상 보내므로 0도 `"ip_version": 0`으로 나가고 유효성은 Neutron이 판정합니다. `UpdateOpts`의 `Name`·`Shared`는 pointer라 빈 이름과 false를 명시해 보낼 수 있고, 둘 다 nil이면 `{"address_scope": {}}`를 보냅니다. 응답의 null 이름이나 `shared`는 빈 값으로 decode하며 알 수 없는 필드는 무시합니다.

## trunk·port forwarding

`service.Trunks`(`extensions/trunks`)의 generated `Create/Get/List/Delete`는 경로 `trunks`, envelope `trunk`을 쓰며 Create는 201·202, Get은 200, Delete는 202·204를 받습니다. `CreateOpts`의 `PortID`(부모 port)는 필수이고, `Subports`가 nil이면 `"sub_ports": []`로 바꿔 보냅니다. `AdminStateUp`은 pointer라 false도 보낼 수 있습니다. 응답 시각은 표준 RFC3339만 받아서 시간대 없는 `2006-01-02T15:04:05` 형식이면 decode 오류입니다.

trunk 목록은 Neutron 형식의 next 링크를 따라가지 않아 실제로는 **첫 페이지만** 읽습니다. Gophercloud의 `TrunkPage`가 `NextPageURL`을 정의하지 않아 기본 linked page 규칙인 `{"links": {"next": "..."}}` 문자열만 다음 페이지로 보고, Neutron이 주는 `trunks_links` 배열은 읽지 않습니다. 그래서 `Limit` 필드도 `ListOpts`에 없으며, 페이지가 나뉘는 배포에서는 `WithListQuery("limit", ...)`와 `marker`를 직접 다뤄야 합니다. `RevisionNumber` 필터는 문자열이라 `"0"`도 보냅니다.

| 메서드 | 요청 | 본문·응답 | 성공 status |
|---|---|---|---|
| `GetSubports(ctx, id)` | `GET trunks/{id}/get_subports` | 응답 `{"sub_ports": [...]}`를 `[]Subport`로 반환 | 200 |
| `AddSubports(ctx, id, opts, options...)` | `PUT trunks/{id}/add_subports` | `{"sub_ports": [...]}`, 응답은 envelope 없는 trunk 객체 | 200 |
| `RemoveSubports(ctx, id, opts, options...)` | `PUT trunks/{id}/remove_subports` | `{"sub_ports": [{"port_id": ...}]}`, 응답은 envelope 없는 trunk 객체 | 200 |

subport 본문에는 envelope가 없어서 확장 필드는 `sub_ports` 옆 최상위에 붙습니다. `AddSubports`는 `Subports`가 nil이면 HTTP 전에 필수 입력 오류이고 빈 slice는 `[]`로 보냅니다. 각 `Subport`의 `PortID`와 `SegmentationType`도 필수입니다. 다만 `SegmentationID`의 0은 필수 검사를 통과해 그대로 보내므로 `inherit` 형식에 쓸 수 있습니다. `RemoveSubports`는 nil 목록을 `"sub_ports": null`로 보내며, 목록 안 `RemoveSubport`의 `PortID`가 비면 HTTP 전에 오류입니다. trunk 수정의 revision 조건은 [revision 조건 수정](revision-updates.md)을 참고합니다.

`service.PortForwarding`(`extensions/layer3/portforwarding`)은 floating IP 아래 경로 `floatingips/{fip}/port_forwardings`를 씁니다. `Create(ctx, fip, opts, options...)`는 201·202, `Get(ctx, fip, id)`와 `Update(ctx, fip, id, opts, options...)`는 200, `Delete(ctx, fip, id)`는 202·204를 받으며 `List(ctx, fip, options...)`는 부모 floating IP ID를 첫 인자로 받습니다. envelope는 `port_forwarding`입니다.

port forwarding `CreateOpts`에는 필수 검사가 없습니다. `InternalPortID`, `InternalIPAddress`, `Protocol`은 omitempty가 없어서 비어 있어도 `""`로 보내고, port와 port range 필드는 비어 있거나 0이면 생략합니다. `UpdateOpts`는 `Description`만 pointer라 빈 설명을 보낼 수 있고 나머지 빈 값은 생략합니다. 모두 비면 `{"port_forwarding": {}}`를 보냅니다.

응답 decode는 `port_forwarding` key를 직접 찾습니다. 본문이 `{}`이거나 값이 null이면 오류 없이 빈 값을 돌려주고, 다른 key만 있거나 envelope가 객체가 아니면 오류입니다. 목록은 Gophercloud가 단수형 `port_forwarding_links`의 next 링크만 따라갑니다. Neutron이 복수형 `port_forwardings_links`로 링크를 주면 다음 페이지를 요청하지 않습니다.

## QoS policy·rule·rule type

`service.QoSPolicies`(`extensions/qos/policies`)의 generated `Create/Get/List/Delete`는 경로 `qos/policies`, envelope `policy`를 씁니다. Create는 201만 받고 Get은 200, Delete는 202·204를 받습니다. `CreateOpts`에는 필수 검사가 없으며 `Name`은 omitempty가 없어 빈 이름도 `""`로 보냅니다. `Shared`·`IsDefault`는 bool이라 false를 명시해 보낼 수 없습니다. 목록은 `policies_links`의 next href를 따라가고, `RevisionNumber` 필터는 pointer라 0도 보냅니다. 응답의 `rules`는 `[]map[string]any`로 그대로 decode하며 시각은 표준 RFC3339만 받습니다. policy 수정의 revision 조건은 [revision 조건 수정](revision-updates.md)을 참고합니다.

`service.QoSRules`(`extensions/qos/rules`)는 bandwidth limit, DSCP marking, minimum bandwidth 세 rule에 같은 형태의 호출 다섯 개씩을 제공합니다. 경로는 `qos/policies/{policy}/bandwidth_limit_rules`·`dscp_marking_rules`·`minimum_bandwidth_rules`이고 envelope는 단수형(`bandwidth_limit_rule` 등)입니다.

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Create{Kind}(ctx, policy, opts, options...)` | `POST qos/policies/{policy}/{collection}` | 201 |
| `Get{Kind}(ctx, policy, id)` | `GET qos/policies/{policy}/{collection}/{id}` | 200 |
| `Update{Kind}(ctx, policy, id, opts, options...)` | `PUT qos/policies/{policy}/{collection}/{id}` | 200 |
| `Delete{Kind}(ctx, policy, id)` | `DELETE qos/policies/{policy}/{collection}/{id}` | 202, 204 |
| `List{Kind}s(ctx, policy, options...)` | `GET qos/policies/{policy}/{collection}` | native pager 200, 204, 300 |

생성 opts의 대표 값(`MaxKBps`, `DSCPMark`, `MinKBps`)은 omitempty가 없어서 0도 그대로 보내고, 유효성은 Neutron이 판정합니다. 수정 opts의 같은 값은 pointer라 nil이면 생략하고 0을 가리키면 0을 보냅니다. `Direction`은 문자열이라 비어 있으면 생략합니다. 모두 비면 빈 envelope를 보냅니다. rule 목록은 trunk와 같이 `NextPageURL`이 없어서 `*_links` 배열을 읽지 않고 `{"links": {"next": "..."}}` 문자열만 따라갑니다. 그래서 Neutron 응답에서는 첫 페이지만 반환합니다. 부모 policy를 이름이나 ID로 먼저 찾는 scope API는 [scoped resource](../../docs/scoped-resources.md)에 설명합니다.

`service.QoSRuleTypes`(`extensions/qos/ruletypes`)의 `GetRuleType(ctx, name)`은 `GET qos/rule-types/{name}`을 보내고 200만 받습니다. `ListRuleTypes(ctx)`는 `GET qos/rule-types` 한 페이지만 읽으며 옵션이 없습니다. driver의 `parameter_values`는 범위 객체나 선택지 배열 같은 JSON 형태를 `any`로 그대로 둡니다.

## address group·RBAC policy

`service.SecurityAddressGroups`(`extensions/security/addressgroups`)의 generated `Create/Get/List/Update/Delete`는 경로 `address-groups`, envelope `address_group`을 쓰며 Create는 201·202, Get·Update는 200, Delete는 202·204를 받습니다. `CreateOpts`의 `Addresses`는 필수라 nil이면 HTTP 전에 오류이고, 빈 slice는 `[]`로 보냅니다. `ID`를 채우면 서버에 그 ID를 요청합니다. `UpdateOpts`의 `Name`·`Description`은 pointer라 빈 문자열도 보낼 수 있습니다. 목록의 `Addresses` 필터는 값마다 `addresses=` query를 반복하고 `address_groups_links`의 next href를 따라갑니다.

| 메서드 | 요청 | 본문 | 성공 status |
|---|---|---|---|
| `AddAddresses(ctx, id, opts, options...)` | `PUT address-groups/{id}/add_addresses` | `{"addresses": [...]}` | 200 |
| `RemoveAddresses(ctx, id, opts, options...)` | `PUT address-groups/{id}/remove_addresses` | `{"addresses": [...]}` | 200 |

두 주소 호출의 본문에는 envelope가 없어서 확장 필드는 `addresses` 옆 최상위에 붙습니다. `Addresses`가 nil이면 HTTP 전에 필수 입력 오류입니다. 응답은 `address_group` envelope의 갱신된 그룹 전체입니다.

`service.RBACPolicies`(`extensions/rbacpolicies`)는 경로 `rbac-policies`, envelope `rbac_policy`를 씁니다. Create는 201·202, Get은 200, Delete는 202·204를 받고 Update는 200과 201을 모두 성공으로 봅니다. `CreateOpts`의 `Action`·`ObjectType`·`ObjectID`·`TargetTenant`와 `UpdateOpts`의 `TargetTenant`는 필수라 비어 있으면 HTTP 전에 오류입니다. `TargetTenant`에는 모든 프로젝트를 뜻하는 `"*"`도 쓸 수 있습니다.

RBAC 응답 decode는 port forwarding과 같이 `rbac_policy` key를 직접 찾습니다. 본문이 `{}`이거나 값이 null이면 오류 없이 빈 값을 돌려주고, 다른 key만 있거나 envelope가 객체가 아니면 오류입니다. RBAC 목록은 Gophercloud가 `NextPageURL`을 정의하지 않아 `rbac_policies_links` 배열을 읽지 않고 `{"links": {"next": "..."}}` 문자열만 따라갑니다. 그래서 Neutron 응답에서는 첫 페이지만 반환합니다.

## attribute tag·API 버전·extension 조회

`service.AttributeTags`(`extensions/attributestags`)는 모든 태그 지원 resource에 대해 `{resourceType}/{id}/tags` 경로를 씁니다. `resourceType`과 태그는 escape 없이 경로에 이어 붙이므로 `qos/policies`처럼 slash가 든 resource type을 그대로 쓸 수 있고, slash가 든 태그는 경로 segment가 늘어납니다.

| 메서드 | 요청 | 결과 | 성공 status |
|---|---|---|---|
| `ReplaceAll(ctx, type, id, opts, options...)` | `PUT {type}/{id}/tags`, `{"tags": [...]}` | 서버의 태그 목록 | 200 |
| `List(ctx, type, id)` | `GET {type}/{id}/tags` | 태그 목록 | 200 |
| `Add(ctx, type, id, tag)` | `PUT {type}/{id}/tags/{tag}`, 본문 없음 | 없음 | 201 |
| `Delete(ctx, type, id, tag)` | `DELETE {type}/{id}/tags/{tag}` | 없음 | 204 |
| `DeleteAll(ctx, type, id)` | `DELETE {type}/{id}/tags` | 없음 | 204 |
| `Confirm(ctx, type, id, tag)` | `GET {type}/{id}/tags/{tag}` | 존재 여부 | 204 |

`ReplaceAll`의 `Tags`는 필수라 nil이면 HTTP 전에 오류이고, 빈 slice는 `{"tags": []}`로 보내 모든 태그를 지웁니다. 본문에 envelope가 없어서 확장 필드는 `tags` 옆 최상위에 붙습니다. 두 삭제 호출은 일반 DELETE와 달리 202를 받지 않습니다. `Confirm`은 404를 오류 없이 `false`로 바꾸고, 204는 `true`, 그 밖의 status는 `false`와 오류를 돌려줍니다.

`service.APIVersions`(`apiversions`)의 `ListVersions(ctx)`와 `ListVersionResources(ctx, version)`은 `ResourceBase`가 아니라 client `Endpoint`에서 버전 segment와 query를 잘라낸 root를 씁니다. `ListVersions`는 `GET {root}/`, `ListVersionResources`는 `GET {root}/{version}/`을 보내며 version 인자 끝의 slash는 하나로 정리합니다. 두 목록은 한 페이지만 읽고 링크를 따라가지 않습니다.

`service.Extensions`(`extensions`)의 `List(ctx)`는 `GET extensions`를 한 페이지로 읽고, `Get(ctx, alias)`는 `GET extensions/{alias}`를 보내 200만 받습니다. 응답의 `updated`는 시각으로 바꾸지 않고 서버가 준 문자열 그대로 둡니다.

## firewall group·policy·rule

`service.FirewallGroups`, `service.FirewallPolicies`, `service.FirewallRules`(`extensions/fwaas_v2/...`)는 FWaaS v2 경로 `fwaas/firewall_groups`, `fwaas/firewall_policies`, `fwaas/firewall_rules`를 씁니다. envelope는 `firewall_group`, `firewall_policy`, `firewall_rule`입니다. 세 resource 모두 `Create`는 201·202, `Get`·`Update`는 200, `Delete`는 202·204를 받고, 목록은 각 `*_links`의 next href를 따라갑니다.

group의 `AdminStateUp`·`Shared`는 pointer라 false를 보낼 수 있습니다. `UpdateOpts.Ports`는 slice pointer라 빈 slice를 가리키면 `"ports": []`로 모든 port를 뺍니다. 목록의 `Ports` 필터도 slice pointer이며 값마다 `ports=` query를 반복하고, 빈 slice는 생략합니다. `RemoveIngressPolicy(ctx, id)`와 `RemoveEgressPolicy(ctx, id)`는 별도 action 경로가 아니라 일반 수정 경로에 `{"firewall_group": {"ingress_firewall_policy_id": null}}`(egress도 같은 형태)을 보내 200만 받습니다.

policy의 `FirewallRules`는 생성 때 비어 있으면 생략하고, 수정 때는 slice pointer라 빈 slice로 규칙을 모두 비울 수 있습니다. 규칙 순서는 두 action으로 바꿉니다.

| 메서드 | 요청 | 본문 | 성공 status |
|---|---|---|---|
| `InsertRule(ctx, id, opts, options...)` | `PUT fwaas/firewall_policies/{id}/insert_rule` | `{"firewall_rule_id": ..., "insert_before"\|"insert_after": ...}` | 200 |
| `RemoveRule(ctx, id, ruleID)` | `PUT fwaas/firewall_policies/{id}/remove_rule` | `{"firewall_rule_id": ...}` | 200 |

`InsertRule`은 `ID`가 필수이고 `InsertBefore`와 `InsertAfter` 중 정확히 하나를 요구하며, 어긋나면 HTTP 전에 오류입니다. 본문에 envelope가 없어서 확장 필드는 최상위에 붙습니다. `RemoveRule`은 옵션이 없고 빈 rule ID도 그대로 보냅니다. 두 호출의 응답은 envelope 없는 policy 객체입니다.

rule의 `Protocol`과 `Action`은 필수입니다. `Protocol`이 `ProtocolAny`(`"any"`)이면 `"protocol": null`로 바꿔 보내 모든 protocol을 뜻합니다. 수정 opts는 모든 필드가 pointer라 빈 문자열이나 false도 보낼 수 있습니다. 응답의 `firewall_policy_id`는 `[]string`으로만 decode하므로 서버가 문자열 하나를 주면 decode 오류입니다. null `protocol`은 빈 문자열입니다.

## VPNaaS

`service.VPNEndpointGroups`, `service.VPNIKEPolicies`, `service.VPNIPsecPolicies`, `service.VPNServices`, `service.VPNSiteConnections`(`extensions/vpnaas/...`)는 모두 `Create/Get/List/Update/Delete`를 제공합니다. 경로와 envelope는 아래와 같고, 다섯 resource 모두 Create는 201·202, Get·Update는 200, Delete는 202·204를 받으며 목록은 각 `*_links`의 next href를 따라갑니다.

| resource | 경로 | envelope | 입력에서 주의할 점 |
|---|---|---|---|
| endpoint group | `vpn/endpoint-groups` | `endpoint_group` | 생성 `Endpoints`는 omitempty가 없어 nil이면 `null` |
| IKE policy | `vpn/ikepolicies` | `ikepolicy` | 생성은 `phase1_negotiation_mode`, 수정과 목록 필터는 `phase_1_negotiation_mode` key |
| IPsec policy | `vpn/ipsecpolicies` | `ipsecpolicy` | 생성 필드는 모두 선택이며 비면 생략 |
| VPN service | `vpn/vpnservices` | `vpnservice` | 생성 `RouterID` 필수, `AdminStateUp`이 nil이면 `"admin_state_up": null` |
| site connection | `vpn/ipsec-site-connections` | `ipsec_site_connection` | 생성의 policy·service·peer ID, peer 주소, PSK는 omitempty가 없어 비어도 `""`로 보냄 |

IKE·IPsec policy의 `Lifetime`은 pointer 객체라 nil이면 생략하고, 빈 객체를 가리키면 `"lifetime": {}`를 보냅니다. IKE policy는 생성과 수정의 negotiation mode key 철자가 다릅니다. 이 차이는 Gophercloud 고정 소스 그대로이며, 수정에서 이 필드를 쓰려면 대상 배포가 `phase_1_negotiation_mode` key를 받는지 먼저 확인해야 합니다.

site connection 목록의 `PSK` 필터는 `psk=` query로 보내므로 사전 공유 키가 URL과 서버·proxy 로그에 남을 수 있습니다. 응답의 `psk`도 그대로 decode합니다. 수정 opts는 `Name`·`Description`·`AdminStateUp`만 pointer이고 나머지 빈 값은 생략합니다. 서비스·IKE·IPsec·endpoint group의 수정 opts도 이름과 설명만 pointer라 빈 문자열을 명시할 수 있습니다.

# Network

Neutron 네트워크·포트·floating IP의 공통 조회 정책과 floating IP 생성·재사용·서버 연결 작업을 제공합니다. `conn`과 `ctx`는 [전체 README](../README.md)처럼 준비합니다. 아래 Go 조각은 `fmt`, `resource` 등을 import한 오류 반환 함수 안에서 사용합니다.

## openstacksdk 대응

| openstacksdk | gophercloudsdk |
|---|---|
| `conn.network.get_network(id)` | `service.Networks.Get(ctx, id)` |
| `conn.network.find_network(name, ignore_missing=False)` | `service.Networks.Find(ctx, resource.Name(name))` |
| `conn.network.networks(status="ACTIVE")` | `service.Networks.List(ctx, resource.WithStatus("ACTIVE"))` |
| `conn.network.networks(is_router_external=True)` | `service.Networks.List(ctx, resource.WithQuery("router:external", "true"))` |
| `conn.network.delete_network(id)` | `service.Networks.Delete(ctx, resource.ID(id))` |
| `conn.create_network("private")` | `service.CreateNetwork(ctx, network.CreateNetworkRequest{Name: "private"})` |
| `conn.update_network(name, name="renamed")` | `service.UpdateNetwork(ctx, resource.Name(name), network.WithNetworkName("renamed"))` |
| `conn.delete_network(name)` | `service.DeleteNetwork(ctx, resource.Name(name))`; `(bool, error)` |
| `conn.network.find_router(name_or_id, ignore_missing=False)` | `service.API.Routers.FindIdentity(ctx, nameOrID, resource.WithIdentityFindIgnoreMissing(false))` |
| `conn.network.find_security_group(name_or_id, project_id=projectID)` | `service.API.SecurityGroups.FindIdentity(ctx, nameOrID, resource.WithIdentityFindQuery("project_id", projectID))` |
| `conn.network.find_subnet_pool(name_or_id)` | `service.API.SubnetPools.FindIdentity(ctx, nameOrID)` |
| `conn.network.find_trunk(name_or_id)` | `service.API.Trunks.FindIdentity(ctx, nameOrID)` |
| `conn.network.find_qos_policy(name_or_id, is_shared=True)` | `service.API.QoSPolicies.FindIdentity(ctx, nameOrID, resource.WithIdentityFindQuery("shared", "true"))` |
| `conn.network.find_address_group(name_or_id)` | `service.API.SecurityAddressGroups.FindIdentity(ctx, nameOrID)` |
| `conn.network.get_port(id)` | `service.Ports.Get(ctx, id)` |
| `conn.network.get_ip(id)` | `service.FloatingIPs.Get(ctx, id)` |
| `conn.create_floating_ip(network="public", server=server, wait=True)` | `service.FloatingIPs.Create(ctx, request, network.WithServer(ref), network.WithWait())` |
| `conn.add_ip_list(server, ips=[address])`의 기존 Neutron IP 한 개 연결 기반 | `service.FloatingIPs.Attach(ctx, request)`; [대상·대기·부분 결과 차이](floating-ip-attach.md) |
| `conn.add_ips_to_server(server, ip_pool="public", reuse=True)`의 Neutron pool 선택·연결 기반 | `service.FloatingIPs.Ensure(ctx, request, network.WithEnsureWait())`; [재사용·결과·대기 차이](floating-ip-ensure.md) |
| 독립 `conn.add_ips_to_server` / `add_ip_list` | [Compute·Connection Add helper](../compute/server-ip-helpers.md): 기본60초·비동기와 raw 주소 wait |
| pool → 순차 명시 IP → automatic 선택과 서버 관측 | [Compute IP dispatch](../compute/server-ip-dispatch.md)의 `WithFloatingIPPool`·`WithFloatingIPAddresses` |

openstacksdk는 일부 query 이름을 Python 속성 이름으로 매핑합니다. `WithQuery`는 Neutron의 실제 HTTP query 이름을 받습니다. [공식 Network API](https://docs.openstack.org/openstacksdk/latest/user/proxies/network.html)

## 조회와 목록

Python:

```python
network = conn.network.find_network("private", ignore_missing=False)
for network in conn.network.networks(is_router_external=True):
    print(network.id)
```

Go:

```go
service, err := conn.Network(ctx)
if err != nil { return err }
network, err := service.Networks.Find(ctx, resource.Name("private"))
if err != nil { return err }
fmt.Println(network.ID)

for network, err := range service.Networks.List(ctx,
    resource.WithQuery("router:external", "true"), resource.WithPageSize(100)) {
    if err != nil { return err }
    fmt.Println(network.ID)
}
```

추가 query를 사용해도 builder interface 구현은 필요하지 않습니다. 이름 검색은 정확한 일치를 확인하며, 같은 이름이 여러 개면 `ErrAmbiguous`를 반환합니다.

## Router·Security Group 자동 조회

`service.API.Routers.FindIdentity`와 `service.API.SecurityGroups.FindIdentity`는 같은
이름·ID 문자열 옵션을 받습니다. 기본 GET400·403·404 뒤 모든 목록 페이지에서 정확한
ID/이름을 검사하고, 같은 이름이 여러 프로젝트에 있으면 `ErrAmbiguous`를 반환합니다.
`WithIdentityFindQuery("project_id", projectID)`로 호출자가 조회 범위를 지정할 수 있습니다.
SDK가 프로젝트를 자동 추론하지 않으며 query는 GET과 목록에 모두 전달됩니다.

반복 `fields`·tags와 확장 query는 concrete 설정의 `Query`로 전달할 수 있습니다.
Security Group의 공통 `Resources.List/All`도 raw query를 보존하며, 로컬 소비량·첫 페이지
옵션을 지원합니다. native typed `List`는 기존 concrete `ListOpts`를 사용합니다.
`FindIdentity`는 wire query를 그대로 사용합니다. ordinary `Resources.List/All`의 Python
속성 별칭과 로컬 Body 분류는 아래 `WithFilter`/`WithFilters` 사용법을 따릅니다.
[Python/Go 사용 예제](../docs/finding-identities.md#neutron-routersecurity-group와-project-query)를 참고하세요.

## Subnet Pool·Trunk 자동 조회

`service.API.SubnetPools.FindIdentity`와 `service.API.Trunks.FindIdentity`도 같은
이름·ID 옵션을 사용합니다. 프로젝트 필터는 `WithIdentityFindQuery("project_id", id)`로
지정하며 SDK가 자동 추론하지 않습니다. native 모델·오류·페이지 경계를 유지합니다.
Subnet Pool은 세 prefix 길이의 string/number decode가 필요하므로 `fields=id,name`처럼
이 값을 제외한 응답은 오류입니다. Trunk의 native pager는 `links.next`, Subnet Pool은
`subnetpools_links`를 따릅니다. [Python/Go 예제와 경계](../docs/finding-identities.md#neutron-subnet-pooltrunk)를 참고하세요.

## QoS Policy·Address Group 자동 조회

QoS Policy·Address Group의 자동 조회는 각각 `service.API.QoSPolicies.FindIdentity`와
`service.API.SecurityAddressGroups.FindIdentity`를 사용합니다. Python의 `is_shared`는
Go wire query `shared`로 지정하며 `rules`·`addresses`의 Body 로컬 필터를 자동 적용하지
않습니다. [Python/Go 예제와 native 모델 경계](../docs/finding-identities.md#neutron-qos-policyaddress-group)를 참고하세요.

ordinary 목록에서는 `service.API.QoSPolicies.Resources.List/All`의 `rules`,
`service.API.SecurityAddressGroups.Resources.List/All`의 `addresses`,
`service.API.SubnetPools.Resources`의 `prefixes`, `service.API.Networks.Resources`와 상위
`service.Networks`의 `subnets`를 `resource.WithBodyFilter` 또는 `WithBodyFilters`로 로컬에서
비교합니다. Network는 Python 속성 이름 `subnet_ids`를 `subnets`의 SDK 별칭으로 받습니다.
SDK가 field 선택과 옵션 복사를 처리하므로 별도 predicate는 필요하지 않습니다. 배열 순서·길이·내부 dict 전체가
일치해야 하며 raw query와는 독립적입니다. [Python/Go Body 필터 사용법](../docs/listing.md#명시적인-native-body-필터)을 참고합니다.

`service.API.Subnets.Resources`는 allocation pools·DNS nameservers·host routes·service types·
두 timestamp·prefix length·tenant ID·revision number의 9개 로컬 필터도 제공합니다.
native 모델이 생략하는 `prefixlen`과 nested 추가 필드는 원본 페이지에서 비교하며,
반환값은 기존 typed Subnet입니다. `resource.WithFilter`/`WithFilters`는 Python 속성 이름을
받아 query 24개와 로컬 Body 9개로 자동 분류합니다. 예를 들어 `is_dhcp_enabled`는
`enable_dhcp` query이고 `prefix_length`는 원본 `prefixlen`의 로컬 비교입니다.
[Subnet Python/Go 사용법](v2/subnets/README.md)에 bulk 교체·별칭 우선순위·충돌 검사를 설명합니다.

`service.API.SecurityAddressGroups.Resources`도 `resource.WithFilter`/`WithFilters`로
query 8개와 로컬 Body 3개를 분류합니다. `name`·`project_id`는 서버 query이고
`id`·`tenant_id`·`addresses`는 원본 응답의 로컬 조건입니다. native 모델에 없는 tenant ID와
주소 배열의 null 요소도 원문으로 비교하지만 반환값은 기존 typed AddressGroup입니다.
[AddressGroup Python/Go 사용법](v2/extensions/security/addressgroups/listing/README.md)에
필드 선택과 이름 hint·raw query·페이지 경계의 차이를 설명합니다.

`service.API.QoSPolicies.Resources`는 query 15개와 로컬 Body 2개를 분류합니다.
`is_shared`→`shared`와 태그 query 별칭을 포함한 19개 이름을 받으며 bulk에서는 canonical 값이 우선합니다.
`rules`·deprecated `tenant_id`는 원문에서 비교하며 `name`·`id`·`project_id`·`is_default`·태그는
서버 query입니다. rule의 큰 숫자 비교와 native float64 반환값을 구분합니다.
[QoS Policy Python/Go 사용법](v2/extensions/qos/policies/listing/README.md)에 옵션·응답·페이지 차이를 설명합니다.

`service.API.SubnetPools.Resources`는 query 16개와 로컬 Body 10개를 분류합니다.
`is_shared`와 태그 별칭을 포함한 20개 query 이름을 받으며 `project_id`는 서버 query입니다.
`id`·deprecated `tenant_id`·prefix 배열·두 timestamp·다섯 정수 속성은 원문에서 비교합니다.
Python prefix length 이름을 wire 필드로 연결하고 정수 응답에는 정확한 정수 정책을 적용합니다.
native 모델의 필수 prefix length 디코드와 timestamp 디코드는 로컬 비교보다 먼저 수행합니다.
[Subnet Pool Python/Go 사용법](v2/extensions/subnetpools/listing/README.md)에 전체 이름과 Python 정수 변환의 차이를 설명합니다.

`service.Networks`와 `service.API.Networks.Resources`는 `resource.WithFilter`/`WithFilters`로
query 23개와 로컬 Body 14개를 분류합니다. query는 wire 별칭을 포함해 35개 이름을 받으며
`name`·`status`·`id`는 서버 조건입니다. 기존 `WithName`의 정확한 로컬 이름 비교와
`WithStatus`의 대소문자 무시 로컬 비교는 별도로 유지합니다. provider·address scope·확장 query도
builder 없이 전달하고 원문 `subnet_ids`·availability zone·segments·timestamp를 비교합니다.
네 boolean 응답은 null을 보존한 truthiness로, `mtu`·`revision_number`는 정확한 정수 정책으로
비교합니다. 반환값은 기존 native Network이며 전체 페이지의 알려진 필드 디코드가 먼저 수행됩니다.
`Networks.ResourceAdapter()`는 두 SDK facade를 조립하는 독립 메타데이터를 제공하며 일반 호출자는
collection 옵션을 사용합니다. 상위 facade의 `network` 오류 이름과 정확한 `ERROR` 상태 대기는
유지합니다. [Network Python/Go 사용법](v2/networks/listing/README.md)에 전체 필터와 변환·충돌·페이지 차이를 설명합니다.

`service.API.Routers.Resources`와 `conn.NetworkV2(ctx).Routers.Resources`는 같은 client와
`resource.WithFilter`/`WithFilters` 옵션으로 query 18개·로컬 Body 10개를 분류합니다.
bool query 세 개와 태그 별칭을 포함한 24개 이름을 받으며 `name`·`status`·`id`는 서버 조건입니다.
gateway·routes·availability zone·timestamp·deprecated tenant ID는 원문에서 비교합니다.
`enable_ndp_proxy`의 응답 truthiness와 `evpn_vni`·`revision`의 정확한 정수 변환을 SDK가 처리합니다.
Python `revision_number`는 원문 `revision`을 선택합니다. native `RevisionNumber`의 응답 필드
`revision_number`는 별도이며 전체 페이지의 native 디코드가 로컬 비교보다 먼저 수행됩니다.
기존 이름·상태 조건과 native typed List·FindIdentity·Get·interface 변경 API는 유지합니다.
[Router Python/Go 사용법](v2/extensions/layer3/routers/listing/README.md)에 전체 필터와 raw 응답 차이를 설명합니다.

`service.API.SecurityGroups.Resources`와 `conn.NetworkV2(ctx).SecurityGroups.Resources`는
query 17개·accepted 이름 21개와 로컬 Body 3개를 분류합니다. `is_shared`→`shared`와 태그 별칭을
변환하며 revision·project·tenant·stateful·이름·ID는 서버 조건입니다. 두 timestamp와
`security_group_rules`를 원문에서 비교하고 native 전체 페이지의 중첩 rule 디코드를 먼저 적용합니다.
추가 rule 필드와 큰 숫자·null 배열 요소는 비교에 남으며 반환값은 기존 native SecGroup입니다.
SDK 소유 pager가 concrete native ListOpts로 표현할 수 없는 반복·nil·확장 query를 보존합니다.
기존 `WithName`은 로컬 이름 비교를 유지하고 `WithStatus`는 미지원이며 raw status는 전달합니다.
[Security Group Python/Go 사용법](v2/extensions/security/groups/listing/README.md)에 전체 이름·입력·native 응답·페이지 차이를 설명합니다.

`service.API.Trunks.Resources`와 `conn.NetworkV2(ctx).Trunks.Resources`는
query 14개·accepted 이름 18개와 로컬 Body 2개를 분류합니다. admin state·태그 별칭은 서버
query로 변환하고 `id`·`tenant_id`는 원문 JSON에서 비교합니다. `project_id`와 로컬 tenant는
독립적이며 `sub_ports`·이름·상태는 서버 조건입니다. native에만 있는 timestamp와 revision은
semantic 이름으로 받지 않습니다. 기존 `WithName`의 정확한 이름 비교와 `WithStatus`의 최종
query 값에 대한 대소문자 무시 비교를 유지하며 raw status 단독은 로컬 조건이 아닙니다.
native 전체 페이지의 RFC3339 timestamp·nested Subport 디코드가 필터와 cap보다 먼저 수행됩니다.
[Trunk Python/Go 사용법](v2/extensions/trunks/listing/README.md)에 전체 이름과
`links.next` 페이지 순회, null·native 응답·호출별 옵션 차이를 설명합니다.

## 생성·수정

[네트워크 CRUD와 Python/Go 비교](network-mutations.md)에 concrete With 옵션, 생성 기본값과 수정의 false·null·빈 값, provider/AZ·extension, revision, 이름/ID·부분 응답과 공유 cache 초기화를 설명합니다. 기존 `Networks` Collection 타입과 direct-ID 삭제 동작은 유지합니다.

## 삭제와 대기

```go
ready, err := service.Networks.Wait(ctx, resource.ID(network.ID), "ACTIVE")
if err != nil { return err }
fmt.Println(ready.Status)

if err := service.Networks.Delete(ctx, resource.Name("private"),
    resource.WithMissingError()); err != nil {
    return err
}
```

`WithMissingError()` 없이 삭제하면 미존재를 성공으로 처리합니다. 연결된 포트 등으로 삭제가 거부되면 실제 HTTP 오류를 반환합니다. `ERROR`는 실패 상태로 처리합니다.

endpoint를 직접 지정할 때는 `sdk.WithEndpoint(sdk.Network, "https://network.example/")`처럼 Neutron의 base URL을 전달합니다. 연결 계층이 `/v2.0/`를 추가하므로 중복으로 붙이지 않습니다.

## Floating IP 생성과 서버 연결

Python:

```python
server = conn.compute.find_server("web", ignore_missing=False)
floating_ip = conn.create_floating_ip(
    network="public", server=server, nat_destination="private", wait=True)
```

Go (`gophercloudsdk/network`를 import):

```go
service, err := conn.Network(ctx)
if err != nil { return err }
created, err := service.FloatingIPs.Create(ctx,
    network.CreateFloatingIPRequest{Network: resource.Name("public")},
    network.WithServer(resource.Name("web")),
    network.WithNATDestination(resource.Name("private")),
    network.WithWait(resource.WithTimeout(time.Minute)))
if err != nil {
    if created != nil {
        fmt.Println("created floating IP remains:", created.ID)
    }
    return err
}
fmt.Println(created.FloatingIP, created.PortID, created.FixedIP)
```

외부 네트워크 이름은 `router:external=true`로 조회한 뒤 응답의 외부 플래그와 정확한 이름을 다시 확인합니다. 같은 이름의 내부 네트워크는 후보가 되지 않습니다. 외부 네트워크 ID를 직접 지정하면 추가 조회 없이 Neutron에 전달하며, 외부 네트워크 여부는 Neutron이 검증합니다. 서버 이름은 Connection이 Compute를 통해 해석합니다. 명시 서버 ID는 Compute 조회를 생략합니다.

서버에 속한 포트의 유효한 fixed IPv4 주소를 모두 검사합니다. IPv6와 잘못된 주소는 제외하고, `WithNATDestination`과 `WithFixedAddress`로 후보를 좁힐 수 있습니다. 서버·네트워크 조건은 API query와 실제 응답 모두에 적용합니다. 여러 포트가 남거나 한 포트에 여러 IPv4 주소가 남으면 `ErrAmbiguous`를 반환하며 생성하지 않습니다. 오류에는 `portID@IPv4` 후보가 포함됩니다. Python의 최근 포트·첫 IPv4 선택처럼 임의의 후보를 선택하지 않습니다.

```go
created, err := service.FloatingIPs.Create(ctx,
    network.CreateFloatingIPRequest{Network: resource.ID("external-network-id")},
    network.WithServer(resource.ID("server-id")),
    network.WithPort(resource.ID("port-id")),
    network.WithFixedAddress("10.0.0.10"),
    network.WithDescription("web ingress"),
    network.WithFloatingIPField("vendor_enabled", false))
```

`WithPort`는 서버 포트 목록을 조회하는 대신 지정한 포트를 조회합니다. 포트 이름도 정확한 일치와 중복 오류 정책을 사용합니다. 서버와 함께 지정하면 해당 포트가 서버에 속하는지도 확인합니다. 서버 없이 포트만 지정해 연결할 수도 있습니다. 포트의 IPv4 후보가 여러 개면 `WithFixedAddress`가 필요합니다. `WithFloatingIPAddress`는 외부 네트워크에서 할당받을 특정 IPv4 주소를 요청합니다. extension 값은 옵션을 만들 때 JSON으로 복사하며, core 필드나 선택된 포트·네트워크를 덮어쓸 수 없습니다.

`WithServer`와 `WithPort`를 모두 생략하면 연결하지 않은 새 floating IP를 할당합니다. fixed 주소·NAT destination·ACTIVE 대기는 연결 대상이 있어야 사용할 수 있습니다. `WithWait`는 생성한 동일 ID가 ACTIVE가 될 때까지 공통 wait 정책으로 조회합니다. 생성 후 상태 대기나 연결 검증이 실패하면 `created`와 오류를 함께 반환합니다. SDK가 자동 삭제하지 않으므로 생성된 리소스를 확인하거나 직접 정리할 수 있습니다.

직접 Service를 구성하는 코드는 기존 `network.New(client)`를 계속 사용할 수 있습니다. 서버 이름을 해석하려면 `network.NewWithDependencies(client, network.Dependencies{Server: resolver})`를 사용합니다. 일반적인 `conn.Network(ctx)` 사용에서는 Connection이 이 의존성을 제공합니다.

## 전체 API와 남은 복합 작업

네트워크 생성·수정, subnet, port, router, security group, floating IP의 개별 호출은 `service.API`의 [Network v2 API](v2/README.md)에서 제공합니다. 예를 들어 `service.API.Networks.Create(ctx, networks.CreateOpts{...})`와 `service.API.FloatingIPs.Create(ctx, floatingips.CreateOpts{...})`는 SDK가 제공하는 concrete options를 사용하며 builder interface 구현이 필요하지 않습니다. 각각 `gophercloudsdk/network/v2/networks`, `gophercloudsdk/network/v2/extensions/layer3/floatingips`를 import합니다.

위 `Create` 작업은 Neutron에서 새 IPv4 floating IP를 생성합니다. [FloatingIPs.Ensure](floating-ip-ensure.md)는 현재 프로젝트의 이미 연결된 IP·가용 IP·새 allocation을 순서대로 선택하고 서버의 fixed IPv4에 연결합니다. 명시 network 또는 공유 floating 역할·router gateway에서 외부 network를 선택하며, 조건에 따라 공유 NAT 역할로 destination을 좁힙니다. 알려진 revision으로 PUT을 보호하고 실패 시 선택/할당 결과를 보존합니다. 자동 필요성 판단·pool·순차 IPv4와 raw Nova 관측은 [Compute IP dispatch](../compute/server-ip-dispatch.md)에서 제공합니다. direct proxy add/remove·unattached cleanup·함수별 fallback과 전체 cloud 서비스·Resource/session/lifecycle은 남은 범위입니다. Neutron의 404·403·409는 원래 HTTP 오류를 보존하며 다른 API로 전환하지 않습니다. 지정한 기존 IP의 고정 연결은 [FloatingIPs.Attach](floating-ip-attach.md)로 제공하며 해제·일반 수정은 `service.API.FloatingIPs.Update`에서 개별 호출할 수 있습니다. 이 범위를 Python cloud 계층 전체와 동일하게 구현했다고 간주하지 않습니다.

[network_test.go](network_test.go)는 추가 query의 URL 인코딩과 이름을 통한 삭제를, [floating_ip_test.go](floating_ip_test.go)는 이름 해석·다중 페이지 포트 선택·IPv4 조건·중복·명시 포트·extension·상태 대기·생성 후 실패 보존을, [Ensure·선택 테스트](floating-ip-ensure.md#고정-소스와-남은-범위)는 기존 IP 재사용·owner·revision·빈 페이지·부분 실패를, [전체 통합 테스트](../collections_test.go)는 서비스 공통 정책을 검증합니다.

선택과 실행을 나누려면 [PrepareEnsure·EnsurePrepared](floating-ip-plan.md)를 사용합니다. 소유 프로젝트를 선조회하지 않고 외부 network·server-owned port·fixed IPv4를 immutable plan으로 준비하고, 실행 시 해당 port를 재검증하여 같은 선택으로 연결합니다. 전체 페이지와 revision·201/202 접수 증거를 보존하며, 원본 변경이나 부분 실패를 재선택·자동 삭제로 바꾸지 않습니다. [Compute의 자동 floating IPv4](../compute/server-automatic-ip.md)는 이 계획을 같은 호출의 필요성 판단·조건부 실행·raw Nova 관측에 사용합니다.

서버 생성부터 floating IP 연결·ACTIVE 대기까지 공통 deadline으로 실행하려면 Compute의 [CreateWithFloatingIP](../compute/create-with-floating-ip.md)를 사용합니다. 실패 시 실제 서버와 이 서비스의 IP assignment를 함께 보존합니다.

[네트워크 역할 조회](network-roles.md)는 외부·내부 IPv4/IPv6, floating source, NAT destination과 기본 인터페이스를 SDK가 설정과 Neutron 응답에서 분류합니다. Getter, configured 기본 NIC, Ensure의 자동 외부 network와 Create/Ensure의 조건부 NAT 선택이 성공 snapshot을 공유하고, 반환한 모델은 각 호출자가 소유합니다. `WithNetworkRoles`로 설정을 교체하고 `ResetNetworkRoles`로 cache를 갱신할 수 있습니다. IP 후보와 서버 port는 호출마다 조회합니다. [상위 네트워크 CRUD](network-mutations.md)의 접수된 변경은 같은 cache를 자동 초기화합니다. Raw/native/API·외부 변경은 명시 Reset이 필요합니다. [서버 주소 계산·보충](../compute/server-addresses.md)은 구현되어 있으며, [기존 서버의 조건부 Neutron assignment·raw Nova 관측](../compute/server-automatic-ip.md)도 제공합니다. 전체 cloud lifecycle·direct proxy add/remove·unattached cleanup·서비스 가용성 정책은 남은 범위입니다.

[Compute의 legacy Nova backend](../compute/server-nova-floating-ip.md)는 Nova pool/list/allocation/add action을 별도 `NovaAssignment`로 반환합니다. 직접 Network API와 immutable Neutron plan의 모델·목적지·revision·ACTIVE 계약은 그대로 사용합니다.

[FloatingIPs.Available](floating-ip-available.md)은 외부 network·project에서 첫 free IP를 반환하거나 새로 할당합니다. free 재사용은 Server를 지정해도 연결하지 않으며, 새 allocation의 optional Server만 ports·fixed IPv4·NAT 선택에 사용합니다. 설정에 따른 backend 선택과 pure NotFound 전환은 [Connection·Compute Available](../compute/floating-ip-available.md)에서 제공합니다.

[Connection·Compute의 Floating IP 조회](../compute/floating-ip-queries.md)는 cloud의 목록·검색·단건·pool API6개입니다. Neutron에서는 known query와 Resource Body descriptor view를 사용하고 legacy Nova fallback·pool은 Compute로 연결합니다. 이 facade와 직접 Network Proxy/Resource 전체 지원은 별도로 추적합니다.

[Connection·Compute Floating IP 삭제](../compute/floating-ip-delete.md)는 cloud의 backend 선택과 기본 재검증을 제공하며, Neutron DELETE404에서 Nova로 전환하지 않습니다. 직접 Network의 generic Delete와 별도 반환 계약입니다.

[독립 Floating IP 생성](../compute/floating-ip-create.md)은 가용 IP를 재사용하지 않고 새 allocation을 생성합니다. SDK가 Neutron/Nova·port 우선·optional server/NAT·공개 Get 대기·wait timeout 정리를 처리하며, 접수된 응답과 compatibility/wait/cleanup 부분 결과를 보존합니다.

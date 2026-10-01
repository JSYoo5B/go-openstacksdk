# Network

Neutron 네트워크·포트·floating IP의 공통 조회 정책과 floating IP 생성·서버 연결 작업을 제공합니다. `conn`과 `ctx`는 [전체 README](../README.md)처럼 준비합니다. 아래 Go 조각은 `fmt`, `resource` 등을 import한 오류 반환 함수 안에서 사용합니다.

## openstacksdk 대응

| openstacksdk | gophercloudsdk |
|---|---|
| `conn.network.get_network(id)` | `service.Networks.Get(ctx, id)` |
| `conn.network.find_network(name, ignore_missing=False)` | `service.Networks.Find(ctx, resource.Name(name))` |
| `conn.network.networks(status="ACTIVE")` | `service.Networks.List(ctx, resource.WithStatus("ACTIVE"))` |
| `conn.network.networks(is_router_external=True)` | `service.Networks.List(ctx, resource.WithQuery("router:external", "true"))` |
| `conn.network.delete_network(id)` | `service.Networks.Delete(ctx, resource.ID(id))` |
| `conn.network.find_router(name_or_id, ignore_missing=False)` | `service.API.Routers.FindIdentity(ctx, nameOrID, resource.WithIdentityFindIgnoreMissing(false))` |
| `conn.network.find_security_group(name_or_id, project_id=projectID)` | `service.API.SecurityGroups.FindIdentity(ctx, nameOrID, resource.WithIdentityFindQuery("project_id", projectID))` |
| `conn.network.find_subnet_pool(name_or_id)` | `service.API.SubnetPools.FindIdentity(ctx, nameOrID)` |
| `conn.network.find_trunk(name_or_id)` | `service.API.Trunks.FindIdentity(ctx, nameOrID)` |
| `conn.network.get_port(id)` | `service.Ports.Get(ctx, id)` |
| `conn.network.get_ip(id)` | `service.FloatingIPs.Get(ctx, id)` |
| `conn.create_floating_ip(network="public", server=server, wait=True)` | `service.FloatingIPs.Create(ctx, request, network.WithServer(ref), network.WithWait())` |

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
Python 속성 별칭과 로컬 Body 필터 분류는 자동 적용하지 않습니다.
[Python/Go 사용 예제](../docs/finding-identities.md#neutron-routersecurity-group와-project-query)를 참고하세요.

## Subnet Pool·Trunk 자동 조회

`service.API.SubnetPools.FindIdentity`와 `service.API.Trunks.FindIdentity`도 같은
이름·ID 옵션을 사용합니다. 프로젝트 필터는 `WithIdentityFindQuery("project_id", id)`로
지정하며 SDK가 자동 추론하지 않습니다. native 모델·오류·페이지 경계를 유지합니다.
Subnet Pool은 세 prefix 길이의 string/number decode가 필요하므로 `fields=id,name`처럼
이 값을 제외한 응답은 오류입니다. Trunk의 native pager는 `links.next`, Subnet Pool은
`subnetpools_links`를 따릅니다. [Python/Go 예제와 경계](../docs/finding-identities.md#neutron-subnet-pooltrunk)를 참고하세요.

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

위 복합 작업은 Neutron에서 새 IPv4 floating IP를 생성합니다. 기존 floating IP 재사용·가용 IP 선택, Nova-network 자동 fallback, clouds.yaml 기반 외부 네트워크·NAT destination 자동 선택은 아직 제공하지 않습니다. Neutron의 404·403·409는 원래 HTTP 오류를 보존하며 다른 API로 전환하지 않습니다. 기존 floating IP 연결·해제·수정은 `service.API.FloatingIPs.Update`에서 개별 호출할 수 있습니다. 이 범위를 Python cloud 계층 전체와 동일하게 구현했다고 간주하지 않습니다.

[network_test.go](network_test.go)는 추가 query의 URL 인코딩과 이름을 통한 삭제를, [floating_ip_test.go](floating_ip_test.go)는 이름 해석·다중 페이지 포트 선택·IPv4 조건·중복·명시 포트·extension·상태 대기·생성 후 실패 보존을, [전체 통합 테스트](../collections_test.go)는 서비스 공통 정책을 검증합니다.

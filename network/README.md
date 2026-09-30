# Network

Neutron 네트워크의 조회, iterator, 삭제, 상태 대기를 제공합니다. `conn`과 `ctx`는 [전체 README](../README.md)처럼 준비합니다. 아래 Go 조각은 `fmt`, `resource` 등을 import한 오류 반환 함수 안에서 사용합니다.

## openstacksdk 대응

| openstacksdk | gophercloudsdk |
|---|---|
| `conn.network.get_network(id)` | `service.Networks.Get(ctx, id)` |
| `conn.network.find_network(name, ignore_missing=False)` | `service.Networks.Find(ctx, resource.Name(name))` |
| `conn.network.networks(status="ACTIVE")` | `service.Networks.List(ctx, resource.WithStatus("ACTIVE"))` |
| `conn.network.networks(is_router_external=True)` | `service.Networks.List(ctx, resource.WithQuery("router:external", "true"))` |
| `conn.network.delete_network(id)` | `service.Networks.Delete(ctx, resource.ID(id))` |

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

## 전체 API와 남은 복합 작업

네트워크 생성·수정, subnet, port, router, security group, floating IP의 개별 호출은 `service.API`의 [Network v2 API](v2/README.md)에서 제공합니다. 예를 들어 `service.API.Networks.Create(ctx, networks.CreateOpts{...})`와 `service.API.FloatingIPs.Create(ctx, floatingips.CreateOpts{...})`는 SDK가 제공하는 concrete options를 사용하며 builder interface 구현이 필요하지 않습니다. 각각 `gophercloudsdk/network/v2/networks`, `gophercloudsdk/network/v2/extensions/layer3/floatingips`를 import합니다.

서버의 포트 선택부터 floating IP 선택·재사용·연결·대기까지 묶는 복합 작업은 아직 없습니다. 개별 FloatingIP API가 이 전체 작업을 수행하는 것으로 간주하지 않습니다.

[network_test.go](network_test.go)는 추가 query의 URL 인코딩과 이름을 통한 삭제를, [전체 통합 테스트](../collections_test.go)는 서비스 공통 정책을 검증합니다.

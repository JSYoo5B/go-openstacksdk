# Compute

Nova 서버와 flavor를 제공합니다. 연결은 [전체 README](../README.md)의 `sdk.Connect(ctx, ...)`로 준비합니다. 아래 Go 조각은 오류를 반환하는 함수 안에서 사용하며, `fmt`, `time`, `compute`, `resource`를 필요한 만큼 import합니다.

## openstacksdk 대응

| openstacksdk | gophercloudsdk |
|---|---|
| `conn.compute.get_server(id)` | `service.Servers.Get(ctx, id)` |
| `conn.compute.find_server(name, ignore_missing=False)` | `service.Servers.Find(ctx, resource.Name(name))` |
| `conn.compute.servers(status="ACTIVE")` | `service.Servers.List(ctx, resource.WithStatus("ACTIVE"))` |
| `conn.compute.delete_server(id)` | `service.Servers.Delete(ctx, resource.ID(id))` |
| `conn.compute.wait_for_server(server, status="ACTIVE", wait=300)` | `service.Servers.Wait(ctx, resource.ID(server.ID), "ACTIVE", resource.WithTimeout(5*time.Minute))` |
| `conn.compute.find_flavor(name, ignore_missing=False)` | `service.Flavors.Find(ctx, resource.Name(name))` |
| `conn.compute.flavors()` | `service.Flavors.List(ctx)` |
| `conn.create_server(...)` | `service.Servers.Create(ctx, compute.CreateServerRequest{...}, ...)` |

마지막 행은 의존 리소스를 해석하는 상위 작업의 대응입니다. Python의 `conn.compute.create_server(**attrs)`는 이미 준비한 API 속성을 전달하는 Proxy 작업이므로 상위 `conn.create_server(...)`와 구분해야 합니다. [공식 Compute API](https://docs.openstack.org/openstacksdk/latest/user/proxies/compute.html)

## 서버 조회와 목록

Python:

```python
server = conn.compute.find_server("web-01", ignore_missing=False)
for server in conn.compute.servers(status="ACTIVE"):
    print(server.id, server.name)
```

Go:

```go
service, err := conn.Compute(ctx)
if err != nil { return err }
server, err := service.Servers.Find(ctx, resource.Name("web-01"))
if err != nil { return err }
fmt.Println(server.ID)

for server, err := range service.Servers.List(ctx, resource.WithStatus("ACTIVE")) {
    if err != nil { return err }
    fmt.Println(server.ID, server.Name)
}
```

`WithName`은 Nova의 정규표현식 대신 정확한 이름으로 처리합니다. `web[1].*`처럼 특수문자가 있는 이름도 문자 그대로 비교합니다.

## 이름 해석을 포함한 생성

Go의 생성 예제:

```go
server, err := service.Servers.Create(ctx, compute.CreateServerRequest{
    Name: "web-01",
    Image: resource.Name("ubuntu"),
    Flavor: resource.Name("small"),
}, compute.WithNetworks(resource.Name("private")),
   compute.WithKeyName("my-key"),
   compute.WithConfigDrive(false),
   compute.WithWait(resource.WithTimeout(5*time.Minute)))
if err != nil {
    if server != nil { fmt.Println("created:", server.ID) }
    return err
}
```

이미지와 네트워크의 이름은 연결이 제공한 서비스에서, flavor는 Compute에서 찾습니다. 이름이 중복되거나 없으면 POST 전에 실패합니다. ID를 지정한 의존성은 사전 조회하지 않습니다. 기본 네트워크 선택은 Nova에 맡기며, `WithNetworks()`의 빈 선택은 오류입니다.

현재 옵션은 `WithMetadata`, `WithKeyName`, `WithUserData`, `WithConfigDrive`, `WithAvailabilityZone`, `WithSecurityGroups`, `WithNetworks`, `WithWait`, `WithField`입니다. UserData의 인코딩은 Gophercloud가 처리합니다. map/slice/확장 JSON 입력은 옵션 생성 시 복사하여 이후 애플리케이션의 변경으로 요청이 달라지지 않게 합니다.

## 대기와 삭제

```go
ready, err := service.Servers.Wait(ctx, resource.ID(server.ID), "ACTIVE",
    resource.WithTimeout(5*time.Minute), resource.WithPollInterval(time.Second))
if err != nil { return err }
fmt.Println(ready.Status)

if err := service.Servers.Delete(ctx, resource.ID(server.ID)); err != nil {
    return err
}
```

`ERROR`면 대기를 즉시 종료합니다. 생성 후 대기에 실패해도 이미 생성된 서버를 자동 삭제하지 않고, 생성 응답과 오류를 함께 반환합니다. Delete는 요청의 성공까지 확인하며 서버가 사라질 때까지 대기하지는 않습니다.

## 확장과 미지원 범위

`compute.WithField("vendor_hint", value)`로 기본 생성 필드와 충돌하지 않는 확장 JSON을 전달할 수 있습니다. 예시 필드는 표준 Nova API가 아닙니다. builder는 SDK 내부에서 구현합니다. 필드 스키마·지원 여부·microversion은 실제 API가 검증합니다.

flavor는 조회만 지원합니다. 서버 Update, reboot/resize 등의 action, keypair 관리, floating IP 연결, boot-from-volume은 아직 상위 계층에 없습니다. `service.RawClient()`를 이용한 Gophercloud 호출은 가능합니다.

테스트는 [compute_test.go](compute_test.go), 전체 생성 흐름은 [server_create_test.go](../server_create_test.go), 페이지·이름·대기 정책은 [collections_test.go](../collections_test.go)에 있습니다.

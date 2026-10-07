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
| `conn.compute.find_flavor(name_or_id, ignore_missing=False)` | `service.Flavors.FindIdentity(ctx, nameOrID, resource.WithIdentityFindIgnoreMissing(false))` |
| `conn.compute.find_flavor(name_or_id, get_extra_specs=True)` | 위 자동 조회에 `resource.WithIdentityFindExtraSpecs(true)` 추가 |
| `conn.compute.flavors()` | `service.Flavors.List(ctx)` |
| `conn.create_server(...)` | `service.Servers.Create(ctx, compute.CreateServerRequest{...}, ...)` |
| `conn.add_ips_to_server(server, ip_pool="public")` | [AddIPsToServer](server-ip-helpers.md): 기본60초·비동기, pool → IP 목록 → auto |
| `conn.add_ip_list(server, ips)` | [AddIPList](server-ip-helpers.md): positional 목록·중복·빈 목록 no-op, 선택적인 raw 주소 관측 |
| `conn.get_server_public_ip(server)` / `get_server_private_ip(server)` | [주소 선택·보충](server-addresses.md): owned view·MAC·IPv6·private 설정과 기존 association 조회 |
| `conn.create_server(..., auto_ip=True, wait=True)` | [CreateWithAutomaticFloatingIP](create-with-automatic-floating-ip.md): 실제 ACTIVE·주소 준비·조건부 IP·Nova 관측 |
| `conn.get_active_server(server, wait=False)` | [GetActiveServer](server-ready.md): supplied 상태 판정·조건부 IP 접수; 선택적 ACTIVE/주소 관측 |
| `conn.wait_for_server(server, timeout=180)` | [Service·Connection WaitForServer](server-ready.md): 같은 ID의 raw 현재 상태·주소 준비·필요한 IP 관측 |
| `conn.create_server(..., ip_pool="public", wait=True)` | [CreateWithAutomaticFloatingIP](create-with-automatic-floating-ip.md)의 `AutomaticIP`에 `compute.WithFloatingIPPool(resource.Name("public"))`; [pool·IP dispatch](server-ip-dispatch.md) |
| `conn.create_server(..., ips=["203.0.113.10", "203.0.113.11"], wait=True)` | 같은 `AutomaticIP`에 `compute.WithFloatingIPAddresses(...)`; 순차 연결·관측과 `Attempts` 부분 결과 |
| `conn.create_server(..., boot_volume=volume, terminate_volume=False)` | `service.Servers.Create(ctx, request, compute.WithBootVolume(volumeRef))` |
| `conn.create_server(..., boot_volume=volume, terminate_volume=True)` | 위 호출에 `compute.WithDeleteBootVolumeOnTermination(true)` 추가 |
| `conn.create_server(..., image=image, boot_from_volume=True, volume_size=50)` | 이미지가 있는 요청에 `compute.WithBootVolumeSize(50)` 추가 |

생성 행들은 의존 리소스를 해석하는 상위 작업의 대응입니다. Python의 `conn.compute.create_server(**attrs)`는 이미 준비한 API 속성을 전달하는 Proxy 작업이므로 상위 `conn.create_server(...)`와 구분해야 합니다. [공식 Compute API](https://docs.openstack.org/openstacksdk/latest/user/proxies/compute.html)

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

## Flavor 자동 조회

`service.Flavors.FindIdentity(ctx, "small")`는 ID GET부터 시도한 뒤 필요하면 전체
상세 목록을 정확한 ID/이름으로 검색합니다. 자동 이름 query를 보내지 않고 목록에서만
기본 `is_public=None`을 적용하며, caller의 명시 wire 값은 보존합니다.
`resource.WithIdentityFindExtraSpecs(true)`를 지정하면 단일 결과의 ExtraSpecs가
비어 있을 때만 반환된 ID의 extra-specs GET을 추가합니다. 기본값은 false이고, 후속
GET 실패는 IgnoreMissing으로 숨기지 않습니다. [Python/Go 예제와 옵션](../docs/finding-identities.md#nova-flavor와-extra-specs)을 참고하세요.

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

이미지와 네트워크의 이름은 연결이 제공한 서비스에서, flavor는 Compute에서 찾습니다. 이름이 중복되거나 없으면 POST 전에 실패합니다. ID를 지정한 의존성은 사전 조회하지 않습니다. 명시한 NIC/mode는 [Connection·YAML 기본 네트워크](server-default-network.md)보다 우선합니다. 이 기본값도 없고 선택된 Compute microversion 2.37 이상에서 네트워크 선택을 생략하면 `networks: "auto"`를 보냅니다. 이전 버전이나 버전 미설정에서는 필드를 생략합니다. `WithNetworks()`의 빈 선택은 오류입니다. [NIC 선택 가이드](server-network-interfaces.md)는 기존 port·고정 IP·tag와 auto/none mode, 선택 교체·기본값과 Python 차이를 설명합니다.

현재 옵션은 `WithMetadata`, `WithKeyName`, `WithUserData`, `WithConfigDrive`, `WithAvailabilityZone`, `WithSecurityGroups`, `WithNetworks`, `WithNetworkInterfaces`, `WithNetworkMode`, `WithBootVolume`, `WithBootVolumeSize`, `WithBootVolumeType`, `WithDeleteBootVolumeOnTermination`, `WithWait`, `WithField`입니다. UserData의 인코딩은 Gophercloud가 처리합니다. map/slice/확장 JSON 입력은 옵션 생성 시 복사하여 이후 애플리케이션의 변경으로 요청이 달라지지 않게 합니다.

## 기존 볼륨으로 부팅

Python:

```python
server = conn.create_server(
    name="volume-server",
    flavor="small",
    boot_volume="root-volume",
    terminate_volume=False,
    network="private",
    auto_ip=False,
    wait=True,
)
```

Go:

```go
server, err := service.Servers.Create(ctx, compute.CreateServerRequest{
    Name: "volume-server",
    Flavor: resource.Name("small"),
}, compute.WithBootVolume(resource.Name("root-volume")),
   compute.WithNetworks(resource.Name("private")),
   compute.WithWait(resource.WithTimeout(5*time.Minute)))
if err != nil {
    if server != nil { fmt.Println("created:", server.ID) }
    return err
}
```

`Image`를 비우고 `WithBootVolume`으로 이미 존재하는 부팅 볼륨을 선택합니다. 연결은 Block Storage의 정확한 이름 조회를 제공하며, 중복 이름이나 누락은 서버 생성 전에 오류로 처리합니다. `resource.ID("volume-id")`는 볼륨 사전 조회를 생략합니다. 볼륨이 bootable이고 사용 가능한지에 대한 검증은 Nova와 Cinder가 수행합니다.

SDK는 `imageRef: ""`와 `source_type: "volume"`, `destination_type: "volume"`, `boot_index: 0`인 `block_device_mapping_v2`를 작성합니다. 별도의 builder나 매핑 구조체 구현은 필요하지 않습니다. 기본 `delete_on_termination`은 `false`이며, 서버 삭제 시 볼륨도 지우려면 `WithDeleteBootVolumeOnTermination(true)`를 추가합니다. `false`도 실제 JSON에 포함됩니다.

이미지와 기존 볼륨을 함께 지정하거나 둘 다 생략하면 조회를 포함한 모든 HTTP 요청 전에 실패합니다. 볼륨 부팅을 선택하지 않고 삭제 옵션만 지정하는 것도 오류입니다. Python의 `boot_volume`은 이미지 입력보다 우선하지만 Go는 상충하는 입력을 조기에 알려줍니다.

생성 후 대기 실패는 생성 응답과 오류를 함께 반환합니다. SDK는 이 경우 서버나 볼륨을 삭제하지 않습니다. 옵션은 이후 서버가 삭제될 때 Nova가 적용하는 정책이며 대기 실패 시의 정리 정책이 아닙니다.

## 이미지에서 새 부팅 볼륨 생성

Python:

```python
server = conn.create_server(
    name="new-volume-server",
    image="ubuntu",
    flavor="small",
    boot_from_volume=True,
    volume_size=50,
    terminate_volume=False,
    auto_ip=False,
    wait=True,
)
```

Go:

```go
server, err := service.Servers.Create(ctx, compute.CreateServerRequest{
    Name: "new-volume-server",
    Image: resource.Name("ubuntu"),
    Flavor: resource.Name("small"),
}, compute.WithBootVolumeSize(50),
   compute.WithWait(resource.WithTimeout(5*time.Minute)))
if err != nil {
    if server != nil { fmt.Println("created:", server.ID) }
    return err
}
```

`WithBootVolumeSize`를 지정하면 해석한 이미지 ID를 `source_type: "image"`, `destination_type: "volume"`, `boot_index: 0`, `volume_size: 50`인 매핑으로 옮기고 `imageRef`는 비웁니다. Nova가 서버 생성 요청의 일부로 볼륨을 생성하므로 SDK는 별도 Cinder 생성 요청을 보내지 않습니다. 이미지 ID는 사전 조회하지 않으며, 이름은 Image 서비스에서 정확하게 찾습니다.

Python은 `volume_size`를 생략하면 50을 사용합니다. Go는 새 볼륨의 용량을 `WithBootVolumeSize`에 양의 GiB 단위 정수로 명시합니다. 기존 `WithBootVolume`과 새 볼륨 용량을 동시에 지정하거나, 이미지 없이 새 볼륨 부팅을 요청하면 HTTP 전에 실패합니다. 이미지 최소 크기와 볼륨 할당량은 클라우드가 검증합니다.

새 볼륨도 `delete_on_termination: false`가 기본값이므로 서버 삭제 후 유지됩니다. 서버 삭제 시 새 볼륨도 삭제하려면 `WithDeleteBootVolumeOnTermination(true)`를 추가합니다. 대기 실패 시 자동 삭제는 수행하지 않습니다.

`WithBootVolumeType("fast")`는 새 부팅 볼륨의 타입을 지정합니다. 이 옵션은 `WithBootVolumeSize`와 Compute microversion 2.67 이상이 필요합니다. 클라이언트 버전이 비설정이거나 2.67 미만이면 이미지 조회를 포함한 HTTP 전에 `resource.ErrUnsupported`를 반환합니다. 연결에 `sdk.WithMicroversion(sdk.Compute, "2.67")`을 설정하거나 해당 하한을 만족하는 버전 범위를 선택하세요. 타입을 생략하면 클라우드의 기본 볼륨 타입을 사용하며, 타입의 존재 여부와 접근 권한은 Cinder가 검증합니다.

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

flavor는 상위 계층에서 조회를 지원합니다. 서버 Update, reboot/resize 등의 action, keypair 관리에는 `service.API`의 [전체 Compute API](v2/README.md)를 사용할 수 있습니다. floating IP 생성·재사용·연결은 Network의 `FloatingIPs`와 아래 `CreateWithFloatingIP`으로 제공합니다. `service.RawClient()`를 이용한 Gophercloud 호출도 가능합니다.

테스트는 [compute_test.go](compute_test.go), 기존 볼륨 부팅의 요청·검증·실패 정책은 [boot_volume_test.go](boot_volume_test.go), 새 부팅 볼륨과 microversion 정책은 [new_boot_volume_test.go](new_boot_volume_test.go), 연결을 통한 전체 생성 흐름은 [server_create_test.go](../server_create_test.go), 페이지·이름·대기 정책은 [collections_test.go](../collections_test.go)에 있습니다.

Server의 `WaitForServer(ctx, ref)`는 ACTIVE·ERROR·120초를 기본으로 사용합니다. `WaitForServerState`는 다른 대상을 120초 기본으로, `WaitForState`는 대상을 명시하고 SDK timeout 없이 기다립니다. `WaitForDelete`는 삭제 요청 없이 기본 120초 동안 삭제 완료를 관찰합니다. 패키지 함수 `compute.WaitForState/WaitForDelete`는 기존 typed collection을 받습니다. [서비스별 대기 비교](../docs/service-waits.md)에 옵션·context·Python 대응과 남은 차이를 설명합니다.

[기존 서버 readiness](server-ready.md)의 `service.WaitForServer(ctx, AutomaticFloatingIPRequest, ...)`와 `conn.WaitForServer`는 별도 상위 작업입니다. 기본180초·5초로 raw 현재 상태와 주소 준비를 기다리고 필요할 때 조건부 Neutron/Nova assignment·Neutron IP ACTIVE·raw Nova 관측까지 수행합니다. `GetActiveServer`는 supplied 상태만 판정하고 기본 비동기 IP 접수 결과를 반환합니다.

서버 생성과 floating IP 재사용·연결을 한 작업으로 수행하려면 [CreateWithFloatingIP](create-with-floating-ip.md)를 사용합니다. 기본으로 실제 서버와 IP의 ACTIVE를 기다리며, Connection이 서비스와 기본 reuse project를 준비합니다. 전체 deadline과 서버/IP 부분 결과를 제공하고 일반 `Create`의 비동기 동작은 유지합니다. 필요성 판단·순차 IP/pool 선택과 raw Nova 관측은 [상위 IP dispatch](server-ip-dispatch.md)로 제공하며, 전체 cloud Resource/session·서비스 정책은 계속 추적합니다.

Connection의 [네트워크 역할 설정·조회](../network/network-roles.md)는 family·NAT·default 역할을 함께 제공합니다. Configured default는 getter 및 floating source/NAT 선택과 같은 성공 snapshot을 사용하며, 명시 NIC와 기존 `WithDefaultNetwork`/`WithoutDefaultNetwork`가 먼저입니다. 명시 `WithDefaultNetwork(resource.Name(...))`은 기존 exact-name 조회를 생성마다 수행합니다.

[서버 주소 가이드](server-addresses.md)는 조회한 Nova 모델의 public/private 주소와 기존 Floating IP 보충, default interface·IPv6·접속 검사 정책을 비교합니다. Connection이 설정과 공유 역할 cache를 제공합니다. [자동 생성 흐름](create-with-automatic-floating-ip.md)에서 생성·대기와 조건부 할당·주소 수렴을 연결합니다. 일반 Create/Wait 전체 계약은 계속 별도로 추적합니다.

[기존 서버의 자동 floating IPv4](server-automatic-ip.md)는 `PlanServerFloatingIP`로 읽기 전용 필요성을 판단하고 `EnsureServerFloatingIP`로 조건부 Neutron/Nova assignment와 raw Nova 주소 관측을 수행합니다. Connection이 설정·서비스·공유 역할 snapshot을 제공하고 오류 시 알려진 Server·Assignment를 보존합니다. [CreateWithAutomaticFloatingIP](create-with-automatic-floating-ip.md)는 이 정책을 생성·ACTIVE 대기에 연결하고 초기 생성 응답과 마지막 서버를 보존합니다. [pool·순차 IP dispatch](server-ip-dispatch.md)는 같은 Plan/Ensure·GetActive/Wait·자동 생성 메서드의 `WithFloatingIPPool`·`WithFloatingIPAddresses`로 사용합니다. 일반 Create/Wait 전체 계약과 Nova의 별도 공개 CRUD·detach/cleanup·함수별 fallback·전체 cloud Resource/session은 별도 remaining입니다.

[Legacy Nova floating IP](server-nova-floating-ip.md)는 같은 Service/Connection IP 소비자의 Nova backend를 제공합니다. 명시 pool/IP의 configured Nova·None 또는 정확한 Network endpoint 부재에서 실행하며, 자동 source=None은 skip을 유지합니다. pool은 literal 값이고 `NovaAssignment`로 실제 모델과 action202 증거를 읽습니다. selected Compute2.36 이상과 Neutron 전용 port/NAT/project override는 `ErrUnsupported`입니다. 동기 상위 entry는 실제 서버 ACTIVE·목표 주소를 확인하며 Neutron IP ACTIVE는 Neutron backend에만 적용합니다.

[독립 Floating IP 조회·할당](floating-ip-available.md)은 `service.AvailableFloatingIP`과 `conn.AvailableFloatingIP`으로 제공합니다. 설정과 호출별 source 옵션, 선택적인 Network 옵션, 실제 backend·재사용/할당 증거를 반환하며 서버 action이나 readiness wait를 수행하지 않습니다. free Neutron IP는 optional Server와 무관하게 반환하고 새 Neutron allocation만 Server를 사용합니다.

[독립 Floating IP 목록·검색·단건·pool 조회](floating-ip-queries.md)는 `ListFloatingIPs`, `SearchFloatingIPs`, `GetFloatingIP`, `GetFloatingIPByID`, `ListFloatingIPPools`, `SearchFloatingIPPools`를 Service와 Connection에 제공합니다. 기본 Get은 목록 검색, UUID direct는 옵션으로 선택합니다. Neutron dict Search는 ID를 무시하며, Nova source 정규화의 합성 ACTIVE는 실제 연결 상태와 별도로 읽습니다.

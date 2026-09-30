# gophercloudsdk

Gophercloud 위에 연결, 서비스, 리소스, 복합 작업의 일관된 사용 방식을 제공하는 Go SDK 프로젝트입니다. 애플리케이션이 기본값, 이름 조회, 페이지네이션, 상태 대기, 요청 builder를 반복해서 구현하지 않도록 하는 것이 목적입니다.

현재는 **개발 중**입니다. 고정한 Gophercloud의 공개 API 호출은 제공하며, openstacksdk 수준의 리소스·복합 작업 계층을 확장하고 있습니다. Go 1.25 이상과 Gophercloud v2.15.0을 사용합니다. 모듈 이름 `gophercloudsdk`는 로컬 개발용이며, 저장소 공개 시 실제 모듈 경로로 변경해야 합니다. 옆 디렉토리의 개발 브랜치에 의존하는 `replace`는 사용하지 않습니다.

## 디렉토리와 지원 범위

```text
gophercloudsdk/
├── connection*.go           # 인증, 설정, 21개 서비스 접근과 캐시
├── compute/                 # 서버, flavor, 서버 생성 흐름
├── network/                 # Neutron 네트워크
├── image/                   # Glance 이미지
├── blockstorage/            # Cinder v3 볼륨
├── identity/, dns/, ...     # 서비스별 버전 패키지와 전체 API
├── resource/                # 공통 조회, iterator, 대기, 옵션, 오류
├── request/                 # SDK 소유 옵션, 확장 필드·query·header
├── api/                     # API/리소스 지원 목록과 HTTP 계약 테스트
├── internal/cmd/sdkgen/     # API, 공통 정책, 참조 문서 생성
├── internal/testcloud/      # HTTP 테스트 fixture
├── examples/create-server/  # 빌드 가능한 실행 예제
└── docs/                    # 설계와 테스트 설명
```

전체 API는 **21개 서비스의 23개 API 버전**, **194개 리소스 패키지**, **1,126개 공개 연산**을 제공합니다. 공통 리소스 정책은 107개 일반 Collection과 15개 부모 범위에 적용합니다. 연산 수에는 인증 함수와 URL 도우미도 포함되며 HTTP endpoint 수를 뜻하지 않습니다. [API 설명](api/README.md), [공통 정책 지원 목록](api/resource_inventory.json), [openstacksdk 비교 기준](api/openstacksdk/README.md)에서 범위를 확인합니다.

아래 표는 추가 이름 해석과 서버 생성 흐름을 제공하는 기존 상위 서비스의 범위입니다. 모든 API와 공통 정책은 이어지는 버전별 서비스 패키지에 있습니다.

| 서비스 | 리소스 | Get / Find / List / All | Delete | Wait | Create |
|---|---|---|---|---|---|
| [Compute](compute/README.md) | 서버 | 지원 | 지원 | 지원 | 이미지·기존 볼륨·이미지에서 만든 새 볼륨 부팅, 이름 해석, 선택적 대기 |
| Compute | flavor | 지원 | 미지원 | 미지원 | 미지원 |
| [Network](network/README.md) | 네트워크 | 지원 | 지원 | 지원 | 미지원 |
| [Image](image/README.md) | 이미지 | 지원 | 지원 | 지원 | 미지원 |
| [Block Storage](blockstorage/README.md) | 볼륨 | 지원 | 지원 | 지원 | 미지원 |

Create/Update와 각 서비스의 API 호출은 버전별 패키지에서 concrete options로 사용합니다. [microversion 범위 협상](docs/microversions.md)과 [볼륨 부팅 옵션](compute/README.md)을 제공하며, 응답 변경 추적과 자동 commit, floating IP 연결을 포함한 복합 작업, 이미지 업로드의 상위 흐름은 계속 구현할 대상입니다. [SDK 지원 판정대장](docs/sdk-support-ledger.md)은 확인한 차이와 전체 완료의 기준을 기록합니다.

## 모든 서비스의 사용 문서

| 서비스 | API 버전 문서 | Connection |
|---|---|---|
| Bare Metal | [v1](baremetal/v1/README.md) | `BareMetal(ctx)` |
| Bare Metal Introspection | [v1](baremetalintrospection/v1/README.md) | `BareMetalIntrospection(ctx)` |
| Block Storage | [v2](blockstorage/v2/README.md), [v3](blockstorage/v3/README.md) | `BlockStorageV2(ctx)`, `BlockStorageV3(ctx)` |
| Compute | [v2](compute/v2/README.md) | `ComputeV2(ctx)` |
| Container | [v1](container/v1/README.md) | `Container(ctx)` |
| Container Infra | [v1](containerinfra/v1/README.md) | `ContainerInfra(ctx)` |
| Database | [v1](db/v1/README.md) | `Database(ctx)` |
| DNS | [v2](dns/v2/README.md) | `DNS(ctx)` |
| Identity | [v2](identity/v2/README.md), [v3](identity/v3/README.md) | `IdentityV2(ctx)`, `Identity(ctx)` |
| Image | [v2](image/v2/README.md) | `ImageV2(ctx)` |
| Key Manager | [v1](keymanager/v1/README.md) | `KeyManager(ctx)` |
| Load Balancer | [v2](loadbalancer/v2/README.md) | `LoadBalancer(ctx)` |
| Messaging | [v2](messaging/v2/README.md) | `Messaging(ctx)` |
| Metric (Aetos) | [v1](metric/v1/README.md) | `Metric(ctx)` |
| Object Storage | [v1](objectstorage/v1/README.md) | `ObjectStorage(ctx)` |
| Orchestration | [v1](orchestration/v1/README.md) | `Orchestration(ctx)` |
| Placement | [v1](placement/v1/README.md) | `Placement(ctx)` |
| Network | [v2](network/v2/README.md) | `NetworkV2(ctx)` |
| Reservation | [v1](reservation/v1/README.md) | `Reservation(ctx)` |
| Shared File System | [v2](sharedfilesystems/v2/README.md) | `SharedFileSystem(ctx)` |
| Workflow | [v2](workflow/v2/README.md) | `Workflow(ctx)` |

각 문서에는 openstacksdk와의 입력·결과 형식 비교, 실제 서비스 필드, API 패키지 링크, 이름 조회·삭제·대기가 적용되는 리소스를 기록합니다. DNS zone이나 Octavia pool의 자식 리소스는 [부모 범위를 지정](docs/scoped-resources.md)해서 사용합니다.

## openstacksdk와 전체 사용 방식 비교

openstacksdk는 `Connection`에서 서비스 Proxy에 접근합니다. 이 프로젝트는 Go에서 오류와 context를 명시하도록 서비스 접근을 메서드로 제공합니다. 리소스 연산은 `Servers`, `Networks` 등의 typed collection에 모읍니다. [openstacksdk 사용 계층](https://docs.openstack.org/openstacksdk/latest/user/)

| 작업 | openstacksdk | gophercloudsdk |
|---|---|---|
| 연결 | `openstack.connect(cloud="dev")` | `sdk.Connect(ctx, sdk.WithCloud("dev"))` |
| 서비스 접근 | `conn.compute` | `conn.Compute(ctx)` → `*compute.Service, error` |
| ID 조회 | `conn.compute.get_server(id)` | `computeService.Servers.Get(ctx, id)` |
| 이름 조회 | `conn.compute.find_server("web", ignore_missing=False)` | `computeService.Servers.Find(ctx, resource.Name("web"))` |
| ID 참조 조회 | `conn.compute.find_server(id)` | `computeService.Servers.Find(ctx, resource.ID(id))` |
| 목록 | `conn.compute.servers(status="ACTIVE")` | `computeService.Servers.List(ctx, resource.WithStatus("ACTIVE"))` |
| 목록 수집 | `list(conn.compute.servers())` | `computeService.Servers.All(ctx)` |
| 선택 인자 | keyword argument / 기본값 | 작업별 `With...` 옵션 / 문서화된 기본값 |
| 확장 입력 | `**attrs`, `**query` | `compute.WithField(...)`, `resource.WithQuery(...)` |
| 오류 | 예외 | 반환된 `error`, `errors.Is`, `errors.As` |
| 리소스 재사용 | Resource 인스턴스 전달 가능 | `resource.ID(server.ID)`로 명시적으로 참조 |

이름과 ID를 하나의 문자열로 추측하지 않습니다. UUID처럼 생긴 이름도 `resource.Name(...)`이면 이름으로 찾습니다. 기본 응답 모델은 Gophercloud 타입의 alias를 사용합니다. 인증·virtual media·Swift 리소스에는 SDK가 추가 정보를 보관하는 모델을 제공합니다. 응답 변경을 추적하고 자동 저장하는 Resource 객체는 아직 제공하지 않습니다.

### 서비스 여러 개를 사용하는 서버 생성

Python의 상위 연결 API:

```python
import openstack

conn = openstack.connect(cloud="dev")
server = conn.create_server(
    name="web-01",
    image="ubuntu",
    flavor="small",
    network="private",
    wait=True,
    timeout=300,
)
print(server.id)
```

Go에서는 연결이 서비스 의존성을 주입하고, 서버 생성 계층이 이미지·flavor·네트워크 이름을 해석합니다:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    sdk "gophercloudsdk"
    "gophercloudsdk/compute"
    "gophercloudsdk/resource"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
    defer cancel()

    conn, err := sdk.Connect(ctx, sdk.WithCloud("dev"))
    if err != nil { log.Fatal(err) }
    computeService, err := conn.Compute(ctx)
    if err != nil { log.Fatal(err) }

    server, err := computeService.Servers.Create(ctx, compute.CreateServerRequest{
        Name: "web-01",
        Image: resource.Name("ubuntu"),
        Flavor: resource.Name("small"),
    }, compute.WithNetworks(resource.Name("private")),
       compute.WithMetadata(map[string]string{"team": "infra"}),
       compute.WithWait(resource.WithTimeout(5*time.Minute)))
    if err != nil {
        if server != nil { log.Printf("created server: %s", server.ID) }
        log.Fatal(err)
    }
    fmt.Println(server.ID)
}
```

ID를 이미 알고 있다면 `resource.ID(...)`를 사용하면 됩니다. 생성 의존성이 ID로 지정되면 사전 조회를 생략합니다. 위 두 예제의 공통 범위는 기존 이미지·flavor·네트워크를 사용한 생성과 상태 대기입니다. Python 상위 API의 추가 네트워크 자동 구성이나 floating IP 관리까지 구현한 것은 아닙니다. [openstacksdk Connection API](https://docs.openstack.org/openstacksdk/latest/user/connection.html)

빌드 가능한 예제는 [examples/create-server](examples/create-server/main.go)에 있습니다. 인자를 주고 실행하면 실제 클라우드에 서버를 생성합니다:

```sh
go run ./examples/create-server -name web-01 -image ubuntu -flavor small -network private
```

## 기본값과 공통 계약

| 항목 | 기본 동작 |
|---|---|
| 인증 | `OS_CLOUD`가 있으면 clouds.yaml, 없으면 Gophercloud의 `OS_*` 인증 파서 |
| 명시 인증 | `sdk.WithAuth(gophercloud.AuthOptions{...})`가 기본 인증 소스를 대체 |
| 명시 cloud | `sdk.WithCloud("dev")`가 환경의 cloud 선택을 대체 |
| 서비스 선택 | cloud/environment의 region/interface, 명시 옵션으로 덮어쓰기 |
| 서비스 연결 | 최초 접근 시 구성하고 성공한 서비스만 캐시 |
| Find | 정확한 이름 또는 명시 ID, 미존재는 `ErrNotFound` |
| 중복 이름 | 항상 `ErrAmbiguous`; 임의로 선택하지 않음 |
| Delete | 미존재는 성공; `WithMissingError()`로 엄격하게 변경 |
| List | 모든 페이지를 lazy iterator로 순회; `break`하면 후속 페이지를 읽지 않음 |
| All | 모든 결과를 메모리에 수집; 빈 목록은 빈 slice |
| 페이지 크기 | `WithPageSize`는 페이지 크기이며 전체 결과 개수 제한이 아님 |
| Wait | 대상 상태를 필수로 지정; 최대 5분, 간격 2초, context로 취소 가능 |
| Create | 생성 응답 즉시 반환; `compute.WithWait()`면 ACTIVE까지 대기 |
| 생성 후 대기 실패 | 생성된 서버와 오류를 함께 반환, 자동 삭제하지 않음 |

같은 설정에 대한 옵션은 뒤에 지정한 값이 우선합니다. `WithName`은 항상 정확한 이름 필터로 적용되며 `WithQuery("name", ...)`보다 우선합니다. `WithQuery`로 전달하는 이름 필터는 서비스 자체의 의미를 따릅니다.

원래 HTTP 오류는 보존됩니다. 403을 미존재로 취급하거나 생성으로 자동 전환하지 않습니다. `WithIgnoreMissing()`을 사용한 Find는 미존재일 때 `nil, nil`을 반환하므로 결과의 nil 여부를 확인해야 합니다.

```go
server, err := computeService.Servers.Find(ctx, resource.Name("web-01"))
switch {
case errors.Is(err, resource.ErrNotFound):
    // 이름이 없음
case errors.Is(err, resource.ErrAmbiguous):
    // 중복 이름: ID를 지정해야 함
case err != nil:
    return err
default:
    fmt.Println(server.ID)
}
```

위 조각은 `errors`, `fmt`, `resource`를 import하고, `ctx`와 `computeService`를 준비한 오류 반환 함수 안에서 사용합니다. 공통 사용법은 [resource README](resource/README.md)를 참고하세요.

## Go에 맞춘 선택 옵션과 확장

필수 입력은 concrete request struct, 선택 입력은 작업별 functional options로 받습니다. `WithConfigDrive(false)`처럼 false를 명시한 경우에도 값을 전송합니다. 옵션 종류를 분리해 List 옵션을 Create에 전달하면 컴파일 오류가 나도록 합니다.

```go
// 서비스별 query 확장: 애플리케이션의 builder 구현이 필요하지 않음
networks, err := networkService.Networks.All(ctx,
    resource.WithQuery("router:external", "true"))

// 서버 생성 확장: server 객체 내부에 JSON 필드를 추가
vendorOption := compute.WithField("vendor_hint", map[string]any{"pool": "fast"})
// vendorOption을 Servers.Create의 옵션으로 전달
```

`vendor_hint`는 확장 경로를 보여주는 예시이며 표준 Nova 필드가 아닙니다. 해당 필드를 지원하는 클라우드에서만 사용해야 합니다. 기본 생성 필드와 충돌하거나 JSON으로 변환할 수 없는 값은 요청 전에 거부합니다. JSON 필드의 실제 스키마와 microversion 요구는 서버가 검증합니다. 현재 Gophercloud CreateOpts의 필드는 core 필드로 보호되며, 일부 core 필드는 아직 SDK의 typed 옵션으로 노출하지 않았습니다.

현재 미지원 API는 서비스의 `RawClient()`로 Gophercloud concrete options를 사용할 수 있습니다. 이를 상위 SDK의 지원 기능으로 간주하지 않습니다. 공유 클라이언트 설정은 요청을 동시에 실행하기 시작한 뒤 변경하지 마세요.

## 개발과 검증

```sh
go mod download
make check
go test -coverpkg=./... ./...
go build ./examples/create-server
```

테스트는 로컬 `httptest.Server`를 사용합니다. 실클라우드 자격 증명이 필요하지 않으며 OpenStack 리소스를 생성하지 않습니다. 테스트 환경은 localhost 포트 바인딩을 허용해야 합니다. [테스트 구성](docs/testing.md), [설계 및 확장 계획](docs/design.md)을 참고하세요.

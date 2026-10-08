# 네트워크 생성·수정·삭제와 역할 cache

`service.CreateNetwork`, `UpdateNetwork`, `DeleteNetwork`는 기본값·필드 옵션·이름 해석·응답 검증과 공유 네트워크 역할 cache 초기화를 처리합니다. `service`는 `conn.Network(ctx)`가 반환하는 `*network.Service`입니다. 애플리케이션에서 Gophercloud builder를 구현할 필요가 없습니다.

기존 `service.Networks`는 그대로 `*resource.Collection[network.Network]`입니다. 목록·조회·이름 해석·삭제·대기 코드를 계속 사용할 수 있습니다. 새 cloud 방식 메서드는 Service에 추가되므로 Collection을 직접 받는 함수나 변수의 타입을 바꾸지 않아도 됩니다.

## Python과 Go의 대응

| 고정 openstacksdk cloud 호출 | go-openstacksdk |
|---|---|
| `conn.create_network("private")` | `service.CreateNetwork(ctx, network.CreateNetworkRequest{Name: "private"})` |
| `admin_state_up=False` | `network.WithNetworkAdminStateUp(false)` |
| `shared=True`, `external=True` | `WithNetworkShared(true)`, `WithNetworkExternal(true)` |
| `port_security_enabled=False` | `WithNetworkPortSecurity(false)` |
| `mtu_size=1500`, `dns_domain="example.org."` | `WithNetworkMTU(1500)`, `WithNetworkDNSDomain("example.org.")` |
| `provider={...}` | `WithNetworkProvider(network.ProviderNetwork{...})` |
| `availability_zone_hints=[]` | `WithNetworkAvailabilityZoneHints()` |
| `project_id="project-id"` | create의 `WithNetworkProjectID("project-id")` |
| `conn.update_network(id, name="renamed")` | `service.UpdateNetwork(ctx, resource.ID(id), WithNetworkName("renamed"))` |
| `conn.update_network(name, external=False)` | `UpdateNetwork(ctx, resource.Name(name), WithNetworkExternal(false))` |
| `conn.delete_network(name_or_id)` | `service.DeleteNetwork(ctx, resource.Name(name))` 또는 `resource.ID(id)`; `(bool, error)` 반환 |
| Proxy `conn.network.update_network(id, if_revision=0)` | `UpdateNetwork(ctx, resource.ID(id), WithNetworkRevision(0))` |

Python cloud helper와 Proxy는 다른 입력 계약입니다. Cloud `create_network`·`update_network`는 description이나 임의의 kwargs를 받지 않습니다. Go의 `WithNetworkDescription`, revision 옵션과 JSON extension은 SDK가 제공하는 추가 기능이며, Python에서 같은 속성을 설정하려면 `conn.network.create_network`·`update_network`의 Proxy 속성을 사용합니다.

## 독립 실행 예제

`dev` cloud 이름을 환경에 맞게 바꿉니다. 예제는 네트워크를 생성하고 이름을 바꾼 뒤 삭제를 요청합니다. 이 세 메서드는 ACTIVE나 실제 삭제 완료를 기다리지 않습니다.

Python:

```python
import openstack

conn = openstack.connect(cloud="dev")
created = conn.create_network("private", port_security_enabled=False)
print(created.id, created.name)
updated = conn.update_network(
    created.id,
    name="private-renamed",
    admin_state_up=False,
    dns_domain="",
)
print(updated.id, updated.name)
deleted = conn.delete_network(created.id)
print("deleted", deleted)
```

Go:

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    if err := run(ctx); err != nil {
        var response *resource.ResponseError
        if errors.As(err, &response) {
            log.Printf("accepted response: status=%d", response.StatusCode)
        }
        log.Fatal(err)
    }
}

func run(ctx context.Context) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud("dev"))
    if err != nil {
        return err
    }
    service, err := conn.Network(ctx)
    if err != nil {
        return err
    }
    created, err := service.CreateNetwork(ctx,
        network.CreateNetworkRequest{Name: "private"},
        network.WithNetworkPortSecurity(false))
    if err != nil {
        if created != nil {
            log.Printf("returned network: id=%q", created.ID)
        }
        return err
    }
    fmt.Println(created.ID, created.Name)
    updated, err := service.UpdateNetwork(ctx, resource.ID(created.ID),
        network.WithNetworkName("private-renamed"),
        network.WithNetworkAdminStateUp(false),
        network.WithNetworkDNSDomain(""))
    if err != nil {
        return err
    }
    fmt.Println(updated.ID, updated.Name)
    deleted, err := service.DeleteNetwork(ctx, resource.ID(created.ID))
    fmt.Println("deleted", deleted)
    return err
}
```

상태 대기가 필요하면 기존 Collection 메서드를 명시적으로 이어서 호출합니다. 예를 들어 생성 후 `service.Networks.Wait(ctx, resource.ID(created.ID), "ACTIVE")`, 삭제 후 `service.Networks.WaitDeleted(ctx, resource.ID(id))`를 사용할 수 있습니다. 기다리는 동안 실패하면 앞서 접수된 생성·삭제를 되돌리지 않습니다.

## 생성 기본값과 수정의 생략·빈 값

Create는 `name`을 항상 보내며 `admin_state_up` 기본값은 true입니다. 빈 이름도 요청에 보존합니다. Update는 옵션으로 선택한 필드만 보내므로 생략한 필드는 바꾸지 않습니다. 옵션은 호출할 때마다 한 번 적용하고 같은 필드의 마지막 값이 우선합니다. nil 옵션이나 잘못된 앞선 옵션은 이후 옵션으로 숨기지 않고 HTTP 전에 실패합니다.

| 입력 | CreateNetwork | UpdateNetwork |
|---|---|---|
| name | Request.Name 기본값, `WithNetworkName`으로 교체 가능; 빈 문자열 보존 | 옵션 없음은 생략; 빈 문자열 보존 |
| admin state | 기본 true; 명시 false 또는 nullable null은 그대로 전송 | 명시 false/null 보존 |
| shared·external | false와 null은 생략; true 전송 | 명시 false/null 보존 |
| port security | 옵션 없으면 생략; false 전송 | false 전송; concrete bool 옵션만 제공 |
| MTU | 0은 생략, 68 이상 전송; 음수와 1–67 거부 | 0과 68 미만 거부 |
| DNS domain | 빈 문자열/null 생략 | 빈 문자열/null 보존 |
| project owner | `WithNetworkProjectID`로 문자열 지정; 빈 문자열도 그대로 전송 | create-only 오류 |
| availability zone hints | 옵션 생략 시 extension 조회 없음; 옵션의 빈 배열도 검사 후 `[]` 전송 | create-only 오류 |
| description·추가 JSON 필드 | 지정한 값 전송; cloud helper보다 넓은 Go 기능 | 지정한 값 전송; cloud helper보다 넓은 Go 기능 |
| revision | update-only 오류 | 0을 포함한 비음수 revision 전송 |

`WithNetworkNameValue`, `WithNetworkAdminStateValue`, `WithNetworkSharedValue`, `WithNetworkExternalValue`, `WithNetworkDNSDomainValue`는 `request.Optional[T]`를 받습니다. Zero Optional은 이 옵션이 값을 설정하지 않는다는 뜻이며, `request.Present`는 false·0·빈 문자열을, `request.Null`은 JSON null을 선택합니다. 위 표의 Create 생략 규칙은 마지막에 적용합니다. 예를 들어 Update의 DNS null은 다음처럼 지정합니다.

```go
updated, err := service.UpdateNetwork(ctx, resource.ID(id),
    network.WithNetworkDNSDomainValue(request.Null[string]()))
```

이 짧은 예제에는 `github.com/JSYoo5B/go-openstacksdk/request` import가 필요합니다. 같은 typed 옵션으로 사용되지 않는 필드의 nullable 동작까지 자동 확장되지는 않습니다. Project ID는 URL 선택에 쓰이지 않는 body 값이므로 빈 문자열도 그대로 보존합니다. 옵션이 없으면 생략하며, 실제 owner 값과 권한은 Neutron이 검사합니다. Python MTU 검사는 코드에서 68을 기준으로 하며 IPv6 subnet에 필요한 추가 제약은 Neutron이 판단합니다.

## Provider·AZ와 extension 값

Provider는 SDK가 소유하는 concrete 객체입니다. Known 필드 세 개에 대해 생략/null/빈 값·0을 구분하고, segmentation ID는 JSON 값으로 받습니다. Python 예제의 문자열 segmentation ID도 표현할 수 있습니다.

```go
provider := network.ProviderNetwork{
    NetworkType: request.Present("vlan"),
    PhysicalNetwork: request.Present("physnet1"),
    SegmentationID: request.Present[any](100),
}
created, err := service.CreateNetwork(ctx,
    network.CreateNetworkRequest{Name: "provider-net"},
    network.WithNetworkProvider(provider))
```

`WithNetworkProvider`는 이전 known provider 필드 세 개를 전체 교체합니다. 빈 ProviderNetwork를 뒤에 지정하면 이전 provider 선택을 지웁니다. 옵션을 만들 때 JSON으로 복사하므로 입력 map·slice나 Provider 필드에 들어간 JSON 객체를 이후 수정해도 준비된 값은 바뀌지 않습니다. Source의 update helper가 입력 provider dict에서 값을 pop하는 것과 달리 Go는 입력을 변경하지 않습니다. Provider network type/physical network는 typed 문자열/null로 제한하며 Python의 임의 객체 coercion 전체를 제공하지 않습니다.

`WithNetworkAvailabilityZoneHints("az1")` 또는 인수 없는 `WithNetworkAvailabilityZoneHints()`는 `network_availability_zone` extension이 있는지 Create 전에 확인합니다. 지원되지 않으면 `ErrUnsupported`, 조회 실패면 해당 오류를 반환하고 POST하지 않습니다. 빈 hints 배열도 검사를 생략하지 않습니다. Go는 role cache와 별개의 extension 조회를 사용하며 body next 링크·HTTP Link 페이지를 읽는 경로로 구성합니다. Python의 extension 캐시와 session/discovery 전체 정책을 재현한다는 뜻은 아닙니다. 고정 Python list는 빈 페이지에서 종료합니다. Go는 빈 중간 페이지에 next 링크가 있으면 계속 읽으며, 뒤 페이지 오류·순환 링크·외부 origin 링크를 성공 또는 미지원으로 숨기지 않습니다. 조회는 호출마다 새로 수행합니다.

`WithNetworkField("vendor_enabled", false)`는 추가 JSON 필드를 보냅니다. 값은 옵션 생성 시 복사하고 null도 보존합니다. 빈 이름과 SDK core wire 필드 이름을 거부하므로 `mtu`·`project_id`·`availability_zone_hints`·provider known 필드 등을 이 경로로 덮어쓸 수 없습니다. Core 필드는 해당 typed 옵션을 사용합니다. 반환 모델에 이 추가 필드가 생기는 기능은 아닙니다.

## 이름 해석과 변경 없는 수정

`resource.ID(id)`는 explicit ID이고 `resource.Name(name)`은 정확한 이름입니다. Go는 UUID 형태의 이름도 ID로 추정하지 않습니다. 상위 CRUD의 lookup은 변경 요청과 같은 SDK 소유 HTTP·source 검사 경로를 사용합니다. 이름 query를 보내고 모든 페이지에서 정확한 이름을 비교하며 중복과 뒤 페이지 오류를 보존합니다. 상위 이름 목록의 204는 빈 목록으로 종료합니다. 기존 `Networks` Collection의 native 읽기 경로는 유지합니다. Python cloud의 단일 문자열 name-or-ID fallback과 동일한 입력 자동 추정은 하지 않습니다.

Update에 필드 또는 revision 옵션이 있으면 explicit ID는 사전 network GET 없이 PUT합니다. Name은 정확한 이름을 한 번 해석한 ID로 PUT합니다. 옵션을 모두 생략하거나 unset Optional만 있으면 현재 network를 조회해 반환하고 PUT 없이 역할 cache를 초기화합니다. 필드를 명시했다면 현재 값과 같아도 PUT합니다. Python Resource의 dirty 비교·동일 값 수정의 no-op 처리는 별도 계약입니다.

DeleteNetwork는 ID도 먼저 조회합니다. 초기 조회에서 없으면 `(false, nil)`이고 DELETE와 cache 초기화를 하지 않습니다. 조회 후 DELETE가 202/204로 접수되거나 원래 DELETE의 clean404 경합이면 true를 반환하고 cache를 초기화합니다. true는 실제 삭제 완료 대기 결과가 아닙니다. 조회·권한·서버 오류와 callback/source 오류를 clean404로 숨기지 않습니다.

고정 Python cloud `delete_network(name_or_id)`도 선행 조회의 미존재는 false, 찾은 network의 삭제는 true를 반환합니다. Go는 `resource.ID`·`resource.Name`으로 입력 해석을 명시하고 DELETE 성공을 202/204로 제한합니다. Python Resource의 응답 검사에는 400 미만 상태가 허용되므로 이 성공 코드 범위는 문서화된 Go 차이입니다. Cloud 함수에는 revision·wait 인자가 없으며 별도 Proxy의 `if_revision` 또는 삭제 대기는 이 bool API의 필수 옵션으로 추가하지 않습니다.

기존 `service.Networks.Delete`는 error-only 반환과 explicit ID direct DELETE를 유지합니다. 기본으로 missing을 무시하고 `resource.WithMissingError()`로 미존재 오류를 선택합니다. 실제 접수된 삭제는 역할 cache를 초기화하지만 missing 응답은 accepted 삭제가 아니므로 이 direct ID 경로의404만으로 초기화하지 않습니다. Cloud DeleteNetwork의 선행 조회 및 bool 계약과 구분합니다.

## Cache와 접수 후 오류

Create의 accepted status는 201/202, Update는 200/201, Delete는 202/204입니다. SDK가 허용한 상태의 실제 응답을 받으면 같은 Service의 공유 역할 cache를 초기화합니다. 새 getter·configured 기본 NIC·floating source/NAT 선택은 다음 역할 탐색에서 바뀐 목록을 읽습니다. 기존에 시작한 역할 탐색을 취소하지는 않지만 이전 generation의 결과는 cache에 넣지 않습니다. [역할 cache와 Reset](network-roles.md#cache-복사-reset과-오류)을 참고하세요.

Validation·AZ 조회·이름 해석·접수 전 HTTP 오류는 cache를 초기화하지 않습니다. 옵션 없는 성공 Update는 조회 후 초기화하는 예외입니다. Create/Update의 응답은 `network` JSON object와 안전한 ID가 필요하며 Update ID는 요청한 ID와 일치해야 합니다.

Service가 생성한 Networks·Roles를 다른 객체로 교체하면 HTTP 전에 오류를 반환합니다. 요청 중 client/provider/endpoint/resource-base 또는 두 공개 객체가 교체되어도 다음 시도와 mutation을 차단합니다. 접수 후 교체는 원래 공유 cache를 초기화하고 응답 오류를 반환합니다. 이 guard는 callback에 의한 변경을 검사하며, public 필드의 동시 쓰기를 thread-safe로 만드는 기능은 아닙니다.

Mutation이 접수된 뒤 body 읽기·Close·decode·ID 검증·context/source 검사가 실패하더라도 cache를 초기화하고 오류를 반환합니다. 읽기와 decode 오류만으로 접수된 POST/PUT을 다시 전송하지 않습니다. Native 인증·HTTP 오류의 재시도 정책과 이 접수 후 decode 경계는 구분됩니다.

Decode 실패는 nil 모델과 오류를 반환할 수 있으므로 모델이 없다는 이유만으로 생성이 접수되지 않았다고 판단할 수 없습니다. Decode된 모델의 ID 검증 실패는 그 모델과 오류를 함께 반환합니다. Delete의 accepted 응답 뒤 read/Close 오류는 `(true, err)`로 접수 증거를 보존합니다. 응답 후 오류의 `*resource.ResponseError`는 status/header/body와 원인을 갖고, `errors.As`로 확인할 수 있습니다. 원래 HTTP 오류는 `OperationError`를 통해 cause를 확인합니다. SDK는 실패 후 만든 network를 자동 삭제하지 않습니다.

고정 Python helper는 proxy가 정상 반환한 뒤 cache를 초기화합니다. 잘못된 응답의 일부 형태에서 예외가 나면 source cache가 초기화되지 않을 수 있습니다. Go의 accepted 응답 후 decode/검증 실패에도 초기화하는 정책은 이 부분을 강화한 동작입니다. 더 넓은 Python의 response 허용·Resource 변환·session 정책과 같다고 판정하지 않습니다.

`service.API.Networks`·`conn.NetworkV2(ctx)`·`RawClient`를 통한 직접 mutation이나 외부 도구의 변경은 이 상위 hook을 통과하지 않습니다. 그 뒤 같은 high Service를 계속 쓰려면 `service.Roles.Reset()` 또는 `conn.ResetNetworkRoles()`를 명시합니다. 별도로 만든 Service끼리 cache를 공유하거나 cloud 전체 변경을 자동 감지하지 않습니다.

## 반환 모델과 권한 범위

반환값은 기존 native Gophercloud의 `*network.Network`입니다. ID·이름·상태·admin state·shared·project·subnet·tag 등의 native 필드를 제공합니다. Router external·provider 필드·MTU·DNS·port-security 같은 확장 필드는 이 native 모델에 포함되지 않으므로 설정에 성공했더라도 반환값에서 읽을 수 없습니다. 역할 discovery의 `RoleNetwork`와 Python mutable Resource, 전체 extension 응답 모델은 별도입니다.

기본 CRUD는 핵심 user API 단위에서 제공하지만 옵션마다 필요한 권한은 해당 cloud의 Neutron policy가 검사합니다. 고정 SDK 문서는 다른 project를 선택하는 project_id를 admin-only 입력으로 설명합니다. Provider·shared·external 설정도 이 API의 존재만으로 관리자 권한을 얻거나 해당 cloud 정책을 우회하지 않습니다. 이 단위는 전체 admin 권한 정책·확장 기능의 검증 완료를 뜻하지 않습니다.

High `WithNetworkRevision(0)`는 조건부 Update를 요청합니다. Cloud helper에는 없는 Go/native/Proxy capability이고 revision은 비음수로 제한합니다. Low native Update는 signed revision 및 raw header escape를 유지합니다. DeleteNetwork에는 revision 옵션이 없으므로 Proxy delete의 if_revision 전체를 지원한다고 판정하지 않습니다. [기존 native revision 동작](v2/revision-updates.md)을 참고하세요.

## 고정 소스 비교와 검증 범위

비교 기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [cloud CRUD](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L491-L692), [network Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L3159-L3307), [역할 cache reset](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L95-L108), [Network 모델](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/network.py#L16-L142)입니다.

Cloud `delete_network`의 named Go 대응은 명시 Ref의 선행 조회, 초기 미존재 false, 접수되거나 조회 후 clean404인 삭제 true, 공유 role cache 초기화와 inspectable 오류를 제공합니다. 접수 후 read·Close·context/source 실패에서도 `(true, err)`와 원래 cache 초기화를 유지하는 것은 Python proxy 정상 반환 후 Reset과 다른 Go 정책입니다. bool 반환에는 Network model의 extension·location 변환이 필요하지 않습니다. 이 판정으로 Create·Update·Proxy의 revision·native family 또는 삭제 waiter까지 완료로 판정하지 않습니다.

Mutable Resource/descriptor coercion·dirty/no-op·전체 session/adapter/discovery 정책, dynamic kwargs와 nullable 입력 전체, Create/Update의 native 모델에 없는 응답 extension은 별도 SDK 범위입니다. raw/native/out-of-band cache 변경은 위의 명시 Reset 정책을 사용합니다. [자동 IP 판단·조건부 연결](../compute/server-automatic-ip.md), [명시 IP·pool](../compute/server-ip-dispatch.md), [Nova backend](../compute/server-nova-floating-ip.md), [서버 주소 view](../compute/server-addresses.md), [생성·수렴](../compute/create-with-automatic-floating-ip.md)과 [ready/wait](../compute/server-ready.md)는 후속으로 구현된 각 API의 가이드에서 범위와 차이를 추적합니다. 이 상위 CRUD와 cache hook만으로 cloud 네트워크·서버 생성 또는 SDK 전체의 지원을 승격하지 않습니다.

검증 상태는 [지원 판정대장](../docs/sdk-support-ledger.md)에 기록합니다. 이 가이드의 독립 Go 예제는 컴파일을, HTTP 동작은 로컬 fixture를 기준으로 검증합니다.

[CRUD 기본 계약](network_mutations_test.go), [응답·source·페이지·cache 경계](network_mutation_boundaries_test.go), [Connection getter·기본 NIC·source/NAT 소비](../connection_network_mutations_test.go)를 로컬 HTTP fixture로 검증했습니다. 위 독립 Go main을 로컬 SDK 코드 `5bfea18`로 컴파일했습니다(source SHA-256 `8ce7d71e42a0bfaa905a160f1e5594e1b276c47080d6c8c49cc96389f40dfc75`). 공통 Resource의 mixed 오류와 REST collection guard·204 처리는 별도 계약 테스트로 확인했습니다. 전체 검사 실행 근거와 남은 범위는 [지원 판정대장](../docs/sdk-support-ledger.md#상위-network-crud와-공유-cache-hook)에 기록합니다. Python 예제 실행이나 인증된 OpenStack 검증 결과는 포함하지 않습니다.

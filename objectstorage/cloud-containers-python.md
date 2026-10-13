# Swift cloud container: openstacksdk와 Go

이 가이드는 고정된 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 cloud `create_container`, `set_container_access`, `get_container_access`와 Go를 비교합니다. `swift.Cloud`가 조회 후 생성, public/private ACL 변환, 갱신 여부와 결과 해석을 처리합니다. `swift.Containers.CreateRecord`는 한 번의 Proxy PUT이므로 cloud의 get-or-create와 사용하는 목적이 다릅니다.

Python의 전체 연결 예제입니다.

```python
import openstack

conn = openstack.connect()
container = conn.create_container("photos", public=True)
if container is not None:
    print(container.name, container.read_ACL)

updated = conn.set_container_access("photos", "private")
observed = conn.set_container_access("photos", "public", refresh=True)
print(conn.get_container_access("photos"))
```

Go의 전체 연결·서비스 예제입니다. `ObjectStorage`와 `ObjectStorageV1` 모두 같은 `Cloud` facade를 제공합니다. 인증 환경과 선택한 Swift account의 권한을 구성한 뒤 실행할 수 있습니다. 이 문서는 예제 빌드와 fixture 검증에 대한 가이드이며 실제 cloud 실행 결과를 주장하지 않습니다.

```go
package main

import (
    "context"
    "fmt"
    "log"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx)
    if err != nil { log.Fatal(err) }
    swift, err := conn.ObjectStorage(ctx)
    if err != nil { log.Fatal(err) }

    created, err := swift.Cloud.CreateContainer(ctx, "photos",
        objectstorage.WithContainerPublic(true))
    if err != nil { log.Fatal(err) }
    if created.Container != nil {
        fmt.Printf("existing=%t name=%s ACL=%s\n", created.Existing,
            created.Container.Resource.Body["name"],
            created.Container.Resource.Body["read_ACL"])
    }

    updated, err := swift.Cloud.SetContainerAccess(ctx, "photos", "private")
    if err != nil { log.Fatal(err) }
    fmt.Printf("POST=%d requested=%s\n", updated.Acknowledgement.StatusCode, updated.Requested)

    observed, err := swift.Cloud.SetContainerAccess(ctx, "photos", "public",
        objectstorage.WithContainerAccessRefresh(true))
    if err != nil { log.Fatal(err) }
    fmt.Printf("POST=%d HEAD=%d observed ACL=%s\n",
        observed.Acknowledgement.StatusCode, observed.Refreshed.StatusCode,
        observed.Container.Resource.Body["read_ACL"])

    access, err := swift.Cloud.GetContainerAccess(ctx, "photos")
    if err != nil { log.Fatal(err) }
    fmt.Printf("access=%s HEAD=%d\n", access.Access, access.Container.StatusCode)
}
```

| Python cloud API | Go service API | SDK가 처리하는 동작 |
| --- | --- | --- |
| `conn.create_container(name)` | `swift.Cloud.CreateContainer(ctx, name)` | HEAD; clean404인 경우에만 PUT; 두 번째 HEAD |
| `public=True` | `WithContainerPublic(true)` | 새 container PUT에 `.r:*,.rlistings` 전송 |
| `conn.set_container_access(name, "private")` | `swift.Cloud.SetContainerAccess(ctx, name, "private")` | 빈 값인 `X-Container-Read`를 포함하는 POST; 기본 refresh false |
| `refresh=True` | `WithContainerAccessRefresh(true)` | POST가 성공하면 HEAD 한 번으로 갱신 |
| `conn.get_container_access(name)` | `swift.Cloud.GetContainerAccess(ctx, name)` | HEAD의 ACL을 정확히 비교하고 public/private 반환 |

직접 인증한 Gophercloud client를 사용한다면 `objectstorage.New(client)`가 같은 concrete cloud API를 제공합니다. 라이브러리 사용자는 builder interface나 요청 순서 분기를 구현하지 않습니다. 전용 `ContainerCreateOpts`, `ContainerAccessOpts`, `ContainerReadOpts`, `With...Opts` 및 `With...Headers`도 제공하며, 전체 옵션 교체로 기본값을 복원할 수 있습니다. 옵션 maps를 생성 시점과 각 callback 경계에서 복사하고 모든 콜백 이후 검증합니다. nil options, 부정확한 service, 유효하지 않은 literal container 이름, routing 변경, HTTP Header alias 충돌과 `X-Container-Read` 직접 덮어쓰기는 HTTP 전에 실패합니다. ACL은 `Public` 및 access 인자로 설정합니다.

`CreateContainer`의 기본 `public=false`는 PUT에 SDK 소유 ACL을 보내지 않습니다. 기존 container가 있으면 HEAD 결과를 즉시 돌려주며 `public=true`로 ACL을 변경하지도 않습니다. 생성한 경우 `Creation`은 PUT 결과, `Refreshed`는 마지막 HEAD 결과이며 `Container`가 Python return 값입니다. 생성 후 container가 사라져 clean404가 나오면 `Container=nil`, `err=nil`이 됩니다. 이때 실제 404 receipt는 `Refreshed`, 이미 성공한 PUT receipt는 `Creation`에 각각 남습니다. 403, descriptor 변환 오류, physical read/Close 오류, 취소, source 변경이 있으면 실패하며 PUT을 다시 수행하지 않습니다.

`SetContainerAccess`는 `public`과 `private`를 정확한 대소문자로만 받습니다. `public`은 `.r:*,.rlistings`, `private`는 **존재하는 빈 Header 값**으로 변환합니다. raw ACL 문자열, 공백과 `PUBLIC` 같은 다른 값은 옵션 실행·HTTP 전에 거부합니다. Source cloud의 기본 refresh는 false로 유지하며 Source Proxy setter의 기본 true와 혼동하지 않습니다. `Acknowledgement`는 실제 POST, `Refreshed`는 명시적으로 선택한 HEAD입니다. `Requested`는 입력값입니다. Source `_base.BaseResource.set_metadata`처럼 POST 요청 ACL을 응답 Resource의 `read_ACL`로 복사하지 않으므로 서버가 ACL Header를 반환하지 않았으면 POST record의 getter는 null입니다. HEAD 실패 시 POST receipt는 유지하며 rollback이나 POST 재전송은 하지 않습니다.

`GetContainerAccess`는 Header 누락·빈 문자열을 private, 정확한 `.r:*,.rlistings`를 public으로 분류합니다. `.r:*`, 순서가 바뀐 문자열, 다른 사용자에게 주는 custom ACL은 unknown으로 실패합니다. Unknown ACL과 missing container 모두 HEAD receipt가 반환값에 남습니다. clean404는 추가로 `resource.ErrNotFound`이며 native404 cause가 유지됩니다. physical 오류가 있는404는 missing으로 취급하지 않습니다.

각 record의 `Resource.Body`는 declared Body/Header getter를 사용하는 Source 형태이고, `Wire.Header`·`Header`·`StatusCode`는 실제 응답 증거입니다. HEAD와 metadata POST는 Source의 `has_body=False`를 따르므로 JSON body가 있더라도 Resource를 바꾸지 않습니다. 원본 bytes는 `Envelope`에 보존합니다. 성공 status는 Source `raise_from_response`의 `<400`을 따릅니다. Go HTTP transport의 중간1xx·redirect 처리와 실제 Swift 성공 status는 별도의 HTTP 동작입니다. 실패한 응답이나 실패한 projection으로부터 성공 Resource를 만들지 않습니다.

기존 leaf의 `captureMetadata`·Source guard·record projection과 shared REST transport를 재사용합니다. 한 compound workflow 전체에서 literal name, service/provider/endpoint/resource base/microversion을 고정하며 read 후 source 변경이 Close에서 복구되어도 실패는 남습니다. 일반 option Header와 operation ACL은 native retry에서 고정하고 bodyless framing을 유지합니다. fault가 있는 rejected body의 read/Close 오류와 취소는 RetryFunc·RetryBackoffFunc·ReauthFunc 호출 전에 차단합니다. clean401 재인증과 clean retry는 기존 native 정책을 사용할 수 있으며 갱신된 token이 다음 물리 요청에 반영됩니다. native `*ErrUnableToReauthenticate`의 `ErrOriginal`·`ErrReauth`는 함께 유지하고 실제401 receipt는 `ErrOriginal`에서 복사합니다.

Go는 workflow별 단계 receipts를 가진 concrete 결과를 돌려줍니다. Source의 mutable Python Resource·session/cache/dirty 전체 재사용 및 `location` 계산 parity는 별도 공통 SDK 계약에 남아 있으며 이 세 helper 구현으로 완료 처리하지 않습니다. Header의 중복 값·alias가 모호하면 실제 응답 증거를 보존한 projection 오류를 반환하는 기존 Swift record 정책을 유지합니다. Go의 bool/string 타입과 사전 container name 검증은 Python의 임의 값·이름 전달과 명시적인 차이입니다. 실패한 마지막 HEAD에서 null success state를 만들지 않으며 개발자가 HTTP 분기를 이어 붙일 필요가 없습니다.

[Cloud 계약 테스트](cloud_containers_test.go)와 [기존 fixture를 재사용한 physical 계약](v1/containers/cloud_records_test.go)은 생성 흐름/default ACL/refresh/strict access classification/option 소유권, 126 physical fault lanes, native401, clean renewal, 취소, source 복구 이후 sticky failure, 확장된404의 거부와 HEAD/POST body 무시를 검증합니다. Swift account ACL·reseller admin·operator role·정책 설정 및 authentication을 포함한 실제 cloud acceptance는 별도 단계입니다.

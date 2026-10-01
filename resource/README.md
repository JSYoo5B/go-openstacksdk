# 공통 리소스 계층

서비스별 호출의 공통 계약을 `Collection[T]`에 둡니다. 애플리케이션은 서비스가 제공한 collection을 사용하고, Adapter 등록은 SDK 서비스 구현이 담당합니다. 애플리케이션의 builder interface 구현은 필요하지 않습니다.

## 참조와 조회

```go
resource.ID("server-id") // ID로만 조회; 실패해도 이름으로 재해석하지 않음
resource.Name("web-01")  // 정확한 이름으로만 조회
resource.ID(server.ID)   // 조회한 응답을 다음 작업에서 참조
```

빈 참조는 오류입니다. ID의 기본 검증은 URL path 한 구간이며 UUID 형식으로 제한하지는 않습니다. 서비스에 따라 식별자 문법이 다르면 SDK의 binding이 검증과 URL 인코딩을 함께 맡습니다. 이름은 query로 인코딩하므로 이름에 포함된 구두점도 그대로 사용할 수 있습니다.

| 메서드 | 동작 |
|---|---|
| `Get(ctx, id)` | 단일 ID 조회, 404면 `ErrNotFound` |
| `Find(ctx, ref, ...LookupOption)` | 명시 ID 조회 또는 정확한 이름 검색, 중복 검사 |
| `ResolveID(ctx, ref)` | ID는 요청 없이 검증, 이름은 정확히 찾아 안정적인 ID 반환 |
| `List(ctx, ...ListOption)` | lazy `iter.Seq2[*T, error]`, 모든 페이지 순회 |
| `All(ctx, ...ListOption)` | iterator를 slice로 수집 |
| `Delete(ctx, ref, ...LookupOption)` | 이름이면 ID 해석 후 삭제, 기본 미존재 무시 |
| `Wait(ctx, ref, status, ...WaitOption)` | 참조를 한 번 해석하고 동일 ID의 상태 확인 |
| `WaitDeleted(ctx, ref, ...WaitOption)` | 동일 ID를 조회하다가 404·nil 결과·deleted 상태면 삭제 완료 |

Find의 이름 검색은 현재 클라이언트의 기본 조회 범위 안에서 수행합니다. Find에 별도 tenant/project 필터를 전달하는 기능은 아직 없습니다. 중복 오류의 IDs는 중복을 확인한 첫 두 리소스입니다.

## 옵션

| 연산 | 옵션 |
|---|---|
| 목록 | `WithName`, `WithStatus`, `WithPageSize`, `WithMaxItems`, `WithPaginated`, `WithQuery` |
| 조회/삭제 | `WithIgnoreMissing`, `WithMissingError` |
| 대기 | `WithTimeout`, `WithUnlimitedWait`, `WithPollInterval`, `WithFailureStates`, `WithStatusAttribute`, `WithProgressCallback` |

페이지 크기와 로컬 행 수 제한을 구분합니다. `WithPageSize(100)`은 페이지마다 서버에 요청하는
크기이며 기본 List/All은 후속 페이지도 읽습니다. `WithMaxItems(250)`은 서버 응답에서 decode한
행을 로컬 name/status 필터 전에 최대 250개 소비합니다. 따라서 필터를 통과해 반환하는 결과는
250개보다 적을 수 있습니다. 0은 무제한이고 음수는 lazy 순회가 시작될 때 HTTP 전에
`ErrInvalidOption`입니다. 같은 설정은 마지막 옵션이 우선합니다.

`WithPaginated(false)`는 첫 페이지만 읽습니다. MaxItems와 함께 사용하면 첫 페이지에서도
cap에서 멈춥니다. 기본값과 `WithPaginated(true)`는 continuation을 허용합니다. cap 또는
`break`에서 중단하면 추가 페이지를 가져오지 않습니다. wire limit hint는 서비스별 opt-in이며,
공통 MaxItems가 모든 서버에 limit을 보내는 것은 아닙니다.

iterator 사용 예제는 오류를 반환하는 함수 안에서 작성합니다:

```go
for server, err := range service.Servers.List(ctx, resource.WithPageSize(100)) {
    if err != nil { return err }
    fmt.Println(server.ID)
    if shouldStop { break }
}
```

에러는 `(nil, error)`로 한 번 전달하고 순회를 종료합니다. iterator를 다시 순회하면 새로운 API 요청이 시작됩니다. All 도중 오류가 발생하면 부분 결과 대신 `nil, error`를 반환합니다.
List를 만들 때 전달한 옵션 slice를 보관하므로 caller가 나중에 slice의 옵션을 바꾸어도
기존 iterator의 동작은 바뀌지 않습니다. 옵션 적용과 검증은 각 순회가 시작될 때 수행합니다.

Senlin REST bindings는 실제 소비한 행의 decode·검증 뒤 cap을 적용하고, 사용하지 않을
후속 행이나 next link를 해석하지 않습니다. native Gophercloud pager는 페이지 전체를 먼저
Extract하므로 cap 뒤 malformed 행도 extraction 오류를 낼 수 있습니다. page 경계가 없는
custom Iterate는 cap을 적용하지만 first-page 제어를 지원하지 않으면
`WithPaginated(false)`에 `ErrUnsupported`입니다. 빈 페이지 continuation 정책도 binding에
따릅니다. Senlin의 typed 옵션·limit hint·Python과의 차이는
[목록 제어](../clustering/v1/listing/README.md)에 있습니다.

서버의 `next` URL 또는 marker가 이전에 요청한 페이지를 반복하면 추가 요청 전에 `ErrPaginationCycle`로 중단합니다. 같은 URL의 query 순서가 바뀌어도 반복으로 판정합니다. `PaginationCycleError.URL`에는 반복한 링크가 들어 있습니다. 이 정책은 서비스의 typed List, 공통 Collection, page 단위 iterator에 함께 적용됩니다. 소비자가 `break`하면 다음 링크 검사와 후속 요청을 하지 않습니다.

## 오류와 상태 대기

`errors.Is`로 `ErrNotFound`, `ErrAmbiguous`, `ErrUnsupported`, `ErrInvalidOption`, `ErrFailedState`, `ErrPaginationCycle`을 구분합니다. `errors.As`로 `NotFoundError`, `AmbiguousError`, `FailedStateError`, `PaginationCycleError`, `OperationError`의 상세 정보를 확인합니다. HTTP에서 발생한 미존재는 원래 Gophercloud 오류도 보존합니다.

SDK 소유 Cyborg·Senlin·Masakari 모델의 `Metadata.Body`는 원래 JSON 필드를 `json.RawMessage`로 보존합니다. 큰 정수를 `float64`로 바꾸지 않으며 null·빈 값·생략도 구별합니다. `Metadata.Header`와 `StatusCode`는 받아들인 응답의 HTTP 근거입니다. Senlin·Masakari의 받아들인 응답을 읽거나 해석하지 못하면 `ResponseError`가 원문·헤더·상태·cause를 보존합니다. 생성 요청이 이미 성공했을 수 있으므로 이 오류만으로 자동 재전송하지 않습니다.

Wait의 기본 timeout은 5분, 간격은 2초입니다. `WithUnlimitedWait()`는 SDK의 timeout을 없애며 부모 context의 취소와 deadline은 유지합니다. 뒤의 `WithTimeout(...)`으로 다시 제한할 수 있습니다. 대상 상태 비교는 대소문자를 구분하지 않습니다. 실패 상태는 서비스별 Adapter가 선언합니다. `WithFailureStates("ERROR", "BROKEN")`은 이를 정확한 대소문자 무시 비교로 교체하고, 인자 없는 `WithFailureStates()`는 상태에 의한 실패 검사를 끕니다. 대상 상태와 실패 상태가 같다면 대상 도달을 먼저 판정합니다. 서비스 조회 자체가 반환하는 실패 오류는 이 옵션으로 무시하지 않습니다.

Python `resource.wait_for_status(..., failures=[], wait=None)`에 대응하는 Go 옵션은 `WithFailureStates(), WithUnlimitedWait()`입니다. Go는 응답 객체를 받아 캐시된 상태를 검사하는 대신 처음부터 HTTP로 조회하고, timeout을 HTTP 요청에도 context deadline으로 전달합니다. 이미 취소된 context에서는 목표 상태 응답도 성공으로 반환하지 않습니다.

Python의 `attribute="provision_state"`는 `WithStatusAttribute("provision_state")`에 대응합니다. SDK가 모델의 JSON tag 또는 exported Go 필드 이름을 찾으므로 별도 getter나 builder를 구현하지 않습니다. embedded 모델도 지원하며, 없는 필드·문자열이 아닌 필드·모호한 필드는 요청 전에 `ErrUnsupported`입니다. `*string`은 지원하지만 실제 응답이 nil이면 명확한 오류를 반환합니다. `WaitUntilFinished`처럼 완료 조건을 고정한 전용 waiter에서는 속성을 교체할 수 없습니다.

```go
node, err := service.Nodes.Resources.Wait(ctx, resource.ID("node-uuid"), "power on",
    resource.WithStatusAttribute("power_state"),
    resource.WithFailureStates("error"),
    resource.WithUnlimitedWait(),
    resource.WithProgressCallback(func(progress int) { fmt.Println(progress) }))
if err != nil { return err }
_ = node
```

`WithProgressCallback`은 초기 조회와 이후 각 비종료 응답의 `progress` 정수 필드를 읽습니다. 필드가 없거나 nil이면 0이며 값의 범위를 보정하지 않습니다. 완료·실패·HTTP 오류에서는 호출하지 않습니다. callback은 호출한 goroutine에서 동기적으로 실행되며, callback에서 context를 취소하면 추가 조회를 하지 않습니다. 서비스 생성 workflow는 `ValidateWaitOptionsFor[Model]`로 속성의 지원 여부까지 검사한 뒤 생성 요청을 보냅니다.

Wait는 없는 ID나 삭제된 리소스를 계속 기다리지 않고 조회 오류를 반환합니다. 성공 응답에서 리소스 객체가 nil이면 `ErrFailedState`입니다. WaitDeleted는 이미 없는 리소스·nil 결과·대소문자 무시 `deleted` 상태에도 성공하며, 인증 오류나 서버 오류를 삭제 완료로 처리하지 않습니다. 삭제 대기에도 callback과 속성 선택을 사용할 수 있습니다. flavor처럼 기본 상태가 없는 리소스는 Wait/WithStatus를 요청하면 `ErrUnsupported`를 반환하며, Wait에서 실제 문자열 속성을 명시적으로 선택할 수는 있습니다.

## SDK 내부 어댑터

`Adapter[T]`는 getter, pager, extractor, ID/name/status 접근 함수를 등록하는 concrete descriptor입니다. 새 서비스 구현은 원래 Gophercloud의 API 호출과 모델 매핑에 집중하고 이름 조회·중복 검사·대기 알고리즘을 재작성하지 않습니다.

`ValidateID`를 지정한 binding은 모든 조회·해석·삭제·대기에 같은 서비스 식별자 문법을 적용합니다. 생략하면 단일 URL path segment 검증을 사용합니다. `NameQueryKey`는 이름 검색의 서버 query 이름을 지정하며 기본값은 `name`입니다. `prefix` 검색을 쓰더라도 공통 계층은 정확한 이름 비교를 추가합니다. 애플리케이션이 이 설정을 구성하지는 않습니다.

Collection과 서비스 객체의 zero value는 사용하지 않습니다. `Connection`이 구성한 서비스에서 가져오는 것이 일반적인 사용 경로입니다. 응답 객체를 수정해도 자동으로 서버에 반영되지 않습니다. 변경은 서비스 API의 typed Update, 또는 부모 범위 객체의 Update에 전달합니다.

부모가 필요한 리소스는 [범위 객체](../docs/scoped-resources.md)를 사용합니다. `dns.RecordSets.InZone(ctx, resource.Name("example.org."))`처럼 부모를 한 번 해석하고, 반환된 객체의 Find/List/Delete/Wait와 Create/Update가 같은 부모를 사용합니다. 기본 식별자 검증은 빈 ID, `.`, `..`와 URL 경로·query 문자가 들어간 ID를 요청 전에 거부합니다.

테스트는 [collection_test.go](collection_test.go)와 [서비스 공통 통합 테스트](../collections_test.go)에 있습니다.

## 목록 제어 함수 예제

준비된 collection을 받아 첫 페이지에서 최대 5개의 raw 행을 소비하는 함수입니다.

```go
package example

import (
    "context"
    "fmt"

    "gophercloudsdk/resource"
)

func PrintFirstPage[T any](ctx context.Context, collection *resource.Collection[T]) error {
    for value, err := range collection.List(ctx,
        resource.WithMaxItems(5), resource.WithPaginated(false)) {
        if err != nil { return err }
        fmt.Println(value)
    }
    return nil
}
```

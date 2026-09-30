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
| `WaitDeleted(ctx, ref, ...WaitOption)` | 동일 ID를 조회하다가 404가 오면 삭제 완료 |

Find의 이름 검색은 현재 클라이언트의 기본 조회 범위 안에서 수행합니다. Find에 별도 tenant/project 필터를 전달하는 기능은 아직 없습니다. 중복 오류의 IDs는 중복을 확인한 첫 두 리소스입니다.

## 옵션

| 연산 | 옵션 |
|---|---|
| 목록 | `WithName`, `WithStatus`, `WithPageSize`, `WithQuery` |
| 조회/삭제 | `WithIgnoreMissing`, `WithMissingError` |
| 대기 | `WithTimeout`, `WithPollInterval` |

페이지 크기와 총 결과 수를 구분합니다. `WithPageSize(100)`은 페이지마다 서버에 요청하는 크기이며 List/All은 후속 페이지도 읽습니다. 원하는 수에서 `break`하면 추가 페이지를 가져오지 않습니다.

iterator 사용 예제는 오류를 반환하는 함수 안에서 작성합니다:

```go
for server, err := range service.Servers.List(ctx, resource.WithPageSize(100)) {
    if err != nil { return err }
    fmt.Println(server.ID)
    if shouldStop { break }
}
```

에러는 `(nil, error)`로 한 번 전달하고 순회를 종료합니다. iterator를 다시 순회하면 새로운 API 요청이 시작됩니다. All 도중 오류가 발생하면 부분 결과 대신 `nil, error`를 반환합니다.

## 오류와 상태 대기

`errors.Is`로 `ErrNotFound`, `ErrAmbiguous`, `ErrUnsupported`, `ErrInvalidOption`, `ErrFailedState`를 구분합니다. `errors.As`로 `NotFoundError`, `AmbiguousError`, `FailedStateError`, `OperationError`의 상세 정보를 확인합니다. HTTP에서 발생한 미존재는 원래 Gophercloud 오류도 보존합니다.

Wait의 기본 timeout은 5분, 간격은 2초입니다. 부모 context가 더 먼저 종료되면 취소됩니다. 대상 상태 비교는 대소문자를 구분하지 않습니다. 실패 상태는 서비스별 Adapter가 선언합니다. 대상 상태와 실패 상태가 같다면 대상 도달을 먼저 판정합니다.

Wait는 없는 ID나 삭제된 리소스를 계속 기다리지 않고 조회 오류를 반환합니다. WaitDeleted는 이미 없는 리소스에도 성공하며, 인증 오류나 서버 오류를 삭제 완료로 처리하지 않습니다. flavor처럼 상태가 없는 리소스는 Wait/WithStatus를 요청하면 `ErrUnsupported`를 반환합니다.

## SDK 내부 어댑터

`Adapter[T]`는 getter, pager, extractor, ID/name/status 접근 함수를 등록하는 concrete descriptor입니다. 새 서비스 구현은 원래 Gophercloud의 API 호출과 모델 매핑에 집중하고 이름 조회·중복 검사·대기 알고리즘을 재작성하지 않습니다.

`ValidateID`를 지정한 binding은 모든 조회·해석·삭제·대기에 같은 서비스 식별자 문법을 적용합니다. 생략하면 단일 URL path segment 검증을 사용합니다. `NameQueryKey`는 이름 검색의 서버 query 이름을 지정하며 기본값은 `name`입니다. `prefix` 검색을 쓰더라도 공통 계층은 정확한 이름 비교를 추가합니다. 애플리케이션이 이 설정을 구성하지는 않습니다.

Collection과 서비스 객체의 zero value는 사용하지 않습니다. `Connection`이 구성한 서비스에서 가져오는 것이 일반적인 사용 경로입니다. 응답 객체를 수정해도 자동으로 서버에 반영되지 않습니다. 변경은 서비스 API의 typed Update, 또는 부모 범위 객체의 Update에 전달합니다.

부모가 필요한 리소스는 [범위 객체](../docs/scoped-resources.md)를 사용합니다. `dns.RecordSets.InZone(ctx, resource.Name("example.org."))`처럼 부모를 한 번 해석하고, 반환된 객체의 Find/List/Delete/Wait와 Create/Update가 같은 부모를 사용합니다. 기본 식별자 검증은 빈 ID, `.`, `..`와 URL 경로·query 문자가 들어간 ID를 요청 전에 거부합니다.

테스트는 [collection_test.go](collection_test.go)와 [서비스 공통 통합 테스트](../collections_test.go)에 있습니다.

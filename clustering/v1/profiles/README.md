# Senlin Profiles

고정 openstacksdk의 `conn.clustering` profile 연산을 concrete 옵션과 `With...` 함수로 사용합니다. 요청 builder interface를 구현할 필요가 없습니다.

| openstacksdk | Go | HTTP 계약 |
|---|---|---|
| `create_profile(name=..., spec=..., metadata=...)` | `Profiles.Create(ctx, CreateOpts, options...)` | POST `/profiles`, 201 |
| `get_profile(identity)` | `Profiles.Get(ctx, identity)` | GET `/profiles/{identity}`, 200 |
| `profiles(**query)` | `Profiles.List/All(ctx, options...)` | GET `/profiles`, 200 |
| `update_profile(profile, **attrs)` | `Profiles.Update(...)` 또는 `Load/Track → Edit → Commit` | 객체 PATCH `/profiles/{identity}`, 200; tracked는 dirty 필드만 보내고 clean이면 HTTP 없음 |
| `delete_profile(profile, ignore_missing=True)` | `Profiles.Delete(ctx, ref, options...)` | DELETE, 204; 기본 미존재 무시 |
| `find_profile(identity, ignore_missing=True)` | `Profiles.FindIdentity(ctx, identity, options...)` | GET 후 400·403·404 목록 fallback, 정확 ID 또는 이름 검색 |
| `validate_profile(spec=...)` | `Profiles.Validate(ctx, ValidateOpts, options...)` | POST `/profiles/validate`, 200; 숫자 1.2 이상 |

## 생성과 검증

다음 Go 예제는 `ctx context.Context`, `conn *gophercloudsdk.Connection`을 사용하는 함수 안에서 실행합니다. `profiles`는 `gophercloudsdk/clustering/v1/profiles`, `resource`는 `gophercloudsdk/resource`입니다.

```go
service, err := conn.Clustering(ctx)
if err != nil { return err }
spec := map[string]any{
    "type": "os.nova.server",
    "version": "1.0",
    "properties": map[string]any{"name": "worker", "flavor": "FLAVOR_ID", "image": "IMAGE_ID"},
}
profile, err := service.Profiles.Create(ctx, profiles.CreateOpts{Name: "worker_template"},
    profiles.WithCreateSpec(spec),
    profiles.WithCreateMetadata(map[string]any{"role": "worker"}))
if err != nil { return err }
fmt.Println(profile.ID, profile.UserMetadata, profile.Header)
```

name은 ASCII 문자로 시작하고 ASCII 문자·숫자·`_`·`.`·`-`로 구성하는 255자 미만 값입니다. spec은 필수 JSON 객체이며 plugin의 properties schema는 Senlin 서버가 검사합니다. 예제의 FLAVOR_ID/IMAGE_ID는 실제 클라우드 값으로 교체합니다. [Nova profile spec](https://docs.openstack.org/senlin/2023.2/user/profile_types/nova.html)을 참고합니다. `WithCreateSpec`, `WithCreateMetadata`, `With...Options`는 생성 시 입력을 깊게 복사하고 재사용할 때 새 복사본을 만듭니다. 정확한 큰 정수와 소수는 `json.Number` 또는 `json.RawMessage`를 사용합니다.

```go
service, err := conn.Clustering(ctx)
if err != nil { return err }
checked, err := service.Profiles.Validate(ctx, profiles.ValidateOpts{},
    profiles.WithValidateSpec(map[string]any{
        "type": "os.nova.server", "version": "1.0",
        "properties": map[string]any{"flavor": "FLAVOR_ID", "image": "IMAGE_ID"},
    }))
if err != nil { return err }
fmt.Println(checked.Spec, checked.Body["id"])
```

Validate 전에 Connection을 `sdk.WithMicroversion(sdk.Clustering, "1.2")` 또는 적합한 숫자 버전을 협상하는 range로 구성합니다. API는 공유 client를 자동으로 업그레이드하지 않습니다. 미선택·1.0·1.1·symbolic `latest`는 요구 버전을 증명하지 못하므로 요청 전에 거부합니다. 검증 응답의 ID나 이름이 null/생략이면 원문에 그대로 보존하며 route ID를 합성하지 않습니다.

## 갱신과 JSON presence

```go
service, err := conn.Clustering(ctx)
if err != nil { return err }
updated, err := service.Profiles.Update(ctx, resource.Name("worker_template"), profiles.UpdateOpts{},
    profiles.WithUpdateMetadata(map[string]any{}),
    profiles.WithUpdateHeader("X-Vendor-Mode", "explicit"))
if err != nil { return err }
fmt.Println(updated.ID, updated.UserMetadata)
```

갱신 가능한 core 필드는 name과 metadata입니다. spec/type/identity/소유자/타임스탬프는 core 갱신 필드가 아닙니다. name은 `WithUpdateName(name)` 또는 `UpdateOpts{Name: &name}`으로 지정합니다. 추가 서버 확장은 `WithUpdateField`로 지정하고 core 필드 덮어쓰기는 거부합니다. body와 header는 이름 조회 전에 고정하며 정확한 이름을 한 번 해석한 ID로 PATCH합니다.

| metadata 입력 | 전송 의미 |
|---|---|
| `UpdateOpts{}`의 nil RawMessage | 생략 |
| `WithUpdateMetadata(nil)` | 명시 JSON null |
| `WithUpdateMetadata(map[string]any{})` | 명시 빈 객체 |
| `WithUpdateMetadata(map[string]any{...})` | 지정한 객체 |

서버의 metadata null 허용 여부와 의미는 deployment가 결정합니다. SDK가 null을 전송할 수 있다는 사실만으로 삭제나 초기화 성공을 보장하지 않습니다. Go Update는 명시적인 stateless 요청이며 빈 갱신을 요청 전에 거부합니다.

변경 추적은 SDK 소유 `TrackedProfile`로 사용합니다. 앱 개발자가 lifecycle interface나 builder를 구현할 필요가 없습니다.

```python
profile = conn.clustering.get_profile("PROFILE_ID")
profile.name = "renamed_template"
profile.metadata = {"role": "worker"}
profile = conn.clustering.update_profile(profile)
```

```go
service, err := conn.Clustering(ctx)
if err != nil { return err }
tracked, err := service.Profiles.Load(ctx, resource.ID("PROFILE_ID"))
if err != nil { return err }
if err := tracked.Edit(profiles.UpdateOpts{},
    profiles.WithUpdateName("renamed_template"),
    profiles.WithUpdateMetadata(map[string]any{"role": "worker"})); err != nil {
    return err
}
profile, err := tracked.Commit(ctx)
if err != nil { return err }
fmt.Println(profile.Name, tracked.Dirty(), tracked.Response().Body)
```

기존 모델은 `service.Profiles.Track(profile)`로 감쌀 수 있습니다. 같은 현재 값의 대입과 clean Commit은 HTTP를 생략하지만 `A → B → A`는 dirty로 남습니다. `RemoveName`·`RemoveMetadata`는 캐시 필드 삭제와 null PATCH를 구분해 처리하며 서버의 null 허용은 별도입니다. 성공 응답은 필드별로 캐시에 병합하고 nested 객체는 전체 교체하며, 응답에서 생략한 전송 필드도 clean으로 바뀝니다. HTTP 처리 중 새 Edit은 보존합니다.

`Value()`는 병합 캐시 snapshot, `Response()`는 마지막 성공 응답의 필드 snapshot입니다. Track 입력의 Body가 있으면 typed 필드보다 우선하며 수동 Body 없는 모델은 HTTP 증거를 만들지 않는 로컬 seed입니다. 요청 ID는 고정하고 getter 모델이나 입력을 변경해도 handle은 바뀌지 않습니다. 실패 시 dirty를 유지하며 accepted 응답 오류 후 SDK가 자동 재전송하지 않습니다. `Refresh`로 서버 상태를 확인할 수 있습니다. [전체 Python/Go 변경 추적 계약](../tracking/README.md)을 참고합니다.

## 목록과 이름

```go
service, err := conn.Clustering(ctx)
if err != nil { return err }
for profile, err := range service.Profiles.List(ctx,
    profiles.WithListOptions(profiles.ListOpts{Limit: 20, Sort: "created_at:desc"}),
    profiles.WithListGlobalProject(false),
    profiles.WithListFilter("metadata", map[string]any{"role": "worker"})) {
    if err != nil { return err }
    fmt.Println(profile.ID, profile.Name)
}
found, err := service.Profiles.Find(ctx, resource.Name("worker_template"))
if err != nil { return err }
if found != nil { fmt.Println(found.ID) }
```

typed server query는 limit/marker/name/type/sort/global_project입니다. 0/빈 값은 생략하며 global_project의 nil과 명시 false를 구분합니다. sort grammar는 `key[:asc|desc]`의 쉼표 목록이며 실제 정렬 키 지원은 서버가 검사합니다. 기본 limit은 생략합니다. 양의 limit을 지정하면 짧은 페이지를 포함한 모든 nonempty 페이지 뒤에 마지막 **wire 응답 ID**로 marker를 만들고 빈 페이지에서 멈춥니다. 필터로 제외되거나 사용자가 수정한 마지막 모델도 wire marker를 바꾸지 않습니다.

`WithListFilter`는 metadata/spec/id/name/type/project/domain/user/created_at/updated_at을 로컬에서 검사하며 project_id/domain_id/user_id 별칭을 지원합니다. 객체는 재귀 부분집합, 배열은 순서와 전체 값으로 비교합니다. 누락 필드는 null로 비교합니다. 빈 실제 객체는 Python dict filter처럼 객체 필터와 일치하지 않습니다. Go는 bool과 숫자를 구분하고 decimal 숫자는 float64 없이 정확하게 비교합니다. 이 필터는 서버 query로 보내지 않습니다.

`WithListQuery`는 명시적인 추가 서버 query이며 알려진 core/local 필드의 덮어쓰기는 거부합니다. Python이 모르는 query를 생략하는 동작과 다릅니다. body/header next 링크는 같은 collection origin/path에만 허용하며 최초 필터·정렬·limit을 유지합니다. 반복 URL/marker와 필터를 바꾸는 continuation은 오류입니다. iterator는 lazy하고 재순회할 수 있으며 break는 후속 요청을 막습니다.

Get은 컨트롤러의 이름·UUID·짧은 ID를 직접 받습니다. `resource.ID(value)`는 UUID 형태를 추측하지 않고 직접 route로 사용합니다. `resource.Name(value)`는 목록에서 정확히 해석하고 중복 이름을 오류로 처리합니다. `API.Find`는 Python proxy처럼 기본 미존재 무시이며 `resource.WithMissingError()`로 바꿀 수 있습니다. `Resources.Find`의 공통 기본값은 미존재 오류입니다. `FindIdentity(ctx, identity)`는 GET-first와 400·403·404 목록 fallback을 제공합니다. [자동 조회와 옵션](../finding/README.md)을 참고합니다.

## 결과와 남은 범위

결과의 wire `metadata`는 `Profile.UserMetadata`이며 HTTP 증거인 embedded `resource.Metadata`와 구분합니다. spec과 사용자 metadata는 raw JSON 숫자를 보존하고 `Body`는 null/생략·unknown 필드, `Header`/`StatusCode`는 HTTP 응답을 보존합니다. 단건 응답은 `profile` 객체 envelope를 요구합니다. Python의 flat/empty fallback을 적용하지 않습니다. 승인된 mutation의 decode/read 실패는 `resource.ResponseError`에 원문·header·status를 남기며 mutation을 재전송하지 않습니다. Delete는 ignored404 외의 권한·사용 중 오류를 숨기지 않으며, 찾은 응답 ID가 없으면 삭제 요청을 보내지 않습니다.

Profile update의 dirty/no-op/null 삭제/응답 병합과 reset은 [tracked lifecycle](../tracking/README.md)로 제공하며 사용법과 테스트 근거를 기준으로 Go mapping 판정합니다. readonly 보호, snapshot 반환, strict envelope와 선택 client의 버전·인증 소유권은 문서화한 Go 정책입니다. 상속한 목록의 per-call base_path와 proxy JMESPath는 남은 비교·구현 범위입니다. combined-string Find는 [FindIdentity](../finding/README.md)로 제공합니다. 직접 연산 7개가 제공된 상태와 Python proxy 전체 계약 완료를 구분합니다.

근거: [고정 Profile](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/profile.py), [고정 proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/_proxy.py), [공식 API](https://docs.openstack.org/api-ref/clustering/). [HTTP 계약](../../../api/clustering_profiles_test.go), [lookup snapshot 회귀](../../../api/clustering_profiles_update_snapshot_test.go), [tracked HTTP 회귀](../../../api/clustering_lifecycle_test.go), [지원 판정](../../../docs/sdk-support-ledger.md).

## 목록 소비 제어

| pinned Python | Go 옵션 | 소비 정책 |
|---|---|---|
| `max_items=n` | `WithListMaxItems(n)` 또는 `ListOpts.MaxItems` | 로컬 필터 이전에 검증한 raw 행을 최대 n개 소비; 0은 무제한, 음수는 사전 오류 |
| `paginated=False` | `WithListPaginated(false)` 또는 `ListOpts.Paginated` | 첫 응답만 소비하고 continuation을 처리하지 않음 |
| 기본 `paginated=True` | nil 또는 `WithListPaginated(true)` | 페이지 순회를 허용; 뒤의 옵션이 앞의 값을 덮어씀 |
| `limit=n` | `WithListOptions(ListOpts{Limit: n})` | 양수는 wire page limit; 로컬 cap과 독립 |

```go
listingAPI := profiles.New(client)
values, err := listingAPI.All(ctx,
    profiles.WithListOptions(profiles.ListOpts{Limit: 20}),
    profiles.WithListMaxItems(50))
if err != nil { return err }
fmt.Println(len(values))
for value, err := range listingAPI.List(ctx, profiles.WithListPaginated(false)) {
    if err != nil { return err }
    fmt.Println(value.ID)
}
```

명시한 wire limit이 없으면 양의 `MaxItems`를 limit hint로 보냅니다. 명시 limit은 그대로
유지하고 로컬 cap은 응답이 그 limit보다 커도 적용합니다. 반환 수는 로컬 필터나 서버의
page 정책에 따라 cap보다 적을 수 있습니다.

`max_items`와 `paginated`는 서버 query로 보내지 않으며 `WithListQuery`에서 같은 이름을
사용하면 사전 오류입니다. `WithListOptions`는 bool pointer도 snapshot으로 소유하고 재사용 시
독립적으로 적용합니다. cap에 도달하면 뒤의 행이나 next link를 처리하지 않지만 소비한 행의
잘못된 JSON·검증 오류는 전체 페이지 증거와 함께 반환합니다. 빈 페이지에서는 next link가
있어도 끝냅니다. `break`, context와 매 페이지의 source/version 검사도 유지합니다.
Pinned Python은 정확한 page 경계에서 cap 검사를 다음 raw 행까지 미뤄 continuation GET을
한 번 더 할 수 있지만 Go는 cap 직후 끝냅니다.

List 전체 계약은 partial입니다. 알려진 `WithListFilter`의 raw JSON 비교와 별도로 Python
Resource field/default/alias 정규화 및 query 소비, per-call base_path와
deprecated JMESPath는 계속 비교합니다. `WithListQuery`는 vendor query를 실제로 전달하는
Go 확장이며 Python unknown query 생략과 구별합니다.

공통 소비 정책과 남은 차이는 [Senlin 목록 제어](../listing/README.md), 실제 HTTP 근거는 [목록 제어 테스트](../../../api/clustering_typed_list_controls_test.go)를 참고합니다.

## 목록 호출별 헤더와 버전

`WithListHeader(key, value)`와 `WithListMicroversion("1.7")`은 이 패키지의 typed `List` /
`All`에만 적용합니다. 기본값은 source client 설정이며 명시 옵션은 공유 클라이언트를
수정하지 않습니다. 실제 wire 헤더·버전 선택, 재순회·페이지·인증 정책과 Python 비교 예제는
[Senlin 목록 호출 옵션](../listing/README.md#목록-호출별-헤더와-버전)을 참고합니다.
`headers`, `microversion`, `base_path`를 `WithListQuery`로 전달하면 HTTP 전에 오류입니다.

## 문자열 이름/ID 자동 조회

Python `find_profile(identity, ignore_missing=False)`는 `FindIdentity`와 `WithFindIgnoreMissing(false)`로 호출합니다. SDK가 GET-first, literal 이름 query, 모든 advertised 페이지의 정확한 ID/이름 일치와 중복·후속 오류를 처리합니다. 기본 미존재는 `nil, nil`이며 아래 예제는 strict입니다.

```go
package example

import (
    "context"

    sdk "gophercloudsdk"
    "gophercloudsdk/clustering/v1/profiles"
)

func FindProfileStrict(ctx context.Context, conn *sdk.Connection) (*profiles.Profile, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    return service.Profiles.FindIdentity(ctx, "worker_template",
        profiles.WithFindIgnoreMissing(false))
}
```

`WithFindFallback`로 404-only·GET-only 정책을 선택하고 `WithFindHeader`·`WithFindMicroversion`으로 GET과 fallback의 동일한 호출 설정을 지정합니다. 원본 client·다른 호출은 변경하지 않습니다. [공통 FindIdentity 계약과 Python/Go 차이](../finding/README.md)에 입력 segment 정책·응답 canonical ID·오류 근거·옵션 snapshot을 설명합니다. 기존 `Find(ctx, resource.ID/Name(...))`는 명시한 경로와 기존 옵션을 유지합니다.

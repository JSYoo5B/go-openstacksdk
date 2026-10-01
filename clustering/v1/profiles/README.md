# Senlin Profiles

고정 openstacksdk의 `conn.clustering` profile 연산을 concrete 옵션과 `With...` 함수로 사용합니다. 요청 builder interface를 구현할 필요가 없습니다.

| openstacksdk | Go | HTTP 계약 |
|---|---|---|
| `create_profile(name=..., spec=..., metadata=...)` | `Profiles.Create(ctx, CreateOpts, options...)` | POST `/profiles`, 201 |
| `get_profile(identity)` | `Profiles.Get(ctx, identity)` | GET `/profiles/{identity}`, 200 |
| `profiles(**query)` | `Profiles.List/All(ctx, options...)` | GET `/profiles`, 200 |
| `update_profile(profile, **attrs)` | `Profiles.Update(ctx, ref, UpdateOpts, options...)` | 객체 PATCH `/profiles/{identity}`, 200 |
| `delete_profile(profile, ignore_missing=True)` | `Profiles.Delete(ctx, ref, options...)` | DELETE, 204; 기본 미존재 무시 |
| `find_profile(identity, ignore_missing=True)` | `Profiles.Find(ctx, ref, options...)` | 명시한 ID 또는 정확한 이름 검색 |
| `validate_profile(spec=...)` | `Profiles.Validate(ctx, ValidateOpts, options...)` | POST `/profiles/validate`, 200; 숫자 1.2 이상 |

## 생성과 검증

다음 Go 예제는 `ctx context.Context`, `conn *gophercloudsdk.Connection`을 사용하는 함수 안에서 실행합니다. `profiles`는 `gophercloudsdk/clustering/v1/profiles`, `resource`는 `gophercloudsdk/resource`입니다.

```go
service, err := conn.Clustering(ctx)
if err != nil { return err }
spec := map[string]any{
    "type": "os.nova.server",
    "version": "1.0",
    "properties": map[string]any{"name": "worker", "flavor": "m1.small"},
}
profile, err := service.Profiles.Create(ctx, profiles.CreateOpts{Name: "worker_template"},
    profiles.WithCreateSpec(spec),
    profiles.WithCreateMetadata(map[string]any{"role": "worker"}))
if err != nil { return err }
fmt.Println(profile.ID, profile.UserMetadata, profile.Header)
```

name은 ASCII 문자로 시작하고 ASCII 문자·숫자·`_`·`.`·`-`로 구성하는 255자 미만 값입니다. spec은 필수 JSON 객체이며 plugin의 properties schema는 Senlin 서버가 검사합니다. `WithCreateSpec`, `WithCreateMetadata`, `With...Options`는 생성 시 입력을 깊게 복사하고 재사용할 때 새 복사본을 만듭니다. 정확한 큰 정수와 소수는 `json.Number` 또는 `json.RawMessage`를 사용합니다.

```go
service, err := conn.Clustering(ctx)
if err != nil { return err }
checked, err := service.Profiles.Validate(ctx, profiles.ValidateOpts{},
    profiles.WithValidateSpec(map[string]any{
        "type": "os.nova.server", "version": "1.0", "properties": map[string]any{},
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

갱신 가능한 core 필드는 name과 metadata입니다. spec/type/identity/소유자/타임스탬프는 core 갱신 필드가 아닙니다. name은 `UpdateOpts{Name: &name}`으로 지정합니다. 추가 서버 확장은 `WithUpdateField`로 지정하고 core 필드 덮어쓰기는 거부합니다. body와 header는 이름 조회 전에 고정하며 정확한 이름을 한 번 해석한 ID로 PATCH합니다.

| metadata 입력 | 전송 의미 |
|---|---|
| `UpdateOpts{}`의 nil RawMessage | 생략 |
| `WithUpdateMetadata(nil)` | 명시 JSON null |
| `WithUpdateMetadata(map[string]any{})` | 명시 빈 객체 |
| `WithUpdateMetadata(map[string]any{...})` | 지정한 객체 |

서버의 metadata null 허용 여부와 의미는 deployment가 결정합니다. SDK가 null을 전송할 수 있다는 사실만으로 삭제나 초기화 성공을 보장하지 않습니다. Go Update는 명시적인 stateless 요청이며 빈 갱신을 요청 전에 거부합니다. Python의 기존 Resource 변경 추적, 같은 값/no-change에서 HTTP 생략, 응답 병합과 dirty reset은 아직 별도 과제입니다.

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

Get은 컨트롤러의 이름·UUID·짧은 ID를 직접 받습니다. `resource.ID(value)`는 UUID 형태를 추측하지 않고 직접 route로 사용합니다. `resource.Name(value)`는 목록에서 정확히 해석하고 중복 이름을 오류로 처리합니다. `API.Find`는 Python proxy처럼 기본 미존재 무시이며 `resource.WithMissingError()`로 바꿀 수 있습니다. `Resources.Find`의 공통 기본값은 미존재 오류입니다. Python combined-string Find의 ID-first GET과 400/403/404 후 목록 fallback은 아직 별도 비교 범위입니다.

## 결과와 남은 범위

결과의 wire `metadata`는 `Profile.UserMetadata`이며 HTTP 증거인 embedded `resource.Metadata`와 구분합니다. spec과 사용자 metadata는 raw JSON 숫자를 보존하고 `Body`는 null/생략·unknown 필드, `Header`/`StatusCode`는 HTTP 응답을 보존합니다. 단건 응답은 `profile` 객체 envelope를 요구합니다. Python의 flat/empty fallback을 적용하지 않습니다. 승인된 mutation의 decode/read 실패는 `resource.ResponseError`에 원문·header·status를 남기며 mutation을 재전송하지 않습니다. Delete는 ignored404 외의 권한·사용 중 오류를 숨기지 않으며, 찾은 응답 ID가 없으면 삭제 요청을 보내지 않습니다.

상속한 paginated/base_path/max_items, per-call microversion/header, proxy JMESPath와 Resource field normalization/cache/dirty lifecycle은 남은 비교·구현 범위입니다. 직접 연산 7개가 제공된 상태와 Python proxy 전체 계약 완료를 구분합니다.

근거: [고정 Profile](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/profile.py), [고정 proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/_proxy.py), [공식 API](https://docs.openstack.org/api-ref/clustering/). [HTTP 계약](../../../api/clustering_profiles_test.go), [lookup snapshot 회귀](../../../api/clustering_profiles_update_snapshot_test.go), [지원 판정](../../../docs/sdk-support-ledger.md).

# Senlin profile types

| Python openstacksdk | Go |
| --- | --- |
| `conn.clustering.profile_types()` | `api.List(ctx)` 또는 `api.All(ctx)` |
| `conn.clustering.get_profile_type(name)` | `api.Get(ctx, name)` |
| `conn.clustering.list_profile_type_operations(name)` | `api.Operations(ctx, name)` |

```go
api := profiletypes.New(client)
profile, err := api.Get(ctx, "os.nova.server-1.0")
if err != nil {
    return err
}
fmt.Println(profile.Name, profile.Schema)

for profile, err := range api.List(ctx) {
    if err != nil {
        return err
    }
    fmt.Println(profile.Name, profile.Version)
}

// Configure the selected version before concurrent client use.
client.Microversion = "1.4"
ops, err := api.Operations(ctx, "os.nova.server-1.0")
```

`client.Type`는 `clustering`이며 endpoint는 Senlin v1입니다. 타입 이름이 경로 ID입니다.
점과 버전 접미사를 허용하며 UUID 추측이나 사전 List 요청을 하지 않습니다.
응답의 `Name`과 별도 `Version`을 그대로 보존하고 둘을 합쳐 새로운 ID를 만들지 않습니다.
`Schema`와 operation별 값은 plugin마다 달라지는 JSON이며 `json.RawMessage`로
정밀도와 확장 필드를 유지합니다. `SupportStatus`도 원본 JSON입니다.

Operation 조회는 선택된 숫자 마이크로버전 1.4 이상을 요구합니다. 미선택, 낮은 버전,
`latest`는 HTTP 전에 거부하며 공유 client를 자동 변경하지 않습니다. 1.5의 catalog
응답은 별도 version과 support status를 추가할 수 있으므로 해당 필드의 부재는 유효합니다.

List는 lazy iterator이고 `break`하면 다음 페이지를 요청하지 않습니다. pinned Python이
상속하는 body next/links와 HTTP Link를 지원하되 동일 origin과 collection 경로만 허용하고
원래 필터를 유지합니다. 타입 catalog의 공식 API는 limit/marker를 선언하지 않습니다.
다음 inherited 옵션과 query 확장은 deployment가 지원하는 경우에만 사용합니다.

```go
values, err := api.All(ctx,
    profiletypes.WithListOptions(profiletypes.ListOpts{Limit: 20}),
    profiletypes.WithListQuery("vendor_filter", "enabled"),
)
```

기본 Limit 0과 빈 Marker는 query를 생략합니다. `limit`/`marker`는 concrete 옵션으로만
설정합니다. 페이지 크기만으로 다음 marker를 생성하지 않습니다. `Resources` 필드는
공유 typed collection이며 Delete와 상태 Wait는 지원하지 않습니다.

결과의 `Body`, `Header`, `StatusCode`는 raw JSON과 HTTP 근거를 담습니다. HTTP 오류의
원래 status/body/header와 malformed accepted response의 `resource.ResponseError`를
보존합니다. Python의 mutable Resource 전체 또는 Senlin 전체 parity를 주장하지 않습니다.
Python의 permissive flat/empty 응답 fallback과 달리 Get은 `profile_type` object,
Operations는 root `operations` object를 검증합니다. Get은 type 이름 문자열을 받고
응답을 변경해도 자동으로 commit하지 않습니다.

근거: pinned openstacksdk revision `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의
`profile_type.py`, `_proxy.py`, 상속 `resource.py`와
[공식 profile type API](https://docs.openstack.org/api-ref/clustering/#profile-types-profile-types).
GET 성공 코드는 200입니다.

## 목록 소비 제어

| pinned Python | Go 옵션 | 소비 정책 |
|---|---|---|
| `max_items=n` | `WithListMaxItems(n)` 또는 `ListOpts.MaxItems` | 로컬 필터 이전에 검증한 raw 행을 최대 n개 소비; 0은 무제한, 음수는 사전 오류 |
| `paginated=False` | `WithListPaginated(false)` 또는 `ListOpts.Paginated` | 첫 응답만 소비하고 continuation을 처리하지 않음 |
| 기본 `paginated=True` | nil 또는 `WithListPaginated(true)` | 페이지 순회를 허용; 뒤의 옵션이 앞의 값을 덮어씀 |
| `limit=n` | `WithListOptions(ListOpts{Limit: n})` | 양수는 wire page limit; 로컬 cap과 독립 |

```go
listingAPI := profiletypes.New(client)
values, err := listingAPI.All(ctx,
    profiletypes.WithListOptions(profiletypes.ListOpts{Limit: 20}),
    profiletypes.WithListMaxItems(50))
if err != nil { return err }
fmt.Println(len(values))
for value, err := range listingAPI.List(ctx, profiletypes.WithListPaginated(false)) {
    if err != nil { return err }
    fmt.Println(value.Name)
}
```

명시한 wire limit이 없으면 양의 `MaxItems`를 limit hint로 보냅니다. 명시 limit은 그대로
유지하고 로컬 cap은 응답이 그 limit보다 커도 적용합니다. 반환 수는 로컬 필터나 서버의
page 정책에 따라 cap보다 적을 수 있습니다.
타입 catalog와 services의 공식 API는 limit/marker를 선언하지 않으므로 hint와 명시 limit의
지원은 deployment가 판정합니다. 이들 목록은 server next link만 따라가고 ID marker fallback을
만들지 않습니다.

`max_items`와 `paginated`는 서버 query로 보내지 않으며 `WithListQuery`에서 같은 이름을
사용하면 사전 오류입니다. `WithListOptions`는 bool pointer도 snapshot으로 소유하고 재사용 시
독립적으로 적용합니다. cap에 도달하면 뒤의 행이나 next link를 처리하지 않지만 소비한 행의
잘못된 JSON·검증 오류는 전체 페이지 증거와 함께 반환합니다. 빈 페이지에서는 next link가
있어도 끝냅니다. `break`, context와 매 페이지의 source/version 검사도 유지합니다.
Pinned Python은 정확한 page 경계에서 cap 검사를 다음 raw 행까지 미뤄 continuation GET을
한 번 더 할 수 있지만 Go는 cap 직후 끝냅니다.

List 전체 계약은 partial입니다. Python의 name/schema/support_status Body local filters와
unknown query 생략, per-call base_path/microversion/header, deprecated JMESPath, inherited
limit/marker fallback은 별도 비교 범위입니다. Go의 `WithListQuery`는 명시한 vendor query를
실제로 전달하며 stock 서버 지원을 보장하지 않습니다.

공통 소비 정책과 남은 차이는 [Senlin 목록 제어](../listing/README.md), 실제 HTTP 근거는 [목록 제어 테스트](../../../api/clustering_catalog_list_controls_test.go)를 참고합니다.

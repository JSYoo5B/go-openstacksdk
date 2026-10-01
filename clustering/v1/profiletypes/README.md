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

근거: pinned openstacksdk revision `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의
`profile_type.py`, `_proxy.py`, 상속 `resource.py`와
[공식 profile type API](https://docs.openstack.org/api-ref/clustering/#profile-types-profile-types).
GET 성공 코드는 200입니다.

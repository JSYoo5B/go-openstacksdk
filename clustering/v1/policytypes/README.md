# Senlin policy types

| Python openstacksdk | Go |
| --- | --- |
| `conn.clustering.policy_types()` | `api.List(ctx)` 또는 `api.All(ctx)` |
| `conn.clustering.get_policy_type(name)` | `api.Get(ctx, name)` |

```go
api := policytypes.New(client)
policy, err := api.Get(ctx, "senlin.policy.scaling-1.0")
if err != nil {
    return err
}
fmt.Println(policy.Name, policy.Schema)

for policy, err := range api.List(ctx) {
    if err != nil {
        return err
    }
    fmt.Println(policy.Name)
}
```

타입 이름이 경로 ID이며 점과 버전 접미사를 그대로 허용합니다. `Name`과 별도 `Version`을
임의로 합치지 않습니다. plugin schema는 `map[string]json.RawMessage`, support status는
원본 JSON으로 보존합니다. 추가 응답 필드, null/생략, 숫자 정밀도와 HTTP 근거는
`Body`, `Header`, `StatusCode`에서 확인합니다.

List는 lazy이며 중단하면 추가 요청을 하지 않습니다. pinned Python이 상속하는 body
next/links와 HTTP Link를 지원하며 다음 URL은 같은 origin/collection 경로에 제한됩니다.
공식 catalog API는 pagination query를 선언하지 않습니다. inherited
`WithListOptions(ListOpts{Limit: 20, Marker: "marker"})` 및 `WithListQuery`는 deployment가
지원하는 경우에만 사용합니다. 기본 Limit 0/빈 Marker는 생략하고, full page만 보고
다음 marker를 만들지 않습니다. `Resources`의 Delete와 상태 Wait는 지원하지 않습니다.

이 패키지는 policy type Get/List만 구현합니다. mutable Python Resource나 Senlin 전체
parity를 의미하지 않습니다. Get은 이름 문자열을 받아 공식 `policy_type` object envelope를
요구하며 Python의 flat/empty 응답 fallback을 적용하지 않습니다. 근거는 pinned openstacksdk revision
`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`와
[공식 policy type API](https://docs.openstack.org/api-ref/clustering/#policy-types-policy-types)이며
GET 성공 코드는 200입니다.

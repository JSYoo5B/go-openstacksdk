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

## 목록 소비 제어

| pinned Python | Go 옵션 | 소비 정책 |
|---|---|---|
| `max_items=n` | `WithListMaxItems(n)` 또는 `ListOpts.MaxItems` | 로컬 필터 이전에 검증한 raw 행을 최대 n개 소비; 0은 무제한, 음수는 사전 오류 |
| `paginated=False` | `WithListPaginated(false)` 또는 `ListOpts.Paginated` | 첫 응답만 소비하고 continuation을 처리하지 않음 |
| 기본 `paginated=True` | nil 또는 `WithListPaginated(true)` | 페이지 순회를 허용; 뒤의 옵션이 앞의 값을 덮어씀 |
| `limit=n` | `WithListOptions(ListOpts{Limit: n})` | 양수는 wire page limit; 로컬 cap과 독립 |

```go
listingAPI := policytypes.New(client)
values, err := listingAPI.All(ctx,
    policytypes.WithListOptions(policytypes.ListOpts{Limit: 20}),
    policytypes.WithListMaxItems(50))
if err != nil { return err }
fmt.Println(len(values))
for value, err := range listingAPI.List(ctx, policytypes.WithListPaginated(false)) {
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

List 전체 계약은 partial입니다. Python의 자동 query/Body 분류와 unknown query 생략,
per-call base_path, deprecated JMESPath, inherited
limit/marker fallback은 별도 비교 범위입니다. Go의 `WithListQuery`는 명시한 vendor query를
실제로 전달하며 stock 서버 지원을 보장하지 않습니다.

공통 소비 정책과 남은 차이는 [Senlin 목록 제어](../listing/README.md), 실제 HTTP 근거는 [목록 제어 테스트](../../../api/clustering_catalog_list_controls_test.go)를 참고합니다.

## 타입 catalog Body 필터

Python `policy_types(name="senlin.policy.scaling-1.0")`의 로컬 필터는 Go의
`policytypes.WithListFilter("name", "senlin.policy.scaling-1.0")`로 명시합니다.
지원 필드는 `id/name/schema/support_status`이며 alias는 없습니다. Go 응답 모델의 추가
`version`은 pinned Body 필터 범위가 아니므로 거부합니다. known Body 키를 확장 query로 보내지
못하며 기존 explicit vendor query 정책은 유지합니다.

```go
package examples

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/clustering/v1/policytypes"
)

func FilterPolicyTypes(ctx context.Context, conn *sdk.Connection, typeName string) error {
	service, err := conn.ClusteringV1(ctx)
	if err != nil {
		return err
	}
	values, err := service.PolicyTypes.All(ctx,
		policytypes.WithListFilter("name", typeName),
		policytypes.WithListMaxItems(50), policytypes.WithListPaginated(false))
	if err != nil {
		return err
	}
	for _, policy := range values {
		fmt.Println(policy.Name, policy.Version)
	}
	return nil
}
```

`id` 필터는 raw 응답의 `id`만 비교합니다. 생략된 ID를 name으로 대신 사용하지 않으므로
Python alternate-ID fallback과 구분합니다. 입력은 생성 시 JSON snapshot이며 cap은 필터 전의
응답 행을 셉니다. 객체는 recursive subset, 배열은 순서와 전체 값, 숫자는 정확한 decimal 값으로
비교합니다. bool은 숫자와 다르고 scalar null은 생략과 같으며 빈 실제 객체는 객체 필터에
일치하지 않습니다. [공통 Body 필터 정책](../listing/README.md)과
[HTTP 계약](../../../api/clustering_body_filters_test.go)을 참고합니다.

## 목록 호출별 헤더와 버전

`WithListHeader(key, value)`와 `WithListMicroversion("1.7")`은 이 패키지의 typed `List` /
`All`에만 적용합니다. 기본값은 source client 설정이며 명시 옵션은 공유 클라이언트를
수정하지 않습니다. 실제 wire 헤더·버전 선택, 재순회·페이지·인증 정책과 Python 비교 예제는
[Senlin 목록 호출 옵션](../listing/README.md#목록-호출별-헤더와-버전)을 참고합니다.
`headers`, `microversion`, `base_path`를 `WithListQuery`로 전달하면 HTTP 전에 오류입니다.

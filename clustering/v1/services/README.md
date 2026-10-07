# Senlin engine services

```python
services = conn.clustering.services()
```

```go
// Configure before sharing the client between goroutines.
client.Type = "clustering"
client.Microversion = "1.7"
api := services.New(client)

for service, err := range api.List(ctx) {
    if err != nil {
        return err
    }
    fmt.Println(service.ID, service.Host, service.Status, service.State)
}
```

List는 선택된 숫자 microversion 1.7 이상을 요구합니다. 미선택/낮은 버전/`latest`는 HTTP
전에 거부하고 client를 자동 변경하지 않습니다. ID, binary, host, topic, status/state,
nullable disabled reason, 원래 updated_at 문자열을 제공합니다. `Body`, `Header`,
`StatusCode`에서 확장 JSON과 HTTP 근거를 확인하며 숫자 정밀도를 유지합니다.

List는 lazy입니다. body next/links와 HTTP Link는 동일 origin/collection 경로에 제한하고
기존 query를 유지합니다. 공식 API는 pagination query를 선언하지 않으므로
`WithListOptions(ListOpts{Limit: 20, Marker: "marker"})` 및 `WithListQuery`는 deployment가
지원하는 경우에만 사용합니다. 기본값은 query 생략이며 full page만 보고 다음 marker를
생성하지 않습니다. 공유 `Resources.List(..., resource.WithStatus("enabled"))`는 응답을
로컬 필터링하며 status query를 보내지 않습니다.

서비스 API에는 단건 Get/변경/삭제가 없습니다. Status 필드만으로 poll 가능한 API를
추가하지 않으며 `Resources.Get`과 ID status Wait도 지원하지 않습니다. Python mutable
Resource 및 Senlin 전체 parity는 별도 범위입니다.

근거: pinned openstacksdk revision `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`와
[공식 Services API](https://docs.openstack.org/api-ref/clustering/#services-services). List 성공은 200입니다.

## 목록 소비 제어

| pinned Python | Go 옵션 | 소비 정책 |
|---|---|---|
| `max_items=n` | `WithListMaxItems(n)` 또는 `ListOpts.MaxItems` | 로컬 필터 이전에 검증한 raw 행을 최대 n개 소비; 0은 무제한, 음수는 사전 오류 |
| `paginated=False` | `WithListPaginated(false)` 또는 `ListOpts.Paginated` | 첫 응답만 소비하고 continuation을 처리하지 않음 |
| 기본 `paginated=True` | nil 또는 `WithListPaginated(true)` | 페이지 순회를 허용; 뒤의 옵션이 앞의 값을 덮어씀 |
| `limit=n` | `WithListOptions(ListOpts{Limit: n})` | 양수는 wire page limit; 로컬 cap과 독립 |

```go
listingAPI := services.New(client)
values, err := listingAPI.All(ctx,
    services.WithListOptions(services.ListOpts{Limit: 20}),
    services.WithListMaxItems(50))
if err != nil { return err }
fmt.Println(len(values))
for value, err := range listingAPI.List(ctx, services.WithListPaginated(false)) {
    if err != nil { return err }
    fmt.Println(value.ID)
}
```

위 예제도 선택된 numeric microversion 1.7 이상에서 실행합니다.

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

List 전체 계약은 partial입니다. 명시적인 typed Body 필터와 별도로 Python의 자동 query/Body
분류와 unknown query 생략, per-call
base_path, deprecated JMESPath와 inherited limit/marker fallback을
계속 비교합니다. `WithListQuery`는 명시한 vendor query를 실제로 전달하는 Go 확장입니다.

공통 소비 정책과 남은 차이는 [Senlin 목록 제어](../listing/README.md), 실제 HTTP 근거는 [목록 제어 테스트](../../../api/clustering_catalog_list_controls_test.go)를 참고합니다.

## 서비스 Body 필터

Python `services(state="up", disabled_reason=None)`의 로컬 비교는
`services.WithListFilter("state", "up")`, `WithListFilter("disabled_reason", nil)`로 지정합니다.
지원 필드는 `id/name/status/state/binary/disabled_reason/host/updated_at`이며 별도 alias는 없습니다.
Go 모델의 추가 `topic`과 unknown 필드는 이 옵션에서 거부합니다. 필터는 wire query를
추가하지 않으며 known Body 키를 `WithListQuery`로 보내는 것도 거부합니다.

아래 함수는 Connection에 선택된 numeric Senlin microversion이 1.7 이상일 때 실행합니다.
필터 옵션이 버전을 자동으로 변경하거나 서비스 Get을 추가하지 않습니다.

```go
package examples

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/clustering/v1/services"
)

func FilterServices(ctx context.Context, conn *sdk.Connection) error {
	service, err := conn.ClusteringV1(ctx)
	if err != nil {
		return err
	}
	values, err := service.Services.All(ctx,
		services.WithListFilter("state", "up"),
		services.WithListFilter("disabled_reason", nil),
		services.WithListMaxItems(50), services.WithListPaginated(false))
	if err != nil {
		return err
	}
	for _, value := range values {
		fmt.Println(value.ID, value.Host, value.State)
	}
	return nil
}
```

필터 입력은 생성 시 snapshot으로 소유하고 cap은 필터 이전의 raw 행을 셉니다.
비교는 raw Body의 JSON 타입을 유지하고 숫자는 decimal 값으로 정확하게 비교하며 bool과
구분합니다. scalar null은 생략과 같고 객체는 recursive subset이지만 빈 실제 객체는
일치하지 않습니다. 공통 `Resources`의 status 필터는 기존의 별도 경로이며 이 typed 옵션이
공통 Collection의 임의 Body 필터 선택을 추가한 것은 아닙니다. 상세한
[공통 정책](../listing/README.md)과 [HTTP 계약](../../../api/clustering_body_filters_test.go)을 참고합니다.

## 목록 호출별 헤더와 버전

`WithListHeader(key, value)`와 `WithListMicroversion("1.7")`은 이 패키지의 typed `List` /
`All`에만 적용합니다. 기본값은 source client 설정이며 명시 옵션은 공유 클라이언트를
수정하지 않습니다. 실제 wire 헤더·버전 선택, 재순회·페이지·인증 정책과 Python 비교 예제는
[Senlin 목록 호출 옵션](../listing/README.md#목록-호출별-헤더와-버전)을 참고합니다.
`headers`, `microversion`, `base_path`를 `WithListQuery`로 전달하면 HTTP 전에 오류입니다.

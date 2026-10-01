# Senlin events: Get / List

```python
event = conn.clustering.get_event(event_id)
events = conn.clustering.events(obj_id=node_id, limit=20)
```

```go
api := events.New(client)
event, err := api.Get(ctx, eventID)
if err != nil {
    return err
}
fmt.Println(event.ObjectID, event.GeneratedAt, event.Level)

records, err := api.All(ctx, events.WithListOptions(events.ListOpts{
    Limit:       20,
    ObjectIDs:   []string{nodeID, otherNodeID},
    ObjectTypes: []string{"NODE", "CLUSTER"},
    Sort:        "timestamp:desc",
}))
```

Python의 obj_id/obj_name/obj_type은 Go의 ObjectIDs/ObjectNames/ObjectTypes이며 wire에서는
oid/oname/otype입니다. 각 slice와 Actions는 반복 query를 보존하고 옵션 생성·적용 시
복제합니다. 문서화된 object type은 CLUSTER와 NODE이고 deployment가 추가 type의 지원을
검증합니다. GlobalProject nil은 false인 기본값을 생략하고 `WithListGlobalProject(false)`는
명시적으로 false를 보냅니다. 추가 query는 `WithListQuery`로 보낼 수 있고 concrete 필드를
덮어쓸 수 없습니다.

Event ID와 관련 ObjectID를 구별합니다. Level은 JSON 문자열·숫자를 모두 받아 정확한
문자열 값으로 제공하고, `Body["level"]`에는 원래 JSON 타입을 보존합니다. 큰 정수를
float64로 바꾸지 않습니다. Python의 무타입 Body는 원래 값의 타입을 유지한다는 차이가
있습니다. GeneratedAt은 원래 timestamp 문자열입니다. 임의 JSON인 MetaData와 추가 응답 필드는 숫자 정밀도를 유지합니다.
`Body`는 생략/null을 구별하고 `Header`, `StatusCode`는 HTTP 근거를 담습니다.

명시한 limit이 있으면 짧은 페이지를 포함해 비어 있지 않은 페이지 뒤에 마지막 응답 Event ID로 marker를 만들고 빈
페이지를 허용합니다. body next/links와 HTTP Link는 동일 origin/collection 경로에
제한하고 기존 필터를 유지합니다. lazy List의 break와 context 취소는 추가 요청을
중단합니다. 기본 limit/marker는 생략합니다.

이 패키지는 `get_event`와 `events` 조회만 구현합니다. Event Status는 과거에 기록한
관련 object 상태이며 작업 완료 waiter로 간주하지 않습니다. Name Find, 변경/삭제와
기본 status Wait는 지원하지 않습니다. Python mutable Resource나 Senlin 전체 parity를
주장하지 않습니다.

근거: pinned openstacksdk revision `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`와
[공식 Events API](https://docs.openstack.org/api-ref/clustering/#events-events). Get/List 성공은 200입니다.

## 목록 소비 제어

| pinned Python | Go 옵션 | 소비 정책 |
|---|---|---|
| `max_items=n` | `WithListMaxItems(n)` 또는 `ListOpts.MaxItems` | 로컬 필터 이전에 검증한 raw 행을 최대 n개 소비; 0은 무제한, 음수는 사전 오류 |
| `paginated=False` | `WithListPaginated(false)` 또는 `ListOpts.Paginated` | 첫 응답만 소비하고 continuation을 처리하지 않음 |
| 기본 `paginated=True` | nil 또는 `WithListPaginated(true)` | 페이지 순회를 허용; 뒤의 옵션이 앞의 값을 덮어씀 |
| `limit=n` | `WithListOptions(ListOpts{Limit: n})` | 양수는 wire page limit; 로컬 cap과 독립 |

```go
listingAPI := events.New(client)
values, err := listingAPI.All(ctx,
    events.WithListOptions(events.ListOpts{Limit: 20}),
    events.WithListMaxItems(50))
if err != nil { return err }
fmt.Println(len(values))
for value, err := range listingAPI.List(ctx, events.WithListPaginated(false)) {
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

List 전체 계약은 partial입니다. Python의 자동 query/Body 분류와 unknown query 생략,
per-call base_path, deprecated JMESPath 및 controller의
default/capped limit 비교는 남아 있습니다. `WithListQuery`는 vendor query를 실제로 전달하는
Go 확장이며 Python unknown query 생략과 구별합니다.

공통 소비 정책과 남은 차이는 [Senlin 목록 제어](../listing/README.md), 실제 HTTP 근거는 [목록 제어 테스트](../../../api/clustering_typed_list_controls_test.go)를 참고합니다.

## raw Body 로컬 필터

Python `events(status="ERROR", meta_data={"team": "infra"})`의 Body 필터는 Go에서
`events.WithListFilter`로 명시합니다. `oid/oname/otype/action/cluster_id/level` 같은 서버 query
필드도 이 옵션으로 지정하면 로컬에서만 비교하며, 기존 `ListOpts`의 query 입력과 독립적입니다.

canonical 필드는 `id/name/timestamp/oid/oname/otype/cluster_id/level/user/project/action/status/status_reason/meta_data`입니다.
Python alias `generated_at/obj_id/obj_name/obj_type/user_id/project_id`는 각각
`timestamp/oid/oname/otype/user/project`를 비교합니다. `WithListFilter("level", 20)`은 raw JSON
숫자 20만 비교하고 문자열 `"20"`은 다른 값입니다. 모델의 문자열 `Level`로 변환한 결과를
필터 값으로 사용하지 않습니다. known Body 키의 확장 query는 거부하고 unknown vendor query는
기존 명시적 opt-in 정책을 유지합니다.

```go
package examples

import (
	"context"
	"encoding/json"
	"fmt"

	sdk "gophercloudsdk"
	"gophercloudsdk/clustering/v1/events"
)

func FilterEvents(ctx context.Context, conn *sdk.Connection) error {
	service, err := conn.ClusteringV1(ctx)
	if err != nil {
		return err
	}
	for event, err := range service.Events.List(ctx,
		events.WithListFilter("status", "ERROR"),
		events.WithListFilter("meta_data", json.RawMessage(`{"count":9007199254740993}`)),
		events.WithListMaxItems(50), events.WithListPaginated(false)) {
		if err != nil {
			return err
		}
		fmt.Println(event.ID, event.Level, event.Status)
	}
	return nil
}
```

snapshot 입력은 재사용마다 독립적으로 적용합니다. cap·마지막 raw Event ID는 필터보다 먼저
계산하므로 모든 결과가 걸러져도 다음 marker는 원래 마지막 행을 기준으로 합니다. 객체는
recursive subset, 배열은 순서와 전체 값, 숫자는 정확한 decimal 값으로 비교합니다. bool은
숫자와 다르고 생략은 scalar null과 같습니다. 빈 실제 객체는 객체 필터에 매칭하지 않습니다.
[공통 Body 필터 정책](../listing/README.md)과 [HTTP 계약](../../../api/clustering_body_filters_test.go)을 참고합니다.

## 목록 호출별 헤더와 버전

`WithListHeader(key, value)`와 `WithListMicroversion("1.7")`은 이 패키지의 typed `List` /
`All`에만 적용합니다. 기본값은 source client 설정이며 명시 옵션은 공유 클라이언트를
수정하지 않습니다. 실제 wire 헤더·버전 선택, 재순회·페이지·인증 정책과 Python 비교 예제는
[Senlin 목록 호출 옵션](../listing/README.md#목록-호출별-헤더와-버전)을 참고합니다.
`headers`, `microversion`, `base_path`를 `WithListQuery`로 전달하면 HTTP 전에 오류입니다.

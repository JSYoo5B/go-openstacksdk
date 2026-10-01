# Heat stack events

`StackEvents.InStack`는 기존 stack SDK로 이름 또는 ID를 한 번 해석하고 canonical `StackIdentity{Name, ID}`를 고정합니다. 이름은 전체 stack 목록에서 정확히 비교하며 중복이면 `resource.ErrAmbiguous`입니다. ID는 Heat의 stack 조회와 redirect를 사용합니다. 이미 identity를 알고 있으면 `ForStack`으로 조회 요청 없이 범위를 만듭니다.

```go
package main

import (
    "context"
    "fmt"

    sdk "gophercloudsdk"
    "gophercloudsdk/orchestration/v1/stackevents"
    "gophercloudsdk/resource"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx)
    if err != nil { panic(err) }
    service, err := conn.OrchestrationV1(ctx)
    if err != nil { panic(err) }
    events, err := service.StackEvents.InStack(ctx, resource.Name("app"))
    if err != nil { panic(err) }

    for event, err := range events.List(ctx,
        stackevents.WithListOptions(stackevents.ListOpts{Limit: 100}),
        stackevents.WithListQuery("nested_depth", "1")) {
        if err != nil { panic(err) }
        fmt.Println(event.ID, event.ResourceName, event.ResourceStatus)
    }

    resourceEvents, err := events.ForResource("server")
    if err != nil { panic(err) }
    values, err := resourceEvents.All(ctx,
        stackevents.WithListResourceEventsOptions(
            stackevents.ListResourceEventsOpts{Limit: 100}))
    if err != nil { panic(err) }
    if len(values) > 0 {
        detail, err := resourceEvents.Get(ctx, values[0].ID)
        if err != nil { panic(err) }
        fmt.Println(detail.ID, detail.ResourceProperties, detail.Header)
    }
}
```

| openstacksdk | Go |
|---|---|
| `conn.orchestration.stack_events(stack, **attrs)` | `events.List(ctx, stackevents.WithListOptions(...), ...)` |
| `stack_events(stack, resource_name="server", **attrs)` | `events.ForResource("server")` → `resourceEvents.List(ctx, ...)` |
| Stack Resource의 `name`, `id` | `StackIdentity`로 `service.StackEvents.ForStack(identity)` |
| generator를 `list(...)`로 수집 | 해당 scope의 `All(ctx, ...)` |

Stack 목록은 `/stacks/{name}/{id}/events`, resource 목록은 `/stacks/{name}/{id}/resources/{resourceName}/events`를 조회합니다. [이벤트 단건 GET](https://docs.openstack.org/api-ref/orchestration/v1/#show-event-details)은 resource 이름까지 필요하므로 `resourceEvents.Get(ctx, eventID)` 또는 `events.Get(ctx, resourceName, eventID)`로 호출합니다. Resource의 물리 ID를 논리 resource 이름으로 추측하지 않습니다. [Pinned Python Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/orchestration/v1/_proxy.py)는 두 목록 경로를 `stack_events`로 제공하며 별도 이벤트 조회 Proxy 메서드는 없습니다. Python `StackEvent`의 generic fetch 선언이 위 단건 경로를 대신 검증한다는 뜻은 아닙니다.

목록의 typed options는 기존 native `ListOpts`와 `ListResourceEventsOpts`를 사용합니다. `ResourceActions`, `ResourceStatuses`, `ResourceNames`, `ResourceTypes`는 반복 query 값을 보존합니다. `Limit`는 페이지 크기이고 전체 결과 수 제한은 아닙니다. `SortKey`는 Heat의 `sort_keys`로 전송합니다. Pinned Python의 `sort_key` keyword를 자동 변환하지 않습니다. Native struct에 없는 `nested_depth` 등 추가 query는 `WithListQuery` 또는 `WithListResourceEventsQuery`로 전달하며 서버가 의미를 검사합니다.

페이지 순회는 [Heat의 `limit`·`marker` 계약](https://docs.openstack.org/api-ref/orchestration/v1/#list-stack-events)과 pinned Gophercloud를 따라 마지막 event ID를 다음 marker로 사용하고 빈 페이지에서 끝납니다. Event 안의 `links`는 event/resource/stack 링크이므로 다음 페이지로 사용하지 않습니다. 반복된 marker URL은 재조회 전에 `resource.ErrPaginationCycle`로 실패합니다. 소비자가 iterator를 중단하면 다음 항목 해석과 다음 페이지 요청도 멈춥니다. Context 취소는 페이지 요청과 항목 사이에 확인하며, `All`의 중간 실패는 완료된 목록으로 반환하지 않습니다. 빈 `All` 결과는 비nil slice입니다.

`EventResource`는 native `Event`의 timestamp·status·reason·logical/physical resource ID·properties·links를 유지합니다. Native 모델에 없는 `resource_type`은 optional `ResourceType`으로, 추가 응답 필드와 omission/null은 `Body`로 보존합니다. `Header`는 각 결과의 독립된 응답 header 복사본입니다. `Detailed`는 단건 GET에서만 true이며 목록에 상세 properties가 항상 들어 있다고 가정하지 않습니다.

`RequestStack`, `RequestResourceName`, `RequestEventID`는 요청에 사용한 identity입니다. Stack 전체 목록에서는 resource 이름이 비어 있고 목록에서는 event ID가 비어 있습니다. 응답의 `ID`와 `ResourceName`은 실제 wire 값으로 별도 유지하며 scope나 다음 요청 경로를 변경하지 않습니다. 알려진 identity를 반환하는 `Identity()`도 복사본입니다.

`Get`의 404는 `resource.ErrNotFound`와 원본 HTTP 오류를 함께 보존합니다. 목록 및 다른 HTTP 실패는 원본 status·body·header·URL을 보존합니다. 부모 stack 또는 resource의 미존재도 서버가 404로 표현할 수 있으며 scope는 이를 성공으로 무시하지 않습니다. 추가 header는 `WithGetHeader`, `WithListHeader`, `WithListResourceEventsHeader`를 사용합니다. 기본 Accept·인증·microversion header 덮어쓰기, 지원하지 않는 body/query/argument 옵션은 요청 전에 거부합니다.

이벤트는 읽기 전용 기록입니다. 이벤트 이름 Find, Create/Update/Delete, 상태 대기와 응답 변경 추적·자동 commit은 제공하지 않습니다. 기존 명시적인 `StackEvents.Find/Get/List/ListResourceEvents` API 호출은 유지됩니다. Scope는 공유 클라이언트의 endpoint·header 설정을 변경하지 않습니다. Stack lookup 정책은 [stack SDK 설명](../stacks/README.md), native 계약은 [Gophercloud v2.15.0](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/orchestration/v1/stackevents/requests.go) 및 [Python StackEvent](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/orchestration/v1/stack_event.py)를 참고합니다.

# Messaging v2 subscriptions

큐 이름을 한 번 고정한 SDK 소유 scope에서 subscription을 생성·조회·목록·삭제합니다. `InQueue`는 큐 GET을 하지 않으며, builder interface를 구현할 필요가 없습니다.

| openstacksdk | Go | HTTP |
|---|---|---|
| `create_subscription(queue_name, **attrs)` | `scope.Create(ctx, opts, options...)` | flat POST, 201 |
| `subscriptions(queue_name, **query)` | `scope.List(ctx, options...)` / `All` | GET, `subscriptions` 배열, 200 |
| `get_subscription(queue_name, subscription)` | `scope.Get(ctx, id, options...)` | flat GET, 200 |
| `delete_subscription(queue_name, value, ignore_missing=True)` | `scope.Delete(ctx, id, options...)` | DELETE, 204 |

```go
package subscriptionexample

import (
    "context"
    "fmt"

    "github.com/gophercloud/gophercloud/v2"
    "gophercloudsdk/messaging/v2/subscriptions"
)

func Subscribe(ctx context.Context, client *gophercloud.ServiceClient) error {
    queue, err := subscriptions.New(client).InQueue(ctx, "events")
    if err != nil { return err }
    created, err := queue.Create(ctx, subscriptions.CreateOpts{
        Subscriber: "mailto:example@example.test",
    }, subscriptions.WithCreateDeliveryOptions(map[string]any{
        "from": "Zaqar", "subject": "new event",
    }))
    if err != nil { return err }
    current, err := queue.Get(ctx, created.SubscriptionID)
    if err != nil { return err }
    fmt.Println(current.Subscriber, string(current.TTL))
    for item, err := range queue.List(ctx,
        subscriptions.WithListOptions(subscriptions.ListOpts{Limit: 10}),
        subscriptions.WithListMaxItems(5)) {
        if err != nil { return err }
        fmt.Println(item.ID)
    }
    return queue.Delete(ctx, created.SubscriptionID,
        subscriptions.WithDeleteIgnoreMissing(false))
}
```

`Subscriber`는 필수 문자열입니다. TTL은 `request.Optional[int64]`이고 생략하면 서버 기본값을 사용합니다. SDK가 3600을 채워 넣지 않습니다. 명시적 null은 거부하고 값의 허용 범위와 destination 정책은 서버에 맡깁니다. Python TTL은 untyped이므로 Go의 integer 입력 제한은 명시적인 차이입니다. `Options`는 생략하거나 JSON 객체로 지정합니다. `WithCreateDeliveryOptions`와 `WithCreateOptions`는 값을 snapshot하며, null/배열 options는 요청 전에 거부합니다. `WithCreateField`는 추가 flat body 필드를 전송하지만 typed 필드·응답 ID·age/source·scope/인증 입력을 덮어쓸 수 없습니다.

Create는 원래 응답만 반환합니다. 요청 subscriber/ttl/options로 cached 응답을 만들거나 후속 GET을 하지 않습니다. 공식 응답의 `subscription_id`는 optional입니다. 생략/null/빈 문자열 또는 ID 없이 반환된 201 객체도 실제 모델과 metadata를 반환합니다. ID의 잘못된 JSON 타입은 decode 오류로 반환하며 `Location`에서 ID를 유추하지 않습니다. 후속 Get/Delete에는 호출자가 유효한 문자열 ID를 명시해야 합니다. Location은 collection URL일 수 있고 외부 URL이어도 수동으로 보관하며 따라가지 않습니다.

응답 모델은 정확한 canonical wire 키만 디코딩합니다. `ID`는 canonical `id`가 존재하면 null/빈 문자열까지 우선하고, id가 없을 때만 alternate `subscription_id`를 사용합니다. case variant 확장은 typed 필드를 바꾸지 않고 `Body`에 남습니다. TTL·age·options는 `json.RawMessage`라 원문 숫자 정밀도, null, 생략, 중첩 metadata를 보존합니다. 응답 options에는 Python dict descriptor의 강제 변환을 적용하지 않습니다. `Body`, `Header`, `StatusCode`는 실제 HTTP 응답 증거입니다. Queue scope와 요청 ID는 응답 source/ID로 바뀌지 않습니다. Python Resource 입력 overload 대신 명시적인 문자열 ID를 사용하고, path segment의 slash/percent/dot/control/공백은 사전 거부합니다. 나머지 큐 이름의 서버 정책을 추측하지 않습니다.

목록은 pinned `MessageResource.list`의 전략을 사용합니다. 처음 limit이 없어도 비어 있지 않은 페이지 뒤에 `limit=행 수`, `marker=마지막 id`로 다음 요청을 만듭니다. 현재 limit보다 짧은 페이지 또는 빈 페이지에서 종료하며 advertised next/links/HTTP Link는 읽지 않습니다. 최초 marker도 query에 전달합니다. 반복 marker와 다음 marker의 누락/null은 Go 오류로 종료하고 `All`은 부분 결과를 반환하지 않습니다.

`MaxItems`는 raw 행 cap이며 기본 0은 무제한입니다. 이 cap은 wire limit hint를 만들지 않습니다. `Paginated=false`는 첫 페이지만 읽습니다. cap과 consumer break는 다음 행의 typed decode 및 후속 HTTP 전에 종료하며, 현재 페이지의 JSON 문법 오류는 cap과 관계없이 보입니다. Python의 이 custom list는 max_items 인자를 실제로 적용하지 않으므로 cap은 명시적인 Go 확장입니다. `WithListQuery`는 raw wire 확장으로 unknown query도 전달합니다. Python custom MessageResource는 unknown query를 discard하며 generic Resource.list의 Body 자동 필터링을 적용하지 않습니다. Go도 Body 자동 필터를 추가하지 않고 unknown query를 wire 확장으로 전달합니다. SDK control과 queue/client/project 이름을 query로 전달하는 우회는 거부합니다. 목록 option slice, pointer, query, header는 각 순회에서 소유하므로 iterator를 독립적으로 재사용할 수 있습니다.

기존 ServiceClient/Provider와 원래 endpoint·ResourceBase·transport·최신 token을 사용합니다. Connection의 안정적인 Client-ID를 기본으로 보존하고 직접 생성한 client에는 Client-ID 설정 또는 작업별 `With...ClientID`가 필요합니다. UUID를 자동으로 새로 만들지 않습니다. 비인증 환경에서는 `WithCreateProjectID`/`WithGetProjectID`/`WithListProjectID`/`WithDeleteProjectID`로 프로젝트를 지정할 수 있습니다. typed client/project 값은 해당 요청의 source header보다 우선하며 공유 client를 바꾸지 않습니다. 빈 typed 값은 기존 값을 유지합니다. 프로젝트를 auth-result에서 추정하지 않으며, 이것은 Python의 session project 자동 선택과 다릅니다. generic header 확장으로 인증·버전·Client-ID·X-PROJECT-ID를 덮어쓰는 요청은 거부합니다.

Delete는 실제 HTTP 404만 기본적으로 무시합니다. `WithDeleteIgnoreMissing(false)`는 이를 오류로 반환합니다. transport의 wrapped404, context 취소, accepted 응답 read 오류는 그대로 오류이며 자동 재전송하지 않습니다. unexpected 성공 코드와 accepted JSON/모델/read 오류도 원래 원인과 응답 증거를 보존합니다. 이 네 연산에 Find·Wait·Update·confirm이나 전체 Python Resource descriptor/cache/dirty 수명주기를 추가한 것으로 주장하지 않습니다.

근거는 [공식 subscription API](https://docs.openstack.org/api-ref/message/#subscriptions-subscriptions), 고정 openstacksdk의 `openstack/message/v2/_proxy.py:191–278`, `subscription.py:17–54`, `_base.py:35–91`과 [HTTP 회귀 테스트](../../../api/messaging_subscriptions_test.go)입니다. 검증은 격리된 HTTP fixture이며 live cloud 테스트가 아닙니다.

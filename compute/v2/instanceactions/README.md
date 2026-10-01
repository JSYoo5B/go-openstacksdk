# 서버의 instance action 이력

Instance action은 서버에 수행한 `reboot`, `stop` 등의 요청 이력입니다. 식별자는 `request_id`이며 같은 action 이름을 가진 요청이 여러 개 있을 수 있습니다. `InServer(ctx, serverRef)`는 서버를 한 번 해석한 뒤 이 범위에서 조회합니다.

| openstacksdk | Go |
|---|---|
| `conn.compute.server_actions(server)` | `scope.List(ctx)` 또는 `scope.All(ctx)` |
| `conn.compute.get_server_action(request_id, server)` | `scope.Find(ctx, resource.ID(requestID))` |
| `ignore_missing=True` | `scope.Find(ctx, resource.ID(requestID), resource.WithIgnoreMissing())` |

```go
// context.Context ctx, *gophercloudsdk.Connection conn을 사용하는 함수 안에서
service, err := conn.ComputeV2(ctx)
if err != nil { return err }
scope, err := service.InstanceActions.InServer(ctx, resource.Name("worker"))
if err != nil { return err }
for action, err := range scope.List(ctx, resource.WithPageSize(100)) {
    if err != nil { return err }
    fmt.Println(scope.ServerID(), action.RequestID, action.Action, action.UpdatedAt)
}
action, err := scope.Find(ctx, resource.ID("req-example"), resource.WithIgnoreMissing())
if err != nil { return err }
if action != nil && action.Events != nil {
    for _, event := range *action.Events {
        fmt.Println(event.Event, event.Result, event.Details)
    }
}
```

예제의 `resource`는 `gophercloudsdk/resource`, `fmt`는 표준 라이브러리입니다. `WithPageSize`에는 Compute microversion 2.58 이상을 선택해야 합니다. `sdk.WithMicroversion(sdk.Compute, "2.84")` 또는 필요한 범위의 `sdk.WithMicroversionRange`를 Connection에 설정하면 예제의 pagination과 최신 event details를 요청할 수 있습니다. 기본 조회 전체에 2.84가 필요한 것은 아닙니다.

서버 `resource.ID`는 사전 조회 없이 사용하고 `resource.Name`은 전체 페이지에서 정확히 찾습니다. 중복 이름·미존재·권한 오류를 숨기지 않습니다. 후속 작업에서 서버를 다시 찾지 않으며 응답의 `instance_uuid`가 달라도 부모 URL을 바꾸지 않습니다. 삭제한 서버의 이력은 이름으로 다시 찾기 어려우므로 보관한 서버 ID로 접근합니다.

Action `resource.ID`는 request ID를 뜻합니다. `ResolveID(ctx, resource.ID(id))`는 요청하지 않고 그대로 해석하며 `Find/Get`은 그 ID의 상세 endpoint를 조회합니다. Go Find의 기본값은 미존재 오류이며 `WithIgnoreMissing()`은 404만 `nil, nil`로 바꿉니다. HTTP 오류의 본문·헤더·status와 decode 오류·context 취소 원인은 보존합니다.

`resource.Name`, `WithName`, `WithStatus`, Delete와 Wait는 `resource.ErrUnsupported`입니다. action 이름이나 확장 `status` 값을 이름·상태 대기 capability로 만들지 않습니다. Inventory의 `find:false`는 이름 조회 capability가 없다는 뜻이며 ID 기반 `Find/ResolveID`는 사용할 수 있습니다.

## 목록과 상세 모델

공통 scope는 SDK의 `ActionResource`를 반환합니다.

- embedded `InstanceAction`은 기본 action 이름, request ID, instance UUID, project/user, message, start time을 보존합니다.
- `ServerID`는 해석한 부모이며 `InstanceUUID`는 응답 그대로입니다.
- `UpdatedAt`은 목록과 상세 모두에 제공되면 해석합니다.
- `Details`는 Get/ID Find에서 받은 native `InstanceActionDetail`입니다. 목록에서는 nil입니다. 목록만 읽었다고 상세를 자동 요청하지 않습니다.
- `Events`는 SDK `ActionEvent` 목록입니다. native event의 host/host ID·traceback·시간과 2.84 `details`를 함께 제공합니다. `EventFields` alias는 원래 native event 값에 접근할 때 사용합니다.
- `Body`는 action 객체의 전체 JSON 필드, 각 event의 `Body`는 event 객체의 전체 필드를 보존합니다. 추가 `status` 같은 필드도 포함합니다. 키 존재 여부로 생략과 JSON null을 구분할 수 있습니다.
- `Header`는 해당 조회 또는 목록 페이지의 전체 응답 헤더를 복사한 값입니다.

`Details != nil`은 상세 조회를 했다는 뜻입니다. events는 버전·권한에 따라 빠질 수 있으므로 Get에서도 `Events == nil`을 이벤트가 전혀 없다는 뜻으로 해석하지 않습니다. 서버가 `events:[]`를 반환하면 길이 0인 non-nil slice pointer를 보존합니다.

Action과 event 항목은 JSON object여야 합니다. null·배열·scalar 항목, 잘못된 events 타입과 잘못된 날짜는 decode 오류로 반환합니다. 목록 `instanceActions`는 배열이어야 하며 `[]`는 정상적인 빈 목록입니다.

Gophercloud v2.15.0의 native 목록은 `SinglePageBase`를 사용합니다. Scope의 목록은 [공식 API의 `links` 배열](https://docs.openstack.org/api-ref/compute/#list-actions-for-server)을 별도 page로 읽고 공통 `resource.Stream`에 연결해 모든 페이지, 원래 next URL, `break`, 취소와 반복 링크 오류를 처리합니다. 기존 `API.Get/List`는 native 모델을 반환하는 하위 호출이며 이 SDK scope의 모델·페이지 정책과 구분합니다.

## Microversion과 권한

정확한 조건은 [Nova API version history](https://docs.openstack.org/nova/latest/reference/api-microversion-history.html)와 [action detail API](https://docs.openstack.org/api-ref/compute/#show-server-action-details)에 근거합니다.

| 버전 | 추가 동작과 필드 |
|---|---|
| 2.21 | 삭제된 서버의 action 이력 조회 |
| 2.51 | events를 일반 사용자에게도 반환. 그 이전에는 정책상 기본 admin 전용. traceback은 별도 정책을 적용 |
| 2.58 | 목록 `limit/marker`, `changes-since`, 응답 `updated_at`과 pagination links |
| 2.62 | event `host`와 `hostId`. host는 정책상 기본 admin 전용 |
| 2.66 | 목록 `changes-before` |
| 2.84 | event `details` fault 메시지. 값은 null일 수 있음 |

Gophercloud의 Events 주석에 있는 2.50을 모든 event 조회의 최소 버전으로 사용하지 않습니다. 공식 API는 2.50까지의 admin events와 2.51부터의 일반 사용자 노출을 구분합니다. Scope는 선택한 microversion을 올리거나 필드·권한 지원을 추정하지 않습니다. 서버가 해당 query와 권한을 검증합니다. 서비스별 정책 설정에 따라 기본 권한은 바뀔 수 있습니다.

Python의 대응 모델은 [pinned ServerAction](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server_action.py)에 있습니다. Go는 Resource의 URI 부모를 명시적 scope로 고정하고 typed 값과 보존된 JSON을 반환합니다. 변경 추적·자동 commit은 제공하지 않습니다.

구현은 [scope와 모델](resources.go), 검증은 [HTTP 계약 테스트](../../../api/instance_actions_scope_test.go)에 있습니다.

## 읽을 행 수와 첫 페이지

Python `conn.compute.server_actions(server, max_items=20, paginated=False)`의 목록 제어는 Go에서 `scope.List/All`에 `resource.WithMaxItems(20)`, `resource.WithPaginated(false)`로 지정합니다. scope는 controlled stream을 사용하며 기존 `resource.Stream`은 제어값이 없는 같은 경로로 위임합니다. 공개 native `service.InstanceActions.List`의 signature와 서비스별 typed options는 유지됩니다.

```go
package examples

import (
	"context"
	"fmt"

	sdk "gophercloudsdk"
	"gophercloudsdk/resource"
)

func ListServerActions(ctx context.Context, conn *sdk.Connection, serverID string) error {
	service, err := conn.ComputeV2(ctx)
	if err != nil {
		return err
	}
	scope, err := service.InstanceActions.InServer(ctx, resource.ID(serverID))
	if err != nil {
		return err
	}
	for action, err := range scope.List(ctx,
		resource.WithMaxItems(20), resource.WithPaginated(false)) {
		if err != nil {
			return err
		}
		fmt.Println(action.ServerID, action.RequestID, action.Action)
	}
	return nil
}
```

이 예제는 wire `limit`을 추가하지 않습니다. `WithPageSize`를 별도로 사용할 때의 2.58 이상 조건은 그대로입니다. 서버가 실제로 제공한 첫 페이지의 action을 최대 20개 읽으며 상세 GET을 하지 않습니다. event·추가 JSON·응답 헤더의 보존과 고정된 ServerID도 기존 목록과 같습니다.

`WithMaxItems(0)`은 무제한이고 음수는 iterator 순회 시 HTTP 전에 실패합니다. 같은 제어는 마지막 옵션이 적용됩니다. `break`와 context 취소는 추가 요청을 중단합니다. 현재 페이지의 action·event 전체 decode 오류는 cap 이후 행에 있어도 반환됩니다. 다음 페이지가 필요하면 기존 `links` continuation과 반복 링크 검사를 사용하며, native 경로에 REST의 origin·path guard를 추가하지 않았습니다. cap이나 첫 페이지 옵션으로 멈추면 다음 링크를 처리하지 않습니다.

공통 정책은 [목록 가이드](../../../docs/listing.md), 실제 scope 연결과 raw 모델 보존은 [native 목록 HTTP 계약](../../../api/native_scope_list_controls_test.go)에서 확인합니다.

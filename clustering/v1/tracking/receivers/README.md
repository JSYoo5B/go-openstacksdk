# Receiver 변경 추적

`Receivers.Load/Track`은 SDK 소유 `TrackedReceiver`를 반환합니다. 기존 `UpdateOpts`와 `WithUpdate...`를 `Edit`에 전달하면 SDK가 현재 값과 비교하고 dirty 필드만 `Commit`에서 전송합니다. 앱에서 builder나 lifecycle interface를 구현하지 않습니다. Receiver는 동기 리소스이며 PATCH 200의 `receiver`를 반환합니다. `Action`은 `CLUSTER_SCALE_OUT` 같은 명령 이름이며 비동기 Action ID나 `Operation`을 만들지 않습니다.

| 작업 | openstacksdk | Go |
|---|---|---|
| 조회 후 편집 | `receiver = conn.clustering.get_receiver(id)` | `tracked, err := service.Receivers.Load(ctx, resource.ID(id))` |
| cached 객체 편집 | `conn.clustering.update_receiver(receiver, **attrs)` | `Receivers.Track(receiver)` → `Edit` → `Commit` |
| 속성 대입 | `receiver.action = "CLUSTER_SCALE_IN"` | `WithUpdateAction("CLUSTER_SCALE_IN")` |
| 명시 null | `receiver.action = None` | `WithUpdateActionNull()` |
| 필드 삭제 | `del receiver.params` | `RemoveParams()`; 다음 Commit에 `params:null` |
| 변경 없음 | cached Resource commit no-op | clean Commit은 HTTP 없이 cache snapshot 반환 |
| 부분 응답 | 같은 Resource에 field 병합 | `Value()`는 cache, `Response()`는 실제 응답 |
| 후속 조회 | `resource.fetch(session)` | `Refresh(ctx)`; 고정 경로 GET 200, 이전 dirty revision reset |
| 경로 지정 | `update_receiver(receiver, base_path=path)` | `Receivers.AtBasePath(path)`의 Update·Load·Track |

```python
receiver = conn.clustering.get_receiver("RECEIVER_ID")
receiver.name = "scale_in"
receiver.action = "CLUSTER_SCALE_IN"
receiver.params = {"team": "platform"}
receiver = conn.clustering.update_receiver(receiver)
```

```go
package example

import (
    "context"

    sdk "github.com/JSYoo5B/gophercloudsdk"
    "github.com/JSYoo5B/gophercloudsdk/clustering/v1/receivers"
    "github.com/JSYoo5B/gophercloudsdk/resource"
)

func RenameTrackedReceiver(ctx context.Context, conn *sdk.Connection) (*receivers.Receiver, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    tracked, err := service.Receivers.Load(ctx, resource.Name("scale_out"))
    if err != nil { return nil, err }
    if err := tracked.Edit(receivers.UpdateOpts{},
        receivers.WithUpdateName("scale_in"),
        receivers.WithUpdateAction("CLUSTER_SCALE_IN"),
        receivers.WithUpdateParams(map[string]any{"team": "platform"})); err != nil {
        return nil, err
    }
    return tracked.Commit(ctx, receivers.WithUpdateHeader("X-Audit-Tag", "rename"))
}
```

```go
package example

import (
    "context"

    sdk "github.com/JSYoo5B/gophercloudsdk"
    "github.com/JSYoo5B/gophercloudsdk/clustering/v1/receivers"
    "github.com/JSYoo5B/gophercloudsdk/resource"
)

func UpdateReceiverAt(ctx context.Context, conn *sdk.Connection, collectionPath, receiverID string) (*receivers.Receiver, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    scope, err := service.Receivers.AtBasePath(collectionPath)
    if err != nil { return nil, err }
    tracked, err := scope.Load(ctx, resource.ID(receiverID))
    if err != nil { return nil, err }
    if err := tracked.Edit(receivers.UpdateOpts{}, receivers.WithUpdateActionNull()); err != nil {
        return nil, err
    }
    return tracked.Commit(ctx)
}
```

## 편집·응답·실패

`Edit`는 로컬 작업으로 Name·Action의 Optional string, Params의 object/null과 extension 보호를 검증합니다. 빈 Edit은 허용하고 잘못된 옵션은 일부 적용하지 않습니다. 같은 현재 값은 clean을 유지하고 `A → B → A`는 sticky dirty로 남습니다. 생략, 명시 empty string, null, 빈 object는 구분합니다. `RemoveName/Action/Params()`는 cached field를 제거하고 null tombstone을 남깁니다. absent 필드 삭제는 no-op이며 `WithUpdate...Null` 또는 `WithUpdateParams(nil)`은 absent 필드에도 null을 대입합니다. nullable 값과 명령별 Params의 수용 여부는 서버가 결정합니다.

body 변경은 Edit에, header는 Commit에 전달합니다. Commit은 body/query/argument 옵션과 보호된 auth/version/transport header를 거부하며 clean 상태에도 context·source·microversion 형식·header를 검증합니다. 유효한 clean Commit은 HTTP를 보내지 않습니다. cached `user`나 `channel`이 있다는 이유로 목록의 `user` query 1.4 gate를 적용하지 않습니다. Update에 별도 field microversion gate를 추측하지 않습니다.

성공 응답은 whole field 단위로 cache에 병합합니다. Params와 vendor nested object는 전체 교체하고, 전송 필드가 응답에서 생략되어도 그 revision은 clean이 되며 로컬 값은 cache에 남습니다. `Value().Body`와 `Response().Body`를 구별하고 모델·Body·Header·Params/Actor/Channel을 입력과 getter마다 복사합니다. Body 없는 수동 모델은 로컬 seed로 감싸고 HTTP 증거를 만들지 않습니다. Location과 Channel URL은 raw 응답에만 보존하며 따라가지 않습니다.

Track과 Name Load는 정확한 lowercase raw `id`를 route로 사용합니다. Name Load는 raw `name`을 모든 선택한 페이지에서 정확히 비교하고 한 번 해석하며 미존재·중복은 오류입니다. explicit ID Load는 응답 ID가 다르거나 null·생략이어도 요청 ID를 이후 Commit/Refresh에 고정합니다. 입력 모델이나 getter의 ID 수정은 route를 바꾸지 않습니다. core·readonly 필드와 대소문자 별칭은 extension 옵션으로 우회하지 않습니다.

HTTP 실패, 잘못된 envelope·JSON·identity의 200 응답에서는 dirty·cache·Response를 바꾸지 않습니다. 응답 검증 오류는 전체 body/header/status를 가진 `resource.ResponseError`이며 SDK가 자동으로 mutation을 재전송하지 않습니다. Refresh는 고정 경로 GET 200을 병합하고 GET 전에 있던 dirty revision을 reset합니다. 실패하면 기존 상태를 보존합니다. Commit/Refresh는 handle별로 직렬화하며 HTTP 중 새 Edit은 revision으로 구분해 보존합니다. Receiver에는 status waiter를 새로 만들지 않습니다.

## 경로 scope와 Python/Go 차이

`Receivers.AtBasePath(path)`는 Update·Load·Track 전용 concrete scope이며 생성 시 HTTP를 하지 않습니다. `AtBasePath("receivers")`는 기본 경로와 같습니다. 경로는 service resource base 아래 unescaped literal UTF-8 segment이며 SDK가 한 번 escape합니다. 빈/선행/후행/중복 slash, dot segment, URL scheme, backslash, query, fragment, percent escape/format template, whitespace/control은 사전 오류입니다. client·Provider·Endpoint·ResourceBase·MoreHeaders를 수정하지 않으며 최신 인증과 선택 microversion을 공유합니다. Create/Delete/List의 경로는 바뀌지 않습니다.

Python [update_receiver](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/_proxy.py#L1114)는 [Proxy._update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L716)의 base_path를 해당 commit에 전달합니다. Go는 URI template 대신 literal scope를 소유하고 handle의 Load·Commit·Refresh까지 collection과 ID를 고정합니다. [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881)의 mutable pointer·permissive descriptor/response·cached ID 수정은 concrete 편집, 독립 snapshot, 명시 error, readonly 보호와 고정 route로 매핑합니다. `prepend_key/has_body/retry_on_conflict`는 이 direct proxy가 전달하는 named commit 제어가 아닙니다.

구현: [TrackedReceiver](../../receivers/lifecycle.go), [UpdateScope](../../receivers/scope.go), [공통 상태](../../../../internal/senlin/tracked.go), [경로 검증](../../../../internal/senlin/collection_path.go). 검증: [Receiver HTTP 계약](../../../../api/clustering_receivers_lifecycle_test.go), [기존 dirty 상태 테스트](../../../../internal/senlin/tracked_test.go). 기본 CRUD와 목록은 [Receiver 사용법](../../receivers/README.md)을 참고합니다.

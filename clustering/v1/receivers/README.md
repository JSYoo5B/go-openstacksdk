# Senlin receivers

`receivers.New(client)`가 webhook/message receiver의 생성·조회·수정·삭제·목록·이름 검색을
제공합니다. Receiver는 동기 Resource이며 Location이나 Channel의 URL을 따라가지 않습니다.
Action은 `CLUSTER_SCALE_OUT` 같은 명령 이름이고 비동기 action UUID가 아닙니다.
Connection을 사용하면 `conn.Clustering(ctx)`가 반환한 서비스의 `Receivers`에서 같은 API를
사용하며, 서비스의 인증·선택 microversion·endpoint prefix를 공유합니다.

| openstacksdk | Go | HTTP 계약 |
|---|---|---|
| `create_receiver(**attrs)` | `Create(ctx, opts, options...)` | POST `/receivers`, 201 + receiver |
| `get_receiver(identity)` | `Get(ctx, identity)` | GET `/receivers/{identity}`, 200 |
| `update_receiver(identity, **attrs)` | `Update(...)` 또는 `Load/Track` → `Edit` → `Commit` | PATCH, 200 + receiver |
| `delete_receiver(identity, ignore_missing=True)` | `Delete(ctx, ref, options...)` | DELETE, 204 |
| `receivers(**query)` | `List` / `All` | GET `/receivers`, 200 + receivers |
| `find_receiver(identity, ignore_missing=True)` | `FindIdentity(ctx, identity, options...)` | GET 후 400·403·404 목록 fallback, 정확 ID 또는 이름 검색 |

## 생성과 응답

```python
receiver = conn.clustering.create_receiver(
    name="scale_out", type="webhook", cluster_id="CLUSTER_ID",
    action="CLUSTER_SCALE_OUT", params={},
)
message = conn.clustering.create_receiver(name="events", type="message")
receiver = conn.clustering.get_receiver(receiver.id)
```

```go
api := receivers.New(client)
receiver, err := api.Create(ctx, receivers.CreateOpts{Name: "scale_out", Type: "webhook"},
    receivers.WithCreateClusterID("CLUSTER_ID"), receivers.WithCreateAction("CLUSTER_SCALE_OUT"),
    receivers.WithCreateParams(map[string]any{}))
if err != nil { return err }
message, err := api.Create(ctx, receivers.CreateOpts{Name: "events", Type: "message"})
if err != nil { return err }
receiver, err = api.Get(ctx, receiver.ID)
if err != nil { return err }
fmt.Println(receiver.ID, receiver.Action, receiver.Channel, message.ID)
```

Name과 Type은 필수이며 서비스의 Type에는 `webhook`과 `message`가 있습니다. 다른 nonempty
Type도 그대로 보내고 서비스가 지원 여부를 검사합니다. 알려진 webhook에는 ClusterID와
Action도 필요하며 message에서는 생략할 수 있습니다. 이 조건 때문에 CRUD나 message에
별도의 microversion gate를 추측하지 않습니다. ClusterID/Action은 `request.Optional[string]`으로
생략·null·빈 문자열을 구별합니다. Actor/Params는 RawMessage로 생략·null·빈 object와 정밀한
숫자를 보존하고 서비스가 nullable 값과 명령별 매개변수의 의미를 검사합니다.

모델은 ID/Name/Type/UserID/ProjectID/DomainID, nullable ClusterID/Action, Actor/Params/Channel과
timestamp를 제공합니다. Body의 raw 추가 필드와 Header/StatusCode를 보존합니다. 숫자는
float64로 바꾸지 않으며 Channel은 인증 정보나 URL을 포함한 raw object로 유지합니다.
Create/Get/Update는 strict receiver envelope를 검사하고 incidental Location은 Header에만
보존합니다. malformed 201/200은 전체 응답 body/header/status를 가진 `resource.ResponseError`이고
이미 accepted된 mutation을 재전송하지 않습니다. 원래 HTTP 오류와 context 원인은 유지합니다.

## 수정과 owned options

```python
receiver = conn.clustering.update_receiver(
    "scale_out", name="scale_in", action="CLUSTER_SCALE_IN", params={},
)
```

```go
api := receivers.New(client)
receiver, err := api.Update(ctx, resource.Name("scale_out"), receivers.UpdateOpts{},
    receivers.WithUpdateName("scale_in"), receivers.WithUpdateAction("CLUSTER_SCALE_IN"),
    receivers.WithUpdateParams(map[string]any{}))
if err != nil { return err }
fmt.Println(receiver.ID, receiver.Action)
```

수정은 Name/Action/Params를 받으며 Type/ClusterID/Actor/Channel은 수정 입력으로 허용하지
않습니다. Optional과 RawMessage의 생략·null·빈 값은 유지하고 stateless 빈 Update는 사전 오류입니다.
SDK 소유 `TrackedReceiver`의 dirty merge·같은 값 no-op·cache/lifecycle과 `AtBasePath` scope는 [변경 추적 사용법](../tracking/receivers/README.md)에 제공합니다.
SDK가 concrete option과 serializer를 제공하며 `With...Options`와 JSON/Field 옵션은 생성 시
snapshot을 만들고 재사용마다 독립적으로 적용합니다. Update body/header는 Name lookup보다 먼저
준비하며 lookup 후에도 source/version을 검사합니다. `WithCreateField`/`WithUpdateField`는 vendor
확장을 보내는 Go 기능이며 concrete/readonly 필드와 SDK 소유 인증/version/transport header를
덮어쓸 수 없습니다. Python의 알려진 Body 속성 기반 갱신과 이 확장 경로를 구별합니다.

## 목록·필터·user microversion

```python
receivers = conn.clustering.receivers(type="message", user_id="USER_ID", limit=20)
```

```go
api := receivers.New(client)
for receiver, err := range api.List(ctx,
    receivers.WithListOptions(receivers.ListOpts{Type: "message", Limit: 20}),
    receivers.WithListUserID("USER_ID"), receivers.WithListGlobalProject(false),
    receivers.WithListFilter("params", map[string]any{"team": "infra"}),
) {
    if err != nil { return err }
    fmt.Println(receiver.ID, receiver.Params)
}
```

이 user 예제에는 numeric microversion 1.4 이상이 필요합니다. UserID는 wire `user`로 보내고
비어 있으면 생략하며 버전을 자동으로 올리지 않습니다. API query 준비와 Resources의 raw query,
모든 후속 page에서 실제 `user` 전송을 검사하므로 query 경로로 gate를 우회할 수 없습니다.
`user_id`는 SDK 별칭이고 raw wire query로 보내지 않습니다.

ListOpts는 limit/marker/name/type/cluster_id/action/sort/global_project/user를 제공합니다.
zero/empty/nil은 생략하고 명시 GlobalProject false는 유지합니다. sort는 공시된
name/type/action/cluster_id/created_at/user와 asc/desc를 검사합니다. `WithListQuery`는 concrete
입력과 local Body 필터 외의 vendor query를 실제로 전달하며 Python의 unknown query 처리와 다른
Go 확장입니다. `WithListFilter`는 알려진 Body 속성의 로컬 필터이며 별칭 project_id/domain_id/user_id도
처리합니다. object subset, 배열, 정밀한 decimal 비교를 제공하고 JSON bool과 숫자는 다른 타입입니다.
Python은 임의 sort 문자열을 서버로 전달하므로 Go의 공시된 키·방향 사전 검사는 정책 차이입니다.

목록은 lazy/reusable이며 break와 취소가 후속 요청을 멈춥니다. server links/next/HTTP Link와 명시
limit의 wire ID marker를 사용하고 짧은 nonempty page도 이어갑니다. marker는 local filter나
consumer의 ID 수정 전에 고정합니다. collection origin/path, 필터와 정렬 유지, 반복 URL/marker는
공유 pager가 검사합니다. max_items/paginated 소비 제어는 아래처럼 제공하며 deprecated JMESPath와 per-call base_path는 미결입니다.

## 이름 검색과 삭제

```python
receiver = conn.clustering.find_receiver("scale_out", ignore_missing=True)
conn.clustering.delete_receiver("scale_out", ignore_missing=True)
```

```go
api := receivers.New(client)
receiver, err := api.Find(ctx, resource.Name("scale_out"))
if err != nil { return err }
if receiver != nil { fmt.Println(receiver.ID) }
if err := api.Delete(ctx, resource.Name("scale_out")); err != nil { return err }
```

Get은 controller identity를 직접 보냅니다. Find/Update/Delete는 `resource.ID`와 `resource.Name`을
명시하고 UUID-shaped 이름도 추측하지 않습니다. Name은 정확한 이름 검색이며 중복을 거부합니다.
API.Find는 기본 미존재 `nil,nil`, Resources.Find는 strict이고 `resource.WithMissingError()` /
`resource.WithIgnoreMissing()`으로 변경합니다. Delete는 기본 미존재를 무시하고 strict 옵션을
제공하며 Name에서 얻은 wire ID를 검사한 뒤 한 번 삭제합니다. 403/409나 잘못된 응답 ID를
미존재로 취급하지 않습니다. 문자열의 GET-first와 400/403/404 목록 fallback은 별도 `FindIdentity`가 처리합니다. [자동 조회](../finding/README.md)를 참고합니다.
Receiver에는 status polling 계약이 없으므로 Resources에 status binding을 만들지 않습니다.

근거는 pinned openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의
`receiver.py:16-80`, `_proxy.py:1102-1215`, `resource.py`와
[공식 Receiver API](https://docs.openstack.org/api-ref/clustering/#receivers-receivers)입니다.

## 목록 소비 제어

| pinned Python | Go 옵션 | 소비 정책 |
|---|---|---|
| `max_items=n` | `WithListMaxItems(n)` 또는 `ListOpts.MaxItems` | 로컬 필터 이전에 검증한 raw 행을 최대 n개 소비; 0은 무제한, 음수는 사전 오류 |
| `paginated=False` | `WithListPaginated(false)` 또는 `ListOpts.Paginated` | 첫 응답만 소비하고 continuation을 처리하지 않음 |
| 기본 `paginated=True` | nil 또는 `WithListPaginated(true)` | 페이지 순회를 허용; 뒤의 옵션이 앞의 값을 덮어씀 |
| `limit=n` | `WithListOptions(ListOpts{Limit: n})` | 양수는 wire page limit; 로컬 cap과 독립 |

```go
listingAPI := receivers.New(client)
values, err := listingAPI.All(ctx,
    receivers.WithListOptions(receivers.ListOpts{Limit: 20}),
    receivers.WithListMaxItems(50))
if err != nil { return err }
fmt.Println(len(values))
for value, err := range listingAPI.List(ctx, receivers.WithListPaginated(false)) {
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

List 전체 계약은 partial입니다. 알려진 `WithListFilter`의 raw JSON 비교와 별도로 Python
Resource field/default/alias 정규화 및 query 소비, per-call base_path와
deprecated JMESPath는 계속 비교합니다. `WithListQuery`는 vendor query를 실제로 전달하는
Go 확장이며 Python unknown query 생략과 구별합니다.

공통 소비 정책과 남은 차이는 [Senlin 목록 제어](../listing/README.md), 실제 HTTP 근거는 [목록 제어 테스트](../../../api/clustering_typed_list_controls_test.go)를 참고합니다.

## 목록 호출별 헤더와 버전

`WithListHeader(key, value)`와 `WithListMicroversion("1.7")`은 이 패키지의 typed `List` /
`All`에만 적용합니다. 기본값은 source client 설정이며 명시 옵션은 공유 클라이언트를
수정하지 않습니다. 실제 wire 헤더·버전 선택, 재순회·페이지·인증 정책과 Python 비교 예제는
[Senlin 목록 호출 옵션](../listing/README.md#목록-호출별-헤더와-버전)을 참고합니다.
`headers`, `microversion`, `base_path`를 `WithListQuery`로 전달하면 HTTP 전에 오류입니다.

## 문자열 이름/ID 자동 조회

Python `find_receiver(identity, ignore_missing=False)`는 `FindIdentity`와 `WithFindIgnoreMissing(false)`로 호출합니다. SDK가 GET-first, literal 이름 query, 모든 advertised 페이지의 정확한 ID/이름 일치와 중복·후속 오류를 처리합니다. 기본 미존재는 `nil, nil`이며 아래 예제는 strict입니다.

```go
package example

import (
    "context"

    sdk "github.com/JSYoo5B/gophercloudsdk"
    "github.com/JSYoo5B/gophercloudsdk/clustering/v1/receivers"
)

func FindReceiverStrict(ctx context.Context, conn *sdk.Connection) (*receivers.Receiver, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    return service.Receivers.FindIdentity(ctx, "scale_out",
        receivers.WithFindIgnoreMissing(false))
}
```

`WithFindFallback`로 404-only·GET-only 정책을 선택하고 `WithFindHeader`·`WithFindMicroversion`으로 GET과 fallback의 동일한 호출 설정을 지정합니다. 원본 client·다른 호출은 변경하지 않습니다. [공통 FindIdentity 계약과 Python/Go 차이](../finding/README.md)에 입력 segment 정책·응답 canonical ID·오류 근거·옵션 snapshot을 설명합니다. 기존 `Find(ctx, resource.ID/Name(...))`는 명시한 경로와 기존 옵션을 유지합니다.

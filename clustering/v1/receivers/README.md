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
| `update_receiver(identity, **attrs)` | `Update(ctx, ref, opts, options...)` | PATCH, 200 + receiver |
| `delete_receiver(identity, ignore_missing=True)` | `Delete(ctx, ref, options...)` | DELETE, 204 |
| `receivers(**query)` | `List` / `All` | GET `/receivers`, 200 + receivers |
| `find_receiver(identity, ignore_missing=True)` | `Find(ctx, ref, options...)` | 명시 ID 조회 또는 정확한 Name 검색 |

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
Python Resource의 dirty merge, 기존 값과 같은 갱신의 no-op, cache/lifecycle은 별도 미결 계약입니다.
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
공유 pager가 검사합니다. Python inherited max_items/paginated 제어와 JMESPath 전체는 미결입니다.

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
미존재로 취급하지 않습니다. Python Find의 ID-first와 400/403/404 fallback 전체는 미결입니다.
Receiver에는 status polling 계약이 없으므로 Resources에 status binding을 만들지 않습니다.

근거는 pinned openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의
`receiver.py:16-80`, `_proxy.py:1102-1215`, `resource.py`와
[공식 Receiver API](https://docs.openstack.org/api-ref/clustering/#receivers-receivers)입니다.

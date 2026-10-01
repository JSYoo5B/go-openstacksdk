# Senlin nodes

`conn.ClusteringV1(ctx).Nodes`가 Senlin node의 생성·조회·목록·수정·삭제·이름 검색을 제공합니다.
Node.ID는 Senlin node이며 PhysicalID는 실제 서버 등 underlying resource의 ID입니다.
아래 tainted 예제에는 `sdk.WithMicroversion(sdk.Clustering, "1.13")`을 설정합니다.

| openstacksdk | Go | HTTP 계약 |
|---|---|---|
| `create_node(**attrs)` | `Nodes.Create(ctx, opts, options...)` | POST `/nodes`, 202 + node와 Location |
| `get_node(identity, details=False)` | `Nodes.Get(ctx, identity, options...)` | GET `/nodes/{identity}`, 200 |
| `nodes(**query)` | `Nodes.List` / `All` | GET `/nodes`, 200 |
| `update_node(identity, **attrs)` | `Nodes.Update(ctx, ref, opts, options...)` | 객체 PATCH, 202 + node와 Location |
| `delete_node(identity, ignore_missing=True, force_delete=False)` | `Nodes.Delete(ctx, ref, options...)` | DELETE, 202 + action Submission |
| `find_node(identity, ignore_missing=True)` | `Nodes.FindIdentity(ctx, identity, options...)` | GET 후 400·403·404 목록 fallback, 정확 ID 또는 이름 검색 |
| `check_node` / `recover_node` | `Check` / `Recover` | POST `/nodes/{id}/actions`, 202 |
| `perform_operation_on_node` | `PerformOperation` | POST `/nodes/{id}/ops`, 202, 1.4 이상 |

## 생성과 physical details

```python
node = conn.clustering.create_node(
    name="worker_1", profile_id="PROFILE_ID", cluster_id="CLUSTER_ID",
    role="worker", metadata={"team": "infra"},
)
node = conn.clustering.get_node(node.id, details=True)
```

```go
service, err := conn.ClusteringV1(ctx)
if err != nil { return err }
node, err := service.Nodes.Create(ctx, nodes.CreateOpts{
    Name: "worker_1", ProfileID: "PROFILE_ID",
}, nodes.WithCreateClusterID("CLUSTER_ID"), nodes.WithCreateRole("worker"),
   nodes.WithCreateMetadata(map[string]any{"team": "infra"}))
if err != nil { return err }
node, err = service.Nodes.Get(ctx, node.ID, nodes.WithGetDetails(true))
if err != nil { return err }
fmt.Println(node.ID, node.PhysicalID, node.Details)
```

Create는 필수 Name/ProfileID를 보내고 ClusterID/Role/Metadata는 명시하면 전송합니다.
ProfileID와 ClusterID의 이름·UUID·short-ID는 서비스가 해석하며 추가 부모 lookup을 하지 않습니다.
이름은 ASCII 문자로 시작하고 ASCII 문자·숫자·`_`·`.`·`-`를 사용하며 255자 미만입니다.
Get은 identity를 직접 route로 보내고 details를 지정하면 `show_details`를 보냅니다.
생략하면 query를 보내지 않으며 명시 true/false는 각각을 전송하는 Go 정책입니다.

응답은 Body/HTTP Header/StatusCode와 raw 추가 필드를 보존하고 index는 정확한
`*json.Number`, nullable timestamp/cluster/domain/role은 포인터로 유지합니다.
사용자 metadata는 `UserMetadata`, physical details/data/dependents는 raw JSON 값입니다.
Create/Update의 `Operation`과 Delete 반환값은 action 참조이며 Node.ID나 PhysicalID와 별개입니다.

## 수정과 생략·null·false

```python
node = conn.clustering.update_node(
    "worker_1", name="worker_2", role=None, metadata={}, tainted=False,
)
```

```go
api := nodes.New(client)
node, err := api.Update(ctx, resource.Name("worker_1"), nodes.UpdateOpts{},
    nodes.WithUpdateName("worker_2"), nodes.WithUpdateRoleNull(),
    nodes.WithUpdateTainted(false), nodes.WithUpdateMetadata(map[string]any{}))
if err != nil { return err }
fmt.Println(node.ID, node.Operation.ActionID)
```

Optional의 zero value는 생략, `request.Null[T]()`은 null, `request.Present(value)`는 값을 보냅니다.
Metadata RawMessage의 nil·raw null·`{}`도 별도로 유지하고 `WithUpdateMetadata(nil)`은 null입니다.
Tainted는 false나 null을 포함해 지정하면 numeric microversion 1.13 이상이 필요합니다.
null의 실제 갱신 의미는 필드별 서버 계약을 따릅니다.
`WithCreateClusterID`, `WithUpdateRoleNull`, `WithUpdateTainted` 등의 함수가 Optional을
만들고 생략·null·false를 구분하므로 호출자에게 별도 builder 구현을 요구하지 않습니다.

수정 body/header는 이름의 한 번 lookup보다 먼저 준비하므로 lookup의 변경을 받지 않습니다.
mutation 직전에도 service/version을 다시 검사합니다. SDK가 concrete options와 serializer를
제공하고 `With...Options`와 JSON 옵션은 생성 시 snapshot을 재사용마다 독립적으로 적용합니다.
응답 ID·PhysicalID·owner·status·ClusterID와 인증/version header를 확장으로 덮어쓸 수 없습니다.

## 목록과 이름 검색

```python
nodes = conn.clustering.nodes(cluster_id="CLUSTER_ID", status="ACTIVE", limit=20)
node = conn.clustering.find_node("worker_1", ignore_missing=True)
```

```go
api := nodes.New(client)
for node, err := range api.List(ctx,
    nodes.WithListOptions(nodes.ListOpts{ClusterID: "CLUSTER_ID", Status: "ACTIVE", Limit: 20}),
    nodes.WithListShowDetails(false), nodes.WithListGlobalProject(false),
    nodes.WithListFilter("metadata", map[string]any{"team": "infra"}),
) {
    if err != nil { return err }
    fmt.Println(node.ID, node.PhysicalID)
}
node, err := api.Find(ctx, resource.Name("worker_1"))
if err != nil { return err }
if node != nil { fmt.Println(node.ID) }
```

ListOpts는 limit/marker/name/cluster_id/status/sort/global_project/show_details를 보냅니다.
zero/empty/nil은 생략하고 bool의 명시 false는 유지합니다. `WithListFilter`는 알려진 Body 속성의
로컬 subset/배열/정확한 decimal 비교이며 project_id/domain_id/user_id 별칭도 처리합니다.
JSON bool과 숫자는 다른 타입입니다. 추가 server query는 `WithListQuery`로 보내고 typed/local
속성을 그 경로로 덮어쓰지 않습니다.

lazy 목록은 next/link/HTTP Link와 명시 limit의 wire ID marker를 사용합니다. 짧은 nonempty
페이지도 이어가고 빈 페이지에서 끝납니다. consumer의 모델 변경, break, context 취소,
URL/marker cycle, collection origin/path와 원래 필터 유지의 계약은 공유 구현이 처리합니다.
API.Find는 미존재를 기본 `nil,nil`로 반환하고 `resource.WithMissingError()`로 변경합니다.
Resources.Find는 strict 기본값입니다. Name은 정확한 이름 검색이며 중복을 거부합니다.

## 비동기 삭제

```python
action = conn.clustering.delete_node("NODE_ID", force_delete=True)
action = conn.clustering.get_action(action.id)
```

```go
api := nodes.New(client)
submission, err := api.Delete(ctx, resource.ID("NODE_ID"), nodes.WithDeleteForce(true))
if err != nil { return err }
if submission == nil { return nil }
action, err := actions.New(client).Get(ctx, submission.ActionID)
if err != nil { return err }
fmt.Println(action.ID, action.Status)
```

일반 Delete는 body 없이 요청하고 미존재를 기본적으로 무시하며 Force true는 `{"force":true}`와
기본 strict404를 사용합니다. `WithDeleteIgnoreMissing`으로 둘 다 명시 변경할 수 있습니다.
403/409와 잘못된 lookup 결과를 미존재로 취급하지 않습니다. 확인한 소스에 force의 최소 버전
근거가 없으므로 추측 gate를 추가하지 않습니다. Delete는 action을 자동 조회·poll하지 않습니다.
반환값은 [Submission의 Location/원문/HTTP 증거](../actions/README.md)입니다.
결과를 버리는 Resources.Delete는 `ErrUnsupported`입니다. Resources의 공유 status polling은
Python proxy wait 전체의 cached Resource/defaults와 별도로 비교합니다.

Get/Create/Update는 strict node envelope와 지정 success code를 검사합니다. 202의 Location이
누락·잘못된 경우와 accepted body decode 실패는 `resource.ResponseError`에 원문을 보존하고
생성·수정·삭제를 재전송하지 않습니다. inherited
per-call base_path와 JMESPath·dirty merge·ID-first Find는 별도 계약으로 계속 추적합니다.
근거는 pinned openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의
`node.py`, `_async_resource.py`, `_proxy.py`, `resource.py`와
[공식 Node API](https://docs.openstack.org/api-ref/clustering/#nodes)입니다.

## check·recover·plugin operation

다음 예제에는 numeric microversion 1.6 이상을 선택합니다. SDK는 명령을 위해 선택한
버전을 자동으로 올리지 않습니다.

```python
checked = conn.clustering.check_node("NODE_ID")
recovered = conn.clustering.recover_node(
    "NODE_ID", operation="REBOOT", operation_params={}, check=False,
)
performed = conn.clustering.perform_operation_on_node("NODE_ID", "reboot", type="SOFT")
```

```go
api := nodes.New(client)
checked, err := api.Check(ctx, resource.ID("NODE_ID"))
if err != nil { return err }
recovered, err := api.Recover(ctx, resource.ID("NODE_ID"), nodes.RecoverOpts{},
    nodes.WithRecoverOperation("REBOOT"), nodes.WithRecoverOperationParams(map[string]any{}),
    nodes.WithRecoverCheck(false))
if err != nil { return err }
performed, err := api.PerformOperation(ctx, resource.ID("NODE_ID"), "reboot",
    nodes.WithPerformOperationField("type", "SOFT"))
if err != nil { return err }
fmt.Println(checked.ActionID, recovered.ActionID, performed.ActionID)
```

기본 Check/Recover는 빈 매개변수 object를 보내며 임의의 최소 버전을 요구하지 않습니다.
Recover의 Check는 false/null을 포함해 명시하면 1.6 이상, PerformOperation 전체는 1.4 이상이
필요합니다. Operation/OperationParams/Check의 생략 여부를 SDK가 관리하고 JSON 객체·정밀한
숫자·명시한 빈 문자열·null은 보존합니다. `WithRecoverOperationNull()`도 명시 null입니다.
null과 operation 이름·매개변수의 지원은 서비스/profile plugin이
검증합니다. `With...Field/Header`로 추가 매개변수와 SDK 소유 외의 header를 전달합니다.

명령은 최상위 action 문자열과 같은 ID를 가리키는 Location을 가진 202 응답만 Submission으로
반환합니다. Python의 JSON dictionary 반환에 비해 엄격한 Go 정책입니다. node ID와 action ID는
구별하며 본문/헤더/status를 보존하고 action을 자동 조회·poll하지 않습니다. 준비한 body와
header 및 버전 조건은 이름 lookup의 변경을 받지 않으며 POST 직전에 source/version을 다시
검사합니다. plugin 매개변수의 id/status 같은 이름은 Resource 속성 갱신으로 취급하지 않습니다.
명시한 ID는 추가 GET 없이 route로 전달하고 `resource.Name`은 정확한 이름을 목록에서 한 번
해석합니다. Python Resource 입력과 combined-string 식별자를 이 두 방식으로 구분합니다.

## 물리 리소스 adopt와 preview

두 분기에는 numeric microversion 1.7 이상을 선택합니다. `Adopt`는 POST `/nodes/adopt`,
`AdoptPreview`는 POST `/nodes/adopt-preview`로 flat JSON body를 보내고 각각 HTTP 200을
받습니다. SDK는 버전을 자동으로 올리지 않습니다.

```python
node = conn.clustering.adopt_node(
    identity="PHYSICAL_RESOURCE_ID", type="os.nova.server-1.0",
    name="adopted_worker", snapshot=False, metadata={},
)
preview = conn.clustering.adopt_node(
    preview=True, identity="PHYSICAL_RESOURCE_ID", type="os.nova.server-1.0",
)
```

```go
api := nodes.New(client)
node, err := api.Adopt(ctx, nodes.AdoptOpts{
    Identity: "PHYSICAL_RESOURCE_ID", Type: "os.nova.server-1.0",
}, nodes.WithAdoptName("adopted_worker"), nodes.WithAdoptSnapshot(false),
   nodes.WithAdoptMetadata(map[string]any{}))
if err != nil { return err }
preview, err := api.AdoptPreview(ctx, nodes.AdoptPreviewOpts{
    Identity: "PHYSICAL_RESOURCE_ID", Type: "os.nova.server-1.0",
})
if err != nil { return err }
fmt.Println(node.ID, node.PhysicalID, preview.Spec.Type, string(preview.Spec.Version))
```

Identity/Type은 필수 body 문자열이며 path identifier 제한을 적용하거나 부모 조회를 하지
않습니다. Identity는 기존 물리 리소스의 이름 또는 ID입니다. Adopt의 Name/Role/Snapshot은
생략·null·명시한 빈 문자열/false를 구별하고 Metadata/Overrides는 생략·null·object를 유지합니다.
생략한 이름과 snapshot 등의 기본값, null과 profile별 속성의 의미는 서비스가 결정합니다.
`WithAdoptOptions`, JSON 옵션과 `WithAdoptField`는 생성 시 snapshot을 만들고 재사용마다
독립적으로 적용합니다. 추가 adopt 속성은 flat body에 들어가며 concrete 입력과 SDK 소유
인증/version/transport header를 덮어쓸 수 없습니다.

Preview는 pinned Python 구현처럼 identity/overrides/type/snapshot 네 필드만 전송합니다.
생략한 Overrides와 Snapshot도 null로 보내며 false와 `{}`는 지정한 그대로 보존합니다.
Python이 버리는 추가 preview 속성을 Go는 사전 오류로 처리합니다. custom Option의 Fields,
Query, Arguments도 허용하지 않으며 `WithAdoptPreviewHeader`로 SDK 소유 외의 header를
전달할 수 있습니다. 두 분기 모두 옵션 적용 후 POST 직전에 source/version을 재검사합니다.

Adopt는 strict node envelope를 반환하고 `Operation`은 nil입니다. Location은 요구하거나
해석하지 않으며 응답에 있으면 Header에 그대로 보존합니다. Preview는 `Spec`에 inner
node_preview, `Body`에 전체 envelope의 raw 필드, `RawBody`에 정확한 응답 원문을 유지합니다.
inner 추가 필드는 `Spec.Body`, Version은 숫자·문자열·null을 구별하는 RawMessage,
Properties는 정밀한 raw JSON map입니다. 양쪽 모두 Header/StatusCode를 보존하고 자동 action
조회나 polling을 하지 않습니다. malformed 200은 전체 원문/header/status를 가진
`resource.ResponseError`이며 adoption 요청을 재전송하지 않습니다.

근거는 pinned `node.py:143-175`, `_proxy.py:875-909`와
[공식 Node API](https://docs.openstack.org/api-ref/clustering/#nodes)입니다.

## 목록 소비 제어

| pinned Python | Go 옵션 | 소비 정책 |
|---|---|---|
| `max_items=n` | `WithListMaxItems(n)` 또는 `ListOpts.MaxItems` | 로컬 필터 이전에 검증한 raw 행을 최대 n개 소비; 0은 무제한, 음수는 사전 오류 |
| `paginated=False` | `WithListPaginated(false)` 또는 `ListOpts.Paginated` | 첫 응답만 소비하고 continuation을 처리하지 않음 |
| 기본 `paginated=True` | nil 또는 `WithListPaginated(true)` | 페이지 순회를 허용; 뒤의 옵션이 앞의 값을 덮어씀 |
| `limit=n` | `WithListOptions(ListOpts{Limit: n})` | 양수는 wire page limit; 로컬 cap과 독립 |

```go
listingAPI := nodes.New(client)
values, err := listingAPI.All(ctx,
    nodes.WithListOptions(nodes.ListOpts{Limit: 20}),
    nodes.WithListMaxItems(50))
if err != nil { return err }
fmt.Println(len(values))
for value, err := range listingAPI.List(ctx, nodes.WithListPaginated(false)) {
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

Python `find_node(identity, ignore_missing=False)`는 `FindIdentity`와 `WithFindIgnoreMissing(false)`로 호출합니다. SDK가 GET-first, literal 이름 query, 모든 advertised 페이지의 정확한 ID/이름 일치와 중복·후속 오류를 처리합니다. 기본 미존재는 `nil, nil`이며 아래 예제는 strict입니다.

```go
package example

import (
    "context"

    sdk "gophercloudsdk"
    "gophercloudsdk/clustering/v1/nodes"
)

func FindNodeStrict(ctx context.Context, conn *sdk.Connection) (*nodes.Node, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    return service.Nodes.FindIdentity(ctx, "worker_1",
        nodes.WithFindIgnoreMissing(false))
}
```

`WithFindFallback`로 404-only·GET-only 정책을 선택하고 `WithFindHeader`·`WithFindMicroversion`으로 GET과 fallback의 동일한 호출 설정을 지정합니다. 원본 client·다른 호출은 변경하지 않습니다. [공통 FindIdentity 계약과 Python/Go 차이](../finding/README.md)에 입력 segment 정책·응답 canonical ID·오류 근거·옵션 snapshot을 설명합니다. 기존 `Find(ctx, resource.ID/Name(...))`는 명시한 경로와 기존 옵션을 유지합니다.

# Senlin actions: Get / List / Update

```python
action = conn.clustering.get_action(action_id)
actions = conn.clustering.actions(target_id=node_id, limit=20)
```

```go
api := actions.New(client)
action, err := api.Get(ctx, actionID)
if err != nil {
    return err
}
fmt.Println(action.ID, action.Status)

for action, err := range api.List(ctx,
    actions.WithListOptions(actions.ListOpts{TargetID: nodeID, Limit: 20}),
) {
    if err != nil {
        return err
    }
    fmt.Println(action.ID, action.Status)
}
```

`ID`는 Action 자체의 ID이며 `TargetID`나 `ClusterID`로 대체하지 않습니다. 이름/UUID/
short-ID 경로는 서버가 해석합니다. 실행 시작·종료 시간은 epoch 숫자이므로
`*json.Number`로 정밀도를 보존합니다. 문자열 timestamp는 변환하지 않고,
inputs/outputs/data 및 추가 응답 필드도 raw JSON으로 보존합니다. null과 생략은
`Body`에서 구분하며 HTTP 근거는 `Header`, `StatusCode`에 있습니다.

Concrete `ListOpts`는 limit/marker, name, target, action, status, sort,
global_project와 pinned Python의 cluster_id 필터를 제공합니다. `GlobalProject=nil`은
false인 서버 기본값을 생략하고, `WithListGlobalProject(false)`는 false를 명시합니다.
Sort는 `created_at:desc,name`처럼 문서화된 key와 asc/desc를 사용합니다. 값과 bool
포인터는 옵션 생성 및 적용 시 복제합니다. `WithListQuery`로 추가 query를 보낼 수
있지만 concrete 필드는 덮어쓸 수 없습니다. ClusterID와 추가 query의 지원은 deployment가
검증합니다.

기본 limit 0은 생략합니다. 명시한 limit이 있으면 짧은 페이지를 포함해 비어 있지 않은
페이지의 마지막 **응답 Action ID**를 marker로 사용하고 빈 페이지에서 멈춥니다. body next/links 및 HTTP Link도
지원하며 origin/경로와 기존 필터를 고정하고 cycle을 거부합니다. List는 lazy이며 break와
context 취소 후 추가 요청을 하지 않습니다.

## 비동기 요청 결과

cluster·node의 비동기 mutation은 `actions.Submission`으로 요청 수락의 근거를 보존합니다.
`ActionID`와 원래 `Location`, 원문 `Body`, 독립적인 `Header`, `StatusCode`를 제공하며,
이 값으로 작업 완료나 Action 상태를 추정하지 않습니다. 상태가 필요하면 같은 서비스의
`Actions.Get(ctx, submission.ActionID)`를 호출합니다. Submission을 만드는 과정에서는
Location을 따라가거나 추가 조회·대기를 하지 않습니다.

Location은 선택한 서비스의 `actions/{ID}`를 가리키는 한 개의 URI여야 합니다.
동일 origin의 절대 URI, root-relative URI와 `actions/{ID}` 상대 URI를 지원하며
reverse-proxy/project 경로를 유지합니다. query·fragment·userinfo, 다른 collection,
빈 ID·추가 경로·dot segment·escaped slash는 거부합니다. Python의 마지막 slash 뒤
문자열 추출보다 엄격한 Go 정책입니다. 서버가 이미 수락한 응답의 Location 해석 실패는
`resource.ResponseError`로 원문/헤더/status를 보존하고 mutation을 다시 보내지 않습니다.
cluster/node 명령 응답은 최상위 action 문자열과 Location의 ID도 일치해야 합니다. 이 검사는
Python의 response dictionary 반환보다 엄격한 Go 정책이며, 실패해도 접수된 요청을 재전송하지
않습니다. command 매개변수와 Resource 갱신 필드는 서로 다른 범위입니다.

## action 취소 요청

```python
action = conn.clustering.update_action("ACTION_ID", status="CANCELLED")
```

```go
service, err := conn.ClusteringV1(ctx)
if err != nil { return err }
accepted, err := service.Actions.Cancel(ctx, resource.ID("ACTION_ID"),
    actions.WithUpdateForce(false))
if err != nil { return err }
fmt.Println(accepted.StatusCode, string(accepted.Body))
action, err := service.Actions.Get(ctx, "ACTION_ID")
if err != nil { return err }
fmt.Println(action.ID, action.Status)
```

numeric microversion 1.12 이상을 설정합니다. `Update(ctx, ref,
actions.UpdateOpts{Status: "CANCELLED"}, options...)`도 같은 PATCH 요청을 보냅니다.
공식 상태 값은 CANCELLED이며 Force는 body 필드가 아닌 query입니다. 생략하면 query가 없고,
`WithUpdateForce(false/true)`는 각각을 명시합니다. 고정한 Python Action 모델에는 force 필드가
없어 `update_action(force=True, ...)`의 force 속성이 누락됩니다. Go 옵션은 공식 query
기능을 제공합니다. 확장 body는 typed·응답 필드, query는 force/status, header는
인증/version을 덮어쓰지 못합니다. 입력은 이름 lookup보다 먼저 복제하고 PATCH 직전에
버전을 재검사합니다.

공식 API는 202 성공을 명시하지만 응답 object schema를 정의하지 않습니다. Go의
`UpdateResult`는 원문 Body·Header·StatusCode를 보존하며 비어 있거나 다른 형식의 본문도
받습니다. command Submission과 달리 Location을 필수 action 참조로 해석하지 않습니다.
Action 객체나 실행 상태를 만들지 않고 자동 GET·재전송도 하지 않습니다. 상태를 확인하려면
예제처럼 별도 Get을 사용합니다. Python의 cached Resource·dirty/no-op commit과 response
merge, 실제 cloud의 응답 모델 비교는 아직 부분 구현 과제입니다.

이 단위는 `get_action`, `actions`, `update_action`에 대응하는 API를 제공합니다.
Action Create/Delete는 제공하지 않습니다. `Resources`의 공유 lookup/wait 기능은 Senlin proxy의 전체 mutable Resource/
wait 기본값을 구현했다는 의미가 아닙니다. 특히 service 수준 `wait_for_status` parity는
별도 단위입니다. HTTP/accepted-response decoding 오류는 원본 응답 근거를 보존합니다.

근거: pinned openstacksdk revision `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의
`action.py`, `_proxy.py`, `resource.py`와
[공식 Actions API](https://docs.openstack.org/api-ref/clustering/#actions). Get/List 성공은 200입니다.

## 목록 소비 제어

| pinned Python | Go 옵션 | 소비 정책 |
|---|---|---|
| `max_items=n` | `WithListMaxItems(n)` 또는 `ListOpts.MaxItems` | 로컬 필터 이전에 검증한 raw 행을 최대 n개 소비; 0은 무제한, 음수는 사전 오류 |
| `paginated=False` | `WithListPaginated(false)` 또는 `ListOpts.Paginated` | 첫 응답만 소비하고 continuation을 처리하지 않음 |
| 기본 `paginated=True` | nil 또는 `WithListPaginated(true)` | 페이지 순회를 허용; 뒤의 옵션이 앞의 값을 덮어씀 |
| `limit=n` | `WithListOptions(ListOpts{Limit: n})` | 양수는 wire page limit; 로컬 cap과 독립 |

```go
listingAPI := actions.New(client)
values, err := listingAPI.All(ctx,
    actions.WithListOptions(actions.ListOpts{Limit: 20}),
    actions.WithListMaxItems(50))
if err != nil { return err }
fmt.Println(len(values))
for value, err := range listingAPI.List(ctx, actions.WithListPaginated(false)) {
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

## 명시적인 Body 필터

`actions.WithListFilter(key, value)`는 known response Body 필드를 로컬에서 비교합니다.
Python `actions(owner_id="engine")`의 Body 필터는 Go에서 `WithListFilter("owner_id", "engine")`로
지정하며, query-capable `name/target/action/status/cluster_id`도 이 옵션으로 명시하면 wire query를
보내지 않습니다. 기존 `ListOpts`는 해당 서버 query를 지정하는 독립적인 입력입니다.

지원 canonical 필드는 `id/name/target/action/cause/owner/user/project/domain/interval/start_time/end_time/timeout/status/status_reason/inputs/outputs/depends_on/depended_by/created_at/updated_at/cluster_id`입니다.
Python alias `target_id/owner_id/user_id/project_id/domain_id/start_at/end_at`는 각각
`target/owner/user/project/domain/start_time/end_time`을 비교합니다. Go 추가 모델 필드 `data`와
unknown 필드는 이 필터에 포함하지 않습니다. known Body 키를 `WithListQuery`로 보내는 것은
거부하며, 기존 typed query와 명시적인 unknown vendor query 정책은 유지합니다.

```go
package examples

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions"
)

func FilterActions(ctx context.Context, conn *sdk.Connection, ownerID string) error {
	service, err := conn.ClusteringV1(ctx)
	if err != nil {
		return err
	}
	for action, err := range service.Actions.List(ctx,
		actions.WithListFilter("owner_id", ownerID),
		actions.WithListFilter("inputs", map[string]any{"region": "region-one"}),
		actions.WithListMaxItems(50), actions.WithListPaginated(false)) {
		if err != nil {
			return err
		}
		fmt.Println(action.ID, action.Status)
	}
	return nil
}
```

필터 입력은 옵션 생성 시 JSON snapshot으로 소유합니다. cap과 마지막 응답 ID의 marker 계산은
필터보다 먼저 진행하므로 걸러진 행도 소비합니다. 객체는 recursive subset, 배열은 순서와 전체
값을 비교하며 숫자는 정확한 decimal 값으로 비교하고 bool과 숫자를 구분합니다. 생략과 null은
scalar null 필터에서 같고, 빈 실제 객체는 객체 필터에 일치하지 않습니다. 상세한 Python 차이는
[Body 필터 가이드](../listing/README.md), HTTP 근거는 [6개 리소스 필터 계약](../../../api/clustering_body_filters_test.go)을 참고합니다.

## 목록 호출별 헤더와 버전

`WithListHeader(key, value)`와 `WithListMicroversion("1.7")`은 이 패키지의 typed `List` /
`All`에만 적용합니다. 기본값은 source client 설정이며 명시 옵션은 공유 클라이언트를
수정하지 않습니다. 실제 wire 헤더·버전 선택, 재순회·페이지·인증 정책과 Python 비교 예제는
[Senlin 목록 호출 옵션](../listing/README.md#목록-호출별-헤더와-버전)을 참고합니다.
`headers`, `microversion`, `base_path`를 `WithListQuery`로 전달하면 HTTP 전에 오류입니다.

`submission.Snapshot()`은 nil을 유지하고 accepted Body/Header를 깊게 복사합니다. 호출자의 수정은 원래 submission에 영향을 주지 않으며 조회나 대기를 하지 않습니다. Cluster·Node [tracked handle](../tracking/async/README.md)은 이 기능으로 JSON 밖의 Operation 근거를 보존합니다.

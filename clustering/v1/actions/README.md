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

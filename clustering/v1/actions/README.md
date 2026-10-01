# Senlin actions: Get / List

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

이 단위는 `get_action`과 `actions`만 구현합니다. Action Update/Create/Delete는 제공하지
않습니다. `Resources`의 공유 lookup/wait 기능은 Senlin proxy의 전체 mutable Resource/
wait 기본값을 구현했다는 의미가 아닙니다. 특히 service 수준 `wait_for_status` parity는
별도 단위입니다. HTTP/accepted-response decoding 오류는 원본 응답 근거를 보존합니다.

근거: pinned openstacksdk revision `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의
`action.py`, `_proxy.py`, `resource.py`와
[공식 Actions API](https://docs.openstack.org/api-ref/clustering/#actions). Get/List 성공은 200입니다.

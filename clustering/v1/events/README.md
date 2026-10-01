# Senlin events: Get / List

```python
event = conn.clustering.get_event(event_id)
events = conn.clustering.events(obj_id=node_id, limit=20)
```

```go
api := events.New(client)
event, err := api.Get(ctx, eventID)
if err != nil {
    return err
}
fmt.Println(event.ObjectID, event.GeneratedAt, event.Level)

records, err := api.All(ctx, events.WithListOptions(events.ListOpts{
    Limit:       20,
    ObjectIDs:   []string{nodeID, otherNodeID},
    ObjectTypes: []string{"NODE", "CLUSTER"},
    Sort:        "timestamp:desc",
}))
```

Python의 obj_id/obj_name/obj_type은 Go의 ObjectIDs/ObjectNames/ObjectTypes이며 wire에서는
oid/oname/otype입니다. 각 slice와 Actions는 반복 query를 보존하고 옵션 생성·적용 시
복제합니다. 문서화된 object type은 CLUSTER와 NODE이고 deployment가 추가 type의 지원을
검증합니다. GlobalProject nil은 false인 기본값을 생략하고 `WithListGlobalProject(false)`는
명시적으로 false를 보냅니다. 추가 query는 `WithListQuery`로 보낼 수 있고 concrete 필드를
덮어쓸 수 없습니다.

Event ID와 관련 ObjectID를 구별합니다. Level은 JSON 문자열·숫자를 모두 받아 정확한
문자열 값으로 제공하고, `Body["level"]`에는 원래 JSON 타입을 보존합니다. 큰 정수를
float64로 바꾸지 않습니다. Python의 무타입 Body는 원래 값의 타입을 유지한다는 차이가
있습니다. GeneratedAt은 원래 timestamp 문자열입니다. 임의 JSON인 MetaData와 추가 응답 필드는 숫자 정밀도를 유지합니다.
`Body`는 생략/null을 구별하고 `Header`, `StatusCode`는 HTTP 근거를 담습니다.

명시한 limit의 full page 뒤에는 마지막 응답 Event ID로 marker를 만들고 마지막 빈
페이지를 허용합니다. body next/links와 HTTP Link는 동일 origin/collection 경로에
제한하고 기존 필터를 유지합니다. lazy List의 break와 context 취소는 추가 요청을
중단합니다. 기본 limit/marker는 생략합니다.

이 패키지는 `get_event`와 `events` 조회만 구현합니다. Event Status는 과거에 기록한
관련 object 상태이며 작업 완료 waiter로 간주하지 않습니다. Name Find, 변경/삭제와
기본 status Wait는 지원하지 않습니다. Python mutable Resource나 Senlin 전체 parity를
주장하지 않습니다.

근거: pinned openstacksdk revision `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`와
[공식 Events API](https://docs.openstack.org/api-ref/clustering/#events-events). Get/List 성공은 200입니다.

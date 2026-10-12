# Aggregate UpdateRecord

`API.UpdateRecord`는 openstacksdk의 [`update_aggregate`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py)를 SDK가 직접 구현한 호출입니다. 생성된 `API.Update`는 Gophercloud v2.15.0 `UpdateOpts`를 그대로 쓰기 때문에 비어 있는 `availability_zone`을 생략하고, `WithUpdateField`로 핵심 키를 덮어쓸 수도 없습니다. 그래서 Nova에서 zone을 제거하는 `availability_zone=None` 요청을 보낼 수 없었고, 이 호출이 그 빈자리를 채웁니다.

## 요청

- `PUT os-aggregates/{id}`에 `{"aggregate": {...}}` 본문을 보냅니다.
- `UpdateRecordOpts`의 필드는 `request.Optional`입니다. 생략한 필드는 보내지 않고, `request.Null`은 JSON null을, `request.Present("")`는 빈 문자열을 그대로 보냅니다.
- `WithUpdateRecordField`로 Aggregate의 다른 선언 속성을 envelope 안에 넣을 수 있습니다. 다만 `name`, `availability_zone`, URI 식별자인 `id`는 HTTP 전에 거부합니다.
- 보낼 속성이 하나도 없으면 Python의 `Resource.commit`처럼 요청을 보내지 않고 `Sent`가 false인 결과를 돌려줍니다.
- 마이크로버전은 Python처럼 서버와 협상하지 않고 클라이언트에 설정된 값을 그대로 보냅니다. Python은 min(2.81, 서버 최대값)을 씁니다.

## 응답과 상태 코드

- Python `raise_from_response`와 같이 400 미만의 모든 상태를 성공으로 받습니다. 400 이상은 Gophercloud 오류를 `resource.OperationError`(Operation `UpdateRecord`, Resource `aggregates`)로 감싸 돌려줍니다.
- 응답에 `aggregate` envelope가 있으면 벗겨 내고, 없으면 최상위 객체를 그대로 씁니다. 본문이 비었거나 JSON이 아니면 필드 없이 성공으로 끝납니다.
- `AggregateRecord.Aggregate`는 Gophercloud 모델로 디코드한 값이고 `Fields`는 응답 객체의 원본 JSON입니다. 헤더와 상태 코드도 함께 담습니다.

## Python과 다른 점

- Python은 Aggregate Resource나 ID 문자열을 받지만 Go는 정수 ID를 받습니다.
- Python은 선언되지 않은 키를 조용히 버리지만 Go의 `WithUpdateRecordField`는 받은 키를 그대로 보냅니다.
- Python은 요청 속성과 응답을 합친 Resource를 돌려주고, Go는 응답 객체만 담은 `AggregateRecord`를 돌려줍니다.

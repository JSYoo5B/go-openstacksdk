# Allocation CreateRecord

`API.CreateRecord`는 openstacksdk의 [`create_allocations(allocations)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/_proxy.py)를 SDK가 직접 구현한 호출입니다. Python의 [`Allocation.set`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/allocation.py)은 호출자가 준 dict를 그대로 `POST allocations`에 보냅니다. 생성된 `API.Manage`는 Gophercloud v2.15.0 `UpdateOpts`로 consumer마다 정해진 키만 만들고, `WithManageField`는 확장 키를 최상위에만 넣습니다. 그래서 1.34의 `mappings`처럼 consumer 안에 들어가는 키를 보낼 수 없었고, 이 호출이 그 빈자리를 채웁니다.

## 요청

- `CreateRecord(ctx, allocations)`는 consumer UUID를 키로 하는 `map[string]any`를 encoding/json으로 직렬화해 그대로 보냅니다. 그래서 `mappings`, null인 `consumer_generation`, consumer를 비우는 빈 `allocations` 객체가 각 consumer 안에 그대로 남습니다.
- 직렬화는 option이나 HTTP보다 먼저 한 번 하므로 호출 뒤 map을 바꿔도 요청은 바뀌지 않습니다. JSON으로 만들 수 없는 값은 HTTP 전에 `resource.ErrInvalidOption`으로 실패합니다.
- nil map과 빈 map은 모두 `{}`를 보냅니다.
- microversion은 client에 설정한 값을 그대로 보냅니다. `WithCreateRecordHeader`로 요청 헤더를 더할 수 있습니다.

## 응답과 상태 코드

- Python `raise_from_response`처럼 400 미만의 모든 상태를 성공으로 받고, `CreateRecordResult`에 응답 헤더와 상태 코드를 담아 돌려줍니다.
- 400 이상, 예를 들어 generation이 맞지 않는 409는 Gophercloud 상태 오류를 `resource.OperationError`(Operation `CreateRecord`, Resource `allocations`)로 감쌉니다. nil option과 nil API도 HTTP 전에 실패합니다.

## Python과 다른 점

- Python은 min(`_max_microversion` 1.38, 서버 최대값)을 협상하거나 session 기본값을 씁니다. Go는 service client의 microversion을 그대로 보내므로 1.38을 보내려면 client를 그렇게 만듭니다.
- Python은 `None`을 돌려주고 Go는 `CreateRecordResult`를 돌려줍니다.
- Python은 `allocations=None`이면 본문 없이 보내고, Go는 nil map에 `{}`를 보냅니다.

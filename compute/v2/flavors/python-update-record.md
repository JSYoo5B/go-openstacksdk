# Flavor UpdateRecord

`API.UpdateRecord`는 openstacksdk의 [`update_flavor`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py)를 SDK가 직접 구현한 호출입니다. 생성된 `API.Update`는 Gophercloud `UpdateOpts`를 그대로 쓰기 때문에 비어 있는 `description`을 생략하고, `WithUpdateField`로 핵심 키를 덮어쓸 수도 없습니다. 그래서 Nova에서 설명을 지우는 `description=None`이나 `description=""` 요청을 보낼 수 없었고, 이 호출이 그 빈자리를 채웁니다. Python 쪽 리소스 정의는 [`flavor.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/flavor.py)에 있습니다.

## 요청

- `PUT flavors/{id}`에 `{"flavor": {...}}` 본문을 보냅니다.
- `UpdateRecordOpts.Description`은 `request.Optional[string]`입니다. 생략하면 보내지 않고, `request.Null`은 JSON null을, `request.Present("")`는 빈 문자열을 그대로 보냅니다.
- `WithUpdateRecordField`로 Flavor의 다른 선언 속성을 서버 키 이름으로 envelope 안에 넣을 수 있습니다. 다만 `description`과 URI 식별자인 `id`는 HTTP 전에 거부합니다.
- `WithUpdateRecordHeader`로 요청 헤더를 더할 수 있습니다.
- 보낼 속성이 하나도 없으면 Python의 `Resource.commit`처럼 요청을 보내지 않고 `Sent`가 false인 결과를 돌려줍니다.
- 비어 있거나 공백뿐인 flavor ID는 HTTP 전에 거부합니다.

## 응답과 상태 코드

- Python `raise_from_response`와 같이 400 미만의 모든 상태를 성공으로 받습니다. 400 이상은 Gophercloud 오류를 `resource.OperationError`(Operation `UpdateRecord`, Resource `flavors`)로 감싸 돌려줍니다.
- 응답에 `flavor` envelope가 있으면 벗겨 내고, 없으면 최상위 객체를 그대로 씁니다. 본문이 비었거나 JSON이 아니면 필드 없이 성공으로 끝납니다.
- `FlavorUpdateRecord.Flavor`는 Gophercloud 모델로 디코드한 값이고 `Fields`는 응답 객체의 원본 JSON입니다. 헤더와 상태 코드도 함께 담습니다.

## Python과 다른 점

- Python은 min(Flavor `_max_microversion` 2.61, 서버 최대값)을 요청마다 협상하지만 Go는 클라이언트에 설정되거나 Connection이 협상한 마이크로버전 하나를 보냅니다.
- Python은 `is_public` 같은 속성 이름을 서버 키로 바꾸고 선언되지 않은 키를 버립니다. Go의 `WithUpdateRecordField`는 받은 키를 서버 키로 보고 그대로 보냅니다.
- Python은 요청 속성과 응답을 합친 Resource를 돌려주고, Go는 응답 객체만 담은 `FlavorUpdateRecord`를 돌려줍니다.

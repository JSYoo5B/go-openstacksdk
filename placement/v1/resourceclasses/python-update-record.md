# Resource class UpdateRecord

`API.UpdateRecord`는 openstacksdk의 [`update_resource_class(resource_class, **attrs)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/_proxy.py)를 SDK가 직접 구현한 호출입니다. Python의 [`ResourceClass`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/resource_class.py)는 `_max_microversion`이 1.2이므로 `Resource.commit`이 microversion 1.2로 `PUT resource_classes/{name}`에 JSON 본문을 보내 class 이름을 바꿉니다. 생성된 `API.Update`는 1.7 이후의 본문 없는 ensure-exists PUT만 보내고 201과 204만 받기 때문에 이 이름 변경 요청을 보낼 수 없었습니다.

## 요청

- `UpdateRecord(ctx, name, ...)`는 `PUT resource_classes/{name}`을 보냅니다. 이름은 빈칸이나 제어 문자가 없는 문자열이어야 합니다.
- `WithUpdateRecordName(new)`는 `{"name": new}`를 보냅니다. 이름을 주지 않으면 Python의 `allow_empty_commit`처럼 `{}`를 보내고, `UpdateRecordOpts{Name: request.Null[string]()}`는 `{"name": null}`을 보냅니다.
- client가 1.39처럼 다른 microversion을 쓰더라도 이 호출은 항상 `OpenStack-API-Version: placement 1.2`를 보냅니다. 1.7부터 같은 PUT이 본문 없는 ensure-exists 호출로 바뀌기 때문입니다. client의 원래 버전 헤더는 이 호출의 복사본에서만 바꾸며 client는 그대로 둡니다. `WithUpdateRecordHeader`로 다른 값의 버전 헤더를 주면 HTTP 전에 거부합니다.

## 응답과 상태 코드

- Python `raise_from_response`처럼 400 미만의 모든 상태를 성공으로 받습니다. 이미 있는 이름으로 바꾸려 할 때의 409 같은 오류는 Gophercloud 상태 오류를 `resource.OperationError`(Operation `UpdateRecord`, Resource `resourceclasses`)로 감쌉니다.
- 200 본문은 Gophercloud `ResourceClass`로 디코드해 `ResourceClass`에, 원본 객체는 `Fields`에 담습니다. 본문이 비었거나 JSON이 아니면 둘 다 nil이고, JSON이지만 객체가 아니면 `resource.ResponseError`로 실패합니다.
- 잘못된 이름, nil option, nil API는 HTTP 전에 `resource.ErrInvalidOption`으로 실패합니다.

## Python과 다른 점

- Python은 min(1.2, 서버 최대값)이나 session 기본 microversion을 보내고, Go는 이 호출에 항상 1.2를 보냅니다.
- Python은 ResourceClass나 이름 문자열을 받고 Go는 현재 이름 문자열을 받습니다.
- Python은 요청 속성과 응답을 합친 Resource를 돌려주므로 id는 이전 이름으로 남습니다. Go는 응답 객체만 담은 `ResourceClassRecord`를 돌려줍니다.

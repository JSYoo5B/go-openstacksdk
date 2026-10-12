# Compute Service UpdateRecord와 UpdateForcedDownRecord

`API.UpdateRecord`와 `API.UpdateForcedDownRecord`는 openstacksdk의 [`update_service`와 `update_service_forced_down`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py)을 SDK가 직접 구현한 호출입니다. 생성된 `API.Update`는 Gophercloud `UpdateOpts`를 그대로 쓰기 때문에 false인 `forced_down`과 빈 `status`, `disabled_reason`을 생략하고, `WithUpdateField`로 이 핵심 키를 덮어쓸 수도 없습니다. 그래서 `forced_down=False` 요청을 보낼 수 없었고, 2.53 미만의 `os-services/force-down` 응답에는 id가 없어 Gophercloud 모델로 디코드하지 못했습니다. 두 호출이 이 빈자리를 채웁니다. Python 쪽 리소스 정의는 [`service.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/service.py)에 있습니다.

## 마이크로버전 판단

Python은 `utils.supports_microversion`으로 서버 범위와 기본 마이크로버전을 함께 보고 분기합니다. Go는 서버를 다시 조회하지 않고 클라이언트에 선택된 마이크로버전 하나로 같은 판단을 합니다. 값이 비어 있으면 Nova 기본값인 2.1로 보고, `latest`는 keystoneauth `version_match`처럼 2.x 요구를 만족하지 않는 것으로 봅니다. 해석할 수 없는 값은 `resource.ErrInvalidOption`으로 거부합니다.

## UpdateRecord

- 클라이언트 마이크로버전이 2.53 미만이면 Python의 `SDKException`처럼 HTTP 없이 `resource.ErrUnsupported`를 돌려줍니다.
- `PUT os-services/{id}`에 감싸지 않은 본문을 보냅니다. `Service.commit`이 `prepend_key=False`를 쓰기 때문입니다.
- `UpdateRecordOpts`의 `Status`, `DisabledReason`, `ForcedDown`은 `request.Optional`입니다. 생략한 필드는 보내지 않고, `request.Present(false)`와 `request.Present("")`는 값을 그대로, `request.Null`은 JSON null을 보냅니다.
- `WithUpdateRecordField`로 `zone` 같은 다른 선언 속성을 서버 키 이름으로 더할 수 있습니다. 다만 위 세 키와 URI 식별자인 `id`는 HTTP 전에 거부합니다. `WithUpdateRecordHeader`로 요청 헤더를 더할 수 있습니다.
- 보낼 속성이 하나도 없으면 요청을 보내지 않고 `Sent`가 false인 결과를 돌려줍니다. 비어 있는 서비스 ID는 HTTP 전에 거부합니다.

## UpdateForcedDownRecord

`ForcedDownRecordTarget`의 `ID`, `Host`, `Binary`와 `forced` 값을 받아 Python `_get_service`와 `update_service_forced_down`의 분기를 그대로 따릅니다.

- 2.53 이상에서 `ID`가 있으면 조회 없이 `PUT os-services/{id}`에 `{"forced_down": forced}`를 보냅니다. `forced`가 false여도 키를 보냅니다.
- 2.53 이상에서 `ID`가 없으면 `Host`와 `Binary`가 모두 있어야 합니다. `GET os-services?binary=...&host=...`로 목록을 읽어 정확히 한 행일 때 그 id로 위 PUT을 보냅니다. 행이 없으면 `resource.ErrNotFound`, 여러 개면 `resource.ErrAmbiguous`를 돌려줍니다. Python은 이 두 경우에 `SDKException`을 냅니다.
- 2.53 미만에서는 `Host`와 `Binary`가 모두 있어야 하고, `Service.set_forced_down`처럼 `PUT os-services/force-down`에 `host`, `binary`를 보냅니다. 클라이언트가 2.11 이상이면 `forced_down`을 더하고 마이크로버전 헤더를 2.11로 고정합니다. 2.11 미만이면 `forced_down` 없이 클라이언트 마이크로버전을 그대로 보냅니다.
- 필요한 식별 정보가 없으면 Python의 `ValueError`처럼 HTTP 전에 `resource.ErrInvalidOption`을 돌려줍니다.

## 응답과 상태 코드

- 두 호출 모두 Python `raise_from_response`와 같이 400 미만의 모든 상태를 성공으로 받습니다. 400 이상은 Gophercloud 오류를 `resource.OperationError`(Resource `services`)로 감싸 돌려줍니다.
- 응답에 `service` envelope가 있으면 벗겨 내고, 없으면 최상위 객체를 그대로 씁니다. 본문이 비었거나 JSON이 아니면 필드 없이 성공으로 끝납니다.
- `ServiceRecord.Service`는 Gophercloud 모델로 디코드한 값이고 `Fields`는 응답 객체의 원본 JSON입니다. 2.53 미만의 force-down 응답은 id가 없고 Python도 None을 돌려주므로 `Service`를 비워 두고 `Fields`만 채웁니다.

## Python과 다른 점

- Python은 min(Service `_max_microversion` 2.69, 서버 최대값)을 요청마다 협상하지만 Go는 클라이언트에 설정되거나 Connection이 협상한 마이크로버전 하나를 보냅니다. 분기 판단도 그 값 하나로 합니다.
- Python은 Service Resource를 받아 그 안의 id나 host, binary를 쓸 수 있습니다. Go는 `ForcedDownRecordTarget`에 값을 직접 받으며, 빈 문자열을 None과 같게 봅니다. Python은 빈 문자열 host나 binary를 받으면 null로 보내지만 Go는 HTTP 전에 거부합니다.
- 2.53 미만에서 Python은 None을 돌려주고, Go는 응답 원본을 담은 `ServiceRecord`를 돌려줍니다.
- Python은 요청 속성과 응답을 합친 Resource를 돌려주고, Go는 응답 객체만 담은 `ServiceRecord`를 돌려줍니다.

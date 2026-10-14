# Compute Service EnableRecord와 DisableRecord

`API.EnableRecord`와 `API.DisableRecord`는 openstacksdk의 [`enable_service`와 `disable_service`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py)를 SDK가 직접 구현한 호출입니다. 생성된 `API.Update`로도 2.53 미만의 `os-services/enable`과 `os-services/disable` 요청은 만들 수 있지만, Nova가 돌려주는 service 객체에 `id`가 없어서 Gophercloud 모델로 디코드하지 못했습니다. 두 호출은 응답을 원본 JSON으로 보존하는 방식으로 이 문제를 피하고, host와 binary로 service를 고르는 Python `_get_service` 흐름도 함께 처리합니다. 2.53 미만 경로는 [`Service.enable`과 `Service.disable`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/service.py)을 따릅니다.

| openstacksdk | go-openstacksdk |
|---|---|
| `conn.compute.enable_service(service, host=None, binary=None)` | `Services.EnableRecord(ctx, services.ServiceActionTarget{ID, Host, Binary})` |
| `conn.compute.disable_service(service, host=None, binary=None, disabled_reason=None)` | `Services.DisableRecord(ctx, services.ServiceActionTarget{ID, Host, Binary}, reason)` |

`ServiceActionTarget`은 `ForcedDownRecordTarget`의 별칭이라서 [UpdateForcedDownRecord](python-update-record.md)와 같은 규칙을 씁니다. `ID`, `Host`, `Binary`, `reason`의 빈 문자열은 Python의 None과 같은 뜻입니다.

## 마이크로버전 판단

Python은 `utils.supports_microversion`으로 서버 범위와 기본 마이크로버전을 함께 보고 분기합니다. Go는 `UpdateRecord`, `UpdateForcedDownRecord`와 마찬가지로 서버를 다시 조회하지 않고 클라이언트에 선택된 마이크로버전 하나로 판단합니다. 값이 비어 있으면 Nova 기본값인 2.1로 보고, `latest`는 2.x 요구를 만족하지 않는 것으로 봅니다. 해석할 수 없는 값은 HTTP 전에 `resource.ErrInvalidOption`으로 거부합니다. 요청에는 클라이언트 마이크로버전을 그대로 보내고, 값이 비어 있으면 버전 헤더를 보내지 않습니다.

## 2.53 이상

- `ID`가 있으면 조회 없이 `PUT os-services/{id}`에 envelope 없는 본문 `{"status": "enabled"}`를 보냅니다. 비활성화는 `{"status": "disabled"}`이고, reason이 비어 있지 않으면 `disabled_reason`을 함께 보냅니다. 이때 `Host`와 `Binary`는 Python처럼 쓰지 않습니다.
- `ID`가 없으면 `Host`와 `Binary`가 모두 있어야 합니다. `GET os-services?binary=...&host=...`로 목록을 읽어 정확히 한 행일 때 그 id로 PUT을 보냅니다. 행이 없으면 `resource.ErrNotFound`, 여러 개면 `resource.ErrAmbiguous`를 돌려주며 PUT은 보내지 않습니다. Python은 이 두 경우에 `SDKException`을 냅니다.
- 목록에서 고른 행은 Python Resource처럼 값이 바뀐 속성만 본문에 담습니다. 그래서 이미 `disabled`인 service에 새 reason을 주면 `{"disabled_reason": ...}`만 보내고, status와 reason이 모두 같으면 PUT 없이 목록 행을 돌려줍니다. 이 경우 `Sent`는 false이고 `Header`와 `StatusCode`는 목록 응답의 값입니다.

## 2.53 미만

- `Host`와 `Binary`가 모두 있어야 하며, `ID`가 있어도 Python `_get_service`처럼 둘 중 하나가 없으면 HTTP 전에 `resource.ErrInvalidOption`을 돌려줍니다.
- 활성화는 `PUT os-services/enable`에 `{"host": ..., "binary": ...}`를 보냅니다. 비활성화는 reason이 없으면 `PUT os-services/disable`, 있으면 `PUT os-services/disable-log-reason`에 같은 본문과 `disabled_reason`을 보냅니다.
- 응답에는 id가 없으므로 `ServiceRecord.Service`를 비워 두고 `Fields`에 원본 service 객체를 담습니다.

## 응답과 상태 코드

- Python `raise_from_response`와 같이 400 미만의 모든 상태를 성공으로 받습니다. 400 이상은 Gophercloud 상태 오류를 원래 본문과 함께 `resource.OperationError`(`Operation`은 `EnableRecord` 또는 `DisableRecord`, `Resource`는 `services`)로 감싸 돌려줍니다.
- 응답에 `service` envelope가 있으면 벗겨 내고, 없으면 최상위 객체를 그대로 씁니다. 본문이 비었거나 JSON이 아니면 필드 없이 성공으로 끝납니다.
- 2.53 이상 응답의 `ServiceRecord.Service`는 Gophercloud 모델로 디코드한 값이고, `Fields`는 응답 객체의 원본 JSON입니다.

## Python과 다른 점

- Python은 min(Service `_max_microversion` 2.69, 서버 최대값)을 요청마다 협상하고 분기도 서버 범위로 판단합니다. Go는 클라이언트에 설정되거나 Connection이 협상한 마이크로버전 하나로 판단하고 그 값을 보냅니다.
- Python은 Service Resource를 받아 그 안의 id나 host, binary를 쓸 수 있지만 Go는 `ServiceActionTarget`에 값을 직접 받습니다. Python은 빈 문자열 host나 binary를 받으면 그대로 보내지만 Go는 HTTP 전에 거부합니다.
- Python은 요청 속성과 응답을 합친 Service Resource를 돌려주고, 2.53 미만에서도 seed id를 가진 Resource를 돌려줍니다. Go는 응답 객체만 담은 `ServiceRecord`를 돌려주며, 2.53 미만에서는 typed `Service` 없이 `Fields`만 채웁니다.
- Python의 host, binary 조회는 `services_links`, `links`, `next`, HTTP `Link`의 다음 page를 따라가지만 Go는 Nova가 돌려주는 한 번의 `os-services` 응답만 읽습니다.

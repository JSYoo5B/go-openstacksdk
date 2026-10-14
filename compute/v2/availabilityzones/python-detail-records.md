# Availability zone 상세 record 목록

이 API는 pinned openstacksdk Compute proxy의 [`availability_zones(details=True)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py)에 대응합니다. `details=False` 쪽은 기존 `ListRecords`와 `compute.Service.ListAvailabilityZones`가 맡고, 상세 쪽은 같은 record 구현을 경로만 바꿔 쓰는 `ListDetailRecords`가 맡습니다.

| openstacksdk | go-openstacksdk | 판정 |
|---|---|---|
| `conn.compute.availability_zones()` | `service.AvailabilityZones.ListRecords(ctx, options...)` | go_mapping |
| `conn.compute.availability_zones(details=True)` | `service.AvailabilityZones.ListDetailRecords(ctx, options...)` | go_mapping |

## 요청

`ListDetailRecords`는 iterator를 소비할 때 `GET os-availability-zone/detail`을 본문과 query 없이 보냅니다. [`AvailabilityZoneDetail`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/availability_zone.py)은 `AvailabilityZone`의 별칭이고 `_max_microversion`이 없으므로, Python처럼 client microversion이 있으면 그 값을 보내고 비어 있으면 버전 헤더를 보내지 않습니다. 옵션은 `ListRecords`와 같아서 `WithAvailabilityZoneMicroversion`으로 호출별 버전을 고르거나 빈 값으로 버전 없는 요청을 만들 수 있고, `WithAvailabilityZoneHeader`로 헤더를 더할 수 있습니다.

응답의 `availabilityZoneInfo_links`, `links`, HTTP `Link` next를 따라 다음 page를 읽습니다. 다음 page도 같은 `/detail` 경로에 있어야 하며, 빈 page를 만나면 next link가 있어도 멈춥니다.

## 결과와 상태 코드

각 행은 `*AvailabilityZoneRecord`입니다. `Resource.Body`에는 Python 속성 `id`, `name`, `state`, `hosts`, `location`이 들어갑니다. wire 이름 `zoneName`, `zoneState`는 `name`, `state`로 옮기고 없는 값은 JSON null로 둡니다. `hosts`는 host별 service 상태(`active`, `available`, `updated_at`)를 변환하지 않은 raw JSON으로 보존합니다. `Wire`와 `Envelope`는 실제 행 전체를 vendor 필드까지 담고, `Header`와 `StatusCode`는 그 행이 들어 있던 page 응답의 값입니다.

성공 상태는 200 이상 400 미만입니다. 그 밖의 상태(관리자 권한이 없을 때의 403 등)는 `gophercloud.ErrUnexpectedResponseCode`를 원래 본문과 함께 보존하고 `*resource.OperationError`(`Operation`은 `ListDetailRecords`, `Resource`는 `availabilityzones`)로 감싸서 iterator 오류로 돌려줍니다.

## Python과 다른 점

- Python은 `details` bool 인자 하나로 경로를 고르지만 Go는 `ListRecords`와 `ListDetailRecords` 두 메서드로 나눕니다.
- Python은 next link를 별도 검증 없이 따라가지만 Go는 같은 origin과 같은 `/detail` 경로의 continuation만 허용합니다.
- Python Resource는 `hosts`를 dict로 돌려주지만 Go는 raw JSON을 보존하므로 필요한 경우 호출자가 직접 decode합니다. 생성된 native `ListDetail`은 Gophercloud 타입으로 decode하는 별도 API로 남아 있습니다.

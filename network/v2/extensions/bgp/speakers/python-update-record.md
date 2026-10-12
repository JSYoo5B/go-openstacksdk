# BGP Speaker UpdateRecord

`API.UpdateRecord`는 openstacksdk의 [`update_bgp_speaker`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)를 SDK가 직접 구현한 호출입니다. 생성된 `API.Update`는 Gophercloud v2.15.0 `UpdateOpts`를 쓰는데, 두 advertise 플래그가 `omitempty` 없는 일반 bool이라 이름만 바꾸는 요청에도 항상 함께 실립니다. 그래서 Python의 `update_bgp_speaker(speaker, name="edge")`처럼 이름만 보내려면 speaker를 먼저 읽어 기존 값을 다시 보내야 했고, 이 호출이 그 빈자리를 채웁니다.

## 요청

- `PUT bgp-speakers/{id}`에 `{"bgp_speaker": {...}}` 본문을 보냅니다.
- `UpdateRecordOpts`의 `Name`, `AdvertiseFloatingIPHostRoutes`, `AdvertiseTenantNetworks`는 `request.Optional`입니다. 생략한 필드는 보내지 않으니 이름만 지정하면 플래그는 본문에 들어가지 않습니다. `request.Present(false)`는 false를, `request.Null`은 JSON null을 그대로 보냅니다. `WithUpdateRecordName`, `WithUpdateRecordAdvertiseFloatingIPHostRoutes`, `WithUpdateRecordAdvertiseTenantNetworks`로 하나씩 지정할 수도 있습니다.
- [`BgpSpeaker`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/bgp_speaker.py)는 `_allow_unknown_attrs_in_body`가 켜져 있어 Python도 선언되지 않은 키를 본문에 넣습니다. Go에서는 `networks`나 `local_as` 같은 다른 키를 `WithUpdateRecordField`로 envelope 안에 넣습니다. 다만 타입이 있는 세 키와 URI 식별자인 `id`는 HTTP 전에 거부합니다.
- `WithUpdateRecordHeader`로 요청 헤더를 더할 수 있습니다.
- 보낼 속성이 하나도 없으면 Python의 [`Resource.commit`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)처럼 요청을 보내지 않고 `Sent`가 false인 결과를 돌려줍니다.
- 비어 있거나 공백뿐인 ID와 `.`, `..`는 HTTP 전에 `resource.ErrInvalidOption`으로 거부합니다.

## 응답과 상태 코드

- Python `raise_from_response`와 같이 400 미만의 모든 상태를 성공으로 받습니다. 400 이상은 Gophercloud 오류를 `resource.OperationError`(Operation `UpdateRecord`, Resource `speakers`)로 감싸 돌려줍니다.
- 응답에 `bgp_speaker` envelope가 있으면 벗겨 내고, 없으면 최상위 객체를 그대로 씁니다. 본문이 비었거나 JSON이 아니면 필드 없이 성공으로 끝납니다.
- `BGPSpeakerRecord.BGPSpeaker`는 Gophercloud 모델로 디코드한 값이고 `Fields`는 응답 객체의 원본 JSON입니다. 헤더와 상태 코드도 함께 담습니다.

## Python과 다른 점

- Python은 BgpSpeaker Resource나 ID 문자열을 받지만 Go는 ID 문자열만 받습니다.
- Python은 `id` 키를 dirty 목록에서 지우고 조용히 넘어가지만 Go는 `WithUpdateRecordField("id", ...)`를 거부합니다.
- Python은 요청 속성과 응답을 합친 Resource를 돌려주고, Go는 응답 객체만 담은 `BGPSpeakerRecord`를 돌려줍니다.

# BGP VPN CreateRecord

`API.CreateRecord`는 openstacksdk의 [`create_bgpvpn`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)을 SDK가 직접 구현한 호출입니다. 생성된 `API.Create`는 Gophercloud v2.15.0 `CreateOpts`를 쓰는데, `LocalPref`가 `omitempty` 정수라 0이 빠지고 `WithCreateField("local_pref", 0)`도 핵심 키라서 거부됩니다. 그래서 networking-bgpvpn이 허용하는 `create_bgpvpn(local_pref=0)`을 보낼 수 없었고, 이 호출이 그 빈자리를 채웁니다.

## 요청

- `POST bgpvpn/bgpvpns`에 `{"bgpvpn": {...}}` 본문을 보냅니다. Python의 `Resource.create`처럼 속성이 하나도 없어도 빈 envelope로 요청을 보냅니다.
- `CreateRecordOpts`는 [`BgpVpn`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/bgpvpn.py)에 선언된 `name`, `project_id`, `type`, `route_distinguishers`, `route_targets`, `import_targets`, `export_targets`, `local_pref`, `vni`를 `request.Optional`로 담습니다. 생략한 필드는 보내지 않고, `request.Present`는 0과 빈 문자열, 빈 목록을 그대로 보내며 `request.Null`은 JSON null을 보냅니다. `WithCreateRecordLocalPref`로 `local_pref`만 따로 지정할 수도 있습니다.
- `BgpVpn`은 `_allow_unknown_attrs_in_body`가 켜져 있어 Python도 선언되지 않은 키를 본문에 넣습니다. Go에서는 선언됐지만 타입 필드가 없는 `tenant_id`나 선언되지 않은 키를 `WithCreateRecordField`로 envelope 안에 넣습니다. 다만 타입이 있는 키는 HTTP 전에 거부합니다.
- `WithCreateRecordHeader`로 요청 헤더를 더할 수 있습니다.

## 응답과 상태 코드

- Python `raise_from_response`와 같이 400 미만의 모든 상태를 성공으로 받습니다. 생성된 `API.Create`가 201과 202만 받는 것과 다릅니다. 400 이상은 Gophercloud 오류를 `resource.OperationError`(Operation `CreateRecord`, Resource `bgpvpns`)로 감싸 돌려줍니다.
- 응답에 `bgpvpn` envelope가 있으면 벗겨 내고, 없으면 최상위 객체를 그대로 씁니다. 본문이 비었거나 JSON이 아니면 필드 없이 성공으로 끝납니다.
- `BGPVPNRecord.BGPVPN`은 Gophercloud 모델로 디코드한 값이고 `Fields`는 응답 객체의 원본 JSON입니다. 헤더와 상태 코드도 함께 담습니다.

## Python과 다른 점

- Python은 `list`로 선언된 속성에 들어온 값을 Resource 타입 규칙에 따라 변환하지만 Go는 Go 타입 그대로 직렬화합니다.
- Python은 요청 속성과 응답을 합친 Resource를 돌려주고, Go는 응답 객체만 담은 `BGPVPNRecord`를 돌려줍니다.

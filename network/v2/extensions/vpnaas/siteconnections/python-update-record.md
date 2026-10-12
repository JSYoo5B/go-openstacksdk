# IPsec Site Connection UpdateRecord

`API.UpdateRecord`는 openstacksdk의 [`update_vpn_ipsec_site_connection`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)을 SDK가 직접 구현한 호출입니다. 생성된 `API.Update`는 Gophercloud v2.15.0 `UpdateOpts`를 그대로 쓰는데, `local_id`, `peer_address`, `peer_id`, `peer_cidrs`, endpoint group ID, `psk`, `mtu`가 `omitempty`라 빈 문자열과 0, 빈 목록이 빠집니다. `WithUpdateField`도 이 핵심 키를 덮어쓰지 못하니 Python의 `local_id=""`나 `peer_cidrs=[]` 요청을 보낼 수 없었고, 이 호출이 그 빈자리를 채웁니다.

## 요청

- `PUT vpn/ipsec-site-connections/{id}`에 `{"ipsec_site_connection": {...}}` 본문을 보냅니다.
- `UpdateRecordOpts`는 `name`, `description`, `local_id`, `peer_address`, `peer_id`, `peer_cidrs`, `local_ep_group_id`, `peer_ep_group_id`, `mtu`, `initiator`, `psk`, `dpd`, `admin_state_up`을 `request.Optional`로 담습니다. 생략한 필드는 보내지 않고, `request.Present`는 빈 문자열과 0, 빈 목록, false를 그대로 보내며 `request.Null`은 JSON null을 보냅니다.
- [`VpnIPSecSiteConnection`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/vpn_ipsec_site_connection.py)에 선언된 나머지 속성(`auth_mode`, `route_mode`, `ikepolicy_id`, `ipsecpolicy_id`, `vpnservice_id` 등)은 `WithUpdateRecordField`로 envelope 안에 넣습니다. 타입이 있는 키와 URI 식별자인 `id`는 HTTP 전에 거부합니다.
- `WithUpdateRecordHeader`로 요청 헤더를 더할 수 있습니다.
- 보낼 속성이 하나도 없으면 Python의 [`Resource.commit`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)처럼 요청을 보내지 않고 `Sent`가 false인 결과를 돌려줍니다.
- 비어 있거나 공백뿐인 ID와 `.`, `..`는 HTTP 전에 `resource.ErrInvalidOption`으로 거부합니다.

## 응답과 상태 코드

- Python `raise_from_response`와 같이 400 미만의 모든 상태를 성공으로 받습니다. 400 이상은 Gophercloud 오류를 `resource.OperationError`(Operation `UpdateRecord`, Resource `siteconnections`)로 감싸 돌려줍니다.
- 응답에 `ipsec_site_connection` envelope가 있으면 벗겨 내고, 없으면 최상위 객체를 그대로 씁니다. 본문이 비었거나 JSON이 아니면 필드 없이 성공으로 끝납니다.
- `ConnectionRecord.Connection`은 Gophercloud 모델로 디코드한 값이고 `Fields`는 응답 객체의 원본 JSON입니다. 헤더와 상태 코드도 함께 담습니다.

## Python과 다른 점

- Python은 VpnIPSecSiteConnection Resource나 ID 문자열을 받지만 Go는 ID 문자열만 받습니다.
- Python 인자 이름은 `is_admin_state_up`이지만 Go 필드는 본문 키를 따라 `AdminStateUp`입니다.
- 이 Resource는 `_allow_unknown_attrs_in_body`가 꺼져 있어 Python은 선언되지 않은 키를 조용히 버립니다. 반면 Go의 `WithUpdateRecordField`는 받은 키를 그대로 보냅니다.
- Python은 `mtu`처럼 타입이 선언된 속성의 값을 보내기 전에 변환하지만 Go는 Go 타입 그대로 직렬화합니다.
- Python은 요청 속성과 응답을 합친 Resource를 돌려주고, Go는 응답 객체만 담은 `ConnectionRecord`를 돌려줍니다.

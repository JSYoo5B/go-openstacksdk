# Firewall rule CreateRecord

`API.CreateRecord`는 openstacksdk의 [`create_firewall_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)과 이를 그대로 부르는 cloud layer의 [`create_firewall_rule(**kwargs)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py)를 SDK가 직접 구현한 호출입니다. 생성된 `API.Create`는 Gophercloud v2.15.0 `CreateOpts`를 쓰는데, 이 타입은 protocol과 action을 필수로 요구하고 빈 문자열을 `omitempty`로 버립니다. 그래서 인자 없는 Python 호출처럼 Neutron 기본값에 맡기는 요청이나 빈 description 같은 값을 보낼 수 없었고, 이 호출이 그 빈자리를 채웁니다.

## 요청

- `POST fwaas/firewall_rules`에 `{"firewall_rule": {...}}` 본문을 보냅니다. 속성을 하나도 주지 않으면 `{"firewall_rule": {}}`를 보내고 Neutron이 protocol any와 action deny 같은 기본값을 정합니다.
- `CreateRecordOpts`는 [`FirewallRule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/firewall_rule.py)에 선언된 `action`, `description`, `destination_ip_address`, `destination_port`, `enabled`, `ip_version`, `name`, `project_id`, `protocol`, `shared`, `source_ip_address`, `source_port`, `firewall_policy_id`를 `request.Optional`로 담습니다. 생략한 필드는 보내지 않고, `request.Present`는 빈 문자열과 false를 그대로 보내며 `request.Null`은 JSON null을 보냅니다.
- `FirewallRule`은 선언되지 않은 본문 속성을 허용하므로 `source_firewall_group_id` 같은 키는 `WithCreateRecordField`로 envelope 안에 넣습니다. `CreateRecordOpts`에서 생략한 선언 키라면 `WithCreateRecordField("protocol", 6)`처럼 다른 JSON 타입으로도 보낼 수 있습니다. 다만 typed 필드에 이미 준 키를 다시 주면 HTTP 전에 거부합니다.
- `WithCreateRecordHeader`로 요청 헤더를 더할 수 있습니다.

## 응답과 상태 코드

- Python `raise_from_response`와 같이 400 미만의 모든 상태를 성공으로 받습니다. 400 이상은 Gophercloud 오류를 `resource.OperationError`(Operation `CreateRecord`, Resource `rules`)로 감싸 돌려줍니다.
- 응답에 `firewall_rule` envelope가 있으면 벗겨 내고, 없으면 최상위 객체를 그대로 씁니다. 본문이 비었거나 JSON이 아니면 필드 없이 성공으로 끝납니다.
- `RuleRecord.Rule`은 Gophercloud 모델로 디코드한 값이고 `Fields`는 응답 객체의 원본 JSON입니다. 헤더와 상태 코드도 함께 담습니다.
- 빈 확장 키, nil option, nil API는 HTTP 전에 `resource.ErrInvalidOption`으로 실패합니다.

## Python과 다른 점

- Python은 요청 속성과 응답을 합친 FirewallRule Resource를 돌려주고, Go는 응답 객체만 담은 `RuleRecord`를 돌려줍니다.
- Go는 같은 키를 `CreateRecordOpts`와 `WithCreateRecordField`에 함께 주면 거부합니다. Python keyword 인자로는 이런 중복을 만들 수 없습니다.
- 문자열 `firewall_policy_id`처럼 Gophercloud `Rule` 모델에 맞지 않는 응답 필드는 Go의 typed 디코드를 실패시키지만 Python은 그대로 보관합니다.

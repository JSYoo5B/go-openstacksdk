# Firewall policy InsertRuleRecord

`API.InsertRuleRecord`는 openstacksdk의 [`insert_rule_into_policy`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)를 SDK가 직접 구현한 호출입니다. Python은 위치를 주지 않아도 `insert_after`와 `insert_before`를 JSON null로 항상 보냅니다. 생성된 `API.InsertRule`은 Gophercloud v2.15.0 `InsertRuleOpts`를 쓰는데, 이 타입은 두 위치 중 정확히 하나를 요구하고 두 키는 확장으로도 null을 넣을 수 없습니다. 그래서 기본 Python 호출을 보낼 수 없었고, 이 호출이 그 빈자리를 채웁니다.

## 요청

- `PUT fwaas/firewall_policies/{id}/insert_rule`에 envelope 없는 `{"firewall_rule_id": ..., "insert_after": ..., "insert_before": ...}`를 보냅니다.
- `InsertRuleRecordOpts`의 nil 위치는 null로, 지정한 빈 문자열은 그대로 보냅니다. `WithInsertRuleRecordAfter`와 `WithInsertRuleRecordBefore`로 위치를 하나씩 지정할 수 있고, `WithInsertRuleRecordOptions`는 option을 만들 때 값을 복사합니다.
- 두 위치를 함께 주거나 둘 다 생략해도 Python처럼 그대로 보내므로 Neutron이 검증합니다. rule ID도 받은 문자열 그대로 보냅니다.
- `WithInsertRuleRecordHeader`로 요청 헤더를 더할 수 있습니다. 본문 확장 키는 받지 않습니다.

## 응답과 상태 코드

- [`FirewallPolicy._put_request`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/firewall_policy.py)처럼 400 미만의 모든 상태를 받되 본문은 JSON 객체여야 합니다. 객체가 아니면 실제 응답을 담은 `resource.ResponseError`로 실패합니다.
- `PolicyRecord.Policy`는 Gophercloud 모델로 디코드한 값이고 `Fields`는 응답 객체의 원본 JSON입니다. 헤더와 상태 코드도 함께 담습니다.
- 400 이상은 Gophercloud 상태 오류를 `resource.OperationError`(Operation `InsertRuleRecord`, Resource `policies`)로 감쌉니다. 빈 policy ID, nil option, nil API는 HTTP 전에 `resource.ErrInvalidOption`으로 실패합니다.

## Python과 다른 점

- Python은 policy와 rule을 ID나 Resource로 받고 Go는 ID 문자열을 받으며 빈 policy ID를 HTTP 전에 거부합니다.
- Python은 FirewallPolicy Resource를 갱신해 돌려주고 Go는 `PolicyRecord`를 돌려줍니다.
- Python은 400 이상에서 NeutronError 메시지를 담은 `HttpException`을 내고 Go는 Gophercloud 상태 오류를 돌려줍니다.

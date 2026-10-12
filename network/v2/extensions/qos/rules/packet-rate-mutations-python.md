# QoS packet rate rule의 Python/Go 생성·수정·삭제

고정 Source는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, Gophercloud는 `v2.15.0`입니다. Gophercloud에는 minimum packet rate rule과 packet rate limit rule의 생성·수정·삭제 바인딩이 없어서 SDK가 `network/v2/extensions/qos/rules`에 직접 구현했습니다. 조회와 find, 목록은 [minimum packet rate 조회](../../../qos-minimum-packet-rate-records-python.md)와 [packet rate limit 조회](../../../qos-packet-rate-limit-records-python.md)에 설명한 기존 Record API이며, 이 문서의 메서드도 같은 `MinimumPacketRateRecord`와 `PacketRateLimitRecord`를 돌려줍니다.

| Python `conn.network` | Go `network.QoSRules` | 요청 |
|---|---|---|
| `create_qos_minimum_packet_rate_rule(qos_policy, **attrs)` | `CreateMinimumPacketRateRecord(ctx, policyID, options...)` | `POST /v2.0/qos/policies/{policy_id}/minimum_packet_rate_rules` |
| `update_qos_minimum_packet_rate_rule(qos_rule, qos_policy, **attrs)` | `UpdateMinimumPacketRateRecord(ctx, policyID, ruleID, options...)` | `PUT .../minimum_packet_rate_rules/{rule_id}` |
| `delete_qos_minimum_packet_rate_rule(qos_rule, qos_policy, ignore_missing=True)` | `DeleteMinimumPacketRateRecord(ctx, policyID, ruleID, options...)` | `DELETE .../minimum_packet_rate_rules/{rule_id}` |
| `create_qos_packet_rate_limit_rule(qos_policy, **attrs)` | `CreatePacketRateLimitRecord(ctx, policyID, options...)` | `POST /v2.0/qos/policies/{policy_id}/packet_rate_limit_rules` |
| `update_qos_packet_rate_limit_rule(qos_rule, qos_policy, **attrs)` | `UpdatePacketRateLimitRecord(ctx, policyID, ruleID, options...)` | `PUT .../packet_rate_limit_rules/{rule_id}` |
| `delete_qos_packet_rate_limit_rule(qos_rule, qos_policy, ignore_missing=True)` | `DeletePacketRateLimitRecord(ctx, policyID, ruleID, options...)` | `DELETE .../packet_rate_limit_rules/{rule_id}` |

Go는 기존 QoS Record API처럼 policy ID를 먼저 받습니다. Python `_get_resource(QoSPolicy, qos_policy)`는 HTTP 없이 ID만 꺼내므로 Go도 policy를 조회하지 않고 받은 ID를 한 path segment로 escape합니다.

```python
rule = conn.network.create_qos_minimum_packet_rate_rule(
    policy, min_kpps=1000, direction="any")
conn.network.update_qos_minimum_packet_rate_rule(rule, policy, min_kpps=2000)
conn.network.delete_qos_minimum_packet_rate_rule(rule, policy)
```

```go
rule, err := network.QoSRules.CreateMinimumPacketRateRecord(ctx, policyID,
    rules.WithPacketRateRuleField("min_kpps", 1000),
    rules.WithPacketRateRuleField("direction", "any"))
if err != nil {
    return err
}
var ruleID string
_ = json.Unmarshal(rule.Resource.Body["id"], &ruleID)
if _, err := network.QoSRules.UpdateMinimumPacketRateRecord(ctx, policyID, ruleID,
    rules.WithPacketRateRuleField("min_kpps", 2000)); err != nil {
    return err
}
_, err = network.QoSRules.DeleteMinimumPacketRateRecord(ctx, policyID, ruleID)
return err
```

## 요청 Body

두 rule 클래스는 `Resource`를 직접 상속하고 `_allow_unknown_attrs_in_body`를 켭니다. 그래서 `WithPacketRateRuleField`로 준 attribute는 선언 여부와 관계없이 `minimum_packet_rate_rule` 또는 `packet_rate_limit_rule` 아래에 그대로 보냅니다. 명시한 `null`, `false`, `0`도 보내고, attribute가 없는 생성은 빈 객체 envelope을 보내며 필수 값 검사는 Neutron이 합니다. `min_kpps`, `max_kpps`, `max_burst_kpps`는 타입 선언이 없어서 Go도 숫자를 바꾸지 않습니다.

`qos_policy_id`와 `qos_policy`는 bound policy 인자와 겹치므로 Python처럼 실패하며, Go는 HTTP 전에 `resource.ErrInvalidOption`을 돌려줍니다. 수정에서 `id`와 `qos_rule`도 같은 이유로 거부합니다. `microversion` attribute는 Python `Resource.new`처럼 버리고, 두 클래스가 `NetworkResource`가 아니므로 수정의 `if_revision`도 Python `_update`처럼 버려서 `If-Match`를 만들지 않습니다. Resource 제어 이름(`base_path`, `connection`, `location` 등)은 거부합니다.

수정할 attribute가 없으면 Python `commit`처럼 HTTP 없이 ID만 담긴 view를 돌려줍니다. 응답 Body가 비어 있거나 JSON이 아니면 요청 값으로 만든 view를 돌려주고 `Wire`는 `nil`입니다. 응답 member에 `qos_policy_id`가 들어 있으면 Source `to_dict()`처럼 view에는 고정 policy ID를 보이고, 서버가 보낸 값은 `Wire`에 남습니다.

## 삭제와 상태 코드

삭제는 Body 없이 보내며 Python `ignore_missing=True`처럼 깨끗한 404를 기본으로 무시합니다. 이때 `PacketRateRuleDeleteResult.IgnoredMissing`이 `true`입니다. `WithPacketRateRuleDeleteIgnoreMissing(false)`이면 404를 돌려줍니다. Python은 `None`을 돌려주지만 Go는 실제 status, header, body를 담은 결과를 돌려줍니다.

200부터 399까지를 성공으로 받고, 400 이상은 원래 status와 body를 담은 Gophercloud `ErrUnexpectedResponseCode`입니다. 모든 오류는 `resource.OperationError`로 감싸며 `Operation`은 Go 메서드 이름, `Resource`는 `rules`입니다.

## Python과 다른 점

- Python은 rule과 policy에 ID나 Resource 객체를 받지만 Go는 ID 문자열만 받고 policy ID를 먼저 받습니다.
- Neutron에는 microversion이 없어서 Python은 version header를 보내지 않습니다. Go는 선택한 client에 `Microversion`이 설정되어 있을 때만 `OpenStack-API-Version`을 보내며, `WithPacketRateRuleMicroversion`과 header 옵션은 Go 확장입니다.
- `location`은 항상 `null`이고, 응답 status 100부터 199는 성공으로 받지 않습니다.

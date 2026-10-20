# Packet Rate Mutations Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/create_qos_minimum_packet_rate_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.CreateMinimumPacketRateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateRuleMutationOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateRuleField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateRuleHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateRuleMicroversion`

- POST /qos/policies/{policy_id}/minimum_packet_rate_rules sends provided attrs (null and unknown attrs kept) under minimum_packet_rate_rule, binds the escaped literal policy, projects the response with the fixed QoSPolicyID, rejects qos_policy_id before HTTP and reports rejections with operation context.
  Tests: `network/v2/extensions/qos/rules/python_packet_rate_mutations_test.go:TestPythonMinimumPacketRateCreateUpdateDelete`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.
- Go takes the QoS policy ID first and the rule ID second; Python update/delete take the rule first. The policy is never fetched, as Python _get_resource only extracts its ID.

### `python:network/v2/update_qos_minimum_packet_rate_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.UpdateMinimumPacketRateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateRuleMutationOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateRuleField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateRuleHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateRuleMicroversion`

- PUT /qos/policies/{policy_id}/minimum_packet_rate_rules/{rule_id} sends only changed attrs, ignores if_revision without If-Match, the empty update sends nothing, id is rejected before HTTP and rejections keep operation context.
  Tests: `network/v2/extensions/qos/rules/python_packet_rate_mutations_test.go:TestPythonMinimumPacketRateCreateUpdateDelete`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.
- Go takes the QoS policy ID first and the rule ID second; Python update/delete take the rule first. The policy is never fetched, as Python _get_resource only extracts its ID.

### `python:network/v2/delete_qos_minimum_packet_rate_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.DeleteMinimumPacketRateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateRuleDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateRuleDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateRuleDeleteMicroversion`

- Bodyless DELETE /qos/policies/{policy_id}/minimum_packet_rate_rules/{rule_id} ignores a clean 404 by default (IgnoredMissing), WithPacketRateRuleDeleteIgnoreMissing(false) or other rejections return the native status with operation context.
  Tests: `network/v2/extensions/qos/rules/python_packet_rate_mutations_test.go:TestPythonMinimumPacketRateCreateUpdateDelete`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.
- Go takes the QoS policy ID first and the rule ID second; Python update/delete take the rule first. The policy is never fetched, as Python _get_resource only extracts its ID.
- Python returns None; Go returns a PacketRateRuleDeleteResult with the actual receipt and IgnoredMissing.

### `python:network/v2/create_qos_packet_rate_limit_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.CreatePacketRateLimitRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateRuleMutationOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateRuleField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateRuleHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateRuleMicroversion`

- POST /qos/policies/{policy_id}/packet_rate_limit_rules sends provided attrs (null and unknown attrs kept) under packet_rate_limit_rule, binds the escaped literal policy, projects the response with the fixed QoSPolicyID, rejects qos_policy_id before HTTP and reports rejections with operation context.
  Tests: `network/v2/extensions/qos/rules/python_packet_rate_mutations_test.go:TestPythonPacketRateLimitCreateUpdateDelete`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.
- Go takes the QoS policy ID first and the rule ID second; Python update/delete take the rule first. The policy is never fetched, as Python _get_resource only extracts its ID.

### `python:network/v2/update_qos_packet_rate_limit_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.UpdatePacketRateLimitRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateRuleMutationOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateRuleField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateRuleHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateRuleMicroversion`

- PUT /qos/policies/{policy_id}/packet_rate_limit_rules/{rule_id} sends only changed attrs, ignores if_revision without If-Match, the empty update sends nothing, id is rejected before HTTP and rejections keep operation context.
  Tests: `network/v2/extensions/qos/rules/python_packet_rate_mutations_test.go:TestPythonPacketRateLimitCreateUpdateDelete`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.
- Go takes the QoS policy ID first and the rule ID second; Python update/delete take the rule first. The policy is never fetched, as Python _get_resource only extracts its ID.

### `python:network/v2/delete_qos_packet_rate_limit_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.DeletePacketRateLimitRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateRuleDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateRuleDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithPacketRateRuleDeleteMicroversion`

- Bodyless DELETE /qos/policies/{policy_id}/packet_rate_limit_rules/{rule_id} ignores a clean 404 by default (IgnoredMissing), WithPacketRateRuleDeleteIgnoreMissing(false) or other rejections return the native status with operation context.
  Tests: `network/v2/extensions/qos/rules/python_packet_rate_mutations_test.go:TestPythonPacketRateLimitCreateUpdateDelete`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.
- Go takes the QoS policy ID first and the rule ID second; Python update/delete take the rule first. The policy is never fetched, as Python _get_resource only extracts its ID.
- Python returns None; Go returns a PacketRateRuleDeleteResult with the actual receipt and IgnoredMissing.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

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

## References

- [QoS Minimum Packet Rate Records Python](../../../qos-minimum-packet-rate-records-python.md)
- [QoS Packet Rate Limit Records Python](../../../qos-packet-rate-limit-records-python.md)

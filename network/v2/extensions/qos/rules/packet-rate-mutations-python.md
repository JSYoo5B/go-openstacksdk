# Packet Rate Mutations Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

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
    request.Field("min_kpps", 1000),
    request.Field("direction", "any"))
if err != nil {
    return err
}
var ruleID string
_ = json.Unmarshal(rule.Resource.Body["id"], &ruleID)
if _, err := network.QoSRules.UpdateMinimumPacketRateRecord(ctx, policyID, ruleID,
    request.Field("min_kpps", 2000)); err != nil {
    return err
}
_, err = network.QoSRules.DeleteMinimumPacketRateRecord(ctx, policyID, ruleID)
return err
```

## References

- [QoS Minimum Packet Rate Records Python](../../../qos-minimum-packet-rate-records-python.md)
- [QoS Packet Rate Limit Records Python](../../../qos-packet-rate-limit-records-python.md)

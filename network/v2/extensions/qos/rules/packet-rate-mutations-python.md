# Packet Rate Mutations Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
rule = conn.network.create_qos_minimum_packet_rate_rule(
    policy, min_kpps=1000, direction="any")
conn.network.update_qos_minimum_packet_rate_rule(rule, policy, min_kpps=2000)
conn.network.delete_qos_minimum_packet_rate_rule(rule, policy)
```

See `Example_minimumPacketRate` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.

## References

- [QoS Minimum Packet Rate Records Python](../../../qos-minimum-packet-rate-records-python.md)
- [QoS Packet Rate Limit Records Python](../../../qos-packet-rate-limit-records-python.md)

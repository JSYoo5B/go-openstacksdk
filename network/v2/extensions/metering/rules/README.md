# Rules

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
rule = conn.network.create_metering_label_rule(
    metering_label_id=label.id, direction="egress",
    destination_ip_prefix="0.0.0.0/0", is_excluded=False)
for item in conn.network.metering_label_rules(metering_label_id=label.id):
    print(item.direction, item.destination_ip_prefix)
conn.network.delete_metering_label_rule(rule)
```

```go
api := rules.New(networkClient)
rule, err := api.Create(ctx,
    request.Field("metering_label_id", labelID),
    request.Field("direction", "egress"),
    request.Field("destination_ip_prefix", "0.0.0.0/0"),
    request.Field("is_excluded", false))
if err != nil { return err }
for item, err := range api.List(ctx, request.Query("metering_label_id", labelID)) {
    if err != nil { return err }
    fmt.Println(*item.Direction, *item.DestinationIPPrefix)
}
_, err = api.Delete(ctx, rule.ID)
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L3005-L3157)
- [`metering_label_rule.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/metering_label_rule.py)
- [`labels`](../labels/README.md)

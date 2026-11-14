# Defaultrules

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
rule = conn.network.create_default_security_group_rule(
    direction="ingress", ether_type="IPv4", protocol="tcp",
    port_range_min=22, port_range_max=22, used_in_default_sg=True)
for item in conn.network.default_security_group_rules(ether_type="IPv4"):
    print(item.id, item.port_range_min)
conn.network.delete_default_security_group_rule(rule)
```

```go
api := defaultrules.New(networkClient)
rule, err := api.Create(ctx,
    request.Field("direction", "ingress"),
    request.Field("ether_type", "IPv4"),
    request.Field("protocol", "tcp"),
    request.Field("port_range_min", 22),
    request.Field("port_range_max", 22),
    request.Field("used_in_default_sg", true))
if err != nil {
    return err
}
for item, err := range api.List(ctx, request.Query("ether_type", "IPv4")) {
    if err != nil {
        return err
    }
    fmt.Println(item.Text("id"), string(item.Resource.Body["port_range_min"]))
}
_, err = api.Delete(ctx, rule.Text("id"))
return err
```

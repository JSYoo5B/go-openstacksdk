# Subnets

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
subnets = conn.network.subnets(
    network_id="network-a",
    is_dhcp_enabled=False,
    prefix_length=24,
    dns_nameservers=["192.0.2.53"],
    max_items=100,
)
for subnet in subnets:
    print(subnet.id, subnet.cidr)
```

See `example_subnets_test.go` in [example_subnets_test.go](example_subnets_test.go), compiled by `go test`.

## References

- [Python Raw Max Items](python-raw-max-items.md)
- [AST manifest](../../../api/openstacksdk/resources/network/v2/subnet.json)
- [SDKgen](../../../internal/cmd/sdkgen/README.md)
- [Listing](../../../docs/listing.md)
- [Finding Identities](../../../docs/finding-identities.md)
- [Network Subnet Semantic Filters Test](../../../api/network_subnet_semantic_filters_test.go)
- [Network Subnet Raw Body Filters Test](../../../api/network_subnet_raw_body_filters_test.go)

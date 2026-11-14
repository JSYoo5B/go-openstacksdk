# Networksegmentranges

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
rng = conn.network.create_network_segment_range(
    name="vlan-100", network_type="vlan", physical_network="physnet1",
    minimum=100, maximum=199, shared=True)
conn.network.update_network_segment_range(rng, maximum=299)
for item in conn.network.network_segment_ranges(network_type="vlan"):
    print(item.name, item.available)
conn.network.delete_network_segment_range(rng)
```

See `Example` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.

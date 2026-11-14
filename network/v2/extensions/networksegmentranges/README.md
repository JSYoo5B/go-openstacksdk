# Networksegmentranges

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
rng = conn.network.create_network_segment_range(
    name="vlan-100", network_type="vlan", physical_network="physnet1",
    minimum=100, maximum=199, shared=True)
conn.network.update_network_segment_range(rng, maximum=299)
for item in conn.network.network_segment_ranges(network_type="vlan"):
    print(item.name, item.available)
conn.network.delete_network_segment_range(rng)
```

```go
api := networksegmentranges.New(networkClient)
rng, err := api.Create(ctx,
    request.Field("name", "vlan-100"),
    request.Field("network_type", "vlan"),
    request.Field("physical_network", "physnet1"),
    request.Field("minimum", 100),
    request.Field("maximum", 199),
    request.Field("shared", true))
if err != nil {
    return err
}
id := rng.Text("id")
if _, err := api.Update(ctx, id, request.Field("maximum", 299)); err != nil {
    return err
}
for item, err := range api.List(ctx, request.Query("network_type", "vlan")) {
    if err != nil {
        return err
    }
    fmt.Println(item.Text("name"), string(item.Resource.Body["available"]))
}
_, err = api.Delete(ctx, id)
return err
```

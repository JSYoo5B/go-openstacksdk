# Labels

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
label = conn.network.create_metering_label(name="web", is_shared=False)
for item in conn.network.metering_labels(project_id=project_id):
    print(item.name, item.is_shared)
conn.network.delete_metering_label(label)
```

```go
api := labels.New(networkClient)
label, err := api.Create(ctx,
    request.Field("name", "web"),
    request.Field("is_shared", false))
if err != nil { return err }
for item, err := range api.List(ctx, request.Query("project_id", projectID)) {
    if err != nil { return err }
    fmt.Println(*item.Name, item.IsShared != nil && *item.IsShared)
}
_, err = api.Delete(ctx, label.ID)
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L2861-L3002)
- [`metering_label.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/metering_label.py)
- [`rules`](../rules/README.md)

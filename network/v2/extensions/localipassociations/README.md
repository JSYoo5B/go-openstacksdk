# Localipassociations

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
assoc = conn.network.create_local_ip_association(lip, fixed_port_id=port.id)
for item in conn.network.local_ip_associations(lip, host="compute-1"):
    print(item.fixed_ip)
conn.network.delete_local_ip_association(lip, port.id)
```

```go
api := localipassociations.New(networkClient)
_, err := api.Create(ctx, lipID, request.Field("fixed_port_id", portID))
if err != nil { return err }
for item, err := range api.List(ctx, lipID, request.Query("host", "compute-1")) {
    if err != nil { return err }
    fmt.Println(*item.FixedIP)
}
err = api.Delete(ctx, lipID, portID)
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L1943-L2109)
- [`local_ip_association.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/local_ip_association.py)
- [`localips`](../localips)

# Python Parity

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
err := secrets.New(client).Update(ctx, "s1", secrets.UpdateOpts{ContentType: "text/plain", Payload: "new"})
```

```go
order, err := orders.New(client).WaitFor(ctx, resource.ID("o1"), "ACTIVE",
	resource.WithUnlimitedWait(),          // Python wait=None
	resource.WithFailureStates("ERROR"),  // Python failures=None
)
err = orders.New(client).WaitForDeletion(ctx, resource.ID("o1"),
	resource.WithTimeout(120*time.Second),
)
```

## References

- [key manager v1 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py)
- [Metadata Fetch](metadata-fetch.md)
- [Metadata Create](metadata-create.md)
- [Metadata Delete](metadata-delete.md)
- [`update_container`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L142)
- [Python Update Record](containers/python-update-record.md)
- [`update_order`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L243)
- [Python Update Record](orders/python-update-record.md)
- [`update_secret`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L345)
- [Python Update Record](secrets/python-update-record.md)
- [`wait_for_status`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L615)
- [`wait_for_delete`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L654)
- [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881)
- [resource.wait_for_status](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2591)
- [resource.wait_for_delete](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2670)
- [Container](containers/python-update-record.md)
- [Order](orders/python-update-record.md)

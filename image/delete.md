# Delete

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_delete1_test.go` in [example_delete1_test.go](example_delete1_test.go), compiled by `go test`.

```python
conn.image.delete_image(image_id)
```

See `example_delete2_test.go` in [example_delete2_test.go](example_delete2_test.go), compiled by `go test`.

```python
conn.image.delete_image(image_id, store=store_id, ignore_missing=False)
```

## References

- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#delete-image-from-store)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#delete-image)
- [Proxy.delete_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L970-L997)
- [Store.delete_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/service_info.py#L46-L75)
- [Delete Contracts Test](delete_contracts_test.go)
- [Delete Core Test](delete_core_test.go)
- [Delete Options Test](delete_options_test.go)
- [Glance Delete Test](../internal/cmd/sdkgen/glance_delete_test.go)

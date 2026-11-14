# Serviceinfo

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_serviceinfo1_test.go` in [example_serviceinfo1_test.go](example_serviceinfo1_test.go), compiled by `go test`.

```python
for store in conn.image.stores(details=True, max_items=20):
    print(store.id, store.description, store.is_default)
```

See `example_serviceinfo2_test.go` in [example_serviceinfo2_test.go](example_serviceinfo2_test.go), compiled by `go test`.

```python
info = conn.image.get_import_info()
methods = info.import_methods
print(methods)
```

## References

- [Records](records.md)
- [Python Store Details](python-store-details.md)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#list-stores)
- [detail API](https://docs.openstack.org/api-ref/image/v2/index.html#list-stores-detail)
- [basic controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/discovery.py#L53-L93)
- [detail controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/discovery.py#L165-L195)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#import-methods-and-values-discovery)
- [Proxy.stores/get_import_info](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L2273-L2295)
- [Store/Import](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/service_info.py#L22-L44)
- [Contracts Test](contracts_test.go)
- [Core Test](core_test.go)
- [Options Test](options_test.go)
- [Connection Image Serviceinfo Test](../../../connectiontest/connection_image_serviceinfo_test.go)
- [Glance Serviceinfo Test](../../../internal/cmd/sdkgen/glance_serviceinfo_test.go)
- [Response Validation Test](../../../internal/rest/response_validation_test.go)
- [Usage](usage.md)

# Image Tasks

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_image_tasks_test.go` in [example_image_tasks_test.go](example_image_tasks_test.go), compiled by `go test`.

```python
def image_tasks_example(conn, image_id):
    return list(conn.image.image_tasks(image_id))
```

## References

- [Tasks](tasks.md)
- [Images Images V2](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/api-ref/source/v2/images-images-v2.inc#L311-L350)
- [ImageTasks Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image_tasks.py#L20-L61)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1181-L1194)
- [Controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L599-L614)
- [API](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/db/sqlalchemy/api.py#L1834-L1873)
- [SQL formatter](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/db/sqlalchemy/api.py#L2018-L2043)
- [Image Tasks Core Test](image_tasks_core_test.go)
- [Image Tasks Options Test](image_tasks_options_test.go)
- [Image Tasks Contracts Test](image_tasks_contracts_test.go)
- [Connection Image Tasks Associated Test](../connectiontest/connection_image_tasks_associated_test.go)
- [Glance Image Tasks Test](../internal/cmd/sdkgen/glance_image_tasks_test.go)

# Tasks

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_tasks_test.go` in [example_tasks_test.go](example_tasks_test.go), compiled by `go test`.

```python
def task_examples(conn):
    created = conn.image.create_task(type="import", input={
        "import_from": "https://example.invalid/image.qcow2",
        "import_from_format": "qcow2",
        "image_properties": {"container_format": "bare", "disk_format": "qcow2"},
    })
    fetched = conn.image.get_task(created.id)
    rows = list(conn.image.tasks(type="import", status="pending", limit=20))
    owned = list(conn.image.tasks(type="import", limit=20, owner_id="project-id",
                                  input={"import_from_format": "qcow2"}))
    return created, fetched, rows, owned
```

## References

- [Tasks](v2/tasks/README.md)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#tasks)
- [Tasks](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/tasks.py#L70-L155)
- [Tasks](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/tasks.py#L354-L430)
- [`WithCreateTaskNullInput`](python-create-task.md)
- [Task Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/task.py#L16-L53)
- [fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1778-L1842)
- [Tasks](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/tasks.py#L308-L351)
- [native v2.15.0 ListOpts](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/tasks/requests.go#L36-L72)
- [Tasks Core Test](tasks_core_test.go)
- [Tasks Options Test](tasks_options_test.go)
- [Tasks Contracts Test](tasks_contracts_test.go)
- [Connection Image Tasks Test](../connectiontest/connection_image_tasks_test.go)
- [Glance Tasks Test](../internal/cmd/sdkgen/glance_tasks_test.go)

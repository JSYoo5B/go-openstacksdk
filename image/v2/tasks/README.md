# Tasks

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_tasks1_test.go` in [example_tasks1_test.go](example_tasks1_test.go), compiled by `go test`.

```python
task = conn.image.wait_for_task(task)
```

See `example_tasks2_test.go` in [example_tasks2_test.go](example_tasks2_test.go), compiled by `go test`.

```python
task = conn.image.wait_for_task(
    task, status="processing", failures=[], interval=0.5, wait=60)
```

## References

- [`wait_for_task`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L2198-L2269)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#tasks)
- [Service Waits](../../../docs/service-waits.md)
- [Glance Task Wait Contracts Test](../../../api/glance_task_wait_contracts_test.go)
- [Wait Options Test](wait_options_test.go)
- [Wait Results Test](wait_results_test.go)
- [Glance Task Wait Test](../../../internal/cmd/sdkgen/glance_task_wait_test.go)

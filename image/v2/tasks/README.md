# Tasks

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/tasks"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func waitForTask(ctx context.Context, client *gophercloud.ServiceClient, id string) (*tasks.TaskWaitResult, error) {
    return tasks.New(client).WaitForTask(ctx, resource.ID(id))
}
```

```python
task = conn.image.wait_for_task(task)
```

```go
package example

import (
    "context"
    "time"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/tasks"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func waitForProcessing(ctx context.Context, client *gophercloud.ServiceClient, id string) (*tasks.TaskWaitResult, error) {
    policy := tasks.WithTaskWait(
        resource.WithTimeout(time.Minute),
        resource.WithPollInterval(500*time.Millisecond),
        resource.WithFailureStates(),
    )
    return tasks.New(client).WaitForTaskState(ctx, resource.ID(id), "processing", policy)
}
```

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

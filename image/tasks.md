# Tasks

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package examples

import (
    "context"
    "encoding/json"

    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func createTask(ctx context.Context, svc *image.Service) (*image.TaskInfo, error) {
    return svc.CreateTask(ctx, "import",
        image.WithCreateTaskOpts(image.CreateTaskOpts{
            Input: map[string]json.RawMessage{"import_from_format": json.RawMessage(`"qcow2"`)},
        }),
        request.Header("X-Request-Id", "task-create-example"),
        image.WithCreateTaskHeaders(map[string]string{"X-Example": "create"}),
        image.WithCreateTaskInput(map[string]any{
            "import_from": "https://example.invalid/image.qcow2",
            "import_from_format": "qcow2",
            "image_properties": map[string]any{
                "container_format": "bare", "disk_format": "qcow2",
            },
        }))
}

func getTask(ctx context.Context, svc *image.Service, taskID string) (*image.TaskInfo, error) {
    return svc.GetTask(ctx, taskID,
        image.WithGetTaskOpts(image.GetTaskOpts{}),
        request.Header("X-Request-Id", "task-get-example"),
        image.WithGetTaskHeaders(map[string]string{"X-Example": "get"}))
}

func listTasks(ctx context.Context, svc *image.Service, marker string) ([]*image.TaskInfo, error) {
    options := []image.ListTasksOption{
        image.WithListTasksOpts(image.ListTasksOpts{}),
        request.Header("X-Request-Id", "task-list-example"),
        image.WithListTasksHeaders(map[string]string{"X-Example": "list"}),
        request.Limit(20),
        request.Marker(marker),
        request.Custom(func(c *image.ListTasksOpts) error { c.Type = "import"; return nil }),
        request.Custom(func(c *image.ListTasksOpts) error { c.Status = "pending"; return nil }),
        request.Custom(func(c *image.ListTasksOpts) error { c.SortKey = "created_at"; return nil }),
        request.Custom(func(c *image.ListTasksOpts) error { c.SortDir = "desc"; return nil }),
        request.MaxItems(40),
        request.Custom(func(c *image.ListTasksOpts) error { c.SinglePage = false; return nil }),
    }
    rows := make([]*image.TaskInfo, 0)
    for task, err := range svc.Tasks(ctx, options...) {
        if err != nil {
            return nil, err
        }
        rows = append(rows, task)
    }
    return rows, nil
}

func allTasks(ctx context.Context, svc *image.Service) ([]*image.TaskInfo, error) {
    return svc.AllTasks(ctx, request.MaxItems(40))
}

// pythonLikeTasks opts in to the pinned Python Resource.list behaviors:
// local Body filters (owner_id compares the response owner field), an RFC 8288
// Link header when the body has no next field, and limit/marker fallback that
// repeats the collection with marker=<last raw row id> until an empty page.
// MaxItems counts raw rows before filtering, as Python's max_items does.
func pythonLikeTasks(ctx context.Context, svc *image.Service, owner string) ([]*image.TaskInfo, error) {
    return svc.AllTasks(ctx,
        request.Custom(func(c *image.ListTasksOpts) error { c.Type = "import"; return nil }),
        request.Limit(20),
        image.WithListTasksBodyFilter("owner_id", owner),
        image.WithListTasksBodyFilter("input", map[string]any{"import_from_format": "qcow2"}),
        image.WithListTasksLinkHeader(true),
        request.Custom(func(c *image.ListTasksOpts) error { c.MarkerFallback = true; return nil }))
}
```

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

package image_test

import (
	"context"
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func createTaskTasks(ctx context.Context, svc *image.Service) (*image.TaskInfo, error) {
	return svc.CreateTask(ctx, "import",
		image.WithCreateTaskOpts(image.CreateTaskOpts{
			Input: map[string]json.RawMessage{"import_from_format": json.RawMessage(`"qcow2"`)},
		}),
		request.Header("X-Request-Id", "task-create-example"),
		image.WithCreateTaskHeaders(map[string]string{"X-Example": "create"}),
		image.WithCreateTaskInput(map[string]any{
			"import_from":        "https://example.invalid/image.qcow2",
			"import_from_format": "qcow2",
			"image_properties": map[string]any{
				"container_format": "bare", "disk_format": "qcow2",
			},
		}))
}

func getTaskTasks(ctx context.Context, svc *image.Service, taskID string) (*image.TaskInfo, error) {
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

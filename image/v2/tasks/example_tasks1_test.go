package tasks_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/image/v2/tasks"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func waitForTaskTasks1(ctx context.Context, client *gophercloud.ServiceClient, id string) (*tasks.TaskWaitResult, error) {
	return tasks.New(client).WaitForTask(ctx, resource.ID(id))
}

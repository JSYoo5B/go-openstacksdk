package tasks_test

import (
	"context"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/image/v2/tasks"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func waitForProcessingTasks2(ctx context.Context, client *gophercloud.ServiceClient, id string) (*tasks.TaskWaitResult, error) {
	policy := tasks.WithTaskWait(
		resource.WithTimeout(time.Minute),
		resource.WithPollInterval(500*time.Millisecond),
		resource.WithFailureStates(),
	)
	return tasks.New(client).WaitForTaskState(ctx, resource.ID(id), "processing", policy)
}

package image_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func imageTaskExamplesImageTasks(ctx context.Context, svc *image.Service, imageID string) ([]*image.ImageTaskInfo, error) {
	options := []image.ListImageTasksOption{
		image.WithListImageTasksOpts(image.ListImageTasksOpts{MaxItems: 0}),
		request.Header("X-Request-Id", "image-tasks-example"),
		image.WithListImageTasksHeaders(map[string]string{"X-Example": "tasks"}),
		request.MaxItems(20),
	}
	rows := make([]*image.ImageTaskInfo, 0)
	for task, err := range svc.ImageTasks(ctx, resource.ID(imageID), options...) {
		if err != nil {
			return nil, err
		}
		rows = append(rows, task)
	}
	return rows, nil
}

func allImageTaskExamplesImageTasks(ctx context.Context, svc *image.Service, imageID string) ([]*image.ImageTaskInfo, error) {
	return svc.AllImageTasks(ctx, resource.ID(imageID), request.MaxItems(20))
}

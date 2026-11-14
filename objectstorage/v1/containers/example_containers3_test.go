package containers_test

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func containerLifecycleContainers3(ctx context.Context, api *containers.API, name string) error {
	createOptions := []containers.CreateContainerOption{
		containers.WithCreateContainerOpts(containers.CreateContainerOpts{
			Headers:  map[string]string{"X-Request-Label": "initial"},
			Metadata: map[string]string{"owner": "initial"},
		}),
		containers.WithCreateContainerHeaders(map[string]string{"X-Trace-Label": "container-create"}),
		request.Header("X-Request-Label", "create"),
		containers.WithCreateContainerMetadata(map[string]string{
			"owner": "ops", "note": "literal %2F 雪",
		}),
	}
	created, err := api.CreateContainer(ctx, name, createOptions...)
	if created != nil {
		fmt.Println(created.StatusCode)
	}
	if err != nil {
		return err
	}

	deleteOptions := []containers.DeleteContainerOption{
		containers.WithDeleteContainerOpts(containers.DeleteContainerOpts{
			Headers: map[string]string{"X-Request-Label": "initial"},
		}),
		containers.WithDeleteContainerHeaders(map[string]string{"X-Trace-Label": "container-delete"}),
		request.Header("X-Request-Label", "delete"),
		request.IgnoreMissing(false),
	}
	deleted, err := api.DeleteContainer(ctx, name, deleteOptions...)
	if deleted != nil {
		fmt.Println(deleted.StatusCode, deleted.IgnoredMissing)
	}
	if err != nil {
		return err
	}

	quietOptions := append([]containers.DeleteContainerOption(nil), deleteOptions...)
	quietOptions = append(quietOptions, request.IgnoreMissing(true))
	missing, err := api.DeleteContainer(ctx, name, quietOptions...)
	if missing != nil {
		fmt.Println(missing.StatusCode, missing.IgnoredMissing)
	}
	return err
}

func Example_containers3() {}

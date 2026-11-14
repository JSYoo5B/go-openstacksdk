package containers_test

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func changeContainerMetadataContainers2(ctx context.Context, api *containers.API, name string) error {
	getOptions := []containers.GetMetadataOption{
		containers.WithGetMetadataOpts(containers.GetMetadataOpts{
			Headers: map[string]string{"X-Request-Label": "initial"},
		}),
		containers.WithGetMetadataHeaders(map[string]string{"X-Trace-Label": "metadata"}),
		request.Header("X-Request-Label", "container-metadata"),
		request.Custom(func(c *containers.GetMetadataOpts) error { v := false; c.Newest = &v; return nil }),
	}
	before, err := api.GetMetadata(ctx, name, getOptions...)
	if before != nil {
		fmt.Println(before.StatusCode)
	}
	if err != nil {
		return err
	}
	if before.Metadata.ObjectCount != nil {
		fmt.Println(*before.Metadata.ObjectCount)
	}

	mutationOptions := []containers.MetadataOption{
		containers.WithMetadataOpts(containers.MetadataOpts{
			Headers: map[string]string{"X-Request-Label": "initial"},
		}),
		containers.WithMetadataHeaders(map[string]string{"X-Trace-Label": "metadata"}),
		request.Header("X-Request-Label", "metadata-mutation"),
	}
	acknowledgement, err := api.SetMetadata(ctx, name, map[string]string{
		"book": "Moby Dick", "note": "literal %2F 雪",
	}, mutationOptions...)
	if acknowledgement != nil {
		fmt.Println(acknowledgement.StatusCode)
	}
	if err != nil {
		return err
	}

	refreshOptions := append([]containers.GetMetadataOption(nil), getOptions...)
	refreshOptions = append(refreshOptions, request.Custom(func(c *containers.GetMetadataOpts) error { c.Newest = nil; return nil }))
	refreshed, err := api.GetMetadata(ctx, name, refreshOptions...)
	if err != nil {
		return err
	}
	fmt.Println(refreshed.Metadata.Values["book"])

	removed, err := api.DeleteMetadata(ctx, name, []string{"note"}, mutationOptions...)
	if removed != nil {
		fmt.Println(removed.StatusCode)
	}
	return err
}

func Example_containers2() {}

package objects_test

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func ChangeObjectMetadataObjects2(ctx context.Context, conn *sdk.Connection, newest bool) error {
	service, err := conn.ObjectStorage(ctx)
	if err != nil {
		return err
	}
	container, key := "assets", "docs/readme ?.txt"
	getOptions := []objects.GetMetadataOption{
		objects.WithGetMetadataOpts(objects.GetMetadataOpts{
			Headers: map[string]string{"X-Trans-Id-Extra": "sdk-read"},
		}),
		request.Header("X-Example", "read"),
		objects.WithGetMetadataHeaders(map[string]string{"X-Example-Trace": "metadata"}),
	}
	if newest {
		getOptions = append(getOptions, request.Custom(func(c *objects.GetMetadataOpts) error { v := true; c.Newest = &v; return nil }))
	} else {
		getOptions = append(getOptions, request.Custom(func(c *objects.GetMetadataOpts) error { c.Newest = nil; return nil }))
	}
	before, err := service.Objects.GetMetadata(ctx, container, key, getOptions...)
	if err != nil {
		if before != nil {
			fmt.Println("HEAD", before.StatusCode)
		}
		return err
	}
	if before.Metadata.ContentLength != nil {
		fmt.Println("object bytes", *before.Metadata.ContentLength)
	}
	mutationOptions := []objects.MetadataOption{
		objects.WithMetadataOpts(objects.MetadataOpts{
			Headers: map[string]string{"X-Trans-Id-Extra": "sdk-write"},
		}),
		request.Header("Content-Disposition", "attachment"),
		objects.WithMetadataHeaders(map[string]string{"Cache-Control": "private"}),
	}
	change, err := service.Objects.SetMetadata(ctx, container, key,
		map[string]string{"owner": "sdk", "empty": ""}, mutationOptions...)
	printObjectMetadataPhasesObjects2(change)
	if err != nil {
		return err
	}
	change, err = service.Objects.DeleteMetadata(ctx, container, key,
		[]string{"owner"}, mutationOptions...)
	printObjectMetadataPhasesObjects2(change)
	if err != nil {
		return err
	}
	refreshed, err := service.Objects.GetMetadata(ctx, container, key, getOptions...)
	if err != nil {
		return err
	}
	fmt.Println("stored empty value", refreshed.Metadata.Values["empty"])
	return nil
}

func printObjectMetadataPhasesObjects2(change *objects.MetadataChangeResult) {
	if change == nil {
		return
	}
	if change.Before != nil {
		fmt.Println("before HEAD", change.Before.StatusCode)
	}
	if change.Acknowledgement != nil {
		fmt.Println("POST", change.Acknowledgement.StatusCode)
	}
}

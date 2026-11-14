package images_test

import (
	"context"
	"fmt"
	"log"

	"errors"
	openstack "github.com/JSYoo5B/go-openstacksdk"
	image "github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/image/v2/images"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

func imageService(ctx context.Context) *image.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Image(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func ExampleAPI_Create() {
	ctx := context.Background()
	visibility := images.ImageVisibilityPrivate
	created, err := imageService(ctx).API.Images.Create(ctx, images.CreateOpts{
		Name: "ubuntu-24.04", DiskFormat: "qcow2", ContainerFormat: "bare",
		Visibility: &visibility, Tags: []string{"base"},
		Properties: map[string]string{"os_distro": "ubuntu"},
	}, request.Field("hw_qemu_guest_agent", "yes"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(created.ID, created.Status)
}

func reportNative(err error) {
	var native gophercloud.ErrUnexpectedResponseCode
	if errors.As(err, &native) {
		fmt.Printf("HTTP %d, expected=%v, response bytes=%d\n", native.Actual, native.Expected, len(native.Body))
	}
	log.Fatal(err)
}

func ExampleAPI_Get() {
	ctx := context.Background()
	got, err := imageService(ctx).API.Images.Get(ctx, "image-id")
	if err != nil {
		reportNative(err)
	}
	fmt.Println(got.ID, got.Status, got.SizeBytes, got.Properties, got.OpenStackImageImportMethods)
}

func ExampleAPI_Delete() {
	ctx := context.Background()
	if err := imageService(ctx).API.Images.Delete(ctx, "image-id"); err != nil {
		reportNative(err)
	}
}

func ExampleAPI_List() {
	ctx := context.Background()
	for found, err := range imageService(ctx).API.Images.List(ctx,
		images.WithListOptions(images.ListOpts{Status: images.ImageStatusActive, Limit: 50}),
		request.Query("size_min", "1073741824")) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(found.ID, found.Name, found.SizeBytes)
	}
}

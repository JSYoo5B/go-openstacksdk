package image_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_imageFragment1() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var service *svc.Service
	service, err = conn.Image(ctx)
	if err != nil {
		log.Fatal(err)
	}
	image, err := service.Images.Find(ctx, resource.Name("ubuntu"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(image.ID, image.SizeBytes)

	for image, err := range service.Images.List(ctx, resource.WithStatus("active")) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(image.ID, image.Name)
	}
}

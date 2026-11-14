package blockstorage_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_blockstorageFragment1() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var service *svc.Service
	service, err = conn.BlockStorage(ctx)
	if err != nil {
		log.Fatal(err)
	}
	volume, err := service.Volumes.Find(ctx, resource.Name("data"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(volume.ID, volume.Size)

	for volume, err := range service.Volumes.List(ctx,
		resource.WithStatus("available"), resource.WithPageSize(100)) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(volume.ID, volume.Size)
	}
}

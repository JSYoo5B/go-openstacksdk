package blockstorage_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	blockstorage "github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"time"
)

func blockStorage(ctx context.Context) *blockstorage.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.BlockStorage(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func Example_waitAndDelete() {
	ctx := context.Background()
	service := blockStorage(ctx)
	ready, err := service.Volumes.Wait(ctx, resource.Name("data"), "available", resource.WithTimeout(5*time.Minute))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(ready.Status)
	if err := service.Volumes.Delete(ctx, resource.ID(ready.ID)); err != nil {
		log.Fatal(err)
	}
}

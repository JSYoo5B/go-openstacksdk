package nodes_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
)

func ExampleAPI_GetVirtualMedia() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx, openstack.WithCloud("dev"), openstack.WithMicroversion("baremetal", "1.93"))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.BareMetal(ctx)
	if err != nil {
		log.Fatal(err)
	}
	media, err := service.Nodes.GetVirtualMedia(ctx, "node-id")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(media.Image, media.Inserted, media.MediaTypes)
}

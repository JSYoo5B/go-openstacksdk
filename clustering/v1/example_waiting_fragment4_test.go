package v1_test

import (
	"context"
	"fmt"
	"log"
	"time"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_waitingFragment4() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var service *svc.Service
	service, err = conn.ClusteringV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	err = service.Nodes.WaitForDelete(ctx, resource.ID("NODE_ID"),
		resource.WithTimeout(10*time.Minute),
		resource.WithUnlimitedWait(),
		resource.WithPollInterval(time.Second),
		resource.WithProgressCallback(func(progress int) error { fmt.Println(progress); return nil }))
	if err != nil {
		log.Fatal(err)
	}
}

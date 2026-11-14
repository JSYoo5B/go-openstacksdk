package buildinfo_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/buildinfo"
)

func Example_buildinfoFragment1() {
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
	client := service.RawClient()
	info, err := buildinfo.New(client).Get(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if info.API != nil && info.API.Revision != nil {
		fmt.Println(*info.API.Revision)
	}
}

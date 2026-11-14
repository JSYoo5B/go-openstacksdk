package services_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/services"
)

func Example_list() {
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
	// Configure before sharing the client between goroutines.
	client.Type = "clustering"
	client.Microversion = "1.7"
	api := services.New(client)

	for service, err := range api.List(ctx) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(service.ID, service.Host, service.Status, service.State)
	}
}

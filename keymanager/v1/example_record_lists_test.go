package v1_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_recordLists() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	name := flag.String("name", "", "optional exact top-level name condition")
	maximum := flag.Int("max-items", 0, "raw row cap; zero is unbounded")
	firstPage := flag.Bool("first-page", false, "consume only the first page")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.KeyManagerV1(ctx)
	if err != nil {
		log.Fatal(err)
	}

	containerOptions := []containers.RecordListOption{
		request.MaxItems(*maximum),
		request.Paginated(!*firstPage),
	}
	orderOptions := []orders.RecordListOption{
		request.MaxItems(*maximum),
		request.Paginated(!*firstPage),
	}
	secretOptions := []secrets.RecordListOption{
		request.MaxItems(*maximum),
		request.Paginated(!*firstPage),
	}
	if *name != "" {
		containerOptions = append(containerOptions, request.Query("name", *name))
		orderOptions = append(orderOptions, request.Query("name", *name))
		secretOptions = append(secretOptions, request.Query("name", *name))
	}
	for value, err := range service.Containers.ListRecords(ctx, containerOptions...) {
		if err != nil {
			log.Fatal(err)
		}
		encoded, err := json.MarshalIndent(value.Resource, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("containers %s\n", encoded)
	}
	for value, err := range service.Orders.ListRecords(ctx, orderOptions...) {
		if err != nil {
			log.Fatal(err)
		}
		encoded, err := json.MarshalIndent(value.Resource, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("orders %s\n", encoded)
	}
	for value, err := range service.Secrets.ListRecords(ctx, secretOptions...) {
		if err != nil {
			log.Fatal(err)
		}
		encoded, err := json.MarshalIndent(value.Resource, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("secrets %s\n", encoded)
	}
}

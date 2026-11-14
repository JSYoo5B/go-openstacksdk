package serverexternalevents_test

import (
	"context"
	"log"
	"os"

	compute "github.com/JSYoo5B/go-openstacksdk/compute/v2"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/serverexternalevents"
	"github.com/gophercloud/gophercloud/v2"
)

func Example_serverexternalevents() {
	endpoint := os.Getenv("OS_COMPUTE_ENDPOINT")
	if endpoint == "" {
		log.Fatal("OS_COMPUTE_ENDPOINT is required")
	}
	provider := &gophercloud.ProviderClient{TokenID: os.Getenv("OS_TOKEN")}
	client := &gophercloud.ServiceClient{ProviderClient: provider, Endpoint: gophercloud.NormalizeURL(endpoint), Type: "compute"}
	service := compute.New(client)
	rows, err := service.ServerExternalEvents.CreateRecords(context.Background(), []serverexternalevents.Event{
		{"name": "network-changed", "server_uuid": "server-id", "status": "completed"},
	})
	if err != nil {
		log.Fatal(err)
	}
	for row, err := range rows {
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("name=%s code=%s status=%d", row.Resource.Body["name"], row.Resource.Body["code"], row.StatusCode)
	}
}

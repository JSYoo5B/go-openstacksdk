package quotas_test

import (
	"context"
	"log"
	"os"

	keymanager "github.com/JSYoo5B/go-openstacksdk/keymanager/v1"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func Example_projectRecords() {
	endpoint := os.Getenv("OS_KEY_MANAGER_ENDPOINT")
	if endpoint == "" {
		log.Fatal("OS_KEY_MANAGER_ENDPOINT is required")
	}
	provider := &gophercloud.ProviderClient{TokenID: os.Getenv("OS_TOKEN")}
	client := &gophercloud.ServiceClient{ProviderClient: provider, Endpoint: gophercloud.NormalizeURL(endpoint), Type: "key-manager"}
	service := keymanager.New(client)
	ctx := context.Background()
	scope, err := service.Quotas.InProject(ctx, resource.ID("project-id"))
	if err != nil {
		log.Fatal(err)
	}
	record, err := scope.UpdateRecord(ctx,
		quotas.WithRecordAttribute("secrets", 10),
		quotas.WithRecordAttribute("orders", 20),
	)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("sent=%t status=%d secrets=%s", record.Sent, record.StatusCode, record.Resource.Body["secrets"])
}

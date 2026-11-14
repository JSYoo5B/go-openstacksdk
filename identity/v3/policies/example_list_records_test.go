package policies_test

import (
	"context"
	"log"
	"os"

	identity "github.com/JSYoo5B/go-openstacksdk/identity/v3"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/policies"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

func Example_listRecords() {
	endpoint := os.Getenv("OS_IDENTITY_ENDPOINT")
	if endpoint == "" {
		log.Fatal("OS_IDENTITY_ENDPOINT is required")
	}
	provider := &gophercloud.ProviderClient{TokenID: os.Getenv("OS_TOKEN")}
	client := &gophercloud.ServiceClient{ProviderClient: provider, Endpoint: gophercloud.NormalizeURL(endpoint), Type: "identity"}
	service := identity.New(client)
	for row, err := range service.Policies.ListRecords(context.Background(),
		request.Limit(100),
		policies.WithRecordListFilter("project_id", "project-id"),
		policies.WithRecordListFilter("type", "application/json"),
	) {
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("id=%s blob=%s", row.Resource.Body["id"], row.Resource.Body["blob"])
	}
}

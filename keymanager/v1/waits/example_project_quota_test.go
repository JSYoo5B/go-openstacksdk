package waits_test

import (
	"context"
	"log"
	"os"
	"time"

	keymanager "github.com/JSYoo5B/go-openstacksdk/keymanager/v1"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func Example_projectQuota() {
	endpoint := os.Getenv("OS_KEY_MANAGER_ENDPOINT")
	if endpoint == "" {
		log.Fatal("OS_KEY_MANAGER_ENDPOINT is required")
	}
	provider := &gophercloud.ProviderClient{TokenID: os.Getenv("OS_TOKEN")}
	client := &gophercloud.ServiceClient{ProviderClient: provider, Endpoint: gophercloud.NormalizeURL(endpoint), Type: "key-manager"}
	service := keymanager.New(client)
	ctx := context.Background()
	projectID := "project-id"
	scope, err := service.Quotas.InProject(ctx, resource.ID(projectID))
	if err != nil {
		log.Fatal(err)
	}
	record, err := scope.GetRecord(ctx)
	if err != nil {
		log.Fatal(err)
	}
	target := waits.ForProjectQuota(record)
	current, err := service.Waits.WaitForStatus(ctx, target, &projectID,
		waits.WithWait(resource.WithStatusAttribute("id"), resource.WithTimeout(120*time.Second)))
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("id=%s status=%d", current.Resource.Body["id"], current.StatusCode)
	if os.Getenv("WAIT_ABSENCE") == "1" {
		if _, err := service.Waits.WaitForDelete(ctx, target, waits.WithWait(resource.WithTimeout(120*time.Second))); err != nil {
			log.Fatal(err)
		}
	}
}

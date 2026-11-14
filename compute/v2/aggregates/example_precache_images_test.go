package aggregates_test

import (
	"context"
	"log"
	"os"

	compute "github.com/JSYoo5B/go-openstacksdk/compute/v2"
	"github.com/gophercloud/gophercloud/v2"
)

func Example_precacheImages() {
	endpoint := os.Getenv("OS_COMPUTE_ENDPOINT")
	if endpoint == "" {
		log.Fatal("OS_COMPUTE_ENDPOINT is required")
	}
	provider := &gophercloud.ProviderClient{TokenID: os.Getenv("OS_TOKEN")}
	client := &gophercloud.ServiceClient{ProviderClient: provider, Endpoint: gophercloud.NormalizeURL(endpoint), Type: "compute"}
	service := compute.New(client)
	if err := service.Aggregates.PrecacheImages(context.Background(), 7, []string{"image-one", "image-two"}); err != nil {
		log.Fatal(err)
	}
}

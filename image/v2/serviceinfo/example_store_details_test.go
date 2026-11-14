package serviceinfo_test

import (
	"context"
	"log"
	"os"

	image "github.com/JSYoo5B/go-openstacksdk/image/v2"
	"github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

func Example_storeDetails() {
	endpoint := os.Getenv("OS_IMAGE_ENDPOINT")
	if endpoint == "" {
		log.Fatal("OS_IMAGE_ENDPOINT is required")
	}
	provider := &gophercloud.ProviderClient{TokenID: os.Getenv("OS_TOKEN")}
	client := &gophercloud.ServiceClient{ProviderClient: provider, Endpoint: gophercloud.NormalizeURL(endpoint), Type: "image"}
	service := image.New(client)
	for row, err := range service.ServiceInfo.ListStoreRecords(context.Background(),
		request.Custom(func(c *serviceinfo.StoreRecordListOpts) error { c.Details = true; return nil }),
		request.Paginated(false),
		serviceinfo.WithStoreRecordListFilter("is_default", true),
	) {
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("id=%s properties=%s status=%d", row.Resource.Body["id"], row.Resource.Body["properties"], row.StatusCode)
	}
}

package services_test

import (
	"context"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/services"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
	"os"
)

func Example_findRecord() {
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken(os.Getenv("OS_TOKEN"))
	client := &gophercloud.ServiceClient{ProviderClient: provider, Type: "compute", Endpoint: os.Getenv("OS_COMPUTE_ENDPOINT")}
	row, err := services.New(client).FindRecord(context.Background(), "nova-compute", request.Query("host", "node"), request.IgnoreMissing(false))
	if err != nil {
		panic(err)
	}
	fmt.Println(string(row.Resource.Body["id"]))
}

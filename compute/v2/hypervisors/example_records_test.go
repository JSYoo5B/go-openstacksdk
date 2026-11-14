package hypervisors_test

import (
	"context"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/hypervisors"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
	"os"
)

func Example_records() {
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken(os.Getenv("OS_TOKEN"))
	client := &gophercloud.ServiceClient{ProviderClient: provider, Type: "compute", Endpoint: os.Getenv("OS_COMPUTE_ENDPOINT")}
	api := hypervisors.New(client)
	rows, err := api.AllRecords(context.Background(), request.Query("with_servers", true))
	if err != nil {
		panic(err)
	}
	node, err := api.FindRecord(context.Background(), "node", request.IgnoreMissing(false))
	if err != nil {
		panic(err)
	}
	fmt.Println(len(rows), string(node.Resource.Body["name"]))
}

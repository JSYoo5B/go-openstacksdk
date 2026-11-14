package peers_test

import (
	"context"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
	"os"
)

func Example_records() {
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken(os.Getenv("OS_TOKEN"))
	client := &gophercloud.ServiceClient{ProviderClient: provider, Type: "network", Endpoint: os.Getenv("OS_NETWORK_ENDPOINT")}
	api := peers.New(client)
	rows, err := api.AllRecords(context.Background(), request.Query("name", "routing"), request.Query("project_id", "project"))
	if err != nil {
		panic(err)
	}
	row, err := api.FindRecord(context.Background(), "routing", request.IgnoreMissing(false))
	if err != nil {
		panic(err)
	}
	fmt.Println(len(rows), string(row.Resource.Body["id"]))
}

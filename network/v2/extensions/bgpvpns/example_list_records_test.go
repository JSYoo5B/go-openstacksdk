package bgpvpns_test

import (
	"context"
	"fmt"
	"os"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

func Example_listRecords() {
	provider := &gophercloud.ProviderClient{}
	provider.SetToken(os.Getenv("OS_TOKEN"))
	client := &gophercloud.ServiceClient{
		ProviderClient: provider, Type: "network",
		Endpoint: os.Getenv("OS_NETWORK_ENDPOINT"),
	}
	api := bgpvpns.New(client)
	for record, err := range api.ListRecords(context.Background(),
		request.Query("project_id", "project-a")) {
		if err != nil {
			panic(err)
		}
		fmt.Println(string(record.Resource.Body["id"]))
	}
	result, err := api.ProxyListRecords(context.Background(),
		bgpvpns.WithRecordListJMESPath("@"))
	if err != nil {
		panic(err)
	}
	fmt.Printf("%T\n", result.Value)
}

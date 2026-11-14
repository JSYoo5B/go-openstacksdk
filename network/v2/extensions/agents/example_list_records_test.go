package agents_test

import (
	"context"
	"fmt"
	"os"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents"
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
	api := agents.New(client)
	for record, err := range api.ListRecords(context.Background(),
		request.Query("host", "node-a")) {
		if err != nil {
			panic(err)
		}
		fmt.Println(string(record.Resource.Body["id"]))
	}
	result, err := api.ProxyListRecords(context.Background(),
		agents.WithRecordListJMESPath("@"))
	if err != nil {
		panic(err)
	}
	fmt.Printf("%T\n", result.Value)
}

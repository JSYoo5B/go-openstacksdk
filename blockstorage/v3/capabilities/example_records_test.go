package capabilities_test

import (
	"context"
	"fmt"
	"os"

	blockstorage "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3"
	"github.com/gophercloud/gophercloud/v2"
)

func Example_records() {
	provider := &gophercloud.ProviderClient{}
	provider.SetToken(os.Getenv("OS_TOKEN"))
	client := &gophercloud.ServiceClient{
		ProviderClient: provider, Type: "volumev3",
		Endpoint: os.Getenv("OS_VOLUME_ENDPOINT"),
	}
	sdk := blockstorage.New(client)
	result, err := sdk.Capabilities.GetRecord(context.Background(), "host@backend")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(result.Resource.Body["properties"]))
}

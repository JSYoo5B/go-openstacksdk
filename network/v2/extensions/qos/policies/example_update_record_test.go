package policies_test

import (
	"context"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
	"os"
)

func Example_updateRecord() {
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken(os.Getenv("OS_TOKEN"))
	client := &gophercloud.ServiceClient{ProviderClient: provider, Type: "network", Endpoint: os.Getenv("OS_NETWORK_ENDPOINT")}
	row, err := policies.New(client).UpdateRecord(context.Background(), "policy-id",
		request.Field("name", "gold"),
		request.Field("is_shared", true),
		request.Field("description", nil))
	if err != nil {
		panic(err)
	}
	fmt.Println(string(row.Resource.Body["id"]), string(row.Resource.Body["is_shared"]))
}

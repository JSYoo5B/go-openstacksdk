package quotasets_test

import (
	"context"
	"fmt"
	"os"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func Example_revert() {
	provider := &gophercloud.ProviderClient{}
	provider.SetToken(os.Getenv("OS_TOKEN"))
	client := &gophercloud.ServiceClient{
		ProviderClient: provider, Type: "volumev3",
		Endpoint: os.Getenv("OS_VOLUME_ENDPOINT"),
	}
	scope, err := quotasets.New(client).InProject(context.Background(), resource.ID("project-id"))
	if err != nil {
		panic(err)
	}
	result, err := scope.RevertRecord(context.Background(),
		request.Query("user_id", "user-id"),
		request.Query("force", false),
		request.Query("optional", nil))
	if err != nil {
		panic(err)
	}
	fmt.Println(result.ProjectID, result.StatusCode)
}

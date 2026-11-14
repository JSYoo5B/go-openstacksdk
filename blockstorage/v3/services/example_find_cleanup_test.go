package services_test

import (
	"context"
	"fmt"
	"os"

	blockstorage "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

func Example_findCleanup() {
	provider := &gophercloud.ProviderClient{}
	provider.SetToken(os.Getenv("OS_TOKEN"))
	client := &gophercloud.ServiceClient{
		ProviderClient: provider, Type: "volumev3",
		Endpoint: os.Getenv("OS_VOLUME_ENDPOINT"), Microversion: "3.24",
	}
	sdk := blockstorage.New(client)
	found, err := sdk.Services.FindRecord(context.Background(), "cinder-volume",
		request.Query("host", "host@lvm"))
	if err != nil {
		panic(err)
	}
	fmt.Println(found)
	result, err := sdk.Services.CleanupServiceWorkers(context.Background(),
		services.WithCleanupWorkerFilter("is_up", false))
	if err != nil {
		panic(err)
	}
	fmt.Println(string(result.JSON))
}

package v3_test

import (
	"context"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_volumeSummary() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.BlockStorageV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for volume, err := range service.Volumes.ListSummaryWithControl(ctx,
		resource.ListControl{MaxItems: 100},
		volumes.WithListOptions(volumes.ListOpts{Limit: 20})) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(volume.ID, volume.Name)
	}
}

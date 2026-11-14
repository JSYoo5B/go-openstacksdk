package blockstorage_test

import (
	"context"
	"encoding/json"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
)

func SearchSearchVolumes(ctx context.Context, conn *sdk.Connection) error {
	result, err := conn.SearchVolumes(ctx,
		blockstorage.SearchVolumesRequest{NameOrID: "data*"},
		blockstorage.WithVolumeSearchFilters(json.RawMessage(
			`{"metadata":{"owner":"worker"},"is_bootable":false}`)))
	if err != nil {
		return err
	}
	fmt.Println(string(result.Value), len(result.Volumes), len(result.Pages))

	ids, err := conn.SearchVolumes(ctx, blockstorage.SearchVolumesRequest{},
		blockstorage.WithVolumeSearchExpression("[?size >= `10`].id"))
	if err != nil {
		return err
	}
	fmt.Println(string(ids.Value))

	volume, err := conn.GetVolume(ctx,
		blockstorage.GetVolumeRequest{NameOrID: "data"})
	if err != nil {
		return err
	}
	if volume.Value == nil {
		return nil
	}
	fmt.Println(string(volume.Value))
	return nil
}

package blockstorage_test

import (
	"context"
	"encoding/json"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func ReadVolumeIDVolumeId(ctx context.Context, conn *sdk.Connection, nameOrID string) error {
	result, err := conn.GetVolumeID(ctx,
		blockstorage.GetVolumeIDRequest{NameOrID: nameOrID})
	if err != nil {
		if result != nil {
			fmt.Println("admitted fallback pages:", len(result.Pages))
			if result.Observed != nil {
				fmt.Println("admitted member status:", result.Observed.StatusCode)
			}
		}
		return err
	}
	if result.ID == nil {
		fmt.Println("volume absent")
		return nil
	}
	fmt.Println("ID JSON:", string(result.ID))
	fmt.Println("normalized volume:", string(result.Value))
	copy := result.Volume.Clone()
	fmt.Println("original ID field:", string(copy.Body["id"]))
	return nil
}

func ReadVolumeIDWithLocationVolumeId(ctx context.Context, conn *sdk.Connection, nameOrID string) error {
	project := "configured-project"
	location := resource.CloudLocation{
		Project: resource.CloudProject{
			ID: json.RawMessage(`"scope-id"`), Name: &project,
		},
	}
	result, err := conn.GetVolumeID(ctx,
		blockstorage.GetVolumeIDRequest{NameOrID: nameOrID},
		func(c *blockstorage.VolumeReadOpts) error { loc := location.Clone(); c.Location = &loc; return nil })
	if err != nil {
		return err
	}
	fmt.Println(string(result.ID))
	return nil
}

func ReadVolumeIDDirectVolumeId(ctx context.Context, cinder *gophercloud.ServiceClient, nameOrID string) error {
	result, err := blockstorage.GetVolumeID(ctx, cinder,
		blockstorage.GetVolumeIDRequest{NameOrID: nameOrID})
	if err != nil {
		return err
	}
	if result.ID != nil {
		fmt.Println(string(result.ID))
	}
	return nil
}

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

func ReadVolumesVolumeReads(ctx context.Context, conn *sdk.Connection, id, name string) error {
	listed, err := conn.ListVolumes(ctx)
	if err != nil {
		if listed != nil {
			fmt.Println("admitted list pages:", len(listed.Pages))
		}
		return err
	}
	fmt.Println(string(listed.Value), len(listed.Volumes))

	found, err := conn.GetVolumeByID(ctx,
		blockstorage.GetVolumeByIDRequest{ID: id})
	if err != nil {
		if found != nil && found.Observed != nil {
			fmt.Println("admitted member response:", found.Observed.StatusCode)
		}
		return err
	}
	fmt.Println(string(found.Value), string(found.Volume.Body["id"]))
	copy := found.Volume.Clone()
	var projection struct {
		ID *string `json:"id"`
	}
	if err := copy.Decode(&projection); err != nil {
		return err
	}

	exists, err := conn.VolumeExists(ctx,
		blockstorage.VolumeExistsRequest{NameOrID: name})
	if err != nil {
		if exists != nil {
			fmt.Println("admitted fallback pages:", len(exists.Pages))
		}
		return err
	}
	if exists.Exists == nil {
		return fmt.Errorf("missing completed existence result")
	}
	fmt.Println("exists:", *exists.Exists)
	return nil
}

func ListWithLocationVolumeReads(ctx context.Context, conn *sdk.Connection) error {
	cloud, project := "configured-cloud", "configured-project"
	location := resource.CloudLocation{
		Cloud: &cloud,
		Project: resource.CloudProject{
			ID: json.RawMessage(`"project-id"`), Name: &project,
		},
	}
	result, err := conn.ListVolumes(ctx, func(c *blockstorage.VolumeReadOpts) error { loc := location.Clone(); c.Location = &loc; return nil })
	if err != nil {
		return err
	}
	fmt.Println(string(result.Value))
	return nil
}

func ReadDirectVolumeReads(ctx context.Context, cinder *gophercloud.ServiceClient, id string) error {
	listed, err := blockstorage.ListVolumes(ctx, cinder)
	if err != nil {
		return err
	}
	fmt.Println(len(listed.Volumes), len(listed.Pages))
	volume, err := blockstorage.GetVolumeByID(ctx, cinder,
		blockstorage.GetVolumeByIDRequest{ID: id})
	if err != nil {
		return err
	}
	exists, err := blockstorage.VolumeExists(ctx, cinder,
		blockstorage.VolumeExistsRequest{NameOrID: id})
	if err != nil {
		return err
	}
	fmt.Println(string(volume.Value), *exists.Exists)
	return nil
}

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

func InspectUpdateVolumeMutations(result *blockstorage.UpdateVolumeResult) {
	if result == nil {
		return
	}
	fmt.Println("fixed mutation ID:", result.VolumeID)
	if result.Resolved != nil {
		fmt.Println("lookup pages:", len(result.Resolved.Pages))
		if result.Resolved.Observed != nil {
			fmt.Println("lookup status:", result.Resolved.Observed.StatusCode)
		}
	}
	if result.Applied != nil {
		fmt.Println("actual PUT response:", result.Applied.StatusCode,
			"raw bytes:", len(result.Applied.Body))
	}
}

func UpdateVolumeMutations(ctx context.Context, conn *sdk.Connection) error {
	result, err := conn.UpdateVolume(ctx,
		blockstorage.UpdateVolumeRequest{NameOrID: "data"},
		func(c *blockstorage.UpdateVolumeOpts) error { v := "renamed"; c.Attributes.Name = &v; return nil },
		blockstorage.WithUpdateVolumeMetadata(map[string]string{}),
		blockstorage.WithUpdateVolumeFields(map[string]json.RawMessage{
			"image_id":        json.RawMessage(`"image-id"`),
			"scheduler_hints": json.RawMessage(`{"same_host":["peer-volume"]}`),
		}))
	InspectUpdateVolumeMutations(result)
	if err != nil {
		return err
	}
	fmt.Println("merged normalized JSON:", string(result.Value))
	owned := result.Volume.Clone()
	fmt.Println("merged raw metadata:", string(owned.Body["metadata"]))

	// Empty kwargs still resolve the volume but do not send a PUT.
	unchanged, err := conn.UpdateVolume(ctx,
		blockstorage.UpdateVolumeRequest{NameOrID: "data"})
	InspectUpdateVolumeMutations(unchanged)
	if err != nil {
		return err
	}
	fmt.Println("no PUT:", unchanged.Applied == nil)
	return nil
}

func BootableVolumeMutations(ctx context.Context, conn *sdk.Connection) error {
	input := blockstorage.SetVolumeBootableRequest{NameOrID: "data"}
	enabled, err := conn.SetVolumeBootable(ctx, input) // Default: true.
	if enabled != nil && enabled.Applied != nil {
		fmt.Println("enable acknowledgement:", enabled.Applied.StatusCode)
	}
	if err != nil {
		return err
	}
	disabled, err := conn.SetVolumeBootable(ctx, input,
		func(c *blockstorage.SetVolumeBootableOpts) error {
			v := false
			c.Bootable = &v
			return nil
		})
	if disabled != nil && disabled.Applied != nil {
		fmt.Println("disable acknowledgement:", disabled.Applied.StatusCode)
	}
	return err
}

func UpdateWithLocationVolumeMutations(ctx context.Context, conn *sdk.Connection) error {
	cloud, project := "configured-cloud", "configured-project"
	name := "renamed"
	location := resource.CloudLocation{
		Cloud:   &cloud,
		Project: resource.CloudProject{ID: json.RawMessage(`"scope-id"`), Name: &project},
	}
	result, err := conn.UpdateVolume(ctx,
		blockstorage.UpdateVolumeRequest{NameOrID: "data"},
		blockstorage.WithUpdateVolumeOptions(blockstorage.UpdateVolumeOpts{
			Attributes: blockstorage.UpdateVolumeAttributes{Name: &name},
			Location:   &location,
		}))
	InspectUpdateVolumeMutations(result)
	return err
}

func DirectVolumeMutations(ctx context.Context, cinder *gophercloud.ServiceClient) error {
	result, err := blockstorage.UpdateVolume(ctx, cinder,
		blockstorage.UpdateVolumeRequest{NameOrID: "data"},
		func(c *blockstorage.UpdateVolumeOpts) error {
			v := "updated"
			c.Attributes.Description = &v
			return nil
		})
	InspectUpdateVolumeMutations(result)
	if err != nil {
		return err
	}
	_, err = blockstorage.SetVolumeBootable(ctx, cinder,
		blockstorage.SetVolumeBootableRequest{NameOrID: "data"},
		func(c *blockstorage.SetVolumeBootableOpts) error {
			v := false
			c.Bootable = &v
			return nil
		})
	return err
}

package blockstorage_test

import (
	"context"
	"encoding/json"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumetypes"
)

func InspectAccessVolumeTypeAccess(ctx context.Context, conn *sdk.Connection) error {
	result, err := conn.GetVolumeTypeAccess(ctx,
		blockstorage.GetVolumeTypeAccessRequest{NameOrID: "fast"})
	if result != nil {
		if result.Resolved != nil {
			fmt.Println("lookup pages:", len(result.Resolved.Pages))
			if result.Resolved.Observed != nil {
				fmt.Println("lookup status:", result.Resolved.Observed.StatusCode)
			}
		}
		if result.Observed != nil {
			fmt.Println("access status:", result.Observed.StatusCode)
		}
	}
	if err != nil {
		return err
	}
	fmt.Println("actual type ID:", result.TypeID)
	fmt.Println("raw access JSON:", string(result.Value))
	// Accesses exists only when the entire value is an array of objects.
	for _, row := range result.Accesses {
		owned := row.Clone()
		fmt.Println("original project_id:", string(owned.Body["project_id"]))
		// This optional native projection enforces two string fields.
		// A projection error does not change the successful raw access result.
		var projection volumetypes.VolumeTypeAccess
		if err := owned.Decode(&projection); err != nil {
			return err
		}
		fmt.Println("project/type:", projection.ProjectID, projection.VolumeTypeID)
	}
	return nil
}

func ChangeAccessVolumeTypeAccess(ctx context.Context, conn *sdk.Connection, projectID string) error {
	input := blockstorage.VolumeTypeAccessRequest{NameOrID: "fast", ProjectID: projectID}
	added, err := conn.AddVolumeTypeAccess(ctx, input)
	if added != nil {
		fmt.Println("resolved target:", added.TypeID)
		if added.Applied != nil {
			fmt.Println("add acknowledgement:", added.Applied.StatusCode,
				"raw bytes:", len(added.Applied.Body))
		}
	}
	if err != nil {
		return err
	}
	removed, err := conn.RemoveVolumeTypeAccess(ctx, input)
	if removed != nil && removed.Applied != nil {
		fmt.Println("remove acknowledgement:", removed.Applied.StatusCode)
	}
	return err
}

func AccessWithLocationVolumeTypeAccess(ctx context.Context, conn *sdk.Connection) error {
	cloud, project := "configured-cloud", "configured-project"
	location := resource.CloudLocation{
		Cloud: &cloud, Zone: json.RawMessage(`"owned-zone"`),
		Project: resource.CloudProject{ID: json.RawMessage(`"scope-id"`), Name: &project},
	}
	result, err := conn.GetVolumeTypeAccess(ctx,
		blockstorage.GetVolumeTypeAccessRequest{NameOrID: "fast"},
		blockstorage.WithVolumeTypeReadOptions(blockstorage.VolumeTypeReadOpts{Location: &location}))
	if err != nil {
		return err
	}
	// Location belongs to the resolved Type view, not to access rows.
	fmt.Println("resolved Type JSON:", string(result.Resolved.Value))
	return nil
}

func DirectAccessVolumeTypeAccess(ctx context.Context, cinder *gophercloud.ServiceClient) error {
	result, err := blockstorage.GetVolumeTypeAccess(ctx, cinder,
		blockstorage.GetVolumeTypeAccessRequest{NameOrID: "fast"})
	if err != nil {
		return err
	}
	fmt.Println("raw access JSON:", string(result.Value))
	return nil
}

package blockstorage_test

import (
	"context"
	"encoding/json"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	nativeLimits "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/limits"
)

func InspectVolumeLimits(result *blockstorage.GetVolumeLimitsResult) {
	if result == nil {
		return
	}
	if result.Project != nil {
		fmt.Println("project ID JSON:", string(result.Project.ID),
			"seeded:", result.Project.SeededID,
			"lookup pages:", len(result.Project.Pages))
		if result.Project.Observed != nil {
			fmt.Println("project member status:", result.Project.Observed.StatusCode)
		}
	}
	fmt.Println("requested project ID JSON:", string(result.RequestedProjectID))
	if result.Observed != nil {
		fmt.Println("limits status:", result.Observed.StatusCode,
			"raw bytes:", len(result.Observed.Body))
	}
}

func CurrentVolumeLimits(ctx context.Context, conn *sdk.Connection) error {
	result, err := conn.GetVolumeLimits(ctx, blockstorage.GetVolumeLimitsRequest{})
	InspectVolumeLimits(result) // Completed physical proof also remains available on error.
	if err != nil {
		return err
	}
	fmt.Println("normalized nullable JSON:", string(result.Value))
	owned := result.Limits.Clone()
	fmt.Println("original absolute JSON:", string(owned.Body["absolute"]))

	// This optional native projection enforces the native int/string schema.
	// A projection error does not invalidate the successful raw workflow.
	var projection nativeLimits.Limits
	if projectionErr := owned.Decode(&projection); projectionErr != nil {
		fmt.Println("native projection:", projectionErr)
	} else {
		fmt.Println("native maximum volumes:", projection.Absolute.MaxTotalVolumes)
	}
	return nil
}

func OtherProjectVolumeLimits(ctx context.Context, conn *sdk.Connection, nameOrID string) error {
	result, err := conn.GetVolumeLimits(ctx,
		blockstorage.GetVolumeLimitsRequest{NameOrID: nameOrID})
	InspectVolumeLimits(result)
	if err != nil {
		return err
	}
	fmt.Println("normalized limits:", string(result.Value))
	return nil
}

func WithLocationVolumeLimits(ctx context.Context, conn *sdk.Connection) error {
	cloud, project := "configured-cloud", "authenticated-project"
	location := resource.CloudLocation{
		Cloud: &cloud, Zone: json.RawMessage(`"owned-zone"`),
		Project: resource.CloudProject{ID: json.RawMessage(`"scope-id"`), Name: &project},
	}
	result, err := conn.GetVolumeLimits(ctx,
		blockstorage.GetVolumeLimitsRequest{NameOrID: "other-project"},
		blockstorage.WithGetVolumeLimitsOptions(blockstorage.GetVolumeLimitsOpts{
			Location: &location,
		}))
	InspectVolumeLimits(result)
	if err != nil {
		return err
	}
	fmt.Println("limits with owned authenticated location:", string(result.Value))
	return nil
}

func DirectVolumeLimits(ctx context.Context, cinder, identity *gophercloud.ServiceClient) error {
	// Empty input needs no Identity client and sends no project filter.
	current, err := blockstorage.GetVolumeLimits(ctx, cinder, nil,
		blockstorage.GetVolumeLimitsRequest{})
	InspectVolumeLimits(current)
	if err != nil {
		return err
	}
	selected, err := blockstorage.GetVolumeLimits(ctx, cinder, identity,
		blockstorage.GetVolumeLimitsRequest{NameOrID: "project-name-or-id"},
		func(c *blockstorage.GetVolumeLimitsOpts) error { c.Location = &resource.CloudLocation{}; return nil })
	InspectVolumeLimits(selected)
	return err
}

package blockstorage_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func ConfiguredVolumeSnapshotMutations(ctx context.Context, cloudName string) (*sdk.Connection, error) {
	return sdk.Connect(ctx, sdk.WithCloud(cloudName))
}

func AdoptVolumeSnapshotMutations(provider *gophercloud.ProviderClient) (*sdk.Connection, error) {
	return sdk.FromProvider(provider, sdk.WithRegion("RegionOne"))
}

func InspectCreateVolumeSnapshotMutations(result *blockstorage.CreateVolumeSnapshotResult) {
	if result == nil {
		return
	}
	fmt.Println("latest logical ID JSON:", string(result.SnapshotID))
	if result.Created != nil {
		fmt.Println("POST:", result.Created.StatusCode, "bytes:", len(result.Created.Body))
	}
	if result.LastAccepted != nil {
		fmt.Println("last admitted response:", result.LastAccepted.StatusCode)
	}
	fmt.Println("created-stage logical JSON:", string(result.CreatedValue))
	if result.CreatedSnapshot != nil {
		actual := result.CreatedSnapshot.Clone()
		fmt.Println("actual POST ID field:", string(actual.Body["id"]))
		var fields map[string]json.RawMessage
		if err := actual.Decode(&fields); err != nil {
			fmt.Println("optional projection:", err)
		} else {
			fmt.Println("actual POST field count:", len(fields))
		}
	}
	fmt.Println("final logical JSON:", string(result.Value), "ready:", len(result.Ready) != 0)
}

func InspectDeleteVolumeSnapshotMutations(result *blockstorage.DeleteVolumeSnapshotResult) {
	if result == nil {
		return
	}
	if result.Deleted != nil {
		fmt.Println("requested deletion completed:", *result.Deleted)
	}
	if result.Resolved != nil {
		fmt.Println("lookup pages:", len(result.Resolved.Pages), "seeded ID:", result.Resolved.SeededID)
		if result.Resolved.Observed != nil {
			fmt.Println("lookup member:", result.Resolved.Observed.StatusCode)
		}
	}
	if result.Applied != nil {
		fmt.Println("DELETE acknowledged:", result.Applied.StatusCode)
	}
	if result.Absent != nil {
		fmt.Println("clean polling absence:", result.Absent.StatusCode)
	}
	fmt.Println("latest logical ID JSON:", string(result.SnapshotID))
}

func InspectErrorVolumeSnapshotMutations(err error) {
	if err == nil {
		return
	}
	var response *resource.ResponseError
	if errors.As(err, &response) {
		fmt.Println("accepted response failure:", response.StatusCode, len(response.Body))
	}
	var sdkTimeout *blockstorage.SnapshotWaitTimeoutError
	if errors.As(err, &sdkTimeout) {
		fmt.Println("SDK wait budget:", sdkTimeout.Timeout)
	}
	fmt.Println("deadline cause:", errors.Is(err, context.DeadlineExceeded))
}

func ViaConnectionVolumeSnapshotMutations(ctx context.Context, cloudName, volumeID string) error {
	conn, err := ConfiguredVolumeSnapshotMutations(ctx, cloudName)
	if err != nil {
		return err
	}
	created, err := conn.CreateVolumeSnapshot(ctx,
		blockstorage.CreateVolumeSnapshotRequest{VolumeID: volumeID},
		blockstorage.WithCreateVolumeSnapshotName("nightly"),
		blockstorage.WithCreateVolumeSnapshotDescription("nightly volume snapshot"))
	InspectCreateVolumeSnapshotMutations(created) // Inspect completed physical stages even when err is nonnil.
	if err != nil {
		InspectErrorVolumeSnapshotMutations(err)
		return err
	}

	// Default deletion resolves exactly one snapshot and returns after acknowledgement.
	deleted, err := conn.DeleteVolumeSnapshot(ctx,
		blockstorage.DeleteVolumeSnapshotRequest{NameOrID: "nightly"})
	InspectDeleteVolumeSnapshotMutations(deleted)
	InspectErrorVolumeSnapshotMutations(err)
	return err
}

func DirectNoWaitVolumeSnapshotMutations(ctx context.Context, cinder *gophercloud.ServiceClient, volumeID string) error {
	result, err := blockstorage.CreateVolumeSnapshot(ctx, cinder,
		blockstorage.CreateVolumeSnapshotRequest{VolumeID: volumeID},
		blockstorage.WithCreateVolumeSnapshotForce(false),
		blockstorage.WithCreateVolumeSnapshotWait(false),
		blockstorage.WithCreateVolumeSnapshotFields(map[string]json.RawMessage{
			"display_name": json.RawMessage(`"nightly"`),
			"description":  json.RawMessage(`"created without polling"`),
		}))
	InspectCreateVolumeSnapshotMutations(result)
	InspectErrorVolumeSnapshotMutations(err)
	return err
}

func WaitForDeletionVolumeSnapshotMutations(ctx context.Context, conn *sdk.Connection, nameOrID string) error {
	budget, interval := 10*time.Minute, time.Second
	result, err := conn.DeleteVolumeSnapshot(ctx,
		blockstorage.DeleteVolumeSnapshotRequest{NameOrID: nameOrID},
		blockstorage.WithDeleteVolumeSnapshotWait(true),
		blockstorage.WithDeleteVolumeSnapshotWaitOptions(
			resource.WithTimeout(budget), resource.WithPollInterval(interval),
		))
	InspectDeleteVolumeSnapshotMutations(result)
	InspectErrorVolumeSnapshotMutations(err)
	return err
}

func WithCallerDeadlineVolumeSnapshotMutations(ctx context.Context, conn *sdk.Connection, volumeID string) error {
	parent, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	result, err := conn.CreateVolumeSnapshot(parent,
		blockstorage.CreateVolumeSnapshotRequest{VolumeID: volumeID})
	InspectCreateVolumeSnapshotMutations(result)
	InspectErrorVolumeSnapshotMutations(err)
	return err
}

func OwnedLocationVolumeSnapshotMutations(ctx context.Context, conn *sdk.Connection, volumeID string) error {
	cloud, project := "configured-cloud", "configured-project"
	location := resource.CloudLocation{
		Cloud: &cloud, Project: resource.CloudProject{
			ID: json.RawMessage(`"token-scope-id"`), Name: &project,
		},
	}
	wait := false
	result, err := conn.CreateVolumeSnapshot(ctx,
		blockstorage.CreateVolumeSnapshotRequest{VolumeID: volumeID},
		blockstorage.WithCreateVolumeSnapshotOptions(blockstorage.CreateVolumeSnapshotOpts{
			Wait: &wait, Location: &location,
		}))
	InspectCreateVolumeSnapshotMutations(result)
	return err
}

func EmptyDeleteVolumeSnapshotMutations(ctx context.Context) error {
	var conn *sdk.Connection
	result, err := conn.DeleteVolumeSnapshot(ctx, blockstorage.DeleteVolumeSnapshotRequest{})
	InspectDeleteVolumeSnapshotMutations(result) // Valid context, no service selection, successful false.
	return err
}

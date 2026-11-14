package blockstorage_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_volumeRevertUnmanage() {
	if err := runVolumeRevertUnmanage(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runVolumeRevertUnmanage() error {
	cloudName, volumeID := os.Getenv("CLOUD_NAME"), os.Getenv("VOLUME_ID")
	if cloudName == "" || volumeID == "" {
		return fmt.Errorf("CLOUD_NAME and VOLUME_ID are required")
	}
	action := os.Getenv("VOLUME_LIFECYCLE_ACTION")
	if action != "revert" && action != "unmanage" {
		return fmt.Errorf("VOLUME_LIFECYCLE_ACTION must be revert or unmanage")
	}
	path := os.Getenv("VOLUME_LIFECYCLE_PATH")
	if path == "" {
		path = "connection"
	}
	if path != "connection" && path != "direct" && path != "service" {
		return fmt.Errorf("unknown VOLUME_LIFECYCLE_PATH %q", path)
	}
	var snapshotID string
	if action == "revert" {
		value, supplied := os.LookupEnv("SNAPSHOT_ID")
		if !supplied {
			return fmt.Errorf("SNAPSHOT_ID is required; explicit empty is allowed")
		}
		snapshotID = value
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
	if err != nil {
		return err
	}
	input := blockstorage.VolumeActionRequest{VolumeID: volumeID}
	result, err := applyVolumeRevertUnmanage(ctx, conn, path, action, input, snapshotID)
	inspectVolumeRevertUnmanage(result, err)
	return err
}

func applyVolumeRevertUnmanage(ctx context.Context, conn *sdk.Connection, path, action string, input blockstorage.VolumeActionRequest, snapshotID string) (*blockstorage.VolumeActionResult, error) {
	if path == "connection" {
		if action == "revert" {
			return conn.RevertVolumeToSnapshot(ctx, input, snapshotID)
		}
		return conn.UnmanageVolume(ctx, input)
	}
	cinder, err := conn.BlockStorageV3(ctx)
	if err != nil {
		return nil, err
	}
	if path == "direct" {
		if action == "revert" {
			return blockstorage.RevertVolumeToSnapshot(ctx, cinder.RawClient(), input, snapshotID)
		}
		return blockstorage.UnmanageVolume(ctx, cinder.RawClient(), input)
	}
	if action == "revert" {
		return cinder.Volumes.RevertVolumeToSnapshot(ctx, input.VolumeID, snapshotID)
	}
	return cinder.Volumes.UnmanageVolume(ctx, input.VolumeID)
}

func inspectVolumeRevertUnmanage(result *blockstorage.VolumeActionResult, err error) {
	if result != nil {
		fmt.Printf("volume ID: %q; microversion: %q\n", result.VolumeID, result.Microversion)
		for index, page := range result.Discovery {
			if page != nil {
				fmt.Println("admitted discovery response:", index, page.StatusCode, len(page.Body))
			}
		}
		if result.Applied != nil {
			fmt.Println("admitted action response:", result.Applied.StatusCode)
			fmt.Printf("opaque body: %q\n", result.Applied.Body)
		}
		fmt.Println("helper acknowledgement completed:", result.Completed)
	}
	if err != nil {
		var proof *resource.ResponseError
		if errors.As(err, &proof) {
			fmt.Println("current admitted error response:", proof.StatusCode, len(proof.Body))
		}
	}
}

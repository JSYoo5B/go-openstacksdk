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
	"github.com/gophercloud/gophercloud/v2"
)

func Example_volumeActions() {
	if err := runVolumeActions(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runVolumeActions() error {
	cloudName, volumeID := os.Getenv("CLOUD_NAME"), os.Getenv("VOLUME_ID")
	if cloudName == "" || volumeID == "" {
		return fmt.Errorf("CLOUD_NAME and VOLUME_ID are required")
	}
	action := os.Getenv("STATE_ACTION")
	if action == "" {
		action = "reserve"
	}
	switch action {
	case "reserve", "unreserve", "begin-detaching", "abort-detaching":
	default:
		return fmt.Errorf("unknown STATE_ACTION %q", action)
	}
	path := os.Getenv("STATE_ACTION_PATH")
	if path == "" {
		path = "connection"
	}
	switch path {
	case "connection", "direct", "service":
	default:
		return fmt.Errorf("unknown STATE_ACTION_PATH %q", path)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
	if err != nil {
		return err
	}
	input := blockstorage.VolumeActionRequest{VolumeID: volumeID}
	result, err := applyVolumeActions(ctx, conn, action, path, input)
	inspectVolumeActions(result, err)
	return err
}

func applyVolumeActions(ctx context.Context, conn *sdk.Connection, action, path string, input blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error) {
	if path == "connection" {
		calls := map[string]func(context.Context, blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error){
			"reserve":         conn.ReserveVolume,
			"unreserve":       conn.UnreserveVolume,
			"begin-detaching": conn.BeginVolumeDetaching,
			"abort-detaching": conn.AbortVolumeDetaching,
		}
		return calls[action](ctx, input)
	}
	cinder, err := conn.BlockStorageV3(ctx)
	if err != nil {
		return nil, err
	}
	if path == "direct" {
		calls := map[string]func(context.Context, *gophercloud.ServiceClient, blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error){
			"reserve":         blockstorage.ReserveVolume,
			"unreserve":       blockstorage.UnreserveVolume,
			"begin-detaching": blockstorage.BeginVolumeDetaching,
			"abort-detaching": blockstorage.AbortVolumeDetaching,
		}
		return calls[action](ctx, cinder.RawClient(), input)
	}
	calls := map[string]func(context.Context, string) (*blockstorage.VolumeActionResult, error){
		"reserve":         cinder.Volumes.ReserveVolume,
		"unreserve":       cinder.Volumes.UnreserveVolume,
		"begin-detaching": cinder.Volumes.BeginVolumeDetaching,
		"abort-detaching": cinder.Volumes.AbortVolumeDetaching,
	}
	return calls[action](ctx, input.VolumeID)
}

func inspectVolumeActions(result *blockstorage.VolumeActionResult, err error) {
	if result != nil {
		fmt.Printf("volume ID: %q; microversion: %q\n", result.VolumeID, result.Microversion)
		for i, page := range result.Discovery {
			if page != nil {
				fmt.Println("admitted discovery response:", i, page.StatusCode, len(page.Body))
			}
		}
		if result.Applied != nil {
			fmt.Println("admitted action response:", result.Applied.StatusCode)
			fmt.Printf("opaque action body: %q\n", result.Applied.Body)
		}
		fmt.Println("helper acknowledgement completed:", result.Completed)
	}
	if err != nil {
		fmt.Println("action error:", err)
		var physical *resource.ResponseError
		if errors.As(err, &physical) {
			fmt.Println("current admitted error response:", physical.StatusCode, len(physical.Body))
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			fmt.Println("parent context interrupted the call")
		}
	}
}

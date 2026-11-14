package blockstorage_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_volumeFlags() {
	if err := runVolumeFlags(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runVolumeFlags() error {
	cloudName, volumeID := os.Getenv("CLOUD_NAME"), os.Getenv("VOLUME_ID")
	if cloudName == "" || volumeID == "" {
		return fmt.Errorf("CLOUD_NAME and VOLUME_ID are required")
	}
	action := os.Getenv("FLAG_ACTION")
	if action != "bootable" && action != "readonly" {
		return fmt.Errorf("FLAG_ACTION must be bootable or readonly")
	}
	path := os.Getenv("FLAG_ACTION_PATH")
	if path == "" {
		path = "connection"
	}
	if path != "connection" && path != "direct" && path != "service" {
		return fmt.Errorf("unknown FLAG_ACTION_PATH %q", path)
	}
	text, supplied := os.LookupEnv("FLAG_VALUE")
	if action == "bootable" && !supplied {
		return fmt.Errorf("FLAG_VALUE is required for bootable")
	}
	flag := true
	if supplied {
		parsed, err := strconv.ParseBool(text)
		if err != nil {
			return fmt.Errorf("FLAG_VALUE: %w", err)
		}
		flag = parsed
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var policy blockstorage.VolumeReadonlyOpts
	if action == "readonly" {
		var originals []blockstorage.VolumeReadonlyOption
		if supplied {
			originals = append(originals, volumes.WithVolumeReadonly(flag))
		}
		prepared, err := blockstorage.PrepareVolumeReadonlyOptions(ctx, originals...)
		if err != nil {
			return err
		}
		policy = prepared
	}
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
	if err != nil {
		return err
	}
	input := blockstorage.VolumeActionRequest{VolumeID: volumeID}
	result, err := applyVolumeFlags(ctx, conn, action, path, input, flag, policy)
	inspectVolumeFlags(result, err)
	return err
}

func applyVolumeFlags(ctx context.Context, conn *sdk.Connection, action, path string, input blockstorage.VolumeActionRequest, flag bool, policy blockstorage.VolumeReadonlyOpts) (*blockstorage.VolumeActionResult, error) {
	options := []blockstorage.VolumeReadonlyOption{volumes.WithVolumeReadonlyOptions(policy)}
	if path == "connection" {
		if action == "bootable" {
			return conn.SetVolumeBootableStatus(ctx, input, flag)
		}
		return conn.SetVolumeReadonly(ctx, input, options...)
	}
	cinder, err := conn.BlockStorageV3(ctx)
	if err != nil {
		return nil, err
	}
	if path == "direct" {
		if action == "bootable" {
			return blockstorage.SetVolumeBootableStatus(ctx, cinder.RawClient(), input, flag)
		}
		return blockstorage.SetVolumeReadonly(ctx, cinder.RawClient(), input, options...)
	}
	if action == "bootable" {
		return cinder.Volumes.SetVolumeBootableStatus(ctx, input.VolumeID, flag)
	}
	return cinder.Volumes.SetVolumeReadonly(ctx, input.VolumeID, options...)
}

func inspectVolumeFlags(result *blockstorage.VolumeActionResult, err error) {
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
	}
}

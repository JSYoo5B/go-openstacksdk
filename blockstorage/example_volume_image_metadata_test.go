package blockstorage_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_volumeImageMetadata() {
	if err := runVolumeImageMetadata(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runVolumeImageMetadata() error {
	cloudName, volumeID := os.Getenv("CLOUD_NAME"), os.Getenv("VOLUME_ID")
	if cloudName == "" || volumeID == "" {
		return fmt.Errorf("CLOUD_NAME and VOLUME_ID are required")
	}
	action, path := os.Getenv("IMAGE_METADATA_ACTION"), os.Getenv("IMAGE_METADATA_PATH")
	if action != "set" && action != "delete" {
		return fmt.Errorf("IMAGE_METADATA_ACTION must be set or delete")
	}
	if path == "" {
		path = "connection"
	}
	if path != "connection" && path != "direct" && path != "service" {
		return fmt.Errorf("unknown IMAGE_METADATA_PATH %q", path)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	input := blockstorage.VolumeActionRequest{VolumeID: volumeID}
	var set blockstorage.VolumeImageMetadataOpts
	var deletion blockstorage.VolumeImageMetadataDeleteOpts
	if action == "set" {
		var values map[string]json.RawMessage
		if text, supplied := os.LookupEnv("IMAGE_METADATA_JSON"); supplied {
			if err := json.Unmarshal([]byte(text), &values); err != nil {
				return fmt.Errorf("IMAGE_METADATA_JSON: %w", err)
			}
		}
		options := []blockstorage.VolumeImageMetadataOption{volumes.WithVolumeImageMetadataRaw(values)}
		if key, supplied := os.LookupEnv("IMAGE_METADATA_KEY"); supplied {
			options = append(options, volumes.WithVolumeImageMetadataValue(key, os.Getenv("IMAGE_METADATA_VALUE")))
		}
		prepared, err := blockstorage.PrepareVolumeImageMetadataOptions(ctx, options...)
		if err != nil {
			return err
		}
		set = prepared
	} else {
		var options []blockstorage.VolumeImageMetadataDeleteOption
		if text, supplied := os.LookupEnv("IMAGE_METADATA_KEYS_JSON"); supplied {
			var keys []string
			if err := json.Unmarshal([]byte(text), &keys); err != nil {
				return fmt.Errorf("IMAGE_METADATA_KEYS_JSON: %w", err)
			}
			options = append(options, volumes.WithVolumeImageMetadataDeleteKeys(keys...))
		}
		prepared, err := blockstorage.PrepareVolumeImageMetadataDeleteOptions(ctx, options...)
		if err != nil {
			return err
		}
		deletion = prepared
	}
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
	if err != nil {
		return err
	}
	if action == "set" {
		result, err := applySetVolumeImageMetadata(ctx, conn, path, input, set)
		inspectSetVolumeImageMetadata(result)
		inspectErrorVolumeImageMetadata(err)
		return err
	}
	result, err := applyDeleteVolumeImageMetadata(ctx, conn, path, input, deletion)
	inspectDeleteVolumeImageMetadata(result)
	inspectErrorVolumeImageMetadata(err)
	return err
}

func applySetVolumeImageMetadata(ctx context.Context, conn *sdk.Connection, path string, input blockstorage.VolumeActionRequest, policy blockstorage.VolumeImageMetadataOpts) (*blockstorage.VolumeActionResult, error) {
	option := volumes.WithVolumeImageMetadataOptions(policy)
	if path == "connection" {
		return conn.SetVolumeImageMetadata(ctx, input, option)
	}
	cinder, err := conn.BlockStorageV3(ctx)
	if err != nil {
		return nil, err
	}
	if path == "direct" {
		return blockstorage.SetVolumeImageMetadata(ctx, cinder.RawClient(), input, option)
	}
	return cinder.Volumes.SetVolumeImageMetadata(ctx, input.VolumeID, option)
}

func applyDeleteVolumeImageMetadata(ctx context.Context, conn *sdk.Connection, path string, input blockstorage.VolumeActionRequest, policy blockstorage.VolumeImageMetadataDeleteOpts) (*blockstorage.VolumeImageMetadataDeleteResult, error) {
	option := volumes.WithVolumeImageMetadataDeleteOptions(policy)
	if path == "connection" {
		return conn.DeleteVolumeImageMetadata(ctx, input, option)
	}
	cinder, err := conn.BlockStorageV3(ctx)
	if err != nil {
		return nil, err
	}
	if path == "direct" {
		return blockstorage.DeleteVolumeImageMetadata(ctx, cinder.RawClient(), input, option)
	}
	return cinder.Volumes.DeleteVolumeImageMetadata(ctx, input.VolumeID, option)
}

func inspectSetVolumeImageMetadata(result *blockstorage.VolumeActionResult) {
	if result == nil {
		return
	}
	fmt.Printf("volume ID: %q; microversion: %q\n", result.VolumeID, result.Microversion)
	for index, page := range result.Discovery {
		if page != nil {
			fmt.Println("admitted discovery:", index, page.StatusCode, len(page.Body))
		}
	}
	if result.Applied != nil {
		fmt.Printf("admitted set: HTTP %d; opaque body %q\n", result.Applied.StatusCode, result.Applied.Body)
	}
	fmt.Println("helper acknowledgement completed:", result.Completed)
}

func inspectDeleteVolumeImageMetadata(result *blockstorage.VolumeImageMetadataDeleteResult) {
	if result == nil {
		return
	}
	fmt.Printf("volume ID: %q; microversion: %q\n", result.VolumeID, result.Microversion)
	for index, page := range result.Discovery {
		if page != nil {
			fmt.Println("admitted discovery:", index, page.StatusCode, len(page.Body))
		}
	}
	if result.Observed != nil {
		fmt.Printf("admitted member GET: HTTP %d; body %q\n", result.Observed.StatusCode, result.Observed.Body)
	}
	for _, entry := range result.Deleted {
		if entry.Response != nil {
			fmt.Printf("acknowledged key %q: HTTP %d; opaque body %q\n", entry.Key, entry.Response.StatusCode, entry.Response.Body)
		}
	}
	if result.Failed != nil {
		fmt.Printf("failed key: %q\n", result.Failed.Key)
		if page := result.Failed.Response; page != nil {
			fmt.Printf("admitted failed-key response: HTTP %d; opaque body %q\n", page.StatusCode, page.Body)
		}
	}
	fmt.Println("helper acknowledgement completed:", result.Completed)
}

func inspectErrorVolumeImageMetadata(err error) {
	if err == nil {
		return
	}
	var proof *resource.ResponseError
	if errors.As(err, &proof) {
		fmt.Println("current admitted error response:", proof.StatusCode, len(proof.Body))
	}
}

package blockstorage_test

import (
	"context"
	"encoding/json"
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

func Example_volumeUploadImage() {
	if err := runVolumeUploadImage(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runVolumeUploadImage() error {
	cloudName, volumeID := os.Getenv("CLOUD_NAME"), os.Getenv("VOLUME_ID")
	if cloudName == "" || volumeID == "" {
		return fmt.Errorf("CLOUD_NAME and VOLUME_ID are required")
	}
	imageName, supplied := os.LookupEnv("IMAGE_NAME")
	if !supplied {
		return fmt.Errorf("IMAGE_NAME is required; explicit empty is allowed")
	}
	path := os.Getenv("IMAGE_UPLOAD_PATH")
	if path == "" {
		path = "connection"
	}
	if path != "connection" && path != "direct" && path != "service" {
		return fmt.Errorf("unknown IMAGE_UPLOAD_PATH %q", path)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var options []blockstorage.VolumeImageUploadOption
	if text, supplied := os.LookupEnv("IMAGE_UPLOAD_FORCE"); supplied {
		value, err := strconv.ParseBool(text)
		if err != nil {
			return fmt.Errorf("IMAGE_UPLOAD_FORCE: %w", err)
		}
		options = append(options, volumes.WithVolumeImageUploadForce(value))
	}
	if text, supplied := os.LookupEnv("IMAGE_UPLOAD_DISK_FORMAT"); supplied {
		options = append(options, volumes.WithVolumeImageUploadDiskFormat(text))
	}
	if text, supplied := os.LookupEnv("IMAGE_UPLOAD_CONTAINER_FORMAT"); supplied {
		options = append(options, volumes.WithVolumeImageUploadContainerFormat(text))
	}
	if text, supplied := os.LookupEnv("IMAGE_UPLOAD_VISIBILITY"); supplied {
		options = append(options, volumes.WithVolumeImageUploadVisibility(text))
	}
	if text, supplied := os.LookupEnv("IMAGE_UPLOAD_PROTECTED"); supplied {
		value, err := strconv.ParseBool(text)
		if err != nil {
			return fmt.Errorf("IMAGE_UPLOAD_PROTECTED: %w", err)
		}
		options = append(options, volumes.WithVolumeImageUploadProtected(value))
	}
	policy, err := blockstorage.PrepareVolumeImageUploadOptions(ctx, options...)
	if err != nil {
		return err
	}
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
	if err != nil {
		return err
	}
	input := blockstorage.VolumeActionRequest{VolumeID: volumeID}
	result, err := applyVolumeUploadImage(ctx, conn, path, input, imageName, policy)
	inspectVolumeUploadImage(result, err)
	return err
}

func applyVolumeUploadImage(ctx context.Context, conn *sdk.Connection, path string, input blockstorage.VolumeActionRequest, imageName string, policy blockstorage.VolumeImageUploadOpts) (*blockstorage.VolumeImageUploadResult, error) {
	option := volumes.WithVolumeImageUploadOptions(policy)
	if path == "connection" {
		return conn.UploadVolumeToImage(ctx, input, imageName, option)
	}
	cinder, err := conn.BlockStorageV3(ctx)
	if err != nil {
		return nil, err
	}
	if path == "direct" {
		return blockstorage.UploadVolumeToImage(ctx, cinder.RawClient(), input, imageName, option)
	}
	return cinder.Volumes.UploadVolumeToImage(ctx, input.VolumeID, imageName, option)
}

func inspectVolumeUploadImage(result *blockstorage.VolumeImageUploadResult, err error) {
	if result != nil {
		fmt.Printf("volume ID: %q; microversion: %q\n", result.VolumeID, result.Microversion)
		for index, page := range result.Discovery {
			if page != nil {
				fmt.Println("admitted discovery:", index, page.StatusCode, len(page.Body))
			}
		}
		if page := result.Applied; page != nil {
			fmt.Printf("admitted action: HTTP %d; actual body %q\n", page.StatusCode, page.Body)
		}
		if result.Upload != nil {
			fmt.Printf("selected upload JSON: %s\n", result.Upload)
			// Optional application inspection; API success does not require image_id.
			var details map[string]json.RawMessage
			if json.Unmarshal(result.Upload, &details) == nil {
				if id, present := details["image_id"]; present {
					fmt.Printf("optional image_id raw value: %s\n", id)
				}
			}
		}
		fmt.Println("helper acknowledgement/parse completed:", result.Completed)
	}
	if err != nil {
		var proof *resource.ResponseError
		if errors.As(err, &proof) {
			fmt.Println("current admitted error response:", proof.StatusCode, len(proof.Body))
		}
	}
}

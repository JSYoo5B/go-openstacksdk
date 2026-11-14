package image_test

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_imageRecordCloud() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
	pattern := flag.String("pattern", "*", "image name or ID glob")
	showAll := flag.Bool("show-all", false, "include shared images that are not accepted")
	identity := flag.String("image", "", "optional image name or ID for Cloud get_image")
	id := flag.String("id", "", "optional image ID for strict get_image_by_id")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := runImageRecordCloud(ctx, *cloud, *pattern, *showAll, *identity, *id); err != nil {
		log.Fatal(err)
	}
}

func recordValueImageRecordCloud(record *image.ImageRecord) any {
	if record == nil {
		return nil
	}
	return map[string]any{
		"resource": record.Resource, "wire": record.Wire,
		"status_code": record.StatusCode, "import_methods": record.ImportMethods,
	}
}

func printQueryImageRecordCloud(operation string, result *image.ImageRecordQueryResult) error {
	if result == nil {
		return nil
	}
	data, err := json.MarshalIndent(map[string]any{
		"operation": operation, "value": result.Value,
		"image_count": len(result.Images), "inventory_count": len(result.Inventory),
	}, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func runImageRecordCloud(ctx context.Context, cloud, pattern string, showAll bool, identity, id string) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return err
	}
	service, err := conn.Image(ctx)
	if err != nil {
		return err
	}
	all, err := service.AllCloudImageRecords(ctx, request.Custom(func(c *image.ImageRecordQueryOpts) error { v := showAll; c.ShowAll = &v; return nil }))
	if printErr := printQueryImageRecordCloud("list_images", all); printErr != nil {
		return errors.Join(err, printErr)
	}
	if err != nil {
		return fmt.Errorf("list images: %w", err)
	}
	matched, err := service.SearchImageRecords(ctx, pattern)
	if printErr := printQueryImageRecordCloud("search_images", matched); printErr != nil {
		return errors.Join(err, printErr)
	}
	if err != nil {
		return fmt.Errorf("search images: %w", err)
	}
	if identity != "" {
		found, err := service.GetCloudImageRecord(ctx, identity)
		if err != nil {
			return fmt.Errorf("get image: %w", err)
		}
		data, err := json.MarshalIndent(map[string]any{"operation": "get_image", "value": found.Value, "record": recordValueImageRecordCloud(found.Image)}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	}
	if id != "" {
		record, err := service.GetImageRecordByID(ctx, id)
		if err != nil {
			return fmt.Errorf("get image by id: %w", err)
		}
		data, err := json.MarshalIndent(map[string]any{"operation": "get_image_by_id", "record": recordValueImageRecordCloud(record)}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	}
	return nil
}

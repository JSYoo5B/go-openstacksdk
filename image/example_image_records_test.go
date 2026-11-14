package image_test

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_imageRecords() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
	imageID := flag.String("image-id", "", "required image ID for literal GET")
	find := flag.String("find", "", "optional exact image name or ID")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := runImageRecords(ctx, *cloud, *imageID, *find); err != nil {
		var response *resource.ResponseError
		if errors.As(err, &response) {
			fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
		}
		log.Fatal(err)
	}
}

func printRecordImageRecords(value *image.ImageRecord) error {
	body, err := json.MarshalIndent(map[string]any{
		"resource": value.Resource, "wire": value.Wire,
		"status_code": value.StatusCode, "import_methods": value.ImportMethods,
	}, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(body))
	return nil
}

func runImageRecords(ctx context.Context, cloud, imageID, nameOrID string) error {
	if imageID == "" {
		return fmt.Errorf("-image-id is required")
	}
	conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
	if err != nil {
		return err
	}
	service, err := conn.Image(ctx)
	if err != nil {
		return err
	}

	value, err := service.GetImageRecord(ctx, image.ImageRecordRequest{ID: imageID},
		request.Header("X-Request-Source", "image-records-example"))
	if err != nil {
		return err
	}
	if err := printRecordImageRecords(value); err != nil {
		return err
	}

	for value, readErr := range service.ListImageRecords(ctx,
		image.WithImageRecordListFilter("status", "active"),
		request.Limit(10), request.MaxItems(20)) {
		if readErr != nil {
			return readErr
		}
		if err := printRecordImageRecords(value); err != nil {
			return err
		}
	}

	if nameOrID != "" {
		found, err := service.FindImageRecord(ctx, nameOrID,
			request.IgnoreMissing(false))
		if err != nil {
			return err
		}
		if err := printRecordImageRecords(found); err != nil {
			return err
		}
	}
	return nil
}

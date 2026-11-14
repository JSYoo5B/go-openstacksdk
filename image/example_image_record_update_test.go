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

func Example_imageRecordUpdate() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
	imageID := flag.String("image-id", "", "required literal image ID")
	name := flag.String("name", "release-image", "new image name")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := runImageRecordUpdate(ctx, *cloud, *imageID, *name); err != nil {
		var response *resource.ResponseError
		if errors.As(err, &response) {
			fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
		}
		log.Fatal(err)
	}
}

func printRecordImageRecordUpdate(label string, value *image.ImageRecord, operationErr error) error {
	output := map[string]any{"step": label, "partial": value != nil && operationErr != nil}
	if value != nil {
		output["resource"] = value.Resource
		output["wire"] = value.Wire
		output["receipt_status_code"] = value.StatusCode
		output["receipt_body_bytes"] = len(value.Envelope)
		output["import_methods"] = value.ImportMethods
	}
	if operationErr != nil {
		output["error"] = operationErr.Error()
	}
	body, marshalErr := json.MarshalIndent(output, "", "  ")
	if marshalErr != nil {
		return errors.Join(operationErr, marshalErr)
	}
	fmt.Println(string(body))
	return operationErr
}

func runImageRecordUpdate(ctx context.Context, cloud, imageID, name string) error {
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

	updated, err := conn.UpdateImageRecord(ctx, image.ImageRecordUpdateRequest{
		ID:         imageID,
		Attributes: map[string]any{"name": name},
	}, request.Header("X-Request-Source", "image-record-update-example"))
	if err := printRecordImageRecordUpdate("update by ID", updated, err); err != nil {
		return err
	}

	// name has no descriptor conversion: use the returned raw JSON spelling.
	sameName := json.RawMessage(updated.Resource.Body["name"])
	unchanged, err := service.UpdateImageRecord(ctx, image.ImageRecordUpdateRequest{
		Record: updated,
	}, image.WithImageRecordAttributes(map[string]any{"name": sameName}))
	return printRecordImageRecordUpdate("reuse record with same name", unchanged, err)
}

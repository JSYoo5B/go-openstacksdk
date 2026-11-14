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
	"github.com/gophercloud/gophercloud/v2"
)

func Example_imageRecordActions() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
	imageID := flag.String("image-id", "", "required literal image ID")
	action := flag.String("action", "deactivate", "deactivate or reactivate")
	fromRecord := flag.Bool("record", false, "explicitly fetch an SDK record first")
	viaService := flag.Bool("service", false, "use image.Service for the selected POST")
	flag.Parse()
	if *imageID == "" {
		log.Fatal("-image-id is required")
	}
	if *action != "deactivate" && *action != "reactivate" {
		log.Fatal("-action must be deactivate or reactivate")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := runImageRecordActions(ctx, *cloud, *imageID, *action, *fromRecord, *viaService); err != nil {
		var native gophercloud.ErrUnexpectedResponseCode
		if errors.As(err, &native) {
			fmt.Fprintf(os.Stderr, "native HTTP %d, response bytes=%d\n", native.Actual, len(native.Body))
		}
		var response *resource.ResponseError
		if errors.As(err, &response) {
			fmt.Fprintf(os.Stderr, "accepted HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
		}
		log.Fatal(err)
	}
}

func runImageRecordActions(ctx context.Context, cloud, imageID, action string, fromRecord, viaService bool) error {
	conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
	if err != nil {
		return err
	}
	input := image.ImageRecordActionRequest{ID: imageID}
	var service *image.Service
	if fromRecord || viaService {
		service, err = conn.Image(ctx)
		if err != nil {
			return err
		}
	}
	if fromRecord {
		record, err := service.GetImageRecord(ctx, image.ImageRecordRequest{ID: imageID})
		if err != nil {
			return err
		}
		input = image.ImageRecordActionRequest{Record: record}
	}
	options := []image.ImageRecordActionOption{
		image.WithImageRecordActionOpts(image.ImageRecordActionOpts{
			Headers: map[string]string{"X-Request-Source": "image-record-actions-example"},
		}),
		request.Header("X-SDK-Example-Action", action),
		image.WithImageRecordActionHeaders(map[string]string{"X-SDK-Example-Mode": "owned"}),
	}
	var result *image.ImageRecordActionResult
	if viaService {
		if action == "deactivate" {
			result, err = service.DeactivateImageRecord(ctx, input, options...)
		} else {
			result, err = service.ReactivateImageRecord(ctx, input, options...)
		}
	} else if action == "deactivate" {
		result, err = conn.DeactivateImageRecord(ctx, input, options...)
	} else {
		result, err = conn.ReactivateImageRecord(ctx, input, options...)
	}
	return printActionImageRecordActions(result, err)
}

func printActionImageRecordActions(result *image.ImageRecordActionResult, operationErr error) error {
	value := map[string]any{"partial": result != nil && operationErr != nil}
	if result != nil {
		value["acknowledgement"] = result.Acknowledgement
		if result.Record != nil {
			value["record"] = map[string]any{
				"resource":                    result.Record.Resource,
				"wire":                        result.Record.Wire,
				"earlier_receipt_status_code": result.Record.StatusCode,
				"earlier_receipt_body_bytes":  len(result.Record.Envelope),
				"import_methods":              result.Record.ImportMethods,
			}
		}
	}
	if operationErr != nil {
		value["error"] = operationErr.Error()
	}
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return errors.Join(operationErr, err)
	}
	fmt.Println(string(body))
	return operationErr
}

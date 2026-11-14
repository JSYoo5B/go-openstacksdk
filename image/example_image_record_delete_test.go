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

func Example_imageRecordDelete() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
	imageID := flag.String("image-id", "", "required literal image ID")
	storeID := flag.String("store", "", "optional explicit store ID; omission deletes whole image")
	ignoreMissing := flag.Bool("ignore-missing", true, "ignore a clean final native404")
	fromRecord := flag.Bool("record", false, "explicitly fetch an SDK record before deletion")
	viaService := flag.Bool("service", false, "use image.Service instead of Connection for the DELETE")
	flag.Parse()
	storeSelected := false
	flag.Visit(func(value *flag.Flag) {
		if value.Name == "store" {
			storeSelected = true
		}
	})
	if *imageID == "" {
		log.Fatal("-image-id is required")
	}
	if storeSelected && *storeID == "" {
		log.Fatal("an explicitly selected store must have a nonempty ID")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := runImageRecordDelete(ctx, *cloud, *imageID, *storeID, storeSelected,
		*ignoreMissing, *fromRecord, *viaService); err != nil {
		var response *resource.ResponseError
		if errors.As(err, &response) {
			fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
		}
		log.Fatal(err)
	}
}

func printDeleteImageRecordDelete(result *image.ImageRecordDeleteResult, operationErr error) error {
	value := map[string]any{"partial": result != nil && operationErr != nil}
	if result == nil && operationErr == nil {
		value["ignored_missing"] = true
	}
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

func runImageRecordDelete(ctx context.Context, cloud, imageID, storeID string,
	storeSelected, ignoreMissing, fromRecord, viaService bool) error {
	conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
	if err != nil {
		return err
	}
	input := image.ImageRecordDeleteRequest{ID: imageID}
	if fromRecord {
		service, err := conn.Image(ctx)
		if err != nil {
			return err
		}
		record, err := service.GetImageRecord(ctx, image.ImageRecordRequest{ID: imageID})
		if err != nil {
			return err
		}
		input = image.ImageRecordDeleteRequest{Record: record}
	}
	options := []image.ImageRecordDeleteOption{
		request.IgnoreMissing(ignoreMissing),
		request.Header("X-Request-Source", "image-record-delete-example"),
	}
	if storeSelected {
		options = append(options, image.WithImageRecordDeleteStoreID(storeID))
	}
	var result *image.ImageRecordDeleteResult
	if viaService {
		service, err := conn.Image(ctx)
		if err != nil {
			return err
		}
		result, err = service.DeleteImageRecord(ctx, input, options...)
		return printDeleteImageRecordDelete(result, err)
	}
	result, err = conn.DeleteImageRecord(ctx, input, options...)
	return printDeleteImageRecordDelete(result, err)
}

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

func Example_imageRecordStage() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
	imageID := flag.String("image-id", "", "required literal image ID")
	filename := flag.String("filename", "", "required binary image filename")
	borrow := flag.Bool("borrow", false, "open a caller-owned file and lend its reader")
	viaService := flag.Bool("service", false, "call image.Service instead of Connection")
	infer := flag.Bool("infer-size", true, "infer total seekable length and restore cursor")
	size := flag.Int64("size", 0, "explicit signed X-OpenStack-Image-Size")
	flag.Parse()
	explicitSize := false
	flag.Visit(func(value *flag.Flag) {
		if value.Name == "size" {
			explicitSize = true
		}
	})
	if *imageID == "" || *filename == "" {
		log.Fatal("-image-id and -filename are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := runImageRecordStage(ctx, *cloud, *imageID, *filename, *borrow, *viaService, *infer, *size, explicitSize); err != nil {
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

func runImageRecordStage(ctx context.Context, cloud, imageID, filename string, borrow, viaService, infer bool, size int64, explicitSize bool) (err error) {
	conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
	if err != nil {
		return err
	}
	service, err := conn.Image(ctx)
	if err != nil {
		return err
	}
	record, err := service.GetImageRecord(ctx, image.ImageRecordRequest{ID: imageID})
	if err != nil {
		return err
	}
	input := image.ImageRecordStageRequest{Record: record, Filename: filename}
	mode := "filename"
	if borrow {
		data, openErr := os.Open(filename)
		if openErr != nil {
			return openErr
		}
		defer func() { err = errors.Join(err, data.Close()) }()
		input = image.ImageRecordStageRequest{Record: record, Data: data}
		mode = "borrowed-reader"
	}
	options := []image.ImageRecordStageOption{
		image.WithImageRecordStageOpts(image.ImageRecordStageOpts{
			Headers: map[string]string{"X-Request-Source": "image-record-stage-example"},
		}),
		request.Header("X-SDK-Example-Mode", mode),
		image.WithImageRecordStageHeaders(map[string]string{"X-SDK-Example-Operation": "stage"}),
		image.WithImageRecordStageSizeInference(infer),
	}
	if explicitSize {
		options = append(options, request.Custom(func(c *image.ImageRecordStageOpts) error { v := size; c.Size = &v; return nil }))
	} else {
		options = append(options, request.Custom(func(c *image.ImageRecordStageOpts) error { c.Size = nil; return nil }))
	}
	var result *image.ImageRecordStageResult
	if viaService {
		result, err = service.StageImageRecord(ctx, input, options...)
	} else {
		result, err = conn.StageImageRecord(ctx, input, options...)
	}
	return printStageImageRecordStage(result, err)
}

func printStageImageRecordStage(result *image.ImageRecordStageResult, operationErr error) error {
	value := map[string]any{"partial": result != nil && operationErr != nil}
	if result != nil {
		value["staged"] = result.Staged
		value["metadata"] = result.Metadata
		if result.Record != nil {
			value["record"] = map[string]any{
				"resource":       result.Record.Resource,
				"wire":           result.Record.Wire,
				"status_code":    result.Record.StatusCode,
				"envelope_bytes": len(result.Record.Envelope),
				"import_methods": result.Record.ImportMethods,
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

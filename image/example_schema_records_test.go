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
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func Example_schemaRecords() {
	cloudName := flag.String("cloud", "dev", "clouds.yaml cloud name")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := runSchemaRecords(ctx, *cloudName); err != nil {
		log.Fatal(err)
	}
}

func printRecordSchemaRecords(record *image.SchemaRecord) error {
	if record == nil {
		return nil
	}
	data, err := json.MarshalIndent(map[string]any{
		"kind": record.Kind, "resource": record.Resource, "wire": record.Wire,
		"envelope": string(record.Envelope),
		"header":   record.Header, "status_code": record.StatusCode,
	}, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func printFailureSchemaRecords(err error) error {
	var data map[string]any
	var accepted *resource.ResponseError
	var unexpected gophercloud.ErrUnexpectedResponseCode
	switch {
	case errors.As(err, &accepted):
		data = map[string]any{
			"status_code": accepted.StatusCode, "header": accepted.Header,
			"body": string(accepted.Body),
		}
	case errors.As(err, &unexpected):
		data = map[string]any{
			"status_code": unexpected.Actual, "header": unexpected.ResponseHeader,
			"body": string(unexpected.Body),
		}
	default:
		return nil
	}
	raw, printErr := json.MarshalIndent(data, "", "  ")
	if printErr != nil {
		return printErr
	}
	fmt.Println(string(raw))
	return nil
}

func runSchemaRecords(ctx context.Context, cloudName string) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
	if err != nil {
		return err
	}
	service, err := conn.Image(ctx)
	if err != nil {
		return err
	}
	options := []image.GetSchemaOption{
		request.Header("X-Request-Source", "schema-record-example"),
	}
	for _, kind := range []image.SchemaKind{image.SchemaMetadefNamespace, image.SchemaTask} {
		record, readErr := service.GetSchemaRecord(ctx, kind, options...)
		if printErr := printRecordSchemaRecords(record); printErr != nil {
			return errors.Join(readErr, printErr)
		}
		if readErr != nil {
			return errors.Join(readErr, printFailureSchemaRecords(readErr))
		}
	}
	return nil
}

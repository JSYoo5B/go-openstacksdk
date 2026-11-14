package image_test

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"log"
	"time"

	"github.com/JSYoo5B/go-openstacksdk"
	properties "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func Example_metadefPropertyRecordList() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
	namespace := flag.String("namespace", "OS::Compute::Libvirt", "literal namespace")
	maxRows := flag.Int("max-rows", 0, "maximum consumed dictionary entries")
	minLength := flag.Int("min-length", 0, "local canonical min_length filter")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := runMetadefPropertyRecordList(ctx, *cloud, *namespace, *maxRows, *minLength); err != nil {
		log.Fatal(err)
	}
}

func printRecordMetadefPropertyRecordList(record *properties.Record) error {
	data, err := json.MarshalIndent(map[string]any{
		"key": record.Key, "namespace": record.Namespace,
		"resource": record.Resource, "wire": record.Wire,
		"envelope": string(record.Envelope),
		"header":   record.Header, "status_code": record.StatusCode,
	}, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func printFailureMetadefPropertyRecordList(err error) error {
	var accepted *resource.ResponseError
	var unexpected gophercloud.ErrUnexpectedResponseCode
	var receipt map[string]any
	switch {
	case errors.As(err, &accepted):
		receipt = map[string]any{
			"status_code": accepted.StatusCode, "header": accepted.Header,
			"body": string(accepted.Body),
		}
	case errors.As(err, &unexpected):
		receipt = map[string]any{
			"status_code": unexpected.Actual, "header": unexpected.ResponseHeader,
			"body": string(unexpected.Body),
		}
	default:
		return nil
	}
	data, printErr := json.MarshalIndent(receipt, "", "  ")
	if printErr != nil {
		return printErr
	}
	fmt.Println(string(data))
	return nil
}

func runMetadefPropertyRecordList(ctx context.Context, cloud, namespace string, maxRows, minLength int) error {
	conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
	if err != nil {
		return err
	}
	service, err := conn.Image(ctx)
	if err != nil {
		return err
	}
	scope, err := service.API.MetadefProperties.InNamespace(ctx, namespace)
	if err != nil {
		return err
	}
	records, readErr := scope.AllRecords(ctx,
		properties.WithRecordListOpts(properties.RecordListOpts{MaxItems: maxRows}),
		properties.WithRecordListFilter("min_length", minLength),
		request.Header("X-Request-Source", "metadef-property-list-example"),
	)
	for _, record := range records {
		if err := printRecordMetadefPropertyRecordList(record); err != nil {
			return errors.Join(readErr, err)
		}
	}
	if readErr != nil {
		return errors.Join(readErr, printFailureMetadefPropertyRecordList(readErr))
	}
	return nil
}

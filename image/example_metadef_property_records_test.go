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

func Example_metadefPropertyRecords() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
	namespace := flag.String("namespace", "OS::Compute::Libvirt", "literal namespace")
	id := flag.String("id", "hw_cpu_policy", "literal property id")
	title := flag.String("seed-title", "fallback title", "title retained when response omits it")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := runMetadefPropertyRecords(ctx, *cloud, *namespace, *id, *title); err != nil {
		log.Fatal(err)
	}
}

func printFailureMetadefPropertyRecords(err error) error {
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

func runMetadefPropertyRecords(ctx context.Context, cloud, namespace, id, title string) error {
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
	record, err := scope.GetRecord(ctx, properties.RecordRequest{ID: id},
		properties.WithRecordGetAttributes(map[string]any{
			"title": title, "min_length": 0,
		}),
		request.Header("X-Request-Source", "metadef-property-record-example"),
	)
	if err != nil {
		return errors.Join(err, printFailureMetadefPropertyRecords(err))
	}
	data, err := json.MarshalIndent(map[string]any{
		"namespace": record.Namespace,
		"resource":  record.Resource, "wire": record.Wire,
		"envelope": string(record.Envelope),
		"header":   record.Header, "status_code": record.StatusCode,
	}, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

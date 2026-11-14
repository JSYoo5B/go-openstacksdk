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

func Example_metadefPropertyRecordWrite() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
	namespace := flag.String("namespace", "OS::Compute::Libvirt", "literal namespace")
	id := flag.String("id", "hw_cpu_policy", "property name")
	action := flag.String("action", "snapshot", "snapshot, create, or update")
	title := flag.String("title", "CPU Policy", "property title")
	description := flag.String("description", "CPU allocation policy", "property description")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := runMetadefPropertyRecordWrite(ctx, *cloud, *namespace, *id, *action, *title, *description); err != nil {
		log.Fatal(err)
	}
}

func printFailureMetadefPropertyRecordWrite(err error) error {
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

func runMetadefPropertyRecordWrite(ctx context.Context, cloud, namespace, id, action, title, description string) error {
	if action != "snapshot" && action != "create" && action != "update" {
		return fmt.Errorf("unknown action %q", action)
	}
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
	attributes := map[string]any{
		"name": id, "type": "string", "title": title,
		"description": description, "is_readonly": false, "min_length": 0,
	}
	var record *properties.Record
	switch action {
	case "create":
		record, err = scope.CreateRecord(ctx,
			properties.WithRecordCreateAttributes(attributes),
			request.Header("X-Request-Source", "metadef-property-write-example"),
		)
	case "update":
		record, err = scope.UpdateRecord(ctx, properties.RecordRequest{ID: id},
			properties.WithRecordUpdateAttributes(attributes),
			request.Header("X-Request-Source", "metadef-property-write-example"),
		)
	default:
		record, err = scope.UpdateRecord(ctx, properties.RecordRequest{ID: id},
			request.Header("X-Request-Source", "metadef-property-write-example"),
		)
	}
	if err != nil {
		return errors.Join(err, printFailureMetadefPropertyRecordWrite(err))
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

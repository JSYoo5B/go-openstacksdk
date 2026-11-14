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
	types "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func Example_metadefResourceTypeAssociationMutations() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
	namespace := flag.String("namespace", "OS::Compute::Libvirt", "literal namespace")
	resourceType := flag.String("resource-type", "OS::Nova::Server", "resource type name")
	action := flag.String("action", "list", "list, create, or delete")
	prefix := flag.String("prefix", "hw:", "association property prefix")
	target := flag.String("properties-target", "metadata", "association target")
	strictMissing := flag.Bool("strict-missing", false, "return an error for deletion404")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := runMetadefResourceTypeAssociationMutations(ctx, *cloud, *namespace, *resourceType, *action, *prefix, *target, *strictMissing); err != nil {
		log.Fatal(err)
	}
}

func printJSONMetadefResourceTypeAssociationMutations(value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func printRecordMetadefResourceTypeAssociationMutations(record *types.Record) error {
	return printJSONMetadefResourceTypeAssociationMutations(map[string]any{
		"namespace": record.Namespace,
		"resource":  record.Resource, "wire": record.Wire,
		"envelope": string(record.Envelope),
		"header":   record.Header, "status_code": record.StatusCode,
	})
}

func printFailureMetadefResourceTypeAssociationMutations(err error) error {
	var accepted *resource.ResponseError
	var unexpected gophercloud.ErrUnexpectedResponseCode
	switch {
	case errors.As(err, &accepted):
		return printJSONMetadefResourceTypeAssociationMutations(map[string]any{
			"status_code": accepted.StatusCode, "header": accepted.Header,
			"body_base64": accepted.Body,
		})
	case errors.As(err, &unexpected):
		return printJSONMetadefResourceTypeAssociationMutations(map[string]any{
			"status_code": unexpected.Actual, "header": unexpected.ResponseHeader,
			"body_base64": unexpected.Body,
		})
	default:
		return nil
	}
}

func runMetadefResourceTypeAssociationMutations(ctx context.Context, cloud, namespace, resourceType, action, prefix, target string, strictMissing bool) error {
	if action != "list" && action != "create" && action != "delete" {
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
	scope, err := service.API.MetadefResourceTypes.InNamespace(ctx, namespace)
	if err != nil {
		return err
	}
	switch action {
	case "create":
		record, createErr := scope.CreateRecord(ctx,
			types.WithRecordCreateAttributes(map[string]any{
				"name": resourceType, "prefix": prefix, "properties_target": target,
			}),
			request.Header("X-Request-Source", "association-example"),
		)
		if createErr != nil {
			return errors.Join(createErr, printFailureMetadefResourceTypeAssociationMutations(createErr))
		}
		return printRecordMetadefResourceTypeAssociationMutations(record)
	case "delete":
		ack, deleteErr := scope.DeleteRecord(ctx, types.RecordRequest{ID: resourceType},
			request.IgnoreMissing(!strictMissing),
			request.Header("X-Request-Source", "association-example"),
		)
		if ack != nil {
			if err := printJSONMetadefResourceTypeAssociationMutations(ack); err != nil {
				return errors.Join(deleteErr, err)
			}
		}
		if deleteErr != nil {
			return errors.Join(deleteErr, printFailureMetadefResourceTypeAssociationMutations(deleteErr))
		}
		return nil
	default:
		records, readErr := scope.AllRecords(ctx,
			request.Paginated(false),
			request.Header("X-Request-Source", "association-example"),
		)
		for _, record := range records {
			if err := printRecordMetadefResourceTypeAssociationMutations(record); err != nil {
				return errors.Join(readErr, err)
			}
		}
		if readErr != nil {
			return errors.Join(readErr, printFailureMetadefResourceTypeAssociationMutations(readErr))
		}
		return nil
	}
}

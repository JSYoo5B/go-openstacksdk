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
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func Example_string2() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
	namespace := flag.String("namespace", "OS::Compute::Libvirt", "literal namespace")
	name := flag.String("name", "hw_cpu_policy", "literal property name")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := runMetadefProperty1(ctx, *cloud, *namespace, *name); err != nil {
		log.Fatal(err)
	}
}

func printFailureMetadefProperty1(err error) error {
	var accepted *resource.ResponseError
	var unexpected gophercloud.ErrUnexpectedResponseCode
	var receipt map[string]any
	switch {
	case errors.As(err, &accepted):
		receipt = map[string]any{"status_code": accepted.StatusCode,
			"header": accepted.Header, "body": string(accepted.Body)}
	case errors.As(err, &unexpected):
		receipt = map[string]any{"status_code": unexpected.Actual,
			"header": unexpected.ResponseHeader, "body": string(unexpected.Body)}
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

func runMetadefProperty1(ctx context.Context, cloud, namespace, name string) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
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
	value, err := scope.Get(ctx, name)
	if err != nil {
		return errors.Join(err, printFailureMetadefProperty1(err))
	}
	data, err := json.MarshalIndent(value.Body, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("namespace=%q requested_name=%q status_code=%d\n%s\n",
		scope.NamespaceName(), name, value.StatusCode, data)
	return nil
}

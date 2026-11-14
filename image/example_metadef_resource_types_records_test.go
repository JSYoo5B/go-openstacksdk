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
)

func Example_metadefResourceTypesRecords() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
	namespace := flag.String("namespace", "OS::Compute::Libvirt", "literal namespace")
	name := flag.String("name", "", "optional association name filter")
	limit := flag.Int("limit", 0, "explicit server page limit")
	maxItems := flag.Int("max-items", 0, "maximum consumed rows per list")
	paginated := flag.Bool("paginated", false, "follow list continuations")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := runMetadefResourceTypesRecords(ctx, *cloud, *namespace, *name, *limit, *maxItems, *paginated); err != nil {
		log.Fatal(err)
	}
}

func printRecordMetadefResourceTypesRecords(record *types.Record) error {
	if record == nil {
		return nil
	}
	data, err := json.MarshalIndent(map[string]any{
		"namespace":   record.Namespace,
		"resource":    record.Resource,
		"wire":        record.Wire,
		"envelope":    record.Envelope,
		"header":      record.Header,
		"status_code": record.StatusCode,
	}, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func runMetadefResourceTypesRecords(ctx context.Context, cloud, namespace, name string, limit, maxItems int, paginated bool) error {
	conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
	if err != nil {
		return err
	}
	service, err := conn.Image(ctx)
	if err != nil {
		return err
	}
	api := service.API.MetadefResourceTypes
	options := []types.RecordListOption{
		types.WithRecordListOpts(types.RecordListOpts{Limit: limit, MaxItems: maxItems}),
		request.Header("X-Request-Source", "metadef-records-example"),
		request.Paginated(paginated),
	}
	for record, readErr := range api.ListRecords(ctx, options...) {
		if readErr != nil {
			return readErr
		}
		if err := printRecordMetadefResourceTypesRecords(record); err != nil {
			return err
		}
	}
	scope, err := api.InNamespace(ctx, namespace)
	if err != nil {
		return err
	}
	scopedOptions := []types.RecordListOption{
		types.WithRecordListOpts(types.RecordListOpts{
			Limit: limit, MaxItems: maxItems,
			Filters: associationFiltersMetadefResourceTypesRecords(name),
		}),
		request.Header("X-Request-Source", "metadef-records-example"),
		request.Paginated(paginated),
	}
	records, readErr := scope.AllRecords(ctx, scopedOptions...)
	for _, record := range records {
		if err := printRecordMetadefResourceTypesRecords(record); err != nil {
			return errors.Join(readErr, err)
		}
	}
	return readErr
}

func associationFiltersMetadefResourceTypesRecords(name string) []resource.ListOption {
	if name == "" {
		return nil
	}
	return []resource.ListOption{resource.WithFilter("name", name)}
}

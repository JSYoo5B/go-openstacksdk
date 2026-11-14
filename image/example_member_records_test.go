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

func Example_memberRecords() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
	imageID := flag.String("image", "", "literal parent image ID")
	find := flag.String("find", "", "optional member ID or inherited name")
	status := flag.String("status", "", "optional local member status filter")
	limit := flag.Int("limit", 0, "optional server page limit")
	maxItems := flag.Int("max-items", 0, "maximum consumed rows before filtering")
	flag.Parse()
	if *imageID == "" {
		log.Fatal("-image is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := runMemberRecords(ctx, *cloud, *imageID, *find, *status, *limit, *maxItems); err != nil {
		var response *resource.ResponseError
		if errors.As(err, &response) {
			fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
		}
		log.Fatal(err)
	}
}

func printRecordMemberRecords(record *image.ImageMemberRecord) error {
	value := map[string]any{"resource": record.Resource, "status_code": record.StatusCode}
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(body))
	return nil
}

func runMemberRecords(ctx context.Context, cloud, imageID, find, status string, limit, maxItems int) error {
	conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
	if err != nil {
		return err
	}
	service, err := conn.Image(ctx)
	if err != nil {
		return err
	}
	parent := resource.ID(imageID)
	options := []image.ImageMemberRecordListOption{
		image.WithImageMemberRecordListOpts(image.ImageMemberRecordListOpts{Limit: limit, MaxItems: maxItems}),
		request.Header("X-Request-Source", "member-records-example"),
	}
	if status != "" {
		options = append(options, image.WithImageMemberRecordListFilter("status", status))
	}
	for record, readErr := range service.ListImageMemberRecords(ctx, parent, options...) {
		if readErr != nil {
			return readErr
		}
		if err := printRecordMemberRecords(record); err != nil {
			return err
		}
	}
	if find != "" {
		record, err := service.FindImageMemberRecord(ctx, parent, find,
			request.IgnoreMissing(false))
		if err != nil {
			return err
		}
		return printRecordMemberRecords(record)
	}
	return nil
}

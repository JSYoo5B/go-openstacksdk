package v2_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/image/v2/images"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_imageTasks() {
	cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud name")
	imageID := flag.String("image", "", "literal image ID")
	status := flag.String("status", "", "optional local status filter")
	limit := flag.Int("limit", -1, "server page hint; -1 omits it")
	cap := flag.Int("max-items", 0, "raw rows consumed before local filtering; 0 is unlimited")
	paginated := flag.Bool("paginated", true, "follow retained continuation")
	flag.Parse()
	if err := runImageTasks(*cloud, *imageID, *status, *limit, *cap, *paginated); err != nil {
		log.Fatal(err)
	}
}

func runImageTasks(cloud, imageID, status string, limit, maxItems int, paginated bool) error {
	if imageID == "" {
		return fmt.Errorf("-image is required")
	}
	if limit < -1 {
		return fmt.Errorf("-limit must be -1 or nonnegative")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
	if err != nil {
		return err
	}
	service, err := conn.ImageV2(ctx)
	if err != nil {
		return err
	}
	options := []images.TaskRecordsOption{
		request.Headers(map[string]string{"X-Request-Source": "image-tasks-example"}),
		request.MaxItems(maxItems),
		request.Paginated(paginated),
	}
	if limit >= 0 {
		options = append(options, request.Limit(limit))
	}
	if status != "" {
		options = append(options, request.Query("status", status))
	}
	out := json.NewEncoder(os.Stdout)
	for record, readErr := range service.Images.TaskRecords(ctx, imageID, options...) {
		if readErr != nil {
			return readErr
		}
		if err := out.Encode(map[string]any{
			"resource":     record.Resource.Body,
			"wire":         record.Wire.Body,
			"envelope":     record.Envelope,
			"headers":      record.Header,
			"status_code":  record.StatusCode,
			"microversion": record.Microversion,
			"discovery":    record.Discovery,
		}); err != nil {
			return err
		}
	}
	return nil
}

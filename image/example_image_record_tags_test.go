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

func Example_imageRecordTags() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
	imageID := flag.String("image-id", "", "required literal image ID")
	tag := flag.String("tag", "reviewed", "literal tag to add and then remove")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := runImageRecordTags(ctx, *cloud, *imageID, *tag); err != nil {
		var response *resource.ResponseError
		if errors.As(err, &response) {
			fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
		}
		log.Fatal(err)
	}
}

func printResultImageRecordTags(value *image.ImageRecordTagResult, operationErr error) error {
	if value != nil {
		output := make(map[string]any)
		if ack := value.Acknowledgement; ack != nil {
			output["acknowledgement"] = map[string]any{
				"image_id": ack.ImageID, "tag": ack.Tag,
				"status_code": ack.StatusCode, "body_bytes": len(ack.Body),
			}
		}
		if value.Record != nil {
			output["resource"] = value.Record.Resource
			output["wire"] = value.Record.Wire
			output["record_status_code"] = value.Record.StatusCode
		}
		body, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return errors.Join(operationErr, err)
		}
		fmt.Println(string(body))
	}
	return operationErr
}

func runImageRecordTags(ctx context.Context, cloud, imageID, tag string) error {
	if imageID == "" {
		return fmt.Errorf("-image-id is required")
	}
	conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
	if err != nil {
		return err
	}
	service, err := conn.Image(ctx)
	if err != nil {
		return err
	}

	added, err := service.AddImageRecordTag(ctx, image.ImageRecordTagRequest{ID: imageID}, tag,
		request.Header("X-Request-Source", "image-record-tags-example"))
	if err := printResultImageRecordTags(added, err); err != nil {
		return err
	}

	seed, err := service.GetImageRecord(ctx, image.ImageRecordRequest{ID: imageID})
	if err != nil {
		return err
	}
	removed, err := service.RemoveImageRecordTag(ctx, image.ImageRecordTagRequest{Record: seed}, tag,
		image.WithImageMutationHeaders(map[string]string{"X-Request-Source": "image-record-tags-example"}))
	return printResultImageRecordTags(removed, err)
}

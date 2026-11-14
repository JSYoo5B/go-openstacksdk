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

func Example_imageRecordWaits() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
	imageID := flag.String("image-id", "", "required image ID")
	target := flag.String("target", "active", "target status")
	waitDelete := flag.Bool("wait-delete", false, "also observe deletion; does not request deletion")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err := runImageRecordWaits(ctx, *cloud, *imageID, *target, *waitDelete); err != nil {
		var response *resource.ResponseError
		if errors.As(err, &response) {
			fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
		}
		log.Fatal(err)
	}
}

func printRecordImageRecordWaits(value *image.ImageRecord) error {
	body, err := json.MarshalIndent(map[string]any{
		"resource": value.Resource, "wire": value.Wire,
		"status_code": value.StatusCode, "import_methods": value.ImportMethods,
	}, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(body))
	return nil
}

func waitResultImageRecordWaits(value *image.ImageRecord, waitErr error) error {
	if waitErr != nil {
		if value != nil {
			fmt.Fprintln(os.Stderr, "wait incomplete; latest projected observation follows")
			if err := printRecordImageRecordWaits(value); err != nil {
				return errors.Join(waitErr, err)
			}
		}
		return waitErr
	}
	return printRecordImageRecordWaits(value)
}

func runImageRecordWaits(ctx context.Context, cloud, imageID, target string, waitDelete bool) error {
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
	seed, err := service.GetImageRecord(ctx, image.ImageRecordRequest{ID: imageID})
	if err != nil {
		return err
	}

	callback := image.WithImageRecordWait(resource.WithProgressCallback(func(progress int) error {
		fmt.Fprintf(os.Stderr, "progress: %d\n", progress)
		return nil
	}))
	ready, err := service.WaitForImageRecordStatus(ctx, seed, target,
		image.WithImageRecordWait(resource.WithTimeout(2*time.Minute)),
		image.WithImageRecordWait(resource.WithPollInterval(2*time.Second)),
		callback,
		request.Header("X-Request-Source", "image-record-waits-example"))
	if err := waitResultImageRecordWaits(ready, err); err != nil {
		return err
	}

	if waitDelete {
		last, err := service.WaitForImageRecordDelete(ctx, ready,
			callback)
		if err := waitResultImageRecordWaits(last, err); err != nil {
			return err
		}
		fmt.Println("deletion observed; returned record retains the last successful receipt")
	}
	return nil
}

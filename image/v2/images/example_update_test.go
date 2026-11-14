package images_test

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
	"github.com/JSYoo5B/go-openstacksdk/image/v2/images"
	"github.com/gophercloud/gophercloud/v2"
)

func Example_update() {
	cloud := flag.String("cloud", "", "required clouds.yaml cloud name")
	imageID := flag.String("image-id", "", "required native image ID")
	name := flag.String("name", "release-image", "new image name")
	tag := flag.String("tag", "reviewed", "one tag replacing the whole tag set")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := runUpdate(ctx, *cloud, *imageID, *name, *tag); err != nil {
		var native gophercloud.ErrUnexpectedResponseCode
		if errors.As(err, &native) {
			fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", native.Actual, len(native.Body))
		}
		log.Fatal(err)
	}
}

func runUpdate(ctx context.Context, cloud, imageID, name, tag string) error {
	if cloud == "" || imageID == "" {
		return fmt.Errorf("-cloud and -image-id are required")
	}
	conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
	if err != nil {
		return err
	}
	service, err := conn.Image(ctx)
	if err != nil {
		return err
	}

	patches := images.UpdateOpts{
		images.ReplaceImageName{NewName: name},
		images.UpdateVisibility{Visibility: images.ImageVisibilityPrivate},
		images.ReplaceImageHidden{NewHidden: false},
		images.ReplaceImageProtected{NewProtected: false},
		images.ReplaceImageMinDisk{NewMinDisk: 0},
		images.ReplaceImageMinRam{NewMinRam: 0},
		images.ReplaceImageTags{NewTags: []string{tag}},
		images.UpdateImageProperty{Op: images.AddOp, Name: "sdk_demo_build", Value: "draft"},
		images.UpdateImageProperty{Op: images.ReplaceOp, Name: "sdk_demo_build", Value: "ready"},
		images.UpdateImageProperty{Op: images.AddOp, Name: "sdk_demo_temporary", Value: "temporary"},
		images.UpdateImageProperty{Op: images.RemoveOp, Name: "sdk_demo_temporary"},
	}
	// WithUpdateOptions replaces the base nil array with these concrete patches.
	value, operationErr := service.API.Images.Update(ctx, imageID, nil, images.WithUpdateOptions(patches))
	output := map[string]any{
		"image":   value,
		"partial": value != nil && operationErr != nil,
	}
	if operationErr != nil {
		output["error"] = operationErr.Error()
	}
	body, marshalErr := json.MarshalIndent(output, "", "  ")
	if marshalErr != nil {
		return errors.Join(operationErr, marshalErr)
	}
	fmt.Println(string(body))
	return operationErr
}

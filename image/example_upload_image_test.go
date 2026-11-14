package image_test

import (
	"context"
	"encoding/json"
	"os"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func uploadFileUploadImage(ctx context.Context, service *image.Service, filename string,
	knownSize *int64, owner *string) (*image.ImageUploadResult, error) {
	source, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer source.Close()

	options := []image.ImageUploadOption{
		image.WithImageUploadOpts(image.ImageUploadOpts{
			Headers: map[string]string{"X-Request-Label": "example"},
			Fields:  map[string]json.RawMessage{"build": json.RawMessage(`"initial"`)},
		}),
		image.WithImageUploadHeaders(map[string]string{"X-Trace-Label": "upload"}),
		request.Header("X-Request-Label", "direct-upload"),
		image.WithImageUploadFields(map[string]json.RawMessage{
			"build": json.RawMessage(`"release"`),
		}),
		image.WithImageUploadField("build", "2026-10-03"),
		image.WithImageUploadField("disk_format", "qcow2"),
		image.WithImageUploadField("container_format", "bare"),
		image.WithImageUploadField("visibility", image.VisibilityPrivate),
		image.WithImageUploadField("protected", false),
		image.WithImageUploadField("os_hidden", false),
		image.WithImageUploadField("min_disk", 0),
		image.WithImageUploadField("min_ram", 0),
		image.WithImageUploadTags("example"),
	}
	if owner != nil {
		options = append(options, image.WithImageUploadField("owner", *owner))
	}
	if knownSize == nil {
		options = append(options, request.Custom(func(c *image.ImageUploadOpts) error { c.Size = nil; return nil }))
	} else {
		options = append(options, request.Custom(func(c *image.ImageUploadOpts) error { v := *knownSize; c.Size = &v; return nil }))
	}
	result, err := service.UploadImage(ctx,
		image.UploadImageRequest{Name: "upload-example", Data: source}, options...)
	return result, err
}

func Example_uploadImage() {}

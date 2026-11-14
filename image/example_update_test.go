package image_test

import (
	"context"
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func updateQueuedImageExampleUpdate(ctx context.Context, svc *image.Service, imageID, ownerID string) (*image.ImageInfo, error) {
	return svc.UpdateImage(ctx, resource.ID(imageID),
		image.WithUpdateImageOpts(image.UpdateImageOpts{
			Headers: map[string]string{"X-Example": "ordered-image-patch"},
		}),
		image.WithUpdateImageHeaders(map[string]string{"X-Trace-Id": "image-update"}),
		request.Header("X-Request-Id", "image-update-example"),
		image.WithUpdateImageChanges(image.ImagePatch{
			Op: "add", Path: "/build", Value: json.RawMessage(`"draft"`),
		}),
		image.WithUpdateImageChange(image.ImagePatch{
			Op: "replace", Path: "/build", Value: json.RawMessage(`"ready"`),
		}),
		image.WithUpdateImageField("name", "release-image"),
		image.WithUpdateImageField("visibility", "private"),
		image.WithUpdateImageField("protected", false),
		image.WithUpdateImageField("os_hidden", false),
		image.WithUpdateImageField("owner", ownerID),
		image.WithUpdateImageField("container_format", "bare"),
		image.WithUpdateImageField("disk_format", "qcow2"),
		image.WithUpdateImageField("min_disk", 0),
		image.WithUpdateImageField("min_ram", 0),
		image.WithUpdateImageTags("release", "ready"),
		image.WithUpdateImageField("build~origin", "ci"),
		image.WithUpdateImageFields(map[string]json.RawMessage{
			"build/id": json.RawMessage(`"2026.10"`),
			"legacy":   json.RawMessage(`"temporary"`),
		}),
		image.WithUpdateImageRemoveField("legacy"),
	)
}

func setImagePropertiesExampleUpdate(ctx context.Context, svc *image.Service, imageID string) (*image.ImageInfo, error) {
	return svc.SetImageProperties(ctx, resource.ID(imageID),
		image.WithSetImagePropertiesOpts(image.SetImagePropertiesOpts{
			Headers: map[string]string{"X-Example": "property-upsert"},
		}),
		image.WithSetImagePropertiesHeaders(map[string]string{"X-Trace-Id": "image-properties"}),
		request.Header("X-Request-Id", "image-properties-example"),
		image.WithSetImagePropertiesProperties(map[string]json.RawMessage{
			"team":    json.RawMessage(`"platform"`),
			"release": json.RawMessage(`"2026"`),
		}),
		image.WithSetImagePropertiesProperty("release", "2026.10"),
	)
}

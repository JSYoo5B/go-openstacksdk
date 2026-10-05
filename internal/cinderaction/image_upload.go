package cinderaction

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/resource"
)

// ImageUploadResult owns the source's selected JSON value without imposing
// image_id or object shape. Completed reports handling, not image availability.
type ImageUploadResult struct {
	VolumeID     string
	Microversion string
	Discovery    []*rest.Response
	Applied      *rest.Response
	Upload       json.RawMessage
	Completed    bool
}

func ValidateImageName(value string) error { return validateBodyStrings(&value) }

func UploadToImage(ctx context.Context, client *gophercloud.ServiceClient, id, imageName string, options ...ImageUploadOption) (*ImageUploadResult, error) {
	const operation = "UploadVolumeToImage"
	source, err := captureAction(ctx, client, id)
	if err == nil {
		err = ValidateImageName(imageName)
	}
	if err != nil {
		return nil, WrapOperation(ctx, operation, err)
	}
	policy, err := prepareImageUpload(options, func() error { return source.Guard(ctx) })
	if err != nil {
		return nil, WrapOperation(ctx, operation, err)
	}
	fields := map[string]any{"image_name": imageName, "force": *policy.Force}
	for key, value := range map[string]*string{"disk_format": policy.DiskFormat, "container_format": policy.ContainerFormat, "visibility": policy.Visibility} {
		if value != nil {
			fields[key] = *value
		}
	}
	if policy.Protected != nil {
		fields["protected"] = *policy.Protected
	}
	required := ""
	if policy.Visibility != nil || policy.Protected != nil {
		required = "3.1"
	}
	body, _ := json.Marshal(map[string]any{"os-volume_upload_image": fields})
	action, err := applyRequired(ctx, source, id, operation, body, required)
	if action == nil {
		return nil, err
	}
	result := &ImageUploadResult{VolumeID: action.VolumeID, Microversion: action.Microversion, Discovery: action.Discovery, Applied: action.Applied}
	if err != nil {
		return result, err
	}
	value, decodeErr := uploadedValue(result.Applied)
	if err := errors.Join(decodeErr, source.Guard(ctx)); err != nil {
		return result, WrapOperation(ctx, operation, result.Applied.Fail(err))
	}
	result.Upload = value
	result.Completed = true
	return result, nil
}

func uploadedValue(response *rest.Response) (json.RawMessage, error) {
	if response == nil {
		return nil, fmt.Errorf("%w: accepted upload response is required", resource.ErrInvalidOption)
	}
	if !utf8.Valid(response.Body) {
		return nil, fmt.Errorf("upload response must be UTF-8 JSON")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(response.Body, &fields); err != nil {
		return nil, err
	}
	if fields == nil {
		return nil, fmt.Errorf("upload response must be a JSON object")
	}
	value, exists := fields["os-volume_upload_image"]
	if !exists {
		return nil, fmt.Errorf("upload response lacks os-volume_upload_image")
	}
	return bytes.Clone(value), nil
}

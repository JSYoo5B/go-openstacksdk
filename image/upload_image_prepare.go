package image

import (
	"context"
	"encoding/json"
	"strings"
	"unicode/utf8"
)

func validateImageUploadField(key string) error {
	if err := validateImagePatchField(key); err != nil {
		return err
	}
	if key == "name" {
		return uploadInvalid("image upload name belongs to the request")
	}
	return nil
}

func (s *Service) prepareImageUpload(ctx context.Context, input UploadImageRequest, options []ImageUploadOption) (*preparedTaskSource, ImageUploadOpts, map[string]json.RawMessage, error) {
	p, err := s.captureTaskSource(ctx)
	if err != nil {
		return nil, ImageUploadOpts{}, nil, err
	}
	if !utf8.ValidString(input.Name) || strings.TrimSpace(input.Name) == "" {
		return nil, ImageUploadOpts{}, nil, uploadInvalid("image upload name must be nonempty valid UTF-8")
	}
	if input.Data == nil || isNilReader(input.Data) {
		return nil, ImageUploadOpts{}, nil, uploadInvalid("image upload reader is required")
	}
	policy, err := applyTaskOptions(ImageUploadOpts{}, options, copyImageUploadOpts)
	if err == nil {
		policy.Headers, err = imageMutationHeaders(policy.Headers, false, "")
	}
	if err == nil && policy.Size != nil && *policy.Size < 0 {
		err = uploadInvalid("image upload size must not be negative")
	}
	body := map[string]json.RawMessage{"disk_format": json.RawMessage(`"qcow2"`), "container_format": json.RawMessage(`"bare"`), "visibility": json.RawMessage(`"private"`)}
	if err == nil {
		for key, value := range policy.Fields {
			if err = validateImageUploadField(key); err != nil {
				break
			}
			if value == nil || !utf8.Valid(value) || !json.Valid(value) {
				err = uploadInvalid("image upload field requires a valid UTF-8 JSON value")
				break
			}
			body[key] = copyImagePatchRaw(value)
		}
	}
	if err == nil {
		for _, key := range []string{"disk_format", "container_format"} {
			var format string
			if json.Unmarshal(body[key], &format) != nil || format == "" || !utf8.ValidString(format) {
				err = uploadInvalid("image upload %s must be a nonempty valid UTF-8 string", key)
				break
			}
		}
	}
	if err = p.finish(ctx, policy.Headers, err); err != nil {
		return nil, policy, nil, err
	}
	body["name"], _ = json.Marshal(input.Name)
	// Native ServiceClient.Request overlays service headers on RequestOpts.
	// Own media on the private snapshot, while leaving the borrowed source alone.
	p.client.MoreHeaders["Content-Type"], p.client.MoreHeaders["Accept"] = "application/json", "application/json"
	return p, policy, body, nil
}

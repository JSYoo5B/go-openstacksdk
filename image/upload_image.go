package image

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/gophercloud/gophercloud/v2"
)

// UploadImage creates metadata and streams borrowed data through one file PUT.
// It defaults to qcow2/bare/private, preserves actual phase evidence, never
// closes or seeks Data, and does not refresh, wait, retry data or delete images.
func (s *Service) UploadImage(ctx context.Context, input UploadImageRequest, options ...ImageUploadOption) (*ImageUploadResult, error) {
	wrap := func(err error) error { return wrapImageMutationError(ctx, "UploadImage", err) }
	p, policy, body, err := s.prepareImageUpload(ctx, input, append([]ImageUploadOption(nil), options...))
	if err != nil {
		return nil, wrap(err)
	}
	response, err := rest.DoJSON(ctx, p.client, http.MethodPost, p.base+"images", body, nil, http.StatusCreated)
	if response == nil {
		return nil, wrap(checkTaskResponse(ctx, p, nil, err))
	}
	result := &ImageUploadResult{Metadata: imageUploadResponse(response)}
	if err = checkTaskResponse(ctx, p, response, err); err != nil {
		return result, wrap(err)
	}
	var image ImageInfo
	if err = json.Unmarshal(response.Body, &image); err != nil {
		return result, wrap(response.Fail(err))
	}
	image.Header, image.StatusCode = response.Header.Clone(), response.StatusCode
	result.Image = &image
	if image.ID != nil {
		result.ImageID = *image.ID
	}
	if err = p.check(ctx); err != nil {
		return result, wrap(response.Fail(err))
	}
	if err = validateTaskID(result.ImageID); err != nil {
		return result, wrap(response.Fail(err))
	}
	endpoint := p.base + "images/" + url.PathEscape(result.ImageID) + "/file"
	if err = p.check(ctx); err != nil {
		return result, wrap(response.Fail(err))
	}
	response, err = uploadImageDataOnce(ctx, p.client, endpoint, input.Data, policy.Size)
	result.Acknowledgement = imageUploadResponse(response)
	return result, wrap(checkTaskResponse(ctx, p, response, err))
}

func uploadImageDataOnce(ctx context.Context, source *gophercloud.ServiceClient, endpoint string, data io.Reader, size *int64) (*rest.Response, error) {
	return imageDataOnce(ctx, source, nil, endpoint, data, size, []int{http.StatusNoContent})
}

func imageUploadErrors(causes ...error) error {
	var present []error
	for _, cause := range causes {
		if cause != nil {
			present = append(present, cause)
		}
	}
	if len(present) == 1 {
		return present[0]
	}
	return errors.Join(present...)
}
func imageUploadContextError(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		for _, cause := range []error{ctx.Err(), context.Cause(ctx)} {
			if cause != nil && !errors.Is(err, cause) {
				err = imageUploadErrors(err, cause)
			}
		}
	}
	return err
}

package image

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/fixedrequest"
	"gophercloudsdk/internal/rest"
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
	if err := ctx.Err(); err != nil {
		return nil, imageUploadContextError(ctx, err)
	}
	if err := rest.ValidateTarget(source, endpoint); err != nil {
		return nil, imageUploadContextError(ctx, err)
	}
	client, err := fixedrequest.New(source, http.MethodPut, endpoint)
	if err != nil {
		return nil, imageUploadContextError(ctx, err)
	}
	// A native retry can replay a consumed reader without rewinding it. Keep the
	// original HTTP transport/timeout/live auth, and disable resend policy here.
	client.ProviderClient.ReauthFunc = nil
	client.ProviderClient.RetryFunc = nil
	client.ProviderClient.RetryBackoffFunc = nil
	client.ProviderClient.MaxBackoffRetries = 0
	client.ProviderClient.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	client.MoreHeaders = maps.Clone(source.MoreHeaders)
	client.MoreHeaders["Content-Type"], client.MoreHeaders["Accept"] = "application/octet-stream", ""
	if size != nil {
		client.MoreHeaders["X-OpenStack-Image-Size"] = strconv.FormatInt(*size, 10)
	}
	wire, err := client.Request(ctx, http.MethodPut, endpoint, &gophercloud.RequestOpts{
		RawBody: borrowedUploadReader{Reader: data}, KeepResponseBody: true, OkCodes: []int{http.StatusNoContent},
	})
	if err != nil {
		return nil, imageUploadContextError(ctx, err)
	}
	response := &rest.Response{Header: wire.Header.Clone(), StatusCode: wire.StatusCode}
	response.Body, err = io.ReadAll(wire.Body)
	err = imageUploadContextError(ctx, imageUploadErrors(err, wire.Body.Close()))
	if err != nil {
		return response, response.Fail(err)
	}
	return response, nil
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

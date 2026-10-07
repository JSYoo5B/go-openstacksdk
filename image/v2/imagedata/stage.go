package imagedata

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/image/v2/images"
	"github.com/JSYoo5B/gophercloudsdk/internal/fixedrequest"
	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// StageImage fetches fresh queued metadata, stages borrowed data once, then
// fetches the same image again. It does not import, wait or close the reader.
func (a *API) StageImage(ctx context.Context, ref resource.Ref, data io.Reader, options ...StageOption) (*StageImageResult, error) {
	return a.stageImage(ctx, ref, nil, data, append([]StageOption(nil), options...))
}

// StageKnownImage copies a supplied image's ID and status before option
// callbacks. It skips the initial GET and leaves the supplied image untouched.
func (a *API) StageKnownImage(ctx context.Context, value *images.Image, data io.Reader, options ...StageOption) (*StageImageResult, error) {
	if value == nil {
		return nil, wrapStageError(ctx, stageInvalid("image is required"))
	}
	seed := &stageImageSeed{id: value.ID, status: string(value.Status)}
	return a.stageImage(ctx, resource.ID(seed.id), seed, data, append([]StageOption(nil), options...))
}

type stageImageSeed struct{ id, status string }

func (a *API) stageImage(ctx context.Context, ref resource.Ref, seed *stageImageSeed, data io.Reader, options []StageOption) (*StageImageResult, error) {
	wrap := func(err error) error { return wrapStageError(ctx, err) }
	var source *gophercloud.ServiceClient
	if a != nil {
		source = a.client
	}
	if err := validateStageSource(ctx, source); err != nil {
		return nil, wrap(err)
	}
	if stageNilReader(data) {
		return nil, wrap(stageInvalid("stage reader is required"))
	}
	if err := ref.Validate(); err != nil {
		return nil, wrap(err)
	}
	if !ref.IsName() {
		if err := stageID(ref.String()); err != nil {
			return nil, wrap(err)
		}
	}
	if seed != nil && seed.status != "queued" {
		return nil, wrap(stageInvalid("image must have exact queued status"))
	}
	policy, err := a.PrepareStageOptions(ctx, options...)
	if err != nil {
		return nil, wrap(err)
	}
	if err := validateStageSource(ctx, source); err != nil {
		return nil, wrap(err)
	}
	headers, err := stageHeaders(maps.Clone(source.MoreHeaders), true, source.Microversion)
	if err != nil {
		return nil, wrap(err)
	}
	for key, value := range policy.Headers {
		headers[key] = value
	}
	client := *source
	client.MoreHeaders = headers
	collection := source.ServiceURL("images")
	check := func() error {
		if err := validateStageSource(ctx, source); err != nil {
			return err
		}
		if source.ProviderClient != client.ProviderClient {
			return stageInvalid("image staging provider changed")
		}
		return rest.ValidateTarget(source, collection)
	}
	if err := check(); err != nil {
		return nil, wrap(err)
	}
	id := ref.String()
	if ref.IsName() {
		id, err = images.New(&client).Resources.ResolveID(ctx, ref)
		if err != nil {
			return nil, wrap(err)
		}
		if err := stageID(id); err != nil {
			return nil, wrap(err)
		}
	}
	endpoint := collection + "/" + url.PathEscape(id)
	if seed == nil {
		if err := check(); err != nil {
			return nil, wrap(err)
		}
		response, err := rest.DoJSON(ctx, &client, http.MethodGet, endpoint, nil, nil, http.StatusOK)
		if err != nil {
			return nil, wrap(err)
		}
		if _, err := decodeStageImage(response, true); err != nil {
			return nil, wrap(err)
		}
	}
	if err := check(); err != nil {
		return nil, wrap(err)
	}
	response, err := stageDataOnce(ctx, &client, endpoint+"/stage", data, policy.Size)
	result := stageImageResult(id, response)
	if err != nil {
		return result, wrap(err)
	}
	if err := check(); err != nil {
		return result, wrap(err)
	}
	response, err = rest.DoJSON(ctx, &client, http.MethodGet, endpoint, nil, nil, http.StatusOK)
	result.setMetadata(response)
	if err != nil {
		return result, wrap(err)
	}
	result.Image, err = decodeStageImage(response, false)
	return result, wrap(err)
}

func wrapStageError(ctx context.Context, err error) error {
	if err != nil && ctx != nil && ctx.Err() != nil {
		for _, cause := range []error{ctx.Err(), context.Cause(ctx)} {
			if cause != nil && !errors.Is(err, cause) {
				err = errors.Join(err, cause)
			}
		}
	}
	return request.Wrap("StageImage", "imagedata", err)
}

func validateStageSource(ctx context.Context, client *gophercloud.ServiceClient) error {
	if ctx == nil {
		return stageInvalid("context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if client == nil || client.ProviderClient == nil {
		return stageInvalid("image service client is required")
	}
	if client.Type != "image" {
		return fmt.Errorf("%w: image service client type is required", resource.ErrUnsupported)
	}
	base := client.ServiceURL()
	if err := rest.ValidateTarget(client, base); err != nil {
		return err
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.RawQuery != "" || !strings.HasSuffix(parsed.Path, "/") {
		return stageInvalid("image service base must be query-free and end in a slash")
	}
	_, err = stageHeaders(maps.Clone(client.MoreHeaders), true, client.Microversion)
	return err
}

func stageID(id string) error {
	if err := resource.ID(id).Validate(); err != nil {
		return err
	}
	if !utf8.ValidString(id) {
		return stageInvalid("image ID must be valid UTF-8")
	}
	for _, char := range id {
		if unicode.IsSpace(char) || unicode.IsControl(char) || strings.ContainsRune(":\\", char) {
			return stageInvalid("image ID must be a single safe URL segment")
		}
	}
	return nil
}

func stageNilReader(data io.Reader) bool {
	if data == nil {
		return true
	}
	value := reflect.ValueOf(data)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	}
	return false
}

func decodeStageImage(response *rest.Response, queued bool) (*images.Image, error) {
	if !utf8.Valid(response.Body) {
		return nil, response.Fail(fmt.Errorf("response must be valid UTF-8"))
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(response.Body, &fields); err != nil {
		return nil, response.Fail(err)
	}
	if fields == nil {
		return nil, response.Fail(fmt.Errorf("response must be a JSON object"))
	}
	var body map[string]any
	decoder := json.NewDecoder(bytes.NewReader(response.Body))
	decoder.UseNumber()
	if err := decoder.Decode(&body); err != nil {
		return nil, response.Fail(err)
	}
	// Native extraction projects import-method and store-id headers into the
	// model. Its mutable decoding map stays separate from the actual raw body.
	var native images.GetResult
	native.Body, native.Header = body, response.Header.Clone()
	image, err := native.Extract()
	if err != nil {
		return nil, response.Fail(err)
	}
	if queued {
		body, exists := fields["status"]
		if !exists || string(body) == "null" {
			return nil, response.Fail(stageInvalid("canonical queued status is required"))
		}
		var status string
		if err := json.Unmarshal(body, &status); err != nil {
			return nil, response.Fail(err)
		}
		if status != "queued" {
			return nil, response.Fail(stageInvalid("image must have exact queued status"))
		}
	}
	return image, nil
}

// borrowedStageReader masks close, seek, length and replay interfaces. The
// caller owns the data and must release a blocked Read after cancellation.
type borrowedStageReader struct{ io.Reader }

func stageDataOnce(ctx context.Context, source *gophercloud.ServiceClient, endpoint string, data io.Reader, size *int64) (*rest.Response, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := rest.ValidateTarget(source, endpoint); err != nil {
		return nil, err
	}
	client, err := fixedrequest.New(source, http.MethodPut, endpoint)
	if err != nil {
		return nil, err
	}
	// Provider callbacks can replay a partially consumed body. Disable them only
	// on this private provider, while the guarded transport keeps live auth.
	client.ProviderClient.ReauthFunc = nil
	client.ProviderClient.RetryFunc = nil
	client.ProviderClient.RetryBackoffFunc = nil
	client.ProviderClient.MaxBackoffRetries = 0
	client.ProviderClient.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	client.MoreHeaders = maps.Clone(source.MoreHeaders)
	if client.MoreHeaders == nil {
		client.MoreHeaders = make(map[string]string)
	}
	client.MoreHeaders["Content-Type"], client.MoreHeaders["Accept"] = "application/octet-stream", ""
	if size != nil {
		client.MoreHeaders["X-OpenStack-Image-Size"] = strconv.FormatInt(*size, 10)
	}
	wire, err := client.Request(ctx, http.MethodPut, endpoint, &gophercloud.RequestOpts{
		RawBody: borrowedStageReader{Reader: data}, KeepResponseBody: true, OkCodes: []int{http.StatusNoContent},
	})
	if err != nil {
		return nil, err
	}
	response := &rest.Response{Header: wire.Header.Clone(), StatusCode: wire.StatusCode}
	defer wire.Body.Close()
	response.Body, err = io.ReadAll(wire.Body)
	if err == nil {
		err = ctx.Err()
	}
	if err != nil {
		return response, response.Fail(err)
	}
	return response, nil
}

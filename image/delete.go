package image

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/fixedrequest"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// DeleteImage deletes a whole image or one store's copy. Explicit IDs are sent
// directly; names use an exact collection lookup. Absence is ignored by default.
// Only an actual 204 creates a result, retained alongside body or context errors.
// This workflow never discovers stores, polls or performs additional cleanup.
func (s *Service) DeleteImage(ctx context.Context, ref resource.Ref, options ...DeleteImageOption) (*DeleteImageResult, error) {
	wrap := func(err error) error { return wrapDeleteImageError(ctx, err) }
	var source *gophercloud.ServiceClient
	if s != nil {
		source = s.client
	}
	headers, err := validateDeleteImageSource(ctx, source)
	if err != nil {
		return nil, wrap(err)
	}
	if err := ref.Validate(); err != nil {
		return nil, wrap(err)
	}
	if !utf8.ValidString(ref.String()) {
		return nil, wrap(uploadInvalid("image reference must be valid UTF-8"))
	}
	if !ref.IsName() {
		if err := createImportID(ref.String()); err != nil {
			return nil, wrap(err)
		}
	}
	client := *source
	client.MoreHeaders = headers
	base := source.ServiceURL()
	policy, err := parseDeleteImageOptions(append([]DeleteImageOption(nil), options...))
	if err != nil {
		return nil, wrap(err)
	}
	check := func() error {
		if s.client != source || source.ProviderClient != client.ProviderClient {
			return uploadInvalid("image delete source or provider changed")
		}
		if _, err := validateDeleteImageSource(ctx, source); err != nil {
			return err
		}
		return rest.ValidateTarget(source, base)
	}
	if err := check(); err != nil {
		return nil, wrap(err)
	}
	id := ref.String()
	if ref.IsName() {
		id, err = New(&client).Images.ResolveID(ctx, ref)
		if err != nil {
			if sourceErr := check(); sourceErr != nil {
				return nil, wrap(errors.Join(err, sourceErr))
			}
			// Only a direct logical absence from a completed exact search is
			// ignorable. Nested HTTP, transport or callback errors are distinct.
			missing, logical := err.(*resource.NotFoundError)
			if logical && missing.Cause == nil && missing.Resource == "image" && missing.Reference == ref.String() && *policy.IgnoreMissing {
				return nil, nil
			}
			return nil, wrap(err)
		}
		if !utf8.ValidString(id) {
			return nil, wrap(uploadInvalid("resolved image ID must be valid UTF-8"))
		}
		if err := createImportID(id); err != nil {
			return nil, wrap(err)
		}
	}
	if err := check(); err != nil {
		return nil, wrap(err)
	}
	storeID := ""
	endpoint := base + "images/" + url.PathEscape(id)
	if policy.StoreID != nil {
		storeID = *policy.StoreID
		endpoint = base + "stores/" + url.PathEscape(storeID) + "/" + url.PathEscape(id)
	}
	clientForDelete, err := fixedrequest.New(&client, http.MethodDelete, endpoint)
	if err != nil {
		return nil, wrap(err)
	}
	clientForDelete.ProviderClient.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	if retry := clientForDelete.ProviderClient.RetryFunc; retry != nil {
		clientForDelete.ProviderClient.RetryFunc = func(ctx context.Context, method, target string, options *gophercloud.RequestOpts, original error, count uint) error {
			err := retry(ctx, method, target, options, original, count)
			// Native retry hooks may edit RequestOpts. Deletion owns the empty
			// request and response body; changing either loses acknowledgement
			// evidence or closes its body before this workflow can own it.
			if !options.KeepResponseBody || options.JSONResponse != nil || options.JSONBody != nil || options.RawBody != nil {
				return errors.Join(err, original, uploadInvalid("retry changes image delete body ownership"))
			}
			return err
		}
	}
	// Privately handling 404 owns its body failures before deciding absence.
	// It bypasses native RetryFunc for 404; other pre-accept policies remain.
	wire, err := clientForDelete.Request(ctx, http.MethodDelete, endpoint, &gophercloud.RequestOpts{
		KeepResponseBody: true, OkCodes: []int{http.StatusNoContent, http.StatusNotFound},
	})
	if err != nil {
		// Native Request owns rejected bodies. They are not acknowledgements.
		return nil, wrap(err)
	}
	header := wire.Header.Clone()
	status := wire.StatusCode
	body, readErr := io.ReadAll(wire.Body)
	bodyErr := errors.Join(readErr, wire.Body.Close(), ctx.Err())
	if status != http.StatusNoContent && status != http.StatusNotFound {
		// A configured native retry callback can alter its RequestOpts. The
		// public acknowledgement policy still depends on the actual status.
		return nil, wrap(errors.Join(gophercloud.ErrUnexpectedResponseCode{
			Method: http.MethodDelete, URL: endpoint, Expected: []int{http.StatusNoContent}, Actual: status,
			Body: append([]byte(nil), body...), ResponseHeader: header.Clone(),
		}, bodyErr))
	}
	if status == http.StatusNotFound {
		native := gophercloud.ErrUnexpectedResponseCode{
			Method: http.MethodDelete, URL: endpoint, Expected: []int{http.StatusNoContent}, Actual: status,
			Body: append([]byte(nil), body...), ResponseHeader: header.Clone(),
		}
		if bodyErr != nil {
			return nil, wrap(errors.Join(native, bodyErr))
		}
		if *policy.IgnoreMissing {
			return nil, nil
		}
		return nil, wrap(&resource.NotFoundError{Resource: "image", Reference: id, Cause: native})
	}
	result := &DeleteImageResult{
		ImageID: id, StoreID: storeID, Body: append([]byte(nil), body...), Header: header.Clone(), StatusCode: status,
	}
	if bodyErr != nil {
		return result, wrap(&resource.ResponseError{
			Body: append([]byte(nil), body...), Header: header.Clone(), StatusCode: status, Cause: bodyErr,
		})
	}
	return result, nil
}

func validateDeleteImageSource(ctx context.Context, client *gophercloud.ServiceClient) (map[string]string, error) {
	if ctx == nil {
		return nil, uploadInvalid("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if client == nil || client.ProviderClient == nil {
		return nil, uploadInvalid("image service client is required")
	}
	if client.Type != "image" {
		return nil, fmt.Errorf("%w: image service client type is required", resource.ErrUnsupported)
	}
	base := client.ServiceURL()
	if err := rest.ValidateTarget(client, base); err != nil {
		return nil, err
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.RawQuery != "" || !strings.HasSuffix(parsed.Path, "/") {
		return nil, uploadInvalid("image service base must be query-free and end in a slash")
	}
	headers := make(map[string]string, len(client.MoreHeaders))
	for key, value := range client.MoreHeaders {
		if key == "" || !utf8.ValidString(value) || !downloadHeaderValue(value) {
			return nil, uploadInvalid("invalid image delete source header")
		}
		for _, char := range []byte(key) {
			if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(char)) {
				continue
			}
			return nil, uploadInvalid("invalid image delete header %q", key)
		}
		name := http.CanonicalHeaderKey(key)
		if old, exists := headers[name]; exists && old != value {
			return nil, uploadInvalid("conflicting image delete header aliases %q", key)
		}
		switch strings.ToLower(key) {
		case "x-auth-token", "x-service-token", "authorization", "host", "cookie", "content-length", "transfer-encoding", "connection", "trailer", "te", "upgrade", "x-openstack-glance-api-version", "x-openstack-image-size":
			return nil, uploadInvalid("delete header %q is owned by the SDK", key)
		case "openstack-api-version":
			if client.Microversion == "" || value != "image "+client.Microversion {
				return nil, uploadInvalid("image version header conflicts with selected microversion")
			}
		}
		headers[name] = value
	}
	if !utf8.ValidString(client.Microversion) || !downloadHeaderValue(client.Microversion) {
		return nil, uploadInvalid("invalid image microversion")
	}
	return headers, nil
}

func wrapDeleteImageError(ctx context.Context, err error) error {
	if err != nil && ctx != nil && ctx.Err() != nil {
		for _, cause := range []error{ctx.Err(), context.Cause(ctx)} {
			if cause != nil && !errors.Is(err, cause) {
				err = errors.Join(err, cause)
			}
		}
	}
	return request.Wrap("DeleteImage", "image", err)
}

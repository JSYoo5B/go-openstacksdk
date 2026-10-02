package imageimport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/image/v2/images"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// ImportImage resolves an explicit reference, fetches fresh image metadata and
// submits one asynchronous import request. It does not stage or wait for data.
func (a *API) ImportImage(ctx context.Context, ref resource.Ref, options ...ImportOption) (*ImportResult, error) {
	return a.importImage(ctx, ref, nil, append([]ImportOption(nil), options...))
}

// ImportKnownImage uses a supplied model without fetching it. ID and formats
// are copied before option callbacks; unrelated model fields are not retained.
func (a *API) ImportKnownImage(ctx context.Context, value *images.Image, options ...ImportOption) (*ImportResult, error) {
	if value == nil {
		return nil, wrapImportError(ctx, importInvalid("image is required"))
	}
	seed := &importImageSeed{id: value.ID, container: value.ContainerFormat, disk: value.DiskFormat}
	return a.importImage(ctx, resource.ID(seed.id), seed, append([]ImportOption(nil), options...))
}

type importImageSeed struct{ id, container, disk string }

func (a *API) importImage(ctx context.Context, ref resource.Ref, seed *importImageSeed, options []ImportOption) (*ImportResult, error) {
	wrap := func(err error) error { return wrapImportError(ctx, err) }
	var source *gophercloud.ServiceClient
	if a != nil {
		source = a.client
	}
	if err := validateImportSource(ctx, source); err != nil {
		return nil, wrap(err)
	}
	if err := ref.Validate(); err != nil {
		return nil, wrap(err)
	}
	if !ref.IsName() {
		if err := importID(ref.String()); err != nil {
			return nil, wrap(err)
		}
	}
	if seed != nil {
		if err := validateImportFormats(seed.container, seed.disk); err != nil {
			return nil, wrap(err)
		}
	}
	policy, err := parseImportOpts(options)
	if err != nil {
		return nil, wrap(err)
	}
	if err := validateImportSource(ctx, source); err != nil {
		return nil, wrap(err)
	}
	headers, err := importHeaders(source.MoreHeaders, true, source.Microversion)
	if err != nil {
		return nil, wrap(err)
	}
	for key, value := range policy.Headers {
		headers[key] = value
	}
	if policy.Store != nil {
		headers["X-Image-Meta-Store"] = *policy.Store
	}
	if _, exists := headers["X-Image-Meta-Store"]; exists && (len(policy.Stores) > 0 || policy.AllStores != nil && *policy.AllStores) {
		return nil, wrap(importInvalid("source store header conflicts with Stores or AllStores"))
	}
	client := *source
	client.MoreHeaders = headers
	collection := source.ServiceURL("images")
	check := func() error {
		if err := validateImportSource(ctx, source); err != nil {
			return err
		}
		if source.ProviderClient != client.ProviderClient {
			return importInvalid("image import provider changed")
		}
		return rest.ValidateTarget(source, collection)
	}
	if err := check(); err != nil {
		return nil, wrap(err)
	}
	id := ref.String()
	if ref.IsName() {
		// Name resolution retains the existing Images collection's exact-name,
		// duplicate and native paging policies, on the captured client.
		id, err = images.New(&client).Resources.ResolveID(ctx, ref)
		if err != nil {
			return nil, wrap(err)
		}
		if err := importID(id); err != nil {
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
		if err := decodeImportFormats(response); err != nil {
			return nil, wrap(err)
		}
	}
	if err := check(); err != nil {
		return nil, wrap(err)
	}
	response, err := rest.DoJSON(ctx, &client, http.MethodPost, endpoint+"/import", importBody(policy), nil, http.StatusAccepted)
	return importResult(id, response), wrap(err)
}

func wrapImportError(ctx context.Context, err error) error {
	if err != nil && ctx != nil && ctx.Err() != nil && !errors.Is(err, ctx.Err()) {
		err = errors.Join(err, ctx.Err())
	}
	return request.Wrap("ImportImage", "imageimport", err)
}

func validateImportSource(ctx context.Context, client *gophercloud.ServiceClient) error {
	if ctx == nil {
		return importInvalid("context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if client == nil || client.ProviderClient == nil {
		return importInvalid("image service client is required")
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
		return importInvalid("image service base must be query-free and end in a slash")
	}
	_, err = importHeaders(maps.Clone(client.MoreHeaders), true, client.Microversion)
	return err
}

func importID(id string) error {
	if err := resource.ID(id).Validate(); err != nil {
		return err
	}
	if !utf8.ValidString(id) {
		return importInvalid("image ID must be valid UTF-8")
	}
	for _, char := range id {
		if unicode.IsSpace(char) || unicode.IsControl(char) || strings.ContainsRune(":\\", char) {
			return importInvalid("image ID must be a single safe URL segment")
		}
	}
	return nil
}

func validateImportFormats(container, disk string) error {
	if !utf8.ValidString(container) || !utf8.ValidString(disk) || container == "" || disk == "" {
		return importInvalid("both container_format and disk_format are required")
	}
	return nil
}

func decodeImportFormats(response *rest.Response) error {
	fields, err := importObject(response.Body)
	if err != nil {
		return response.Fail(err)
	}
	var image images.Image
	if err := json.Unmarshal(response.Body, &image); err != nil {
		return response.Fail(err)
	}
	formats := make([]string, 0, 2)
	for _, key := range []string{"container_format", "disk_format"} {
		body, exists := fields[key]
		if !exists || string(body) == "null" {
			return response.Fail(importInvalid("canonical %s is required", key))
		}
		var value string
		if err := json.Unmarshal(body, &value); err != nil {
			return response.Fail(err)
		}
		formats = append(formats, value)
	}
	if err := validateImportFormats(formats[0], formats[1]); err != nil {
		return response.Fail(err)
	}
	return nil
}

package objectstorage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// AutocreatedImageObjectsOpts replaces Python connection configuration with an
// explicit owned policy. ImageAPIUseTasks defaults false; nil Container means
// images and nil SegmentPrefix omits the server prefix query.
type AutocreatedImageObjectsOpts struct {
	ImageAPIUseTasks         bool
	Container, SegmentPrefix *string
}
type AutocreatedImageObjectsOption func(*AutocreatedImageObjectsOpts) error

func ownAutocreatedImageObjects(value AutocreatedImageObjectsOpts) AutocreatedImageObjectsOpts {
	for _, p := range []**string{&value.Container, &value.SegmentPrefix} {
		if *p != nil {
			x := **p
			*p = &x
		}
	}
	return value
}
func WithAutocreatedImageObjectsOpts(value AutocreatedImageObjectsOpts) AutocreatedImageObjectsOption {
	owned := ownAutocreatedImageObjects(value)
	return func(cfg *AutocreatedImageObjectsOpts) error { *cfg = ownAutocreatedImageObjects(owned); return nil }
}
func WithAutocreatedImageTasks(value bool) AutocreatedImageObjectsOption {
	return func(cfg *AutocreatedImageObjectsOpts) error { cfg.ImageAPIUseTasks = value; return nil }
}
func WithAutocreatedImageContainer(value string) AutocreatedImageObjectsOption {
	return func(cfg *AutocreatedImageObjectsOpts) error { x := value; cfg.Container = &x; return nil }
}
func WithAutocreatedImagePrefix(value string) AutocreatedImageObjectsOption {
	return func(cfg *AutocreatedImageObjectsOpts) error { x := value; cfg.SegmentPrefix = &x; return nil }
}

// AutocreatedImageObjectObservation records the list row, fresh metadata and
// optional discovery/DELETE without treating requested deletion as completion.
type AutocreatedImageObjectObservation struct {
	Object                           *objects.ObjectRecord
	Metadata, Discovery              *objects.GetMetadataResult
	Deletion                         *objects.MetadataResponse
	Matched, Deleted, IgnoredMissing bool
}
type AutocreatedImageObjectsResult struct {
	Container    string
	Deleted      bool
	Observations []*AutocreatedImageObjectObservation
	LastPage     *objects.ObjectRecord
}

// DeleteAutocreatedImageObjects deletes only objects whose fresh metadata has
// x-sdk-autocreated exactly equal to "true". Errors propagate with completed
// phases; no mutation is automatically replayed by this workflow.
func (c *Cloud) DeleteAutocreatedImageObjects(ctx context.Context, options ...AutocreatedImageObjectsOption) (*AutocreatedImageObjectsResult, error) {
	fail := func(result *AutocreatedImageObjectsResult, err error) (*AutocreatedImageObjectsResult, error) {
		return result, request.Wrap("DeleteAutocreatedImageObjects", "objects", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return fail(nil, err)
	}
	if c == nil {
		return fail(nil, fmt.Errorf("%w: object-store cloud is required", resource.ErrInvalidOption))
	}
	selected := c.client
	var source *gophercloud.ServiceClient
	if selected != nil {
		owned := *selected
		owned.MoreHeaders = maps.Clone(selected.MoreHeaders)
		source = &owned
	}
	var changed error
	sourceGuard := func(ctx context.Context) error {
		if selected != nil && (c.client != selected || selected.ProviderClient != source.ProviderClient || selected.Endpoint != source.Endpoint || selected.ResourceBase != source.ResourceBase || selected.Type != source.Type || selected.Microversion != source.Microversion || !maps.Equal(selected.MoreHeaders, source.MoreHeaders)) {
			changed = fmt.Errorf("%w: object-store source or route changed", resource.ErrInvalidOption)
		}
		return errors.Join(changed, cloudread.Context(ctx))
	}
	cfg := AutocreatedImageObjectsOpts{}
	for _, apply := range slices.Clone(options) {
		if apply == nil {
			return fail(nil, fmt.Errorf("%w: nil cleanup option", resource.ErrInvalidOption))
		}
		next := ownAutocreatedImageObjects(cfg)
		if err := errors.Join(apply(&next), sourceGuard(ctx)); err != nil {
			return fail(nil, err)
		}
		cfg = ownAutocreatedImageObjects(next)
	}
	container := "images"
	if cfg.Container != nil {
		container = *cfg.Container
	}
	result := &AutocreatedImageObjectsResult{Container: container}
	if !cfg.ImageAPIUseTasks {
		return result, nil
	}
	ctx = rest.WithOperationSources(ctx)
	if source == nil || source.ProviderClient == nil {
		return fail(result, fmt.Errorf("%w: object-store service client is required", resource.ErrInvalidOption))
	}
	rest.RegisterOperationSource(ctx, sourceGuard)
	guard := func(ctx context.Context) error { return errors.Join(sourceGuard(ctx), rest.CheckOperationGuard(ctx)) }
	api := objects.New(selected)
	var list []objects.RecordListOption
	if cfg.SegmentPrefix != nil {
		list = append(list, objects.WithRecordListPrefix(*cfg.SegmentPrefix))
	}
	for row, listErr := range api.ListRecords(ctx, container, list...) {
		result.LastPage = copyObjectSearchRecord(row)
		if listErr != nil {
			return fail(result, listErr)
		}
		if err := guard(ctx); err != nil {
			return fail(result, err)
		}
		observation := &AutocreatedImageObjectObservation{Object: copyObjectSearchRecord(row)}
		result.Observations = append(result.Observations, observation)
		var name string
		if row == nil || row.Resource == nil || json.Unmarshal(row.Resource.Body["id"], &name) != nil || name == "" {
			return fail(result, fmt.Errorf("%w: listed object requires a nonempty literal string id", resource.ErrInvalidOption))
		}
		metadata, err := api.HeadMetadata(ctx, container, name)
		if metadata == nil {
			metadata = autocreatedImageFailedMetadata(err)
		}
		observation.Metadata = metadata
		if err != nil {
			return fail(result, err)
		}
		if metadata.Metadata.Values["x-sdk-autocreated"] != "true" {
			continue
		}
		observation.Matched = true
		slo, known := autocreatedImageSLO(row, metadata)
		latest := metadata
		if !known {
			latest, err = api.HeadMetadata(ctx, container, name)
			if latest == nil {
				latest = autocreatedImageFailedMetadata(err)
			}
			observation.Discovery = latest
			if err != nil {
				if resource.IgnoreMissing(err) == nil && guard(ctx) == nil {
					observation.Deleted = true
					observation.IgnoredMissing = true
					result.Deleted = true
					continue
				}
				return fail(result, err)
			}
			slo, _ = autocreatedImageSLO(row, latest)
		}
		headers := maps.Clone(source.MoreHeaders)
		if headers == nil {
			headers = map[string]string{}
		}
		for key, value := range latest.Metadata.Values {
			headers[http.CanonicalHeaderKey("X-Object-Meta-"+key)] = value
		}
		client := *source
		client.MoreHeaders = headers
		target := client.ServiceURL(url.PathEscape(container), url.PathEscape(name))
		if slo {
			target += "?multipart-manifest=delete"
		}
		codes := make([]int, 301)
		for i := range 300 {
			codes[i] = 100 + i
		}
		codes[300] = 404
		response, err := rest.DoJSONGuarded(ctx, &client, guard, http.MethodDelete, target, nil, nil, codes...)
		if response != nil {
			observation.Deletion = &objects.MetadataResponse{Body: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
		} else {
			observation.Deletion = autocreatedImageFailureReceipt(err)
		}
		if err != nil {
			return fail(result, err)
		}
		observation.Deleted = true
		observation.IgnoredMissing = response.StatusCode == 404
		result.Deleted = true
	}
	return result, nil
}

func autocreatedImageSLO(row *objects.ObjectRecord, metadata *objects.GetMetadataResult) (bool, bool) {
	if values, present := metadata.Header[http.CanonicalHeaderKey("X-Static-Large-Object")]; present && len(values) > 0 {
		return values[0] != "", true
	}
	if row != nil && row.Resource != nil {
		raw := bytes.TrimSpace(row.Resource.Body["is_static_large_object"])
		if len(raw) > 0 && !bytes.Equal(raw, []byte("null")) {
			return bytes.Equal(raw, []byte("true")), true
		}
	}
	return false, false
}

func autocreatedImageFailureReceipt(err error) *objects.MetadataResponse {
	var auth *gophercloud.ErrUnableToReauthenticate
	if errors.As(err, &auth) {
		err = auth.ErrOriginal
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) {
		return nil
	}
	return &objects.MetadataResponse{Body: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual}
}
func autocreatedImageFailedMetadata(err error) *objects.GetMetadataResult {
	proof := autocreatedImageFailureReceipt(err)
	if proof == nil {
		return nil
	}
	return &objects.GetMetadataResult{Body: proof.Body, Header: proof.Header, StatusCode: proof.StatusCode}
}

package backups

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/metadata"
	"github.com/JSYoo5B/go-openstacksdk/internal/cindermetadata"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudsnapshot"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// MetadataResponse is the opaque HTTP evidence of a backup metadata request.
type MetadataResponse = ExportRecordResponse

// MetadataResult separates an observed metadata JSON value from its response.
// Metadata is populated only by FetchMetadata/GetMetadata: nil means omission,
// while the nonnil bytes "null" mean an explicit JSON null. Mutations retain
// their actual opaque response and do not synthesize metadata from their input.
type MetadataResult struct {
	BackupID string
	Metadata json.RawMessage
	Response *MetadataResponse
}

// MetadataDeleteResult retains the clear PUT or ordered acknowledged deletes.
// A partial result accompanies the first failed key; deletion is not atomic.
type MetadataDeleteResult = metadata.DeleteResult

type MetadataOption = metadata.Option

func WithMetadataHeader(key, value string) MetadataOption {
	return metadata.WithHeader(key, value)
}
func WithMetadataHeaders(headers map[string]string) MetadataOption {
	return metadata.WithHeaders(headers)
}

// PythonMetadataProfile is an explicit compatibility profile for the Python
// backup MetadataMixin child routes absent from stock Cinder 27.0.0. Creating
// this profile is an opt-in to those routes, not a server capability claim.
type PythonMetadataProfile struct{ api *API }

// PythonMetadata opts into the pinned Python SDK's backup metadata routes.
// Stock Cinder 27.0.0 does not expose these routes and may return HTTP 404.
// Use ordinary Update with a complete Metadata map for stock backup replacement.
func (a *API) PythonMetadata() *PythonMetadataProfile { return &PythonMetadataProfile{api: a} }

func (p *PythonMetadataProfile) parent() *API {
	if p == nil {
		return nil
	}
	return p.api
}

// FetchMetadata follows Python's backup MetadataMixin child route. Stock Cinder
// 27.0.0 does not register this route; its physical HTTP failure is preserved.
// This operation does not fall back to the different backup Update request.
func (p *PythonMetadataProfile) FetchMetadata(ctx context.Context, id string, options ...MetadataOption) (*MetadataResult, error) {
	return p.parent().fetchMetadata(ctx, "FetchMetadata", id, options)
}

// GetMetadata is the deprecated Python spelling of FetchMetadata.
// Deprecated: use FetchMetadata. Go does not emit Python runtime warnings.
func (p *PythonMetadataProfile) GetMetadata(ctx context.Context, id string, options ...MetadataOption) (*MetadataResult, error) {
	return p.parent().fetchMetadata(ctx, "GetMetadata", id, options)
}

func (a *API) fetchMetadata(ctx context.Context, operation, id string, options []MetadataOption) (*MetadataResult, error) {
	call, err := a.prepareMetadata(ctx, id, options)
	if err != nil {
		return nil, metadataWrap(ctx, operation, err)
	}
	response, err := call.perform(ctx, http.MethodGet, call.endpoint, nil)
	result := metadataResult(id, response)
	if err != nil {
		return result, metadataWrap(ctx, operation, err)
	}
	if !utf8.Valid(response.Body) {
		return result, metadataWrap(ctx, operation, response.Fail(fmt.Errorf("backup metadata response must contain UTF-8 JSON")))
	}
	raw, err := response.Object("")
	if err != nil {
		return result, metadataWrap(ctx, operation, err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return result, metadataWrap(ctx, operation, response.Fail(err))
	}
	if err := call.source.Guard(ctx); err != nil {
		return result, metadataWrap(ctx, operation, response.Fail(err))
	}
	result.Metadata = bytes.Clone(fields["metadata"])
	return result, nil
}

// SetMetadata POSTs an explicit metadata object, including for nil/empty input.
// It follows MetadataMixin's merge request and accepts an opaque success body.
func (p *PythonMetadataProfile) SetMetadata(ctx context.Context, id string, values map[string]string, options ...MetadataOption) (*MetadataResult, error) {
	snapshot := maps.Clone(values)
	if snapshot == nil {
		snapshot = make(map[string]string)
	}
	for key, value := range snapshot {
		if !utf8.ValidString(key) || !utf8.ValidString(value) {
			return nil, metadataWrap(ctx, "SetMetadata", metadataInvalid("metadata keys and values must be UTF-8 strings"))
		}
	}
	call, err := p.parent().prepareMetadata(ctx, id, options)
	if err != nil {
		return nil, metadataWrap(ctx, "SetMetadata", err)
	}
	response, err := call.perform(ctx, http.MethodPost, call.endpoint, map[string]any{"metadata": snapshot})
	return metadataResult(id, response), metadataWrap(ctx, "SetMetadata", err)
}

// DeleteMetadata clears all keys by PUT when keys is nil. A nonnil empty slice
// performs no HTTP; other slices retain order and duplicates and stop on error.
func (p *PythonMetadataProfile) DeleteMetadata(ctx context.Context, id string, keys []string, options ...MetadataOption) (*MetadataDeleteResult, error) {
	clearAll := keys == nil
	snapshot := append([]string(nil), keys...)
	call, err := p.parent().prepareMetadata(ctx, id, options)
	if err != nil {
		return nil, metadataWrap(ctx, "DeleteMetadata", err)
	}
	for _, key := range snapshot {
		if err := metadataDeleteKey(key); err != nil {
			return nil, metadataWrap(ctx, "DeleteMetadata", err)
		}
	}
	result := &MetadataDeleteResult{}
	if clearAll {
		response, err := call.perform(ctx, http.MethodPut, call.endpoint, map[string]any{"metadata": map[string]string{}})
		if response != nil {
			result.Cleared = &metadata.Result{Body: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
		}
		return result, metadataWrap(ctx, "DeleteMetadata", err)
	}
	result.Deleted = make([]metadata.DeletedKey, 0, len(snapshot))
	for _, key := range snapshot {
		response, err := call.perform(ctx, http.MethodDelete, call.endpoint+"/"+url.PathEscape(key), nil)
		if err != nil {
			return result, metadataWrap(ctx, "DeleteMetadata", err)
		}
		result.Deleted = append(result.Deleted, metadata.DeletedKey{Key: key, Body: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode})
	}
	return result, metadataWrap(ctx, "DeleteMetadata", call.source.Guard(ctx))
}

type metadataCall struct {
	source   *cloudread.Source
	endpoint string
	headers  map[string]string
}

func (a *API) prepareMetadata(ctx context.Context, id string, options []MetadataOption) (*metadataCall, error) {
	var client *gophercloud.ServiceClient
	if a != nil {
		client = a.client
	}
	source, err := cloudread.Capture(ctx, client, "volume")
	if err != nil {
		return nil, err
	}
	if err := cloudsnapshot.ValidateBackupExportID(id); err != nil {
		return nil, err
	}
	config, err := request.Apply(metadata.Opts{}, append([]MetadataOption(nil), options...)...)
	if err != nil {
		return nil, err
	}
	if err := request.ValidateCapabilities(config, false, false, true); err != nil {
		return nil, err
	}
	// Own a distinct map before a custom callback can retain its config.
	headers := maps.Clone(config.Headers)
	probe := source.Client
	probe.MoreHeaders = headers
	probe.Microversion = ""
	if err := cindermetadata.Validate(ctx, &probe); err != nil {
		return nil, err
	}
	canonical := make(map[string]string, len(headers))
	for key, value := range headers {
		canonical[http.CanonicalHeaderKey(key)] = value
	}
	if explicit, present := canonical["If-Match"]; present {
		if configured, present := source.Client.MoreHeaders["If-Match"]; present && configured != explicit {
			return nil, metadataInvalid("If-Match conflicts with configured service header")
		}
	}
	if err := source.Guard(ctx); err != nil {
		return nil, err
	}
	endpoint := source.Client.ServiceURL("backups", url.PathEscape(id), "metadata")
	if err := rest.ValidateTarget(&source.Client, endpoint); err != nil {
		return nil, err
	}
	return &metadataCall{source: source, endpoint: endpoint, headers: canonical}, nil
}

func (call *metadataCall) perform(ctx context.Context, method, endpoint string, body any) (*rest.Response, error) {
	// Source raise_from_response accepts every final 200..399 response. Keep
	// rejected-response Read/Close faults alongside their native HTTP evidence.
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = 200 + i
	}
	rejected := make([]int, 200)
	for i := range rejected {
		rejected[i] = 400 + i
	}
	return rest.DoJSONGuardedRejections(ctx, &call.source.Client, call.source.Guard, method, endpoint, body, call.headers,
		rest.RejectionPolicy{Codes: rejected}, codes...)
}

func metadataResult(id string, response *rest.Response) *MetadataResult {
	result := &MetadataResult{BackupID: id}
	if response != nil {
		result.Response = &MetadataResponse{Body: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	}
	return result
}

func metadataDeleteKey(key string) error {
	if key == "" || key == "." || key == ".." || !utf8.ValidString(key) {
		return metadataInvalid("metadata key must be a nonempty literal UTF-8 path component")
	}
	for _, char := range key {
		if unicode.IsControl(char) {
			return metadataInvalid("metadata key must not contain controls")
		}
	}
	return nil
}

func metadataWrap(ctx context.Context, operation string, err error) error {
	return request.Wrap(operation, "backups.metadata", cloudread.ContextError(ctx, err))
}
func metadataInvalid(message string) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, message)
}

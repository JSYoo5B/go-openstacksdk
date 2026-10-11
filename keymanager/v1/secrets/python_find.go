package secrets

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// FoundSecretRecord keeps the Source descriptor view separate from the actual
// metadata row and complete HTTP envelope. RequestID is populated only by the
// direct read. Payload holds actual bytes/headers/status, never a followed ref.
// Binary payload bytes are JSON base64 in Resource and exact bytes in Payload.
type FoundSecretRecord struct {
	RequestID      string
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	Payload        *FetchedPayload
}

// FindRecord maps the named find_secret flow onto SDK-owned defaults and views.
// Safe identities attempt metadata plus conditional payload first. Clean
// HTTP400/403/404 may fall back to a complete, exact ID-or-name list search;
// fallback rows are metadata-only. HTTP errors are strict by default, unlike
// pinned Secret.fetch's unchecked custom session.get(...).json() implementation.
// On failure, including strict logical absence, an optional receipt has
// Resource=nil and must never be treated as a found Secret. Missing success
// returns nil,nil.
func (a *API) FindRecord(ctx context.Context, identity string, options ...FindRecordOption) (*FoundSecretRecord, error) {
	wrap := func(err error) error { return request.Wrap("FindRecord", "secrets", err) }
	var client *gophercloud.ServiceClient
	if a != nil {
		client = a.client
	}
	source, err := cloudread.Capture(ctx, client, "key-manager")
	if err != nil {
		return nil, wrap(err)
	}
	guard := func(ctx context.Context) error { return errors.Join(source.Guard(ctx), rest.CheckOperationGuard(ctx)) }
	config := FindRecordOpts{}
	for _, apply := range slices.Clone(options) {
		if err := guard(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(resource.ErrInvalidOption)
		}
		config = ownFindRecordOptions(config)
		applied := apply(&config)
		config = ownFindRecordOptions(config)
		if err := errors.Join(applied, guard(ctx)); err != nil {
			return nil, wrap(cloudread.ContextError(ctx, err))
		}
	}
	prepared, err := resource.PrepareIdentityFindOptions(resource.WithIdentityFindOptions(resource.IdentityFindOpts{
		IgnoreMissing: config.IgnoreMissing, Fallback: config.Fallback,
	}))
	if err := errors.Join(err, guard(ctx)); err != nil {
		return nil, wrap(cloudread.ContextError(ctx, err))
	}
	var receipt *FoundSecretRecord
	binding := resource.NewCollection(resource.Adapter[FoundSecretRecord]{
		Kind: "secrets", IdentityFind: true, ValidateID: validateFetchID,
		IdentityResponseID: func(value *FoundSecretRecord) (string, error) {
			if value == nil || value.Resource == nil {
				return "", resource.ErrInvalidOption
			}
			return findRecordString(value.Resource.Body["id"]), nil
		},
		Name:      func(value *FoundSecretRecord) string { return findRecordString(value.Resource.Body["name"]) },
		NameQuery: func(value string) string { return value },
		Get: func(ctx context.Context, id string) (*FoundSecretRecord, error) {
			value, err := fetchFindRecord(ctx, source, guard, id)
			receipt = value
			return value, err
		},
		Iterate: func(ctx context.Context, query url.Values) iter.Seq2[*FoundSecretRecord, error] {
			return listFindRecords(ctx, source, guard, query, &receipt)
		},
	})
	value, err := binding.FindIdentity(ctx, identity, resource.WithIdentityFindOptions(prepared))
	if err == nil {
		err = guard(ctx)
	}
	if err != nil {
		if receipt != nil {
			receipt.Resource = nil
		}
		return receipt, wrap(cloudread.ContextError(ctx, err))
	}
	return value, nil
}

func findRecordString(raw json.RawMessage) string {
	var value string
	_ = json.Unmarshal(raw, &value)
	return value
}
func findRecordAcceptedCodes() []int {
	values := make([]int, 300)
	for i := range values {
		values[i] = 100 + i
	}
	return values
}
func findRecordRejections() rest.RejectionPolicy {
	values := make([]int, 200)
	for i := range values {
		values[i] = 400 + i
	}
	return rest.RejectionPolicy{Codes: values, PreserveCleanRetry: true}
}
func findRecordReceipt(response *rest.Response, requestID string) *FoundSecretRecord {
	if response == nil {
		return nil
	}
	return &FoundSecretRecord{RequestID: requestID, Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
}
func findRecordReader(ctx context.Context, client *gophercloud.ServiceClient, guard func(context.Context) error, target string, headers map[string]string) (*rest.Response, error) {
	return rest.DoJSONGuardedRejectionsHeaders(ctx, client, guard, http.MethodGet, target, nil, headers, findRecordRejections(), findRecordAcceptedCodes()...)
}

func fetchFindRecord(ctx context.Context, source *cloudread.Source, guard func(context.Context) error, id string) (*FoundSecretRecord, error) {
	metadataURL := source.Client.ServiceURL("secrets", url.PathEscape(id))
	payloadURL := source.Client.ServiceURL("secrets", url.PathEscape(id), "payload")
	response, err := findRecordReader(ctx, &source.Client, guard, metadataURL, nil)
	result := findRecordReceipt(response, id)
	if err != nil {
		return result, cloudread.ContextError(ctx, err)
	}
	wire, err := rest.Decode(response, "", func(value *resource.RawResource) *resource.Metadata { return &value.Metadata })
	if err != nil {
		return result, err
	}
	result.Wire = wire.Clone()
	// A fresh Source existing(id=...) has no payload_content_type seed. A
	// response payload_content_type is therefore NOT a payload selector.
	contentType, err := fetchContentType(wire.Body, nil)
	if err != nil {
		return result, response.Fail(err)
	}
	fields := findRecordFields(wire.Body)
	if contentType != nil {
		if strings.ContainsAny(*contentType, "\r\n") {
			return result, response.Fail(resource.ErrInvalidOption)
		}
		payloadClient := source.Client
		payloadClient.MoreHeaders = maps.Clone(source.Client.MoreHeaders)
		for key := range payloadClient.MoreHeaders {
			if strings.EqualFold(key, "Accept") {
				delete(payloadClient.MoreHeaders, key)
			}
		}
		payload, err := findRecordReader(ctx, &payloadClient, guard, payloadURL, map[string]string{"Accept": *contentType})
		if payload != nil {
			result.Payload = &FetchedPayload{Body: bytes.Clone(payload.Body), Accept: *contentType, Header: payload.Header.Clone(), StatusCode: payload.StatusCode}
		}
		if err != nil {
			return result, cloudread.ContextError(ctx, err)
		}
		if *contentType == "text/plain" {
			if !utf8.Valid(payload.Body) {
				return result, payload.Fail(resource.ErrInvalidOption)
			}
			text := string(payload.Body)
			result.Payload.Text = &text
			fields["payload"], _ = json.Marshal(text)
		} else {
			// Response.Body is json.RawMessage even for opaque payload bytes.
			// Convert to []byte so JSON encodes base64 rather than parsing bytes
			// as an embedded JSON value. Payload.Body keeps the original bytes.
			fields["payload"], _ = json.Marshal([]byte(payload.Body))
		}
	}
	view, err := projectFindRecord(wire, fields, &id)
	if err != nil {
		return result, response.Fail(err)
	}
	if err := guard(ctx); err != nil {
		return result, response.Fail(cloudread.ContextError(ctx, err))
	}
	result.Resource = view
	return result, nil
}

func listFindRecords(ctx context.Context, source *cloudread.Source, guard func(context.Context) error, query url.Values, receipt **FoundSecretRecord) iter.Seq2[*FoundSecretRecord, error] {
	return func(yield func(*FoundSecretRecord, error) bool) {
		listClient := source.Client
		listClient.MoreHeaders = maps.Clone(source.Client.MoreHeaders)
		for key := range listClient.MoreHeaders {
			if strings.EqualFold(key, "Accept") {
				delete(listClient.MoreHeaders, key)
			}
		}
		var page *rest.Response
		spec := rest.CollectionSpec[resource.RawResource]{
			Client: &listClient, Path: "secrets", Kind: "secrets", PluralKey: "secrets",
			Metadata: func(value *resource.RawResource) *resource.Metadata { return &value.Metadata },
			Validate: guard, SourceGuard: guard, ListCodes: findRecordAcceptedCodes(),
			Paging: rest.PagePolicy[resource.RawResource]{OffsetPagination: true, HTTPLink: true, StopOnEmptyPage: true, SingletonObject: true, IgnoreFalseyNext: true, VersionedPath: "/v1/secrets", DecodeNoContent: true},
			ReadPage: func(ctx context.Context, target string, codes ...int) (*rest.Response, error) {
				response, err := findRecordReader(ctx, &listClient, guard, target, map[string]string{"Accept": "application/json"})
				page = response
				*receipt = findRecordReceipt(response, "")
				return response, err
			},
		}
		for wire, err := range rest.List(ctx, spec, query) {
			if err != nil {
				yield(nil, cloudread.ContextError(ctx, errors.Join(err, guard(ctx))))
				return
			}
			// Resource.list passes these constructor controls explicitly. A row
			// containing one duplicates an argument to existing/__init__ even
			// when its value is null or false. Direct Secret.fetch instead consumes
			// Body attributes and silently ignores these undeclared metadata keys.
			for _, key := range []string{"connection", "microversion", "_synchronized"} {
				if _, exists := wire.Body[key]; exists {
					cause := fmt.Errorf("%w: Source secret list row duplicates constructor key %q", resource.ErrInvalidOption, key)
					yield(nil, page.Fail(cloudread.ContextError(ctx, errors.Join(cause, guard(ctx)))))
					return
				}
			}
			view, err := projectFindRecord(wire, findRecordFields(wire.Body), nil)
			if err != nil {
				yield(nil, page.Fail(err))
				return
			}
			if err := guard(ctx); err != nil {
				yield(nil, page.Fail(cloudread.ContextError(ctx, err)))
				return
			}
			value := findRecordReceipt(page, "")
			value.Wire, value.Resource = wire.Clone(), view
			*receipt = value
			if !yield(value, nil) {
				return
			}
			if err := guard(ctx); err != nil {
				yield(nil, page.Fail(cloudread.ContextError(ctx, err)))
				return
			}
		}
	}
}

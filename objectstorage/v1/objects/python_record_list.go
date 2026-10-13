package objects

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

var objectListAbsent = []string{"Authorization", "X-Service-Token", "Host", "Cookie", "Content-Length", "Transfer-Encoding", "Connection", "Proxy-Connection", "Proxy-Authorization", "Trailer", "TE", "Upgrade", "X-OpenStack-Nova-API-Version", "X-OpenStack-Volume-API-Version", "X-OpenStack-Manila-API-Version", "X-OpenStack-Ironic-API-Version", "X-OpenStack-Ironic-Inspector-API-Version"}

func objectListHeaders(input map[string]string, source bool, version string) (map[string]string, error) {
	headers := make(map[string]string, len(input))
	for key, value := range input {
		if !metadataToken(key) || !metadataFieldValue(value) {
			return nil, metadataInvalid("invalid object list header %q", key)
		}
		if strings.EqualFold(key, "X-Auth-Token") {
			return nil, metadataInvalid("authentication is provider owned")
		}
		for _, reserved := range objectListAbsent {
			if strings.EqualFold(key, reserved) {
				return nil, metadataInvalid("object list header %q is SDK owned", key)
			}
		}
		if strings.EqualFold(key, "OpenStack-API-Version") && (!source || version == "" || value != "object-store "+version) {
			return nil, metadataInvalid("Swift version header conflicts with selected microversion")
		}
		canonical := http.CanonicalHeaderKey(key)
		if old, exists := headers[canonical]; exists && old != value {
			return nil, metadataInvalid("conflicting object list header aliases")
		}
		headers[canonical] = value
	}
	return headers, nil
}

type objectListSource struct {
	api                                      *API
	ctx                                      context.Context
	source                                   *gophercloud.ServiceClient
	client                                   gophercloud.ServiceClient
	provider                                 *gophercloud.ProviderClient
	endpoint, base, kind, version, container string
	target                                   string
	outer                                    func(context.Context) error
	mu                                       sync.Mutex
	observed                                 error
}

func (a *API) captureObjectList(ctx context.Context, container string) (*objectListSource, error) {
	var selected *gophercloud.ServiceClient
	if a != nil {
		selected = a.client
	}
	if selected == nil {
		return nil, metadataInvalid("object list client required")
	}
	// Reuse fixed Swift source facts without the metadata-mutation header
	// exclusions; listing has its own operation header policy.
	facts := *selected
	facts.MoreHeaders = nil
	if _, err := validateMetadataSource(ctx, &facts); err != nil {
		return nil, err
	}
	if err := validateMetadataContainer(container); err != nil {
		return nil, metadataContextError(ctx, err)
	}
	headers, err := objectListHeaders(selected.MoreHeaders, true, selected.Microversion)
	if err != nil {
		return nil, err
	}
	target := selected.ServiceURL(url.PathEscape(container))
	if err := rest.ValidateTarget(selected, target); err != nil {
		return nil, metadataContextError(ctx, err)
	}
	client := *selected
	client.MoreHeaders = headers
	if client.Type == "" {
		client.Type = "object-store"
	}
	p := &objectListSource{api: a, ctx: ctx, source: selected, client: client, provider: selected.ProviderClient, endpoint: selected.Endpoint, base: selected.ResourceBase, kind: selected.Type, version: selected.Microversion, container: container, target: target, outer: rest.OperationGuard(ctx)}
	rest.RegisterOperationSource(ctx, p.sourceCheck)
	return p, p.check(ctx)
}
func (p *objectListSource) sourceCheck(ctx context.Context) error {
	var err error
	if p.api.client != p.source || p.source.ProviderClient != p.provider || p.source.Endpoint != p.endpoint || p.source.ResourceBase != p.base || p.source.Type != p.kind || p.source.Microversion != p.version || p.source.ServiceURL(url.PathEscape(p.container)) != p.target {
		err = metadataInvalid("object list source changed")
	}
	facts := *p.source
	facts.MoreHeaders = nil
	_, factsErr := validateMetadataSource(p.ctx, &facts)
	_, headerErr := objectListHeaders(p.source.MoreHeaders, true, p.source.Microversion)
	err = errors.Join(err, factsErr, headerErr)
	p.mu.Lock()
	defer p.mu.Unlock()
	p.observed = errors.Join(p.observed, err)
	return metadataContextError(ctx, metadataContextError(p.ctx, p.observed))
}
func (p *objectListSource) check(ctx context.Context) error {
	err := p.sourceCheck(ctx)
	if p.outer != nil {
		err = errors.Join(err, p.outer(p.ctx))
	}
	return metadataContextError(ctx, err)
}
func objectListStatuses() ([]int, []int) {
	accepted, rejected := make([]int, 300), make([]int, 200)
	for index := range accepted {
		accepted[index] = 100 + index
	}
	for index := range rejected {
		rejected[index] = 400 + index
	}
	return accepted, rejected
}
func (p *objectListSource) read(ctx context.Context, target string) (*rest.Response, error) {
	headers := rest.RequestHeaderPolicy{Values: map[string]string{"Accept": p.client.MoreHeaders["Accept"]}, Absent: slices.Clone(objectListAbsent)}
	if version := p.client.Microversion; version != "" {
		headers.Values["OpenStack-API-Version"] = "object-store " + version
	} else {
		headers.Absent = append(headers.Absent, "OpenStack-API-Version")
	}
	for _, key := range []string{"If-Match", "If-None-Match", "If-Modified-Since", "If-Unmodified-Since"} {
		if value, exists := p.client.MoreHeaders[key]; exists {
			headers.Values[key] = value
		} else {
			headers.Absent = append(headers.Absent, key)
		}
	}
	accepted, rejected := objectListStatuses()
	response, err := rest.DoJSONGuardedRejectionsHeaderPolicy(ctx, &p.client, p.check, http.MethodGet, target, nil, headers, rest.RejectionPolicy{Codes: rejected, PreserveCleanRetry: true, StopFaultRetryCallbacks: true}, accepted...)
	// A native rejection has no shared REST Response; project only its actual
	// receipt without reading or closing the body again.
	if response == nil && err != nil {
		var native gophercloud.ErrUnexpectedResponseCode
		receiptErr := err
		var reauth *gophercloud.ErrUnableToReauthenticate
		if errors.As(err, &reauth) {
			receiptErr = reauth.ErrOriginal
		}
		var after *gophercloud.ErrErrorAfterReauthentication
		if errors.As(receiptErr, &after) {
			receiptErr = after.ErrOriginal
		}
		if errors.As(receiptErr, &native) {
			response = &rest.Response{Body: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual}
		}
	}
	return response, err
}

// objectListQuery validates, transposes and classifies the Source **query.
// Server keys follow Object._query_mapping (client name preferred over the
// server spelling), declared Body keys become local filters and format=json
// is always added by the proxy.
func objectListQuery(cfg RecordListOpts) (url.Values, map[string]json.RawMessage, bool, error) {
	if cfg.MaxItems < 0 || cfg.MaxPages < 0 || cfg.Limit != nil && *cfg.Limit < 0 {
		return nil, nil, false, metadataInvalid("negative object list control")
	}
	raw := cloneObjectListFields(cfg.Query)
	if raw == nil {
		raw = make(map[string]json.RawMessage)
	}
	for key := range raw {
		if err := objectListReservedQuery(key); err != nil {
			return nil, nil, false, err
		}
		if !objectListServerKey(key) && !objectListBodyKey(key) {
			delete(raw, key)
		}
	}
	if cfg.Limit != nil {
		raw["limit"], _ = json.Marshal(*cfg.Limit)
	}
	for key, ptr := range map[string]*string{"marker": cfg.Marker, "prefix": cfg.Prefix} {
		if ptr != nil {
			if !utf8.ValidString(*ptr) {
				return nil, nil, false, metadataInvalid("object list query must be UTF-8")
			}
			raw[key], _ = json.Marshal(*ptr)
		}
	}
	for key, value := range raw {
		if !utf8.Valid(value) || !json.Valid(value) {
			return nil, nil, false, metadataInvalid("object list query %q needs valid UTF-8 JSON", key)
		}
	}
	server := make(map[string]json.RawMessage)
	for _, pair := range objectListQueryMapping {
		if value, exists := raw[pair[0]]; exists {
			server[pair[1]] = value
		} else if value, exists := raw[pair[1]]; exists {
			server[pair[1]] = value
		}
	}
	server["format"] = json.RawMessage(`"json"`)
	limitTruthy, err := cloudfilter.PythonJSONTruthy(server["limit"])
	if err != nil {
		return nil, nil, false, err
	}
	if cfg.MaxItems > 0 && !limitTruthy {
		server["limit"], _ = json.Marshal(cfg.MaxItems)
		limitTruthy = true
	}
	query := make(url.Values)
	for key, value := range server {
		encoded, err := cloudfilter.RequestQueryValues(value)
		if err != nil {
			return nil, nil, false, err
		}
		if len(encoded) > 0 {
			query[key] = encoded
		}
	}
	filters := make(map[string]json.RawMessage)
	for key, value := range raw {
		if objectListBodyKey(key) {
			filters[key] = bytes.Clone(value)
		}
	}
	return query, filters, limitTruthy, nil
}
func objectListFilterOrder(cfg RecordListOpts, filters map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(filters))
	seen := make(map[string]bool)
	for _, key := range cfg.queryOrder {
		if _, exists := filters[key]; exists && !seen[key] {
			keys = append(keys, key)
			seen[key] = true
		}
	}
	remainder := make([]string, 0, len(filters))
	for key := range filters {
		if !seen[key] {
			remainder = append(remainder, key)
		}
	}
	sort.Strings(remainder)
	return append(keys, remainder...)
}

// ListRecords implements the named Source objects(container, **query) listing:
// GET on the escaped container path with format=json and Accept
// application/json, Object query mapping, local Body filters, X-Container-
// Object-Count and limit/marker continuation with the last row name as marker,
// and an Object-shaped record for every row (subdir rows included). Errors are
// wrapped as resource.OperationError with operation ListRecords and keep the
// actual page receipt in the yielded record.
func (a *API) ListRecords(ctx context.Context, container string, options ...RecordListOption) iter.Seq2[*ObjectRecord, error] {
	owned := slices.Clone(options)
	return func(yield func(*ObjectRecord, error) bool) {
		fail := func(page *rest.Response, err error) {
			if page != nil {
				err = page.Fail(err)
			}
			yield(objectListReceipt(container, page), request.Wrap("ListRecords", "objects", metadataContextError(ctx, err)))
		}
		p, err := a.captureObjectList(ctx, container)
		if err != nil {
			fail(nil, err)
			return
		}
		cfg := copyObjectListOpts(RecordListOpts{})
		for _, option := range owned {
			if err := p.check(ctx); err != nil {
				fail(nil, err)
				return
			}
			if option == nil {
				fail(nil, metadataInvalid("nil object list option"))
				return
			}
			next := copyObjectListOpts(cfg)
			if err := errors.Join(option(&next), p.check(ctx)); err != nil {
				fail(nil, err)
				return
			}
			cfg = copyObjectListOpts(next)
		}
		query, filters, limitTruthy, err := objectListQuery(cfg)
		if err != nil {
			fail(nil, err)
			return
		}
		headers, err := objectListHeaders(cfg.Headers, false, "")
		if err != nil {
			fail(nil, err)
			return
		}
		// Resource.list supplies its JSON Accept before method headers, so a
		// captured service-level Accept cannot replace the operation default.
		p.client.MoreHeaders["Accept"] = "application/json"
		maps.Copy(p.client.MoreHeaders, headers)
		if cfg.Microversion != nil {
			if !metadataFieldValue(*cfg.Microversion) {
				fail(nil, metadataInvalid("invalid Swift microversion"))
				return
			}
			p.client.Microversion = *cfg.Microversion
		}
		for key := range p.client.MoreHeaders {
			if strings.EqualFold(key, "OpenStack-API-Version") {
				delete(p.client.MoreHeaders, key)
			}
		}
		collection, err := url.Parse(p.target)
		if err != nil {
			fail(nil, err)
			return
		}
		current := *collection
		current.RawQuery = query.Encode()
		seen := map[string]bool{objectListPageKey(&current): true}
		markers := make(map[string]bool)
		if query.Has("marker") {
			markers[objectListMarkerKey(query)] = true
		}
		pages := cfg.MaxPages
		if pages == 0 {
			pages = 10000
		}
		keys := objectListFilterOrder(cfg, filters)
		count := 0
		for pageNumber := 0; ; pageNumber++ {
			response, err := p.read(ctx, current.String())
			if err != nil {
				fail(response, err)
				return
			}
			if !utf8.Valid(response.Body) {
				fail(response, metadataInvalid("object list response must be UTF-8"))
				return
			}
			rows, err := objectListRows(response)
			if err != nil {
				fail(response, err)
				return
			}
			var last json.RawMessage
			for _, raw := range rows {
				if err := p.check(ctx); err != nil {
					fail(response, err)
					return
				}
				value, err := objectListProject(container, raw, response)
				if err != nil {
					fail(response, err)
					return
				}
				last = bytes.Clone(value.Resource.Body["id"])
				count++
				matched := true
				for _, key := range keys {
					yes, err := objectListMatch(value.Resource.Body[key], filters[key])
					if err != nil {
						fail(response, err)
						return
					}
					if !yes {
						matched = false
						break
					}
				}
				if matched && !yield(value, nil) {
					return
				}
				if err := p.check(ctx); err != nil {
					fail(response, err)
					return
				}
				if cfg.MaxItems > 0 && count >= cfg.MaxItems {
					return
				}
			}
			if len(rows) == 0 || cfg.SinglePage {
				return
			}
			next, err := objectListNext(response, collection, &current, query, limitTruthy, last, count)
			if err != nil {
				fail(response, err)
				return
			}
			if next == nil {
				return
			}
			if pageNumber+1 >= pages {
				fail(response, metadataInvalid("object listing exceeded owned page bound"))
				return
			}
			key, marker := objectListPageKey(next), objectListMarkerKey(next.Query())
			if seen[key] || next.Query().Has("marker") && markers[marker] {
				fail(response, &resource.PaginationCycleError{URL: next.String()})
				return
			}
			seen[key] = true
			if next.Query().Has("marker") {
				markers[marker] = true
			}
			current = *next
		}
	}
}

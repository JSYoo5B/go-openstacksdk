package containers

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

var listRecordAbsent = []string{"Authorization", "X-Service-Token", "Host", "Cookie", "Content-Length", "Transfer-Encoding", "Connection", "Proxy-Connection", "Proxy-Authorization", "Trailer", "TE", "Upgrade", "X-OpenStack-Nova-API-Version", "X-OpenStack-Volume-API-Version", "X-OpenStack-Manila-API-Version", "X-OpenStack-Ironic-API-Version", "X-OpenStack-Ironic-Inspector-API-Version"}

func listRecordHeaders(input map[string]string, source bool, version string) (map[string]string, error) {
	headers := make(map[string]string, len(input))
	for key, value := range input {
		if !metadataToken(key) || !metadataFieldValue(value) {
			return nil, metadataInvalid("invalid container list header %q", key)
		}
		if strings.EqualFold(key, "X-Auth-Token") {
			return nil, metadataInvalid("authentication is provider owned")
		}
		for _, reserved := range listRecordAbsent {
			if strings.EqualFold(key, reserved) {
				return nil, metadataInvalid("container list header %q is SDK owned", key)
			}
		}
		if strings.EqualFold(key, "OpenStack-API-Version") && (!source || version == "" || value != "object-store "+version) {
			return nil, metadataInvalid("Swift version header conflicts with selected microversion")
		}
		canonical := http.CanonicalHeaderKey(key)
		if old, exists := headers[canonical]; exists && old != value {
			return nil, metadataInvalid("conflicting container list header aliases")
		}
		headers[canonical] = value
	}
	return headers, nil
}

type listRecordSource struct {
	api                           *API
	source                        *gophercloud.ServiceClient
	client                        gophercloud.ServiceClient
	provider                      *gophercloud.ProviderClient
	endpoint, base, kind, version string
	outer                         func(context.Context) error
	mu                            sync.Mutex
	observed                      error
}

func (a *API) captureListRecord(ctx context.Context) (*listRecordSource, error) {
	var selected *gophercloud.ServiceClient
	if a != nil {
		selected = a.client
	}
	if selected == nil {
		return nil, metadataInvalid("container list client required")
	}
	// Reuse fixed Swift source facts without applying metadata-mutation-only
	// header exclusions (X-Newest and arbitrary source metadata have no writes).
	facts := *selected
	facts.MoreHeaders = nil
	if _, err := validateMetadataSource(ctx, &facts); err != nil {
		return nil, err
	}
	headers, err := listRecordHeaders(selected.MoreHeaders, true, selected.Microversion)
	if err != nil {
		return nil, err
	}
	client := *selected
	client.MoreHeaders = headers
	if client.Type == "" {
		client.Type = "object-store"
	}
	p := &listRecordSource{api: a, source: selected, client: client, provider: selected.ProviderClient, endpoint: selected.Endpoint, base: selected.ResourceBase, kind: selected.Type, version: selected.Microversion, outer: rest.OperationGuard(ctx)}
	rest.RegisterOperationSource(ctx, p.sourceCheck)
	return p, p.check(ctx)
}
func (p *listRecordSource) sourceCheck(ctx context.Context) error {
	var err error
	if p.api.client != p.source || p.source.ProviderClient != p.provider || p.source.Endpoint != p.endpoint || p.source.ResourceBase != p.base || p.source.Type != p.kind || p.source.Microversion != p.version {
		err = metadataInvalid("container list source changed")
	}
	facts := *p.source
	facts.MoreHeaders = nil
	_, factsErr := validateMetadataSource(ctx, &facts)
	_, headerErr := listRecordHeaders(p.source.MoreHeaders, true, p.source.Microversion)
	err = errors.Join(err, factsErr, headerErr)
	p.mu.Lock()
	defer p.mu.Unlock()
	p.observed = errors.Join(p.observed, err)
	return metadataContextError(ctx, p.observed)
}
func (p *listRecordSource) check(ctx context.Context) error {
	err := p.sourceCheck(ctx)
	if p.outer != nil {
		err = errors.Join(err, p.outer(ctx))
	}
	return metadataContextError(ctx, err)
}
func (p *listRecordSource) read(ctx context.Context, target string) (*rest.Response, error) {
	headers := rest.RequestHeaderPolicy{Values: map[string]string{"Accept": p.client.MoreHeaders["Accept"]}, Absent: slices.Clone(listRecordAbsent)}
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
	accepted, rejected := recordStatuses()
	response, err := rest.DoJSONGuardedRejectionsHeaderPolicy(ctx, &p.client, p.check, http.MethodGet, target, nil, headers, rest.RejectionPolicy{Codes: rejected, PreserveCleanRetry: true}, accepted...)
	// Native rejected responses are errors, not accepted pages. The shared REST
	// reader deliberately returns no Response for those errors; project only the
	// actual native receipt without reading, closing or replaying the body again.
	if response == nil && err != nil {
		var native gophercloud.ErrUnexpectedResponseCode
		if errors.As(err, &native) {
			response = &rest.Response{Body: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual}
		}
	}
	return response, err
}
func listRecordQuery(cfg RecordListOpts) (url.Values, map[string]json.RawMessage, bool, error) {
	if cfg.MaxItems < 0 || cfg.MaxPages < 0 || cfg.Limit != nil && *cfg.Limit < 0 {
		return nil, nil, false, metadataInvalid("negative container list control")
	}
	raw := cloneRecordFields(cfg.Query)
	if _, exists := raw["paginated"]; exists {
		return nil, nil, false, metadataInvalid("named Source containers fixes paginated=True")
	}
	if cfg.Limit != nil {
		raw["limit"], _ = json.Marshal(*cfg.Limit)
	}
	for key, ptr := range map[string]*string{"marker": cfg.Marker, "prefix": cfg.Prefix, "format": cfg.Format} {
		if ptr != nil {
			if !utf8.ValidString(*ptr) {
				return nil, nil, false, metadataInvalid("container list query must be UTF-8")
			}
			raw[key], _ = json.Marshal(*ptr)
		}
	}
	limitTruthy, err := cloudfilter.PythonTruthy(raw["limit"])
	if err != nil {
		return nil, nil, false, err
	}
	if cfg.MaxItems > 0 && !limitTruthy {
		raw["limit"], _ = json.Marshal(cfg.MaxItems)
		limitTruthy = true
	}
	query, filters := make(url.Values), make(map[string]json.RawMessage)
	for key, value := range raw {
		if !listRecordServerKey(key) && !listRecordBodyKey(key) {
			continue
		}
		if !utf8.Valid(value) || !json.Valid(value) {
			return nil, nil, false, metadataInvalid("container list query needs valid UTF-8 JSON")
		}
		if listRecordBodyKey(key) {
			filters[key] = bytes.Clone(value)
			continue
		}
		encoded, err := cloudfilter.RequestQueryValues(value)
		if err != nil {
			return nil, nil, false, err
		}
		if len(encoded) > 0 {
			query[key] = encoded
		}
	}
	return query, filters, limitTruthy, nil
}

// ListRecords implements the named Source containers query with owned rows and
// actual account page receipts. It never fetches per-container metadata.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*ContainerRecord, error] {
	owned := slices.Clone(options)
	return func(yield func(*ContainerRecord, error) bool) {
		fail := func(page *rest.Response, err error) {
			if page != nil {
				err = page.Fail(err)
			}
			yield(listRecordReceipt(page), request.Wrap("ListRecords", "containers", metadataContextError(ctx, err)))
		}
		p, err := a.captureListRecord(ctx)
		if err != nil {
			fail(nil, err)
			return
		}
		cfg := copyListRecordOpts(RecordListOpts{})
		for _, option := range owned {
			if err := p.check(ctx); err != nil {
				fail(nil, err)
				return
			}
			if option == nil {
				fail(nil, metadataInvalid("nil container list option"))
				return
			}
			next := copyListRecordOpts(cfg)
			if err := errors.Join(option(&next), p.check(ctx)); err != nil {
				fail(nil, err)
				return
			}
			cfg = copyListRecordOpts(next)
		}
		query, filters, limitTruthy, err := listRecordQuery(cfg)
		if err != nil {
			fail(nil, err)
			return
		}
		headers, err := listRecordHeaders(cfg.Headers, false, "")
		if err != nil {
			fail(nil, err)
			return
		}
		// Resource.list supplies its own JSON default before method headers.
		// A captured service-level Accept cannot replace that operation default.
		p.client.MoreHeaders["Accept"] = "application/json"
		maps.Copy(p.client.MoreHeaders, headers)
		if cfg.Microversion != nil {
			if !metadataFieldValue(*cfg.Microversion) {
				fail(nil, metadataInvalid("invalid Swift microversion"))
				return
			}
			p.client.Microversion = *cfg.Microversion
			delete(p.client.MoreHeaders, "Openstack-Api-Version")
		}
		for key := range p.client.MoreHeaders {
			if strings.EqualFold(key, "OpenStack-API-Version") {
				delete(p.client.MoreHeaders, key)
			}
		}
		collection, err := url.Parse(p.client.ServiceURL(""))
		if err != nil {
			fail(nil, err)
			return
		}
		current := *collection
		current.RawQuery = query.Encode()
		seen := map[string]bool{listRecordPageKey(&current): true}
		markers := make(map[string]bool)
		if query.Has("marker") {
			markers[listRecordMarkerKey(query)] = true
		}
		pages := cfg.MaxPages
		if pages == 0 {
			pages = 10000
		}
		keys := make([]string, 0, len(filters))
		seenFilters := make(map[string]bool)
		for _, key := range cfg.queryOrder {
			if _, exists := filters[key]; exists && !seenFilters[key] {
				keys = append(keys, key)
				seenFilters[key] = true
			}
		}
		remainder := make([]string, 0, len(filters))
		for key := range filters {
			if !seenFilters[key] {
				remainder = append(remainder, key)
			}
		}
		sort.Strings(remainder)
		keys = append(keys, remainder...)
		count := 0
		for pageNumber := 0; ; pageNumber++ {
			response, err := p.read(ctx, current.String())
			if err != nil {
				fail(response, err)
				return
			}
			if !utf8.Valid(response.Body) {
				fail(response, metadataInvalid("container list response must be UTF-8"))
				return
			}
			rows, err := listRecordRows(response)
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
				value, err := listRecordProject(raw, response)
				if err != nil {
					fail(response, err)
					return
				}
				last = bytes.Clone(value.Resource.Body["id"])
				count++
				matched := true
				for _, key := range keys {
					yes, err := listRecordMatch(value.Resource.Body[key], filters[key])
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
			next, err := listRecordNext(response, collection, &current, query, limitTruthy, last, count)
			if err != nil {
				fail(response, err)
				return
			}
			if next == nil {
				return
			}
			if pageNumber+1 >= pages {
				fail(response, metadataInvalid("container listing exceeded owned page bound"))
				return
			}
			key, marker := listRecordPageKey(next), listRecordMarkerKey(next.Query())
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

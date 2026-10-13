package shareattachments

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"maps"
	"net/http"
	"net/url"
	"strconv"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func descriptor() *resource.FilterDescriptor {
	body := make(map[string]string)
	for _, key := range recordFields() {
		body[key] = key
	}
	return &resource.FilterDescriptor{Query: map[string]string{"limit": "limit", "marker": "marker"}, Body: body, Reserved: []string{"cls", "self", "session", "resource_type", "connection", "server_id", "base_path", "list_base_path", "microversion", "headers", "__conflicting_attrs", "jmespath_filters", "paginated", "max_items", "allow_unknown_params"}}
}
func listMarker(value *Record) (string, error) {
	raw, present := value.Wire.Body["id"]
	if !present {
		raw = value.Wire.Body["share_id"]
	}
	var marker string
	if err := json.Unmarshal(raw, &marker); err != nil {
		return "", invalid("pagination id/share_id must be a string")
	}
	if err := validIdentity(marker); err != nil {
		return "", err
	}
	return marker, nil
}

// ListRecords is lazy. Source Body filters run locally after a raw-row cap;
// unknown filters are discarded without invoking their JSON marshaler.
func (a *API) ListRecords(ctx context.Context, server string, options ...ListOption) iter.Seq2[*Record, error] {
	owned := append([]ListOption(nil), options...)
	return func(yield func(*Record, error) bool) {
		fail := func(err error) { yield(nil, wrap(ctx, "ListRecords", err)) }
		p, err := a.capture(ctx, server)
		if err != nil {
			fail(err)
			return
		}
		cfg, err := cloudread.ApplyReadOptions(p.ctx, ListOpts{}, owned, ownList, p.guard)
		if err == nil {
			err = request.ValidateCapabilities(cfg, false, false, true)
		}
		if err != nil {
			fail(err)
			return
		}
		policy := cfg.Options
		if policy.Limit < 0 || policy.MaxItems < 0 {
			fail(invalid("limit and max items must be nonnegative"))
			return
		}
		if policy.Marker != "" {
			if err := validIdentity(policy.Marker); err != nil {
				fail(err)
				return
			}
		}
		if cfg.Headers == nil {
			cfg.Headers = make(map[string]string)
		}
		if !hasHeader(cfg.Headers, "Accept") {
			cfg.Headers["Accept"] = "application/json"
		}
		// Prepare the semantic classifier before discovery, including reserved controls.
		selection, err := resource.PrepareFilterSelectionGuarded(descriptor(), func() error { return p.guard(p.ctx) }, policy.Filters...)
		if err != nil {
			fail(err)
			return
		}
		query := make(url.Values)
		if policy.Limit != 0 {
			query.Set("limit", strconv.Itoa(policy.Limit))
		}
		if policy.Marker != "" {
			query.Set("marker", policy.Marker)
		}
		for key, values := range selection.Query {
			if _, exists := query[key]; exists {
				fail(invalid("semantic %q conflicts with typed list query", key))
				return
			}
			query[key] = append([]string(nil), values...)
		}
		// The REST pager validates primitive query values before any network activity.
		if value, exists := query["limit"]; exists {
			if len(value) != 1 {
				fail(invalid("limit must be one integer"))
				return
			}
			n, e := strconv.Atoi(value[0])
			if e != nil || n < 0 {
				fail(invalid("limit must be nonnegative"))
				return
			}
		}
		if value, exists := query["marker"]; exists {
			if len(value) != 1 {
				fail(invalid("marker must be one literal ID"))
				return
			}
			if err := validIdentity(value[0]); err != nil {
				fail(err)
				return
			}
		}
		if err = p.selectPolicy(policy.Microversion, cfg.Headers); err != nil {
			fail(err)
			return
		}
		var page *rest.Response
		spec := rest.CollectionSpec[Record]{Client: &p.source.Client, Path: p.path, Kind: "shareattachments", PluralKey: "shares", Metadata: recordMetadata, Validate: p.guard, SourceGuard: p.guard, ListCodes: codes(200),
			ReadPage: func(ctx context.Context, target string, _ ...int) (*rest.Response, error) {
				return p.do(http.MethodGet, target, nil)
			},
			ValidateResponse: func(response *rest.Response) error { page = response; return p.guard(p.ctx) },
			ValidateItem: func(value *Record) error {
				if err := p.guard(p.ctx); err != nil {
					return err
				}
				value.ServerID = server
				value.Discovery = p.discoveryCopy()
				value.Header = value.Wire.Header.Clone()
				value.StatusCode = value.Wire.StatusCode
				value.Resource = project(value.Wire.Body, value.Header, value.StatusCode)
				if page != nil {
					value.Envelope = bytes.Clone(page.Body)
				}
				return p.guard(p.ctx)
			},
			Paging: rest.PagePolicy[Record]{LinkKeys: []string{"links", "shares_links"}, HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, NextTruthy: cloudfilter.PythonJSONTruthy, VersionedPath: "/v2.1/" + p.path, MarkerFallback: true, Marker: listMarker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true},
		}
		for value, err := range rest.ListWithControl(p.ctx, spec, maps.Clone(query), rest.ListControl{MaxItems: policy.MaxItems, SinglePage: policy.Paginated != nil && !*policy.Paginated, LimitHint: true}) {
			if err != nil {
				yield(value, wrap(ctx, "ListRecords", err))
				return
			}
			if err := p.guard(p.ctx); err != nil {
				fail(page.Fail(err))
				return
			}
			matches, err := sourceLocalFilters(value, selection.Body)
			if err = errors.Join(err, p.guard(p.ctx)); err != nil {
				fail(page.Fail(cloudread.ContextError(p.ctx, err)))
				return
			}
			if matches && !yield(value, nil) {
				return
			}
			if err := errors.Join(p.guard(p.ctx), rest.CheckOperationGuard(p.ctx)); err != nil {
				fail(page.Fail(cloudread.ContextError(p.ctx, err)))
				return
			}
		}
	}
}

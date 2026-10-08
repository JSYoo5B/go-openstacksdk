package metadefnamespaces

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// ListRecords lazily lists Source-shaped namespaces using declared semantic
// filters and generic guarded pagination. Each iteration owns source and options.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*Record, error] {
	return listRecords(ctx, options, func() (*preparedSource, error) { return a.capture(ctx) })
}

// AllRecords retains already consumed records when a later row or page fails.
func (a *API) AllRecords(ctx context.Context, options ...RecordListOption) ([]*Record, error) {
	return collectRecords(a.ListRecords(ctx, options...))
}
func collectRecords(stream iter.Seq2[*Record, error]) ([]*Record, error) {
	values := make([]*Record, 0)
	for value, err := range stream {
		if err != nil {
			return values, err
		}
		values = append(values, value)
	}
	return values, nil
}
func listRecords(ctx context.Context, options []RecordListOption, capture func() (*preparedSource, error)) iter.Seq2[*Record, error] {
	owned := slices.Clone(options)
	return func(yield func(*Record, error) bool) {
		fail := func(err error) { yield(nil, wrap(ctx, "ListRecords", err)) }
		p, err := capture()
		if err != nil {
			fail(err)
			return
		}
		outer := rest.OperationGuard(ctx)
		var observed error
		check := func(checkCtx context.Context) error {
			if observed != nil {
				return observed
			}
			var parent error
			if outer != nil {
				parent = outer(checkCtx)
			}
			observed = errors.Join(p.check(checkCtx), parent)
			return observed
		}
		opctx := rest.WithOperationGuard(ctx, check)
		if err := check(opctx); err != nil {
			fail(err)
			return
		}
		location := json.RawMessage("null")
		if p.api.dependencies.CloudLocation != nil {
			facts, readErr := p.api.dependencies.CloudLocation()
			err = readErr
			if err == nil {
				location, err = facts.ForResource(nil, facts.Zone)
			}
		}
		if err = errors.Join(err, check(opctx)); err != nil {
			fail(err)
			return
		}
		parameters, err := prepareRecordList(opctx, check, owned)
		if err = errors.Join(err, check(opctx)); err != nil {
			fail(err)
			return
		}
		for key, value := range parameters.headers {
			p.client.MoreHeaders[key] = value
		}
		if _, present := p.client.MoreHeaders["Accept"]; !present {
			p.client.MoreHeaders["Accept"] = "application/json"
		}
		codes := make([]int, 200)
		for i := range codes {
			codes[i] = 200 + i
		}
		var origin *rest.Response
		spec := rest.CollectionSpec[recordRow]{Client: p.client, Path: strings.TrimPrefix(p.url(nil), p.base), Kind: kind,
			PluralKey: "namespaces", Metadata: recordMetadata, Validate: check, SourceGuard: check, ListCodes: codes,
			ValidateResponse: func(response *rest.Response) error {
				origin = response
				if !utf8.Valid(response.Body) {
					return invalid("record page must be UTF-8")
				}
				return check(opctx)
			},
			ValidateItem: func(value *recordRow) error {
				if err := check(opctx); err != nil {
					return err
				}
				return errors.Join(prepareRecord(value, location, origin.Body), check(opctx))
			},
			Paging: rest.PagePolicy[recordRow]{HTTPLink: true, DictionaryLinks: true, MarkerFallback: true, Marker: recordMarker, MarkerOnShortPage: true,
				AllowFirstServerLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true, AllowZeroLimit: true, VersionedPath: "/v2/metadefs/namespaces"},
		}
		filters := make(map[string]json.RawMessage, len(parameters.filters))
		for key, raw := range parameters.filters {
			filters[key] = json.RawMessage(raw)
		}
		for value, readErr := range rest.ListWithControl(opctx, spec, parameters.query, parameters.control) {
			if readErr != nil {
				fail(readErr)
				return
			}
			matched, matchErr := jsonfilter.MatchFilters(value.Resource.Body, filters)
			if matchErr != nil {
				fail(origin.Fail(matchErr))
				return
			}
			if matched && !yield(&value.Record, nil) {
				return
			}
			if err := check(opctx); err != nil {
				fail(origin.Fail(err))
				return
			}
		}
	}
}

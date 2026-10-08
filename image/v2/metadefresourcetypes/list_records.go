package metadefresourcetypes

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

// ListRecords lazily lists declared global resource records with semantic
// filters and guarded pagination. Each iteration owns its source and options.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*Record, error] {
	return listRecords(ctx, options, func() (*preparedSource, error) { return a.capture(ctx) })
}

// ListRecords retains the namespace URI in every declared association record.
func (s *NamespaceScope) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*Record, error] {
	return listRecords(ctx, options, func() (*preparedSource, error) { return s.capture(ctx, nil) })
}

// AllRecords retains already consumed records when a later page fails.
func (a *API) AllRecords(ctx context.Context, options ...RecordListOption) ([]*Record, error) {
	return collectRecords(a.ListRecords(ctx, options...))
}
func (s *NamespaceScope) AllRecords(ctx context.Context, options ...RecordListOption) ([]*Record, error) {
	return collectRecords(s.ListRecords(ctx, options...))
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
		var namespace *string
		if p.scope != nil {
			namespace = copyPointer(&p.scope.namespace)
		}
		outer := rest.OperationGuard(ctx)
		var observed error
		check := func(checkCtx context.Context) error {
			if observed != nil {
				return observed
			}
			var parent, scope error
			if outer != nil {
				parent = outer(checkCtx)
			}
			if namespace != nil && p.scope.namespace != *namespace {
				scope = invalid("association namespace changed")
			}
			observed = errors.Join(p.check(checkCtx), parent, scope)
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
		parameters, err := prepareRecordList(opctx, check, namespace != nil, owned)
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
		plural := "resource_types"
		if namespace != nil {
			plural = "resource_type_associations"
		}
		codes := make([]int, 200)
		for i := range codes {
			codes[i] = 200 + i
		}
		var origin *rest.Response
		spec := rest.CollectionSpec[recordRow]{Client: p.client, Path: strings.TrimPrefix(p.url(nil), p.base), Kind: kind,
			PluralKey: plural, Metadata: recordMetadata, Validate: check, SourceGuard: check, ListCodes: codes,
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
				return errors.Join(prepareRecord(value, namespace, location, origin.Body), check(opctx))
			},
			Paging: rest.PagePolicy[recordRow]{HTTPLink: true, DictionaryLinks: true, MarkerFallback: true, Marker: recordMarker, MarkerOnShortPage: true,
				AllowFirstServerLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true},
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

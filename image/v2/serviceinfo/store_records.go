package serviceinfo

import (
	"context"
	"errors"
	"iter"
	"slices"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// ListStoreRecords lazily lists declared basic Store resources from /info/stores.
// Local filters use canonical Body names, default values are owned by the SDK,
// and pagination follows inherited Resource behavior through shared guards.
func (a *API) ListStoreRecords(ctx context.Context, options ...StoreRecordListOption) iter.Seq2[*StoreRecord, error] {
	owned := slices.Clone(options)
	return func(yield func(*StoreRecord, error) bool) {
		fail := func(err error) { yield(nil, wrapInfoError(ctx, "ListStoreRecords", err)) }
		p, err := a.captureRecord(ctx)
		if err != nil {
			fail(err)
			return
		}
		parameters, err := prepareStoreRecordList(p.ctx, p.check, owned)
		if err = errors.Join(err, p.check(p.ctx)); err != nil {
			fail(err)
			return
		}
		p.addHeaders(parameters.headers)
		if _, present := p.client.MoreHeaders["Accept"]; !present {
			p.client.MoreHeaders["Accept"] = "application/json"
		}
		var origin *rest.Response
		spec := rest.CollectionSpec[storeRecordRow]{Client: p.client, Path: "info/stores", Kind: kind, PluralKey: "stores",
			ListCodes: discoveryRecordCodes(), Metadata: storeRecordMetadata, Validate: p.check, SourceGuard: p.check,
			ValidateResponse: func(response *rest.Response) error {
				origin = response
				if !utf8.Valid(response.Body) {
					return infoInvalid("store record page must be UTF-8")
				}
				return p.check(p.ctx)
			},
			ValidateItem: func(value *storeRecordRow) error {
				if err := p.check(p.ctx); err != nil {
					return err
				}
				return errors.Join(prepareStoreRecord(value, p.location, origin.Body), p.check(p.ctx))
			},
			Paging: rest.PagePolicy[storeRecordRow]{VersionedPath: "/v2/info/stores", HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true,
				MarkerFallback: true, Marker: storeRecordMarker, MarkerOnShortPage: true, AllowFirstServerLimit: true,
				MaxItemsLimitHint: true, AllowZeroLimit: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true},
		}
		for value, readErr := range rest.ListWithControl(p.ctx, spec, parameters.query, parameters.control) {
			if readErr != nil {
				fail(readErr)
				return
			}
			matched, err := jsonfilter.MatchFilters(value.Resource.Body, parameters.filters)
			if err != nil {
				fail(origin.Fail(err))
				return
			}
			if matched && !yield(&value.StoreRecord, nil) {
				return
			}
			if err := p.check(p.ctx); err != nil {
				fail(origin.Fail(err))
				return
			}
		}
	}
}

// AllStoreRecords preserves consumed rows on a later row/page failure. A
// successfully empty discovery returns a nonnil empty slice.
func (a *API) AllStoreRecords(ctx context.Context, options ...StoreRecordListOption) ([]*StoreRecord, error) {
	values := make([]*StoreRecord, 0)
	for value, err := range a.ListStoreRecords(ctx, options...) {
		if err != nil {
			return values, err
		}
		values = append(values, value)
	}
	return values, nil
}

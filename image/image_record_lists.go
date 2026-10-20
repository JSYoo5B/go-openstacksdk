package image

import (
	"context"
	"errors"
	"iter"
	"slices"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// ListImageRecords lazily lists descriptor-shaped Images with Source server
// queries, canonical local filters, inherited paging and a raw consumption cap.
// The option functions and Connection location run once per iteration.
func (s *Service) ListImageRecords(ctx context.Context, options ...ImageRecordListOption) iter.Seq2[*ImageRecord, error] {
	owned := slices.Clone(options)
	return func(yield func(*ImageRecord, error) bool) {
		var parameters imageRecordListParameters
		p, err := s.prepareImageRecord(ctx, func(opctx context.Context, check func(context.Context) error) (map[string]string, error) {
			var err error
			parameters, err = prepareImageRecordList(opctx, check, owned)
			// List headers belong to its own phase client. Find uses this same helper
			// after its direct GET, without changing the direct request snapshot.
			return nil, err
		})
		if err != nil {
			yield(nil, wrapImageMutationError(ctx, "ListImageRecords", err))
			return
		}
		for value, err := range listImageRecordsPrepared(p, parameters) {
			if !yield(value, err) {
				return
			}
		}
	}
}

// AllImageRecords retains already consumed rows when a later page fails. Empty
// successful results are represented by a nonnil empty slice.
func (s *Service) AllImageRecords(ctx context.Context, options ...ImageRecordListOption) ([]*ImageRecord, error) {
	values := make([]*ImageRecord, 0)
	for value, err := range s.ListImageRecords(ctx, options...) {
		if err != nil {
			return values, err
		}
		values = append(values, value)
	}
	return values, nil
}

// Find reuses one prepared source/location/option snapshot across GET and both
// collection scans. Headers in parameters apply only to this listing phase.
func listImageRecordsPrepared(p *preparedImageRecord, parameters imageRecordListParameters) iter.Seq2[*ImageRecord, error] {
	return func(yield func(*ImageRecord, error) bool) {
		fail := func(err error) { yield(nil, wrapImageMutationError(p.ctx, "ListImageRecords", err)) }
		if err := p.check(p.ctx); err != nil {
			fail(err)
			return
		}
		client := *p.client
		client.MoreHeaders = copyImageRecordHeaders(p.client.MoreHeaders)
		for key, value := range parameters.headers {
			client.MoreHeaders[key] = value
		}
		if _, present := client.MoreHeaders["Accept"]; !present {
			client.MoreHeaders["Accept"] = "application/json"
		}
		var origin *rest.Response
		spec := rest.CollectionSpec[imageRecordRow]{Client: &client, Path: "images", Kind: "image", PluralKey: "images",
			Metadata: imageRecordMetadata, ListCodes: imageRecordCodes(), Validate: p.check, SourceGuard: p.check,
			ValidateResponse: func(response *rest.Response) error {
				origin = response
				if !utf8.Valid(response.Body) {
					return uploadInvalid("image record page must be UTF-8")
				}
				return p.check(p.ctx)
			},
			ValidateItem: func(value *imageRecordRow) error {
				if err := p.check(p.ctx); err != nil {
					return err
				}
				return errors.Join(prepareImageRecordRow(value, p.location, origin.Body), p.check(p.ctx))
			},
			Paging: rest.PagePolicy[imageRecordRow]{VersionedPath: "/v2/images", HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true,
				MarkerFallback: true, Marker: imageRecordMarker, MarkerOnShortPage: true, AllowFirstServerLimit: true,
				AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true},
		}
		for value, err := range rest.ListWithControl(p.ctx, spec, parameters.query, parameters.control) {
			if err != nil {
				fail(err)
				return
			}
			matched, err := jsonfilter.MatchFilters(value.Resource.Body, parameters.filters)
			if err != nil {
				fail(origin.Fail(err))
				return
			}
			if matched && !yield(&value.ImageRecord, nil) {
				return
			}
			if err := p.check(p.ctx); err != nil {
				fail(origin.Fail(err))
				return
			}
		}
	}
}

package image

import (
	"context"
	"errors"
	"iter"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ListImageMemberRecords lazily follows the pinned members proxy default,
// including inherited Resource pagination and descriptor-shaped values. Query
// controls, local filters and parent-name resolution are explicit Go extensions.
func (s *Service) ListImageMemberRecords(ctx context.Context, parent resource.Ref, options ...ImageMemberRecordListOption) iter.Seq2[*ImageMemberRecord, error] {
	owned := slices.Clone(options)
	return func(yield func(*ImageMemberRecord, error) bool) {
		var parameters imageMemberRecordListParameters
		p, err := s.prepareImageMemberRecord(ctx, parent, func(opctx context.Context, check func(context.Context) error) (map[string]string, error) {
			var err error
			parameters, err = prepareImageMemberRecordList(opctx, check, owned)
			return parameters.headers, err
		})
		if err != nil {
			yield(nil, wrapImageMutationError(ctx, "ListImageMemberRecords", err))
			return
		}
		for value, err := range listImageMemberRecordsPrepared(p, parameters) {
			if !yield(value, err) {
				return
			}
		}
	}
}

// AllImageMemberRecords preserves already consumed rows when a later page
// fails. A successfully empty collection returns a nonnil empty slice.
func (s *Service) AllImageMemberRecords(ctx context.Context, parent resource.Ref, options ...ImageMemberRecordListOption) ([]*ImageMemberRecord, error) {
	values := make([]*ImageMemberRecord, 0)
	for value, err := range s.ListImageMemberRecords(ctx, parent, options...) {
		if err != nil {
			return values, err
		}
		values = append(values, value)
	}
	return values, nil
}

// Find reuses this exact prepared operation for fallback, so options and
// location callbacks execute once across the direct GET and collection scan.
func listImageMemberRecordsPrepared(p *preparedImageMemberRecord, parameters imageMemberRecordListParameters) iter.Seq2[*ImageMemberRecord, error] {
	return func(yield func(*ImageMemberRecord, error) bool) {
		fail := func(err error) { yield(nil, wrapImageMutationError(p.ctx, "ListImageMemberRecords", err)) }
		if err := p.check(p.ctx); err != nil {
			fail(err)
			return
		}
		if _, present := p.client.MoreHeaders["Accept"]; !present {
			p.client.MoreHeaders["Accept"] = "application/json"
		}
		codes := make([]int, 200)
		for i := range codes {
			codes[i] = 200 + i
		}
		path := strings.TrimPrefix(imageMemberEndpoint(p.preparedImageMutation, nil), p.base)
		var origin *rest.Response
		spec := rest.CollectionSpec[imageMemberRecordRow]{Client: p.client, Path: path, Kind: "image member", PluralKey: "members",
			Metadata: imageMemberRecordMetadata, Validate: p.check, SourceGuard: p.check, ListCodes: codes,
			ValidateResponse: func(response *rest.Response) error {
				origin = response
				if !utf8.Valid(response.Body) {
					return uploadInvalid("member record page must be UTF-8")
				}
				return p.check(p.ctx)
			},
			ValidateItem: func(value *imageMemberRecordRow) error {
				if err := p.check(p.ctx); err != nil {
					return err
				}
				return errors.Join(prepareImageMemberRecordRow(value, p.id, p.location, origin.Body), p.check(p.ctx))
			},
			Paging: rest.PagePolicy[imageMemberRecordRow]{VersionedPath: "/v2/" + path, HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true,
				MarkerFallback: true, Marker: imageMemberRecordMarker, MarkerOnShortPage: true,
				AllowFirstServerLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true},
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
			if matched && !yield(&value.ImageMemberRecord, nil) {
				return
			}
			if err := p.check(p.ctx); err != nil {
				fail(origin.Fail(err))
				return
			}
		}
	}
}

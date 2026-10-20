package metadefproperties

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"net/http"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ListRecords lazily reads one finite property dictionary. Each iteration
// captures its own source/location/options; continuation advertisements remain
// passive. Dictionary keys seed name before response attributes replace it.
func (s *NamespaceScope) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*Record, error] {
	owned := slices.Clone(options)
	return func(yield func(*Record, error) bool) {
		fail := func(err error) { yield(nil, wrap(ctx, "ListRecords", err)) }
		p, err := s.capture(ctx, nil)
		if err != nil {
			fail(err)
			return
		}
		namespace := s.namespace
		check := p.operationGuard(ctx)
		opctx := rest.WithOperationGuard(ctx, check)
		if err := check(opctx); err != nil {
			fail(err)
			return
		}
		location, err := p.recordLocation(opctx, check)
		if err != nil {
			fail(err)
			return
		}
		parameters, err := prepareRecordList(opctx, check, owned)
		if err = errors.Join(p.finish(opctx, parameters.headers, err), check(opctx)); err != nil {
			fail(err)
			return
		}
		p.client.MoreHeaders["Accept"] = "application/json"
		codes := make([]int, 200)
		for index := range codes {
			codes[index] = 200 + index
		}
		response, err := rest.DoJSONGuardedHeaders(opctx, p.client, check, http.MethodGet, p.url(nil), nil, map[string]string{"Accept": "application/json"}, codes...)
		if err != nil {
			fail(err)
			return
		}
		fields, err := object(response.Body)
		if err != nil {
			fail(response.Fail(err))
			return
		}
		dictionary, exists := present(fields, "properties")
		if !exists {
			fail(response.Fail(invalid("response requires a nonnull properties dictionary")))
			return
		}
		entries, err := orderedDictionary(dictionary)
		if err != nil {
			fail(response.Fail(err))
			return
		}
		for index, entry := range entries {
			if err := check(opctx); err != nil {
				fail(response.Fail(err))
				return
			}
			if parameters.maxItems > 0 && index >= parameters.maxItems {
				return
			}
			wire := &resource.RawResource{Metadata: resource.Metadata{Header: response.Header.Clone(), StatusCode: response.StatusCode}}
			if err := json.Unmarshal(entry.raw, wire); err != nil {
				fail(response.Fail(err))
				return
			}
			seedName, err := json.Marshal(entry.key)
			if err != nil {
				fail(response.Fail(err))
				return
			}
			seed := map[string]json.RawMessage{"name": seedName}
			attributes, err := normalizePropertyRecord(wire.Body, entry.raw)
			if err != nil {
				fail(response.Fail(err))
				return
			}
			for key, raw := range attributes {
				seed[key] = bytes.Clone(raw)
			}
			projected, err := projectPropertyRecord(seed, namespace, location, resource.Metadata{Header: response.Header.Clone(), StatusCode: response.StatusCode})
			if err = errors.Join(err, check(opctx)); err != nil {
				fail(response.Fail(err))
				return
			}
			matched, err := jsonfilter.MatchFilters(projected.Body, parameters.filters)
			if err = errors.Join(err, check(opctx)); err != nil {
				fail(response.Fail(err))
				return
			}
			if matched {
				record := &Record{Namespace: namespace, Key: copyPointer(&entry.key), Resource: projected, Wire: wire,
					Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
				if !yield(record, nil) {
					return
				}
			}
			if err := check(opctx); err != nil {
				fail(response.Fail(err))
				return
			}
		}
		if err := check(opctx); err != nil {
			fail(response.Fail(err))
		}
	}
}

// AllRecords retains consumed matching records if a later entry fails.
func (s *NamespaceScope) AllRecords(ctx context.Context, options ...RecordListOption) ([]*Record, error) {
	values := make([]*Record, 0)
	for value, err := range s.ListRecords(ctx, options...) {
		if err != nil {
			return values, err
		}
		values = append(values, value)
	}
	return values, nil
}

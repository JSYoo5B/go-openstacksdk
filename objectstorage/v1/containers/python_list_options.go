package containers

import (
	"bytes"
	"encoding/json"
	"maps"
	"slices"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// RecordListOpts owns named containers query/control inputs. Unknown Source
// queries (including end_marker/full_listing) are ignored. MaxPages zero uses
// the SDK's 10000-page safety bound; MaxItems zero is unbounded within it.
type RecordListOpts struct {
	Query              map[string]json.RawMessage
	Headers            map[string]string
	Limit              *int
	Marker, Prefix     *string
	Format             *string
	Microversion       *string
	MaxItems, MaxPages int
	SinglePage         bool
	queryOrder         []string
}

// RecordListOption is one option of ListRecords: the WithRecordList* options,
// request.Query (one Source **query value as a JSON keyword), request.Limit,
// request.Marker, request.Microversion, request.MaxItems, request.Paginated
// (false reads only the first page) or request.Custom over *RecordListOpts.
// Other shared calls, including request.Header and request.Headers (use
// WithRecordListHeaders, which merges), fail with resource.ErrInvalidOption
// before HTTP.
type RecordListOption = request.ListCall

var recordListCalls = calls.Spec[RecordListOpts]{
	Query:        calls.JSONQuery(listRecordKeyword),
	Microversion: calls.Pointer(func(cfg *RecordListOpts) **string { return &cfg.Microversion }),
	Paging: calls.Paging[RecordListOpts]{
		Limit:     calls.Pointer(func(cfg *RecordListOpts) **int { return &cfg.Limit }),
		Marker:    calls.Pointer(func(cfg *RecordListOpts) **string { return &cfg.Marker }),
		MaxItems:  calls.Value(func(cfg *RecordListOpts) *int { return &cfg.MaxItems }),
		Paginated: func(cfg *RecordListOpts, paginated bool) error { cfg.SinglePage = !paginated; return nil },
	},
}

// listRecordKeyword applies one request.Query call as a Source **query value.
// Source ignores unknown parameters, with their value's marshal error, before
// Requests encodes their values.
func listRecordKeyword(cfg *RecordListOpts, key string, raw json.RawMessage, err error) error {
	if key == "paginated" {
		return metadataInvalid("named Source containers fixes paginated=True")
	}
	if !listRecordServerKey(key) && !listRecordBodyKey(key) {
		return nil
	}
	if err != nil {
		return err
	}
	if !utf8.Valid(raw) {
		return metadataInvalid("container attrs must encode UTF-8 JSON")
	}
	if cfg.Query == nil {
		cfg.Query = make(map[string]json.RawMessage)
	}
	if !slices.Contains(cfg.queryOrder, key) {
		cfg.queryOrder = append(cfg.queryOrder, key)
	}
	cfg.Query[key] = bytes.Clone(raw)
	return nil
}

func copyListRecordOpts(cfg RecordListOpts) RecordListOpts {
	cfg.Query = cloneRecordFields(cfg.Query)
	cfg.queryOrder = slices.Clone(cfg.queryOrder)
	cfg.Headers = maps.Clone(cfg.Headers)
	if cfg.Headers == nil {
		cfg.Headers = make(map[string]string)
	}
	if cfg.Limit != nil {
		value := *cfg.Limit
		cfg.Limit = &value
	}
	for _, ptr := range []**string{&cfg.Marker, &cfg.Prefix, &cfg.Format, &cfg.Microversion} {
		if *ptr != nil {
			value := **ptr
			*ptr = &value
		}
	}
	return cfg
}
func WithRecordListOpts(value RecordListOpts) RecordListOption {
	owned := copyListRecordOpts(value)
	return request.Custom(func(cfg *RecordListOpts) error { *cfg = copyListRecordOpts(owned); return nil })
}
func WithRecordListHeaders(value map[string]string) RecordListOption {
	owned := maps.Clone(value)
	return request.Custom(func(cfg *RecordListOpts) error {
		headers, err := listRecordHeaders(owned, false, "")
		if err != nil {
			return err
		}
		maps.Copy(cfg.Headers, headers)
		return nil
	})
}
func WithRecordListPrefix(value string) RecordListOption {
	return request.Custom(func(cfg *RecordListOpts) error { owned := value; cfg.Prefix = &owned; return nil })
}
func WithRecordListFormat(value string) RecordListOption {
	return request.Custom(func(cfg *RecordListOpts) error { owned := value; cfg.Format = &owned; return nil })
}
func WithRecordListMaxPages(value int) RecordListOption {
	return request.Custom(func(cfg *RecordListOpts) error { cfg.MaxPages = value; return nil })
}

func listRecordServerKey(key string) bool {
	return key == "prefix" || key == "format" || key == "limit" || key == "marker"
}
func listRecordBodyKey(key string) bool {
	return key == "id" || key == "name" || key == "count" || key == "bytes"
}

package containers

import (
	"bytes"
	"encoding/json"
	"maps"
	"slices"
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
type RecordListOption func(*RecordListOpts) error

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
	return func(cfg *RecordListOpts) error { *cfg = copyListRecordOpts(owned); return nil }
}
func WithRecordListQuery(key string, value any) RecordListOption {
	if key == "paginated" {
		return func(*RecordListOpts) error { return metadataInvalid("named Source containers fixes paginated=True") }
	}
	// Source ignores unknown parameters before Requests encodes their values.
	if !listRecordServerKey(key) && !listRecordBodyKey(key) {
		return func(*RecordListOpts) error { return nil }
	}
	raw, err := recordJSON(value)
	return func(cfg *RecordListOpts) error {
		if err != nil {
			return err
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
}
func WithRecordListHeaders(value map[string]string) RecordListOption {
	owned := maps.Clone(value)
	return func(cfg *RecordListOpts) error {
		headers, err := listRecordHeaders(owned, false, "")
		if err != nil {
			return err
		}
		maps.Copy(cfg.Headers, headers)
		return nil
	}
}
func WithRecordListLimit(value int) RecordListOption {
	return func(cfg *RecordListOpts) error { owned := value; cfg.Limit = &owned; return nil }
}
func WithRecordListMarker(value string) RecordListOption {
	return func(cfg *RecordListOpts) error { owned := value; cfg.Marker = &owned; return nil }
}
func WithRecordListPrefix(value string) RecordListOption {
	return func(cfg *RecordListOpts) error { owned := value; cfg.Prefix = &owned; return nil }
}
func WithRecordListFormat(value string) RecordListOption {
	return func(cfg *RecordListOpts) error { owned := value; cfg.Format = &owned; return nil }
}
func WithRecordListMicroversion(value string) RecordListOption {
	return func(cfg *RecordListOpts) error { owned := value; cfg.Microversion = &owned; return nil }
}
func WithRecordListMaxItems(value int) RecordListOption {
	return func(cfg *RecordListOpts) error { cfg.MaxItems = value; return nil }
}
func WithRecordListMaxPages(value int) RecordListOption {
	return func(cfg *RecordListOpts) error { cfg.MaxPages = value; return nil }
}

// WithRecordListSinglePage is an explicit Go control. Source named containers
// fixes paginated=True; full_listing=False is an ignored query, not this control.
func WithRecordListSinglePage(value bool) RecordListOption {
	return func(cfg *RecordListOpts) error { cfg.SinglePage = value; return nil }
}
func listRecordServerKey(key string) bool {
	return key == "prefix" || key == "format" || key == "limit" || key == "marker"
}
func listRecordBodyKey(key string) bool {
	return key == "id" || key == "name" || key == "count" || key == "bytes"
}

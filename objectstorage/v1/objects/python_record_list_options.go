package objects

import (
	"bytes"
	"encoding/json"
	"maps"
	"slices"
	"unicode/utf8"
)

// RecordListOpts owns the query and control inputs of the named Source
// objects(container, **query) listing. format=json, paginated=True and the
// container URI value are fixed by the pinned proxy. Unknown Source queries
// (delimiter, end_marker, path and similar) are ignored before encoding.
// MaxPages zero uses the SDK's 10000-page safety bound; MaxItems zero is
// unbounded within it.
type RecordListOpts struct {
	Query              map[string]json.RawMessage
	Headers            map[string]string
	Limit              *int
	Marker, Prefix     *string
	Microversion       *string
	MaxItems, MaxPages int
	SinglePage         bool
	queryOrder         []string
}

// RecordListOption configures ListRecords.
type RecordListOption func(*RecordListOpts) error

func copyObjectListOpts(cfg RecordListOpts) RecordListOpts {
	cfg.Query = cloneObjectListFields(cfg.Query)
	cfg.queryOrder = slices.Clone(cfg.queryOrder)
	cfg.Headers = maps.Clone(cfg.Headers)
	if cfg.Headers == nil {
		cfg.Headers = make(map[string]string)
	}
	if cfg.Limit != nil {
		value := *cfg.Limit
		cfg.Limit = &value
	}
	for _, ptr := range []**string{&cfg.Marker, &cfg.Prefix, &cfg.Microversion} {
		if *ptr != nil {
			value := **ptr
			*ptr = &value
		}
	}
	return cfg
}

// WithRecordListOpts replaces the whole option state with an owned copy.
func WithRecordListOpts(value RecordListOpts) RecordListOption {
	owned := copyObjectListOpts(value)
	return func(cfg *RecordListOpts) error { *cfg = copyObjectListOpts(owned); return nil }
}

// WithRecordListQuery adds one Source **query value. Server keys (limit,
// marker, prefix, temp_url_sig, temp_url_expires, filename,
// multipart_manifest or multipart-manifest, symlink) are sent; declared Body
// attributes (id, name, _hash, _bytes, _last_modified, _content_type) become
// local filters; unknown keys are ignored like Source allow_unknown_params.
func WithRecordListQuery(key string, value any) RecordListOption {
	if err := objectListReservedQuery(key); err != nil {
		return func(*RecordListOpts) error { return err }
	}
	if !objectListServerKey(key) && !objectListBodyKey(key) {
		return func(*RecordListOpts) error { return nil }
	}
	raw, err := objectListJSON(value)
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

// WithRecordListHeaders adds Resource.list method headers. They override the
// operation Accept default exactly like Source headers_final.
func WithRecordListHeaders(value map[string]string) RecordListOption {
	owned := maps.Clone(value)
	return func(cfg *RecordListOpts) error {
		headers, err := objectListHeaders(owned, false, "")
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
func WithRecordListMicroversion(value string) RecordListOption {
	return func(cfg *RecordListOpts) error { owned := value; cfg.Microversion = &owned; return nil }
}
func WithRecordListMaxItems(value int) RecordListOption {
	return func(cfg *RecordListOpts) error { cfg.MaxItems = value; return nil }
}
func WithRecordListMaxPages(value int) RecordListOption {
	return func(cfg *RecordListOpts) error { cfg.MaxPages = value; return nil }
}

// WithRecordListSinglePage is an explicit Go control. The pinned objects proxy
// fixes paginated=True, so Python itself has no first-page-only form.
func WithRecordListSinglePage(value bool) RecordListOption {
	return func(cfg *RecordListOpts) error { cfg.SinglePage = value; return nil }
}

// objectListQueryMapping is Object._query_mapping (client name to server name),
// including the default limit/marker pagination parameters.
var objectListQueryMapping = [][2]string{
	{"limit", "limit"}, {"marker", "marker"}, {"prefix", "prefix"}, {"format", "format"},
	{"temp_url_sig", "temp_url_sig"}, {"temp_url_expires", "temp_url_expires"},
	{"filename", "filename"}, {"multipart_manifest", "multipart-manifest"}, {"symlink", "symlink"},
}

func objectListServerKey(key string) bool {
	for _, pair := range objectListQueryMapping {
		if key == pair[0] || key == pair[1] {
			return true
		}
	}
	return false
}

// objectListBodyKey lists declared Object Body attributes outside the query
// mapping; Resource.list applies them as local client filters.
func objectListBodyKey(key string) bool {
	switch key {
	case "id", "name", "_hash", "_bytes", "_last_modified", "_content_type":
		return true
	}
	return false
}
func objectListReservedQuery(key string) error {
	switch key {
	case "container", "paginated", "format":
		return metadataInvalid("Source objects() fixes the %q keyword", key)
	case "headers", "microversion", "max_items":
		return metadataInvalid("Source %q is a Resource.list control; use the dedicated option", key)
	case "base_path", "jmespath_filters", "session", "cls", "self", "resource_type", "__conflicting_attrs":
		return metadataInvalid("Source %q control is not supported by object ListRecords", key)
	}
	return nil
}
func objectListJSON(value any) (json.RawMessage, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if !utf8.Valid(encoded) {
		return nil, metadataInvalid("object list query must encode UTF-8 JSON")
	}
	return json.RawMessage(bytes.Clone(encoded)), nil
}
func cloneObjectListFields(values map[string]json.RawMessage) map[string]json.RawMessage {
	if values == nil {
		return nil
	}
	out := make(map[string]json.RawMessage, len(values))
	for key, raw := range values {
		out[key] = bytes.Clone(raw)
	}
	return out
}

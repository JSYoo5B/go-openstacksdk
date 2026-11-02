package serviceinfo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ImportRecordOpts provides ordinary caller headers for the fixed import route.
type ImportRecordOpts struct{ Headers map[string]string }

// ImportRecordOption is one option of GetImportRecord: WithImportRecordOpts,
// WithImportRecordHeaders (validates and merges the map), request.Header or
// request.Custom over *ImportRecordOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type ImportRecordOption = request.GetCall

// StoreRecordListOpts controls owned basic store discovery. Nil Limit is
// omitted; an explicit zero is sent without marker fallback. Nil Paginated
// means true and zero MaxItems is unlimited. Caps count raw consumed rows before
// local canonical Body filters. Details selects the fixed detail discovery route.
type StoreRecordListOpts struct {
	Details   bool
	Headers   map[string]string
	Limit     *int
	Marker    string
	MaxItems  int
	Paginated *bool
	Filters   map[string]json.RawMessage
}

// StoreRecordListOption is one option of ListStoreRecords:
// WithStoreRecordListOpts, WithStoreRecordListHeaders (validates and merges
// the map), WithStoreRecordListDetails, WithStoreRecordListFilter,
// WithStoreRecordListFilters, request.Header, request.Limit, request.Marker,
// request.MaxItems, request.Paginated or request.Custom over
// *StoreRecordListOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type StoreRecordListOption = request.ListCall

func copyRecordHeaders(value map[string]string) map[string]string {
	result := maps.Clone(value)
	if result == nil {
		result = make(map[string]string)
	}
	return result
}
func copyImportRecordOpts(value ImportRecordOpts) ImportRecordOpts {
	value.Headers = copyRecordHeaders(value.Headers)
	return value
}
func copyStoreRecordListOpts(value StoreRecordListOpts) StoreRecordListOpts {
	value.Headers = copyRecordHeaders(value.Headers)
	if value.Limit != nil {
		owned := *value.Limit
		value.Limit = &owned
	}
	if value.Paginated != nil {
		owned := *value.Paginated
		value.Paginated = &owned
	}
	filters := make(map[string]json.RawMessage, len(value.Filters))
	for key, raw := range value.Filters {
		filters[key] = bytes.Clone(raw)
	}
	value.Filters = filters
	return value
}
func mergeRecordHeaders(target *map[string]string, values map[string]string) error {
	prepared, err := infoHeaders(values, false, "")
	if err != nil {
		return err
	}
	if *target == nil {
		*target = make(map[string]string)
	}
	// Conflicting aliases inside this supplied map fail above. Separate
	// helper applications replace all earlier casing aliases for that header.
	for key, value := range prepared {
		for previous := range *target {
			if strings.EqualFold(previous, key) {
				delete(*target, previous)
			}
		}
		(*target)[key] = value
	}
	return nil
}

// WithImportRecordOpts snapshots and replaces the complete concrete policy.
func WithImportRecordOpts(value ImportRecordOpts) ImportRecordOption {
	owned := copyImportRecordOpts(value)
	return request.Custom(func(config *ImportRecordOpts) error { *config = copyImportRecordOpts(owned); return nil })
}
func WithImportRecordHeaders(values map[string]string) ImportRecordOption {
	owned := copyRecordHeaders(values)
	return request.Custom(func(config *ImportRecordOpts) error { return mergeRecordHeaders(&config.Headers, owned) })
}
func WithStoreRecordListOpts(value StoreRecordListOpts) StoreRecordListOption {
	owned := copyStoreRecordListOpts(value)
	return request.Custom(func(config *StoreRecordListOpts) error { *config = copyStoreRecordListOpts(owned); return nil })
}
func WithStoreRecordListHeaders(values map[string]string) StoreRecordListOption {
	owned := copyRecordHeaders(values)
	return request.Custom(func(config *StoreRecordListOpts) error { return mergeRecordHeaders(&config.Headers, owned) })
}

var (
	importRecordCalls    = calls.Spec[ImportRecordOpts]{Header: calls.MergeHeader[ImportRecordOpts](WithImportRecordHeaders)}
	storeRecordListCalls = calls.Spec[StoreRecordListOpts]{
		Header: calls.MergeHeader[StoreRecordListOpts](WithStoreRecordListHeaders),
		Paging: calls.Paging[StoreRecordListOpts]{
			Limit:     calls.Pointer(func(o *StoreRecordListOpts) **int { return &o.Limit }),
			Marker:    calls.Value(func(o *StoreRecordListOpts) *string { return &o.Marker }),
			MaxItems:  calls.Value(func(o *StoreRecordListOpts) *int { return &o.MaxItems }),
			Paginated: calls.Pointer(func(o *StoreRecordListOpts) **bool { return &o.Paginated }),
		},
	}
)

func storeRecordFilterKnown(key string) bool {
	// Source Body filtering tests class attribute names; the remote spelling
	// "default" is not a declared class attribute and remains unknown here.
	for _, field := range storeRecordFields {
		if key == field.canonical {
			return true
		}
	}
	return false
}
func WithStoreRecordListFilter(field string, value any) StoreRecordListOption {
	if !storeRecordFilterKnown(field) {
		return request.Custom(func(*StoreRecordListOpts) error { return nil })
	}
	raw, encodeErr := json.Marshal(value)
	owned := bytes.Clone(raw)
	return request.Custom(func(config *StoreRecordListOpts) error {
		if encodeErr != nil {
			return fmt.Errorf("%w: store filter %q: %w", resource.ErrInvalidOption, field, encodeErr)
		}
		if config.Filters == nil {
			config.Filters = make(map[string]json.RawMessage)
		}
		config.Filters[field] = bytes.Clone(owned)
		return nil
	})
}
func WithStoreRecordListFilters(values map[string]any) StoreRecordListOption {
	owned := make(map[string]json.RawMessage, len(values))
	var encodeErr error
	for key, value := range values {
		// Unknown values are discarded before any marshaler can be invoked.
		if !storeRecordFilterKnown(key) {
			continue
		}
		raw, err := json.Marshal(value)
		if err != nil {
			encodeErr = errors.Join(encodeErr, fmt.Errorf("%w: store filter %q: %w", resource.ErrInvalidOption, key, err))
		}
		owned[key] = bytes.Clone(raw)
	}
	return request.Custom(func(config *StoreRecordListOpts) error {
		if encodeErr != nil {
			return encodeErr
		}
		if config.Filters == nil {
			config.Filters = make(map[string]json.RawMessage)
		}
		for key, raw := range owned {
			config.Filters[key] = bytes.Clone(raw)
		}
		return nil
	})
}

func prepareImportRecord(ctx context.Context, check func(context.Context) error, options []ImportRecordOption) (map[string]string, error) {
	value := copyImportRecordOpts(ImportRecordOpts{})
	for _, apply := range calls.Translate(options, importRecordCalls) {
		if err := check(ctx); err != nil {
			return nil, err
		}
		if apply == nil {
			return nil, infoInvalid("nil import record option")
		}
		owned := copyImportRecordOpts(value)
		if err := errors.Join(apply(&owned), check(ctx)); err != nil {
			return nil, err
		}
		value = copyImportRecordOpts(owned)
	}
	headers, err := infoHeaders(value.Headers, false, "")
	return headers, errors.Join(err, check(ctx))
}

type storeRecordListParameters struct {
	details bool
	query   url.Values
	headers map[string]string
	filters map[string]json.RawMessage
	control rest.ListControl
}

func prepareStoreRecordList(ctx context.Context, check func(context.Context) error, options []StoreRecordListOption) (storeRecordListParameters, error) {
	value := copyStoreRecordListOpts(StoreRecordListOpts{})
	for _, apply := range calls.Translate(options, storeRecordListCalls) {
		if err := check(ctx); err != nil {
			return storeRecordListParameters{}, err
		}
		if apply == nil {
			return storeRecordListParameters{}, infoInvalid("nil store record list option")
		}
		owned := copyStoreRecordListOpts(value)
		if err := errors.Join(apply(&owned), check(ctx)); err != nil {
			return storeRecordListParameters{}, err
		}
		value = copyStoreRecordListOpts(owned)
	}
	if value.Limit != nil && *value.Limit < 0 || value.MaxItems < 0 {
		return storeRecordListParameters{}, infoInvalid("limit and max items must be nonnegative")
	}
	query := make(url.Values)
	if value.Limit != nil {
		query.Set("limit", strconv.Itoa(*value.Limit))
	}
	if value.Marker != "" {
		if strings.TrimSpace(value.Marker) == "" {
			return storeRecordListParameters{}, infoInvalid("store marker must be nonempty")
		}
		if err := recordQueryText(value.Marker); err != nil {
			return storeRecordListParameters{}, err
		}
		query.Set("marker", value.Marker)
	}
	filters := make(map[string]json.RawMessage, len(value.Filters))
	for key, raw := range value.Filters {
		if !storeRecordFilterKnown(key) {
			continue
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return storeRecordListParameters{}, infoInvalid("store filter %q must be complete UTF-8 JSON", key)
		}
		filters[key] = bytes.Clone(raw)
	}
	headers, err := infoHeaders(value.Headers, false, "")
	if err != nil {
		return storeRecordListParameters{}, err
	}
	return storeRecordListParameters{details: value.Details, query: query, headers: headers, filters: filters,
		control: rest.ListControl{MaxItems: value.MaxItems, SinglePage: value.Paginated != nil && !*value.Paginated, LimitHint: true}}, check(ctx)
}
func recordQueryText(value string) error {
	if !utf8.ValidString(value) {
		return infoInvalid("store query must be valid UTF-8")
	}
	for _, char := range value {
		if unicode.IsControl(char) {
			return infoInvalid("store query must not contain controls")
		}
	}
	return nil
}

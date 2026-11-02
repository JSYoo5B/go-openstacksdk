package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// ImageMemberRecordListOpts supplies explicit Go query and consumption controls.
// The pinned members proxy drops its query kwargs, so the no-option request is
// query-free. Nil Paginated means true, zero Limit is omitted and zero MaxItems
// is unlimited. Filters match declared Body attributes locally after raw caps.
type ImageMemberRecordListOpts struct {
	Headers         map[string]string
	Limit, MaxItems int
	Marker          string
	Paginated       *bool
	Filters         map[string]json.RawMessage
}

// ImageMemberRecordListOption is one option of the member record lists:
// WithImageMemberRecordListOpts, WithImageMemberRecordListHeaders (merges),
// WithImageMemberRecordListFilter(s), request.Header (merges one header),
// request.Limit, request.Marker, request.MaxItems, request.Paginated or
// request.Custom over *ImageMemberRecordListOpts. Other shared calls,
// including request.Headers and request.Query, fail with
// resource.ErrInvalidOption before HTTP.
type ImageMemberRecordListOption = request.ListCall

// FindImageMemberRecordOpts controls owned GET-first member discovery. Nil
// IgnoreMissing means true. Ordinary headers apply to GET and fallback pages.
type FindImageMemberRecordOpts struct {
	Headers       map[string]string
	IgnoreMissing *bool
}

// FindImageMemberRecordOption is one option of FindImageMemberRecord:
// WithFindImageMemberRecordOpts, WithFindImageMemberRecordHeaders (merges),
// request.Header (merges one header), request.IgnoreMissing or
// request.Custom over *FindImageMemberRecordOpts. Other shared calls fail
// with resource.ErrInvalidOption before HTTP.
type FindImageMemberRecordOption = request.FindCall

var (
	imageMemberRecordListCalls = calls.Spec[ImageMemberRecordListOpts]{
		Header: calls.MergeHeader[ImageMemberRecordListOpts](WithImageMemberRecordListHeaders),
		Paging: calls.Paging[ImageMemberRecordListOpts]{
			Limit:     calls.Value(func(c *ImageMemberRecordListOpts) *int { return &c.Limit }),
			Marker:    calls.Value(func(c *ImageMemberRecordListOpts) *string { return &c.Marker }),
			MaxItems:  calls.Value(func(c *ImageMemberRecordListOpts) *int { return &c.MaxItems }),
			Paginated: calls.Pointer(func(c *ImageMemberRecordListOpts) **bool { return &c.Paginated }),
		},
	}
	findImageMemberRecordCalls = calls.Spec[FindImageMemberRecordOpts]{
		Header:        calls.MergeHeader[FindImageMemberRecordOpts](WithFindImageMemberRecordHeaders),
		IgnoreMissing: calls.Pointer(func(c *FindImageMemberRecordOpts) **bool { return &c.IgnoreMissing }),
	}
)

func copyImageMemberRecordList(value ImageMemberRecordListOpts) ImageMemberRecordListOpts {
	value.Headers = copyImageMemberOpts(ImageMemberOpts{Headers: value.Headers}).Headers
	if value.Paginated != nil {
		copy := *value.Paginated
		value.Paginated = &copy
	}
	filters := make(map[string]json.RawMessage, len(value.Filters))
	for key, raw := range value.Filters {
		filters[key] = bytes.Clone(raw)
	}
	value.Filters = filters
	return value
}
func copyFindImageMemberRecord(value FindImageMemberRecordOpts) FindImageMemberRecordOpts {
	value.Headers = copyImageMemberOpts(ImageMemberOpts{Headers: value.Headers}).Headers
	if value.IgnoreMissing != nil {
		copy := *value.IgnoreMissing
		value.IgnoreMissing = &copy
	}
	return value
}

// WithImageMemberRecordListOpts snapshots and replaces the entire configuration.
func WithImageMemberRecordListOpts(value ImageMemberRecordListOpts) ImageMemberRecordListOption {
	owned := copyImageMemberRecordList(value)
	return request.Custom(func(config *ImageMemberRecordListOpts) error { *config = copyImageMemberRecordList(owned); return nil })
}
func WithImageMemberRecordListHeaders(values map[string]string) ImageMemberRecordListOption {
	apply := mergeImageMutationHeaders(values)
	return request.Custom(func(config *ImageMemberRecordListOpts) error { return applyImageMemberHeader(&config.Headers, apply) })
}
func WithImageMemberRecordListFilter(field string, value any) ImageMemberRecordListOption {
	if imageMemberRecordFilterKey(field) == "" {
		return request.Custom(func(*ImageMemberRecordListOpts) error { return nil })
	}
	raw, encodeErr := json.Marshal(value)
	owned := bytes.Clone(raw)
	return request.Custom(func(config *ImageMemberRecordListOpts) error {
		if encodeErr != nil {
			return encodeErr
		}
		if config.Filters == nil {
			config.Filters = make(map[string]json.RawMessage)
		}
		config.Filters[field] = bytes.Clone(owned)
		return nil
	})
}
func WithImageMemberRecordListFilters(values map[string]any) ImageMemberRecordListOption {
	owned := make(map[string]json.RawMessage, len(values))
	var encodeErr error
	for key, value := range values {
		if imageMemberRecordFilterKey(key) == "" {
			continue
		}
		raw, err := json.Marshal(value)
		encodeErr = errors.Join(encodeErr, err)
		owned[key] = bytes.Clone(raw)
	}
	return request.Custom(func(config *ImageMemberRecordListOpts) error {
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
func WithFindImageMemberRecordOpts(value FindImageMemberRecordOpts) FindImageMemberRecordOption {
	owned := copyFindImageMemberRecord(value)
	return request.Custom(func(config *FindImageMemberRecordOpts) error { *config = copyFindImageMemberRecord(owned); return nil })
}
func WithFindImageMemberRecordHeaders(values map[string]string) FindImageMemberRecordOption {
	apply := mergeImageMutationHeaders(values)
	return request.Custom(func(config *FindImageMemberRecordOpts) error { return applyImageMemberHeader(&config.Headers, apply) })
}

func prepareFindImageMemberRecord(ctx context.Context, check func(context.Context) error, options []FindImageMemberRecordOption) (FindImageMemberRecordOpts, error) {
	value := copyFindImageMemberRecord(FindImageMemberRecordOpts{})
	for _, apply := range calls.Translate(options, findImageMemberRecordCalls) {
		if err := check(ctx); err != nil {
			return value, err
		}
		if apply == nil {
			return value, uploadInvalid("nil find member record option")
		}
		owned := copyFindImageMemberRecord(value)
		if err := errors.Join(apply(&owned), check(ctx)); err != nil {
			return value, err
		}
		value = copyFindImageMemberRecord(owned)
	}
	var err error
	value.Headers, err = imageMutationHeaders(value.Headers, false, "")
	return value, errors.Join(err, check(ctx))
}

type imageMemberRecordListParameters struct {
	query   url.Values
	filters map[string]json.RawMessage
	headers map[string]string
	control rest.ListControl
}

func prepareImageMemberRecordList(ctx context.Context, check func(context.Context) error, options []ImageMemberRecordListOption) (imageMemberRecordListParameters, error) {
	value := copyImageMemberRecordList(ImageMemberRecordListOpts{})
	for _, apply := range calls.Translate(options, imageMemberRecordListCalls) {
		if err := check(ctx); err != nil {
			return imageMemberRecordListParameters{}, err
		}
		if apply == nil {
			return imageMemberRecordListParameters{}, uploadInvalid("nil member record list option")
		}
		owned := copyImageMemberRecordList(value)
		if err := errors.Join(apply(&owned), check(ctx)); err != nil {
			return imageMemberRecordListParameters{}, err
		}
		value = copyImageMemberRecordList(owned)
	}
	if value.Limit < 0 || value.MaxItems < 0 {
		return imageMemberRecordListParameters{}, uploadInvalid("member limit and max items must be nonnegative")
	}
	query := make(url.Values)
	if value.Limit > 0 {
		query.Set("limit", strconv.Itoa(value.Limit))
	}
	if value.Marker != "" {
		if strings.TrimSpace(value.Marker) == "" {
			return imageMemberRecordListParameters{}, uploadInvalid("member marker must be nonempty")
		}
		if err := taskQueryText(value.Marker); err != nil {
			return imageMemberRecordListParameters{}, err
		}
		query.Set("marker", value.Marker)
	}
	filters := make(map[string]json.RawMessage, len(value.Filters))
	for key, raw := range value.Filters {
		canonical := imageMemberRecordFilterKey(key)
		if canonical == "" {
			continue
		}
		if _, duplicate := filters[canonical]; duplicate {
			return imageMemberRecordListParameters{}, uploadInvalid("conflicting member filter aliases %q", key)
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return imageMemberRecordListParameters{}, uploadInvalid("member filter %q must be complete UTF-8 JSON", key)
		}
		filters[canonical] = slices.Clone(raw)
	}
	headers, err := imageMutationHeaders(value.Headers, false, "")
	if err != nil {
		return imageMemberRecordListParameters{}, err
	}
	return imageMemberRecordListParameters{query: query, filters: filters, headers: headers,
		control: rest.ListControl{MaxItems: value.MaxItems, SinglePage: value.Paginated != nil && !*value.Paginated, LimitHint: true}}, check(ctx)
}

// Ignore unknown attributes before inspecting captured JSON, matching the
// semantic resource filter boundary used by owned list APIs.
func imageMemberRecordFilterKey(key string) string {
	for _, field := range imageMemberRecordFields {
		if key == field.canonical || key == field.wire {
			return field.canonical
		}
	}
	return ""
}

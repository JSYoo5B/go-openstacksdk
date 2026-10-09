package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ImageRecordQueryOpts controls the Cloud image list/search/get helpers.
// FilterDeleted (nil means true) and ShowAll (nil means false) belong only to
// AllCloudImageRecords. Filters belong only to SearchImageRecords and
// GetCloudImageRecord. Headers are a Go extension applied to every request of
// one logical call. GetImageRecordByID accepts headers only.
type ImageRecordQueryOpts struct {
	Headers       map[string]string
	FilterDeleted *bool
	ShowAll       *bool
	Filters       *json.RawMessage
}
type ImageRecordQueryOption func(*ImageRecordQueryOpts) error

func copyImageRecordQuery(value ImageRecordQueryOpts) ImageRecordQueryOpts {
	value.Headers = copyImageRecordHeaders(value.Headers)
	if value.FilterDeleted != nil {
		owned := *value.FilterDeleted
		value.FilterDeleted = &owned
	}
	if value.ShowAll != nil {
		owned := *value.ShowAll
		value.ShowAll = &owned
	}
	if value.Filters != nil {
		owned := json.RawMessage(bytes.Clone(*value.Filters))
		value.Filters = &owned
	}
	return value
}
func WithImageRecordQueryOpts(value ImageRecordQueryOpts) ImageRecordQueryOption {
	owned := copyImageRecordQuery(value)
	return func(config *ImageRecordQueryOpts) error { *config = copyImageRecordQuery(owned); return nil }
}
func WithImageRecordQueryHeader(key, value string) ImageRecordQueryOption {
	return func(config *ImageRecordQueryOpts) error {
		return mergeImageRecordHeaders(&config.Headers, map[string]string{key: value})
	}
}
func WithImageRecordQueryHeaders(values map[string]string) ImageRecordQueryOption {
	owned := copyImageRecordHeaders(values)
	return func(config *ImageRecordQueryOpts) error { return mergeImageRecordHeaders(&config.Headers, owned) }
}
func WithImageRecordQueryFilterDeleted(value bool) ImageRecordQueryOption {
	return func(config *ImageRecordQueryOpts) error { owned := value; config.FilterDeleted = &owned; return nil }
}
func WithImageRecordQueryShowAll(value bool) ImageRecordQueryOption {
	return func(config *ImageRecordQueryOpts) error { owned := value; config.ShowAll = &owned; return nil }
}

// WithImageRecordQueryFilters owns raw JSON. Nil clears the option; explicit
// null remains present. Search preserves dictionary order and expressions.
func WithImageRecordQueryFilters(value json.RawMessage) ImageRecordQueryOption {
	owned := bytes.Clone(value)
	return func(config *ImageRecordQueryOpts) error {
		config.Filters = nil
		if owned != nil {
			raw := json.RawMessage(bytes.Clone(owned))
			config.Filters = &raw
		}
		return nil
	}
}
func WithImageRecordQueryExpression(value string) ImageRecordQueryOption {
	raw, _ := json.Marshal(value)
	return WithImageRecordQueryFilters(raw)
}

// imageRecordQueryArguments names the Source arguments of one Cloud helper.
type imageRecordQueryArguments struct {
	listControls, filters bool
}

func prepareImageRecordQuery(ctx context.Context, check func(context.Context) error, options []ImageRecordQueryOption, accepted imageRecordQueryArguments) (ImageRecordQueryOpts, error) {
	value := copyImageRecordQuery(ImageRecordQueryOpts{})
	for _, apply := range options {
		if err := check(ctx); err != nil {
			return value, err
		}
		if apply == nil {
			return value, uploadInvalid("nil image record query option")
		}
		owned := copyImageRecordQuery(value)
		if err := errors.Join(apply(&owned), check(ctx)); err != nil {
			return value, err
		}
		value = copyImageRecordQuery(owned)
	}
	if !accepted.listControls && (value.FilterDeleted != nil || value.ShowAll != nil) {
		return value, fmt.Errorf("%w: this Cloud image helper has no filter_deleted or show_all argument", resource.ErrInvalidOption)
	}
	if value.Filters != nil {
		if !accepted.filters {
			return value, fmt.Errorf("%w: this Cloud image helper has no filters argument", resource.ErrInvalidOption)
		}
		if raw := *value.Filters; !utf8.Valid(raw) || !json.Valid(raw) {
			return value, uploadInvalid("Cloud image filters must be complete UTF-8 JSON")
		}
	}
	headers, err := imageMutationHeaders(value.Headers, false, "")
	value.Headers = headers
	return value, errors.Join(err, check(ctx))
}

// ImageRecordQueryResult retains every consumed list row in Inventory, including
// rows later removed by the deleted-status filter, and partial work on error.
// Value and Images represent a completed collection or selection only. A
// JMESPath expression returns arbitrary JSON in Value without inventing Images.
type ImageRecordQueryResult struct {
	Value     json.RawMessage
	Images    []*ImageRecord
	Inventory []*ImageRecord
}

// CloudImageRecordResult is the result of Cloud get_image and the
// get_image_exclude/name/id helpers. Value is the Source return value and nil
// when no image is selected; a selected false, zero, empty string or null name
// remains present. Image is the actual record for direct find, one ordinary
// selected row or the exclude selection. Inventory is populated by every
// search-based path, including partial work on error.
type CloudImageRecordResult struct {
	Value     json.RawMessage
	Image     *ImageRecord
	Inventory []*ImageRecord
}

// ImageRecordSelectionError reports Cloud get_image's len(value)>1 check
// without inventing IDs for an arbitrary expression result.
type ImageRecordSelectionError struct {
	NameOrID string
	Length   int
}

func (e *ImageRecordSelectionError) Error() string {
	return fmt.Sprintf("image %q has multiple matches (length %d)", e.NameOrID, e.Length)
}
func (e *ImageRecordSelectionError) Unwrap() error { return resource.ErrAmbiguous }

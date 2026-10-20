package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ImageRecordRequest selects a literal ID or a private resource seed. Attributes
// form constructor/update data, not an HTTP query. The caller Resource is copied.
type ImageRecordRequest struct {
	ID         string
	Resource   *resource.RawResource
	Attributes map[string]any
}

// ImageRecordOpts supplies ordinary headers and additional seed attributes.
// Later attributes replace earlier names. Maps have deterministic sorted order;
// original JSON response aliases retain parsed dictionary insertion order.
type ImageRecordOpts struct {
	Headers    map[string]string
	Attributes map[string]any
}
type ImageRecordOption func(*ImageRecordOpts) error

// ImageRecordListOpts controls Source Image listing. Filters select declared
// server queries first and otherwise canonical Body filters. Unknown names are
// discarded before encoding. Nil Limit is omitted; explicit zero is retained.
// Nil Paginated means true, and zero MaxItems is unlimited. Caps count raw rows.
type ImageRecordListOpts struct {
	Headers   map[string]string
	Filters   map[string]json.RawMessage
	Limit     *int
	Marker    string
	MaxItems  int
	Paginated *bool
}
type ImageRecordListOption func(*ImageRecordListOpts) error

func copyImageRecordHeaders(values map[string]string) map[string]string {
	owned := maps.Clone(values)
	if owned == nil {
		owned = make(map[string]string)
	}
	return owned
}
func mergeImageRecordHeaders(target *map[string]string, values map[string]string) error {
	headers, err := imageMutationHeaders(values, false, "")
	if err != nil {
		return err
	}
	if *target == nil {
		*target = make(map[string]string)
	}
	for key, value := range headers {
		for previous := range *target {
			if strings.EqualFold(previous, key) {
				delete(*target, previous)
			}
		}
		(*target)[key] = value
	}
	return nil
}
func captureImageRecordAttributes(ctx context.Context, check func(context.Context) error, values map[string]any) (map[string]json.RawMessage, error) {
	values = maps.Clone(values)
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	owned := make(map[string]json.RawMessage, len(values))
	for _, key := range keys {
		if check != nil {
			if err := check(ctx); err != nil {
				return nil, err
			}
		}
		if !utf8.ValidString(key) {
			return nil, uploadInvalid("image attribute name must be UTF-8")
		}
		switch key {
		case "connection", "_synchronized", "microversion":
			return nil, uploadInvalid("image attribute %q collides with a source constructor argument", key)
		}
		raw, err := json.Marshal(values[key])
		if check != nil {
			err = errors.Join(err, check(ctx))
		}
		if err != nil {
			return nil, errors.Join(uploadInvalid("image attribute %q cannot be encoded", key), err)
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, uploadInvalid("image attribute %q must be complete UTF-8 JSON", key)
		}
		owned[key] = bytes.Clone(raw)
	}
	return owned, nil
}
func imageRecordAnyAttributes(values map[string]json.RawMessage) map[string]any {
	owned := make(map[string]any, len(values))
	for key, raw := range values {
		owned[key] = json.RawMessage(bytes.Clone(raw))
	}
	return owned
}
func copyImageRecordOpts(ctx context.Context, check func(context.Context) error, value ImageRecordOpts) (ImageRecordOpts, error) {
	value.Headers = copyImageRecordHeaders(value.Headers)
	attrs, err := captureImageRecordAttributes(ctx, check, value.Attributes)
	value.Attributes = imageRecordAnyAttributes(attrs)
	return value, err
}
func copyImageRecordList(value ImageRecordListOpts) ImageRecordListOpts {
	value.Headers = copyImageRecordHeaders(value.Headers)
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
func WithImageRecordOpts(value ImageRecordOpts) ImageRecordOption {
	owned, captureErr := copyImageRecordOpts(nil, nil, value)
	return func(config *ImageRecordOpts) error {
		if captureErr != nil {
			return captureErr
		}
		copy, err := copyImageRecordOpts(nil, nil, owned)
		*config = copy
		return err
	}
}
func WithImageRecordHeader(key, value string) ImageRecordOption {
	return func(config *ImageRecordOpts) error {
		return mergeImageRecordHeaders(&config.Headers, map[string]string{key: value})
	}
}
func WithImageRecordHeaders(values map[string]string) ImageRecordOption {
	owned := copyImageRecordHeaders(values)
	return func(config *ImageRecordOpts) error { return mergeImageRecordHeaders(&config.Headers, owned) }
}
func WithImageRecordAttribute(field string, value any) ImageRecordOption {
	return WithImageRecordAttributes(map[string]any{field: value})
}
func WithImageRecordAttributes(values map[string]any) ImageRecordOption {
	owned, captureErr := captureImageRecordAttributes(nil, nil, values)
	return func(config *ImageRecordOpts) error {
		if captureErr != nil {
			return captureErr
		}
		if config.Attributes == nil {
			config.Attributes = make(map[string]any)
		}
		for key, raw := range owned {
			config.Attributes[key] = json.RawMessage(bytes.Clone(raw))
		}
		return nil
	}
}
func WithImageRecordListOpts(value ImageRecordListOpts) ImageRecordListOption {
	owned := copyImageRecordList(value)
	return func(config *ImageRecordListOpts) error { *config = copyImageRecordList(owned); return nil }
}
func WithImageRecordListHeader(key, value string) ImageRecordListOption {
	return func(config *ImageRecordListOpts) error {
		return mergeImageRecordHeaders(&config.Headers, map[string]string{key: value})
	}
}
func WithImageRecordListHeaders(values map[string]string) ImageRecordListOption {
	owned := copyImageRecordHeaders(values)
	return func(config *ImageRecordListOpts) error { return mergeImageRecordHeaders(&config.Headers, owned) }
}
func WithImageRecordListLimit(value int) ImageRecordListOption {
	return func(config *ImageRecordListOpts) error { owned := value; config.Limit = &owned; return nil }
}
func WithImageRecordListMarker(value string) ImageRecordListOption {
	return func(config *ImageRecordListOpts) error { config.Marker = value; return nil }
}
func WithImageRecordListMaxItems(value int) ImageRecordListOption {
	return func(config *ImageRecordListOpts) error { config.MaxItems = value; return nil }
}
func WithImageRecordListPaginated(value bool) ImageRecordListOption {
	return func(config *ImageRecordListOpts) error { owned := value; config.Paginated = &owned; return nil }
}

var imageRecordQueries = map[string]string{
	"id": "id", "name": "name", "visibility": "visibility", "member_status": "member_status",
	"owner": "owner", "status": "status", "size_min": "size_min", "size_max": "size_max",
	"protected": "protected", "is_hidden": "os_hidden", "sort_key": "sort_key", "sort_dir": "sort_dir",
	"sort": "sort", "tag": "tag", "created_at": "created_at", "updated_at": "updated_at",
}

func imageRecordQueryKey(key string) (string, string) {
	if wire, known := imageRecordQueries[key]; known {
		return key, wire
	}
	for canonical, wire := range imageRecordQueries {
		if key == wire {
			return canonical, wire
		}
	}
	return "", ""
}
func imageRecordLocalKey(key string) string {
	if _, wire := imageRecordQueryKey(key); wire != "" {
		return ""
	}
	for _, field := range imageRecordFields {
		if key == field.canonical {
			return key
		}
	}
	return ""
}
func imageRecordFilterControl(key string) bool {
	switch key {
	case "limit", "marker", "max_items", "paginated", "base_path", "session", "microversion", "headers", "jmespath_filters":
		return true
	}
	return false
}
func imageRecordFilterKnown(key string) bool {
	_, wire := imageRecordQueryKey(key)
	return wire != "" || imageRecordLocalKey(key) != "" || imageRecordFilterControl(key)
}
func WithImageRecordListFilter(field string, value any) ImageRecordListOption {
	if !imageRecordFilterKnown(field) {
		return func(*ImageRecordListOpts) error { return nil }
	}
	if imageRecordFilterControl(field) {
		return func(*ImageRecordListOpts) error {
			return fmt.Errorf("%w: image filter %q requires a dedicated control", resource.ErrUnsupported, field)
		}
	}
	raw, encodeErr := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(config *ImageRecordListOpts) error {
		if encodeErr != nil {
			return errors.Join(uploadInvalid("image filter %q cannot be encoded", field), encodeErr)
		}
		if config.Filters == nil {
			config.Filters = make(map[string]json.RawMessage)
		}
		canonical, wire := imageRecordQueryKey(field)
		if wire != "" {
			delete(config.Filters, canonical)
			delete(config.Filters, wire)
		}
		config.Filters[field] = bytes.Clone(owned)
		return nil
	}
}
func WithImageRecordListFilters(values map[string]any) ImageRecordListOption {
	owned := make(map[string]json.RawMessage)
	var captureErr error
	for key, value := range values {
		if !imageRecordFilterKnown(key) {
			continue
		}
		if imageRecordFilterControl(key) {
			captureErr = errors.Join(captureErr, fmt.Errorf("%w: image filter %q requires a dedicated control", resource.ErrUnsupported, key))
			continue
		}
		canonical, wire := imageRecordQueryKey(key)
		if wire != "" && key != canonical {
			if _, selected := values[canonical]; selected {
				continue
			}
		}
		raw, err := json.Marshal(value)
		if err != nil {
			captureErr = errors.Join(captureErr, uploadInvalid("image filter %q cannot be encoded", key), err)
		}
		owned[key] = bytes.Clone(raw)
	}
	return func(config *ImageRecordListOpts) error {
		if captureErr != nil {
			return captureErr
		}
		if config.Filters == nil {
			config.Filters = make(map[string]json.RawMessage)
		}
		for key, raw := range owned {
			canonical, wire := imageRecordQueryKey(key)
			if wire != "" {
				delete(config.Filters, canonical)
				delete(config.Filters, wire)
			}
			config.Filters[key] = bytes.Clone(raw)
		}
		return nil
	}
}
func prepareImageRecordGet(ctx context.Context, check func(context.Context) error, options []ImageRecordOption) (map[string]json.RawMessage, map[string]string, error) {
	value := ImageRecordOpts{Headers: make(map[string]string), Attributes: make(map[string]any)}
	for _, apply := range options {
		if err := check(ctx); err != nil {
			return nil, nil, err
		}
		if apply == nil {
			return nil, nil, uploadInvalid("nil image record option")
		}
		owned, err := copyImageRecordOpts(ctx, check, value)
		if err != nil {
			return nil, nil, err
		}
		if err := errors.Join(apply(&owned), check(ctx)); err != nil {
			return nil, nil, err
		}
		value, err = copyImageRecordOpts(ctx, check, owned)
		if err != nil {
			return nil, nil, err
		}
	}
	attrs, err := captureImageRecordAttributes(ctx, check, value.Attributes)
	if err != nil {
		return nil, nil, err
	}
	headers, err := imageMutationHeaders(value.Headers, false, "")
	return attrs, headers, errors.Join(err, check(ctx))
}

type imageRecordListParameters struct {
	query   url.Values
	headers map[string]string
	filters map[string]json.RawMessage
	control rest.ListControl
}

func prepareImageRecordList(ctx context.Context, check func(context.Context) error, options []ImageRecordListOption) (imageRecordListParameters, error) {
	value := copyImageRecordList(ImageRecordListOpts{})
	for _, apply := range options {
		if err := check(ctx); err != nil {
			return imageRecordListParameters{}, err
		}
		if apply == nil {
			return imageRecordListParameters{}, uploadInvalid("nil image record list option")
		}
		owned := copyImageRecordList(value)
		if err := errors.Join(apply(&owned), check(ctx)); err != nil {
			return imageRecordListParameters{}, err
		}
		value = copyImageRecordList(owned)
	}
	if value.Limit != nil && *value.Limit < 0 || value.MaxItems < 0 {
		return imageRecordListParameters{}, uploadInvalid("image limit and max items must be nonnegative")
	}
	query := make(url.Values)
	if value.Limit != nil {
		query.Set("limit", strconv.Itoa(*value.Limit))
	}
	if value.Marker != "" {
		if strings.TrimSpace(value.Marker) == "" {
			return imageRecordListParameters{}, uploadInvalid("image marker must be nonempty")
		}
		if err := taskQueryText(value.Marker); err != nil {
			return imageRecordListParameters{}, err
		}
		query.Set("marker", value.Marker)
	}
	filters := make(map[string]json.RawMessage)
	keys := make([]string, 0, len(value.Filters))
	for key := range value.Filters {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		if !imageRecordFilterKnown(key) {
			continue
		}
		if imageRecordFilterControl(key) {
			return imageRecordListParameters{}, fmt.Errorf("%w: image filter %q requires a dedicated control", resource.ErrUnsupported, key)
		}
		canonical, wire := imageRecordQueryKey(key)
		if wire != "" && key != canonical {
			if _, selected := value.Filters[canonical]; selected {
				continue
			}
		}
		raw := value.Filters[key]
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return imageRecordListParameters{}, uploadInvalid("image filter %q must be complete UTF-8 JSON", key)
		}
		if wire != "" {
			values, err := cloudfilter.RequestQueryValues(raw)
			if err != nil {
				return imageRecordListParameters{}, uploadInvalid("image query %q: %v", key, err)
			}
			for _, text := range values {
				if err := taskQueryText(text); err != nil {
					return imageRecordListParameters{}, err
				}
			}
			query[wire] = values
		} else {
			filters[key] = bytes.Clone(raw)
		}
	}
	headers, err := imageMutationHeaders(value.Headers, false, "")
	if err != nil {
		return imageRecordListParameters{}, err
	}
	return imageRecordListParameters{query: query, headers: headers, filters: filters,
		control: rest.ListControl{MaxItems: value.MaxItems, SinglePage: value.Paginated != nil && !*value.Paginated, LimitHint: true}}, check(ctx)
}

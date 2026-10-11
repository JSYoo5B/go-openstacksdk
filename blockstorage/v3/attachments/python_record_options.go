package attachments

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/url"
	"strconv"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// RecordCreateOpts selects Source Body attributes. Unknown attributes are
// ignored, as Attachment does not allow unknown request attributes.
type RecordCreateOpts struct {
	Fields  map[string]json.RawMessage
	Headers map[string]string
	// ModeSupported supplies the discovered supports_microversion("3.54")
	// result when a configured version cannot establish it (empty or latest).
	ModeSupported *bool
}
type RecordCreateOption func(*RecordCreateOpts) error

func pythonAttachmentCopyCreate(v RecordCreateOpts) RecordCreateOpts {
	v.Fields = pythonAttachmentClone(v.Fields)
	v.Headers = maps.Clone(v.Headers)
	if v.ModeSupported != nil {
		x := *v.ModeSupported
		v.ModeSupported = &x
	}
	return v
}
func WithRecordCreateOpts(v RecordCreateOpts) RecordCreateOption {
	owned := pythonAttachmentCopyCreate(v)
	return func(c *RecordCreateOpts) error { *c = pythonAttachmentCopyCreate(owned); return nil }
}

// WithRecordCreateField accepts declared attribute or wire names. Ordered
// options give aliases deterministic last-write precedence.
func WithRecordCreateField(key string, value any) RecordCreateOption {
	name, ok := pythonAttachmentField(key)
	if !ok {
		return func(*RecordCreateOpts) error { return nil }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordCreateOpts) error {
		if err != nil {
			return err
		}
		if c.Fields == nil {
			c.Fields = make(map[string]json.RawMessage)
		}
		c.Fields[name] = bytes.Clone(owned)
		return nil
	}
}
func WithRecordCreateHeaders(v map[string]string) RecordCreateOption {
	owned := maps.Clone(v)
	return func(c *RecordCreateOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithRecordCreateModeSupported(v bool) RecordCreateOption {
	return func(c *RecordCreateOpts) error { x := v; c.ModeSupported = &x; return nil }
}

// RecordListOpts splits known server parameters from declared local filters.
// MaxItems counts decoded rows before filtering, matching Resource.list.
type RecordListOpts struct {
	Query        map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	Limit        *int
	Marker       string
	MaxItems     int
	Paginated    *bool
}
type RecordListOption func(*RecordListOpts) error

func pythonAttachmentCopyList(v RecordListOpts) RecordListOpts {
	v.Query = pythonAttachmentClone(v.Query)
	v.Headers = maps.Clone(v.Headers)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	if v.Limit != nil {
		x := *v.Limit
		v.Limit = &x
	}
	if v.Paginated != nil {
		x := *v.Paginated
		v.Paginated = &x
	}
	return v
}
func WithRecordListOpts(v RecordListOpts) RecordListOption {
	owned := pythonAttachmentCopyList(v)
	return func(c *RecordListOpts) error { *c = pythonAttachmentCopyList(owned); return nil }
}
func pythonAttachmentServerQuery(key string) (string, bool) {
	switch key {
	case "id", "status", "project_id", "volume_id", "instance_id", "limit", "marker", "all_tenants":
		return key, true
	case "all_projects":
		return "all_tenants", true
	default:
		return "", false
	}
}
func pythonAttachmentLocalQuery(key string) bool {
	for _, field := range pythonAttachmentFields {
		if key == field.attribute {
			_, server := pythonAttachmentServerQuery(key)
			return !server
		}
	}
	return false
}
func WithRecordListQuery(key string, value any) RecordListOption {
	_, server := pythonAttachmentServerQuery(key)
	if !server && !pythonAttachmentLocalQuery(key) {
		return func(*RecordListOpts) error { return nil }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordListOpts) error {
		if err != nil {
			return err
		}
		if c.Query == nil {
			c.Query = make(map[string]json.RawMessage)
		}
		c.Query[key] = bytes.Clone(owned)
		return nil
	}
}
func WithRecordListLimit(v int) RecordListOption {
	return func(c *RecordListOpts) error { x := v; c.Limit = &x; return nil }
}
func WithRecordListMarker(v string) RecordListOption {
	return func(c *RecordListOpts) error { c.Marker = v; return nil }
}
func WithRecordListMaxItems(v int) RecordListOption {
	return func(c *RecordListOpts) error { c.MaxItems = v; return nil }
}
func WithRecordListPaginated(v bool) RecordListOption {
	return func(c *RecordListOpts) error { x := v; c.Paginated = &x; return nil }
}
func WithRecordListMicroversion(v string) RecordListOption {
	return func(c *RecordListOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithRecordListHeaders(v map[string]string) RecordListOption {
	owned := maps.Clone(v)
	return func(c *RecordListOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func pythonAttachmentApply[T any, O ~func(*T) error](ctx context.Context, guard func(context.Context) error, value T, copyValue func(T) T, options []O) (T, error) {
	for _, option := range options {
		if err := guard(ctx); err != nil {
			return value, err
		}
		if option == nil {
			return value, pythonAttachmentInvalid("nil attachment option")
		}
		next := copyValue(value)
		if err := errors.Join(option(&next), guard(ctx)); err != nil {
			return value, err
		}
		value = copyValue(next)
	}
	return value, guard(ctx)
}
func pythonAttachmentPrepareList(v RecordListOpts) (url.Values, map[string]json.RawMessage, rest.ListControl, error) {
	query, filters := make(url.Values), make(map[string]json.RawMessage)
	if v.MaxItems < 0 || v.Limit != nil && *v.Limit < 0 {
		return nil, nil, rest.ListControl{}, pythonAttachmentInvalid("negative limit or maximum items")
	}
	for key, raw := range v.Query {
		if key == "all_tenants" {
			if _, exists := v.Query["all_projects"]; exists {
				continue
			}
		}
		wire, server := pythonAttachmentServerQuery(key)
		if !server && !pythonAttachmentLocalQuery(key) {
			continue
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, rest.ListControl{}, pythonAttachmentInvalid("query %q must be UTF-8 JSON", key)
		}
		if server {
			values, err := cloudfilter.RequestQueryValues(raw)
			if err != nil {
				return nil, nil, rest.ListControl{}, err
			}
			if len(values) != 0 {
				query[wire] = values
			}
		} else {
			filters[key] = bytes.Clone(raw)
		}
	}
	if v.Limit != nil {
		query.Set("limit", strconv.Itoa(*v.Limit))
	}
	if v.Marker != "" {
		if !utf8.ValidString(v.Marker) {
			return nil, nil, rest.ListControl{}, pythonAttachmentInvalid("marker must be UTF-8")
		}
		query.Set("marker", v.Marker)
	}
	return query, filters, rest.ListControl{MaxItems: v.MaxItems, SinglePage: v.Paginated != nil && !*v.Paginated, LimitHint: true}, nil
}

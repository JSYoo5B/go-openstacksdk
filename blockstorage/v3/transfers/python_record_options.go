package transfers

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

// RecordReadOpts applies owned headers while discovery and routing remain SDK-owned.
type RecordReadOpts struct{ Headers map[string]string }
type RecordReadOption func(*RecordReadOpts) error

func copyRecordRead(v RecordReadOpts) RecordReadOpts { v.Headers = maps.Clone(v.Headers); return v }
func WithRecordReadHeaders(values map[string]string) RecordReadOption {
	owned := maps.Clone(values)
	return func(c *RecordReadOpts) error { c.Headers = maps.Clone(owned); return nil }
}

// RecordCreateOpts preserves absent versus explicit JSON values. Extensions are
// an explicit Go escape hatch; ordinary Fields retain Source declared-attribute policy.
type RecordCreateOpts struct {
	Headers                 map[string]string
	Fields                  map[string]json.RawMessage
	Extensions              map[string]json.RawMessage
	SuppressLegacySnapshots bool
}
type RecordCreateOption func(*RecordCreateOpts) error

func copyRecordCreate(v RecordCreateOpts) RecordCreateOpts {
	v.Headers = maps.Clone(v.Headers)
	v.Fields = cloneRecordFields(v.Fields)
	v.Extensions = cloneRecordFields(v.Extensions)
	return v
}
func WithRecordCreateOpts(value RecordCreateOpts) RecordCreateOption {
	owned := copyRecordCreate(value)
	return func(c *RecordCreateOpts) error { *c = copyRecordCreate(owned); return nil }
}
func WithRecordCreateField(key string, value any) RecordCreateOption {
	if !recordKnown(key) {
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
		c.Fields[key] = bytes.Clone(owned)
		return nil
	}
}
func WithRecordCreateVolumeID(value string) RecordCreateOption {
	return WithRecordCreateField("volume_id", value)
}
func WithRecordCreateName(value string) RecordCreateOption {
	return WithRecordCreateField("name", value)
}
func WithRecordCreateNoSnapshots(value bool) RecordCreateOption {
	return WithRecordCreateField("no_snapshots", value)
}
func WithRecordCreateHeaders(values map[string]string) RecordCreateOption {
	owned := maps.Clone(values)
	return func(c *RecordCreateOpts) error { c.Headers = maps.Clone(owned); return nil }
}

// WithRecordCreateExtension sends an otherwise undeclared body key. It cannot
// override a declared field: use WithRecordCreateField for that field.
func WithRecordCreateExtension(key string, value any) RecordCreateOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordCreateOpts) error {
		if err != nil {
			return err
		}
		if key == "" || !utf8.ValidString(key) || recordKnown(key) {
			return recordInvalid("extension must be an undeclared nonempty UTF-8 key")
		}
		if c.Extensions == nil {
			c.Extensions = make(map[string]json.RawMessage)
		}
		c.Extensions[key] = bytes.Clone(owned)
		return nil
	}
}

// WithRecordCreateSuppressLegacySnapshots opts into omitting no_snapshots when
// 3.55 is unavailable. The default preserves the pinned proxy's actual body:
// its cleanup only examines params, although proxy attrs live on the Resource.
func WithRecordCreateSuppressLegacySnapshots(value bool) RecordCreateOption {
	return func(c *RecordCreateOpts) error { c.SuppressLegacySnapshots = value; return nil }
}

type RecordDeleteOpts struct {
	Headers       map[string]string
	IgnoreMissing *bool
}
type RecordDeleteOption func(*RecordDeleteOpts) error

func copyRecordDelete(v RecordDeleteOpts) RecordDeleteOpts {
	v.Headers = maps.Clone(v.Headers)
	if v.IgnoreMissing != nil {
		x := *v.IgnoreMissing
		v.IgnoreMissing = &x
	}
	return v
}
func WithRecordDeleteIgnoreMissing(value bool) RecordDeleteOption {
	return func(c *RecordDeleteOpts) error { x := value; c.IgnoreMissing = &x; return nil }
}
func WithRecordDeleteHeaders(values map[string]string) RecordDeleteOption {
	owned := maps.Clone(values)
	return func(c *RecordDeleteOpts) error { c.Headers = maps.Clone(owned); return nil }
}

type RecordAcceptOpts struct {
	Headers map[string]string
	AuthKey json.RawMessage
}
type RecordAcceptOption func(*RecordAcceptOpts) error

func copyRecordAccept(v RecordAcceptOpts) RecordAcceptOpts {
	v.Headers = maps.Clone(v.Headers)
	v.AuthKey = bytes.Clone(v.AuthKey)
	return v
}
func WithRecordAcceptAuthKey(value any) RecordAcceptOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordAcceptOpts) error {
		if err != nil {
			return err
		}
		c.AuthKey = bytes.Clone(owned)
		return nil
	}
}
func WithRecordAcceptHeaders(values map[string]string) RecordAcceptOption {
	owned := maps.Clone(values)
	return func(c *RecordAcceptOpts) error { c.Headers = maps.Clone(owned); return nil }
}

type RecordFindOpts struct{ IgnoreMissing *bool }
type RecordFindOption func(*RecordFindOpts) error

func copyRecordFind(v RecordFindOpts) RecordFindOpts {
	if v.IgnoreMissing != nil {
		x := *v.IgnoreMissing
		v.IgnoreMissing = &x
	}
	return v
}
func WithRecordFindIgnoreMissing(value bool) RecordFindOption {
	return func(c *RecordFindOpts) error { x := value; c.IgnoreMissing = &x; return nil }
}

// RecordListOpts defaults Details and Paginated to true. Only limit/marker are
// server query descriptors in Source; declared Body attrs filter locally.
// AllProjects mirrors the pinned proxy parameter, which Source discards because
// Transfer declares neither an all_projects Body field nor a query mapping.
type RecordListOpts struct {
	Headers     map[string]string
	Query       map[string]json.RawMessage
	ServerQuery url.Values
	Details     *bool
	AllProjects bool
	Limit       *int
	Marker      string
	MaxItems    int
	Paginated   *bool
}
type RecordListOption func(*RecordListOpts) error

func copyRecordList(v RecordListOpts) RecordListOpts {
	v.Headers = maps.Clone(v.Headers)
	v.Query = cloneRecordFields(v.Query)
	v.ServerQuery = recordQueryCopy(v.ServerQuery)
	if v.Details != nil {
		x := *v.Details
		v.Details = &x
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
func recordQueryCopy(values url.Values) url.Values {
	result := make(url.Values, len(values))
	for key, items := range values {
		result[key] = append([]string(nil), items...)
	}
	return result
}
func WithRecordListOpts(value RecordListOpts) RecordListOption {
	owned := copyRecordList(value)
	return func(c *RecordListOpts) error { *c = copyRecordList(owned); return nil }
}
func WithRecordListQuery(key string, value any) RecordListOption {
	if !recordKnown(key) && key != "limit" && key != "marker" {
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
func WithRecordListDetails(value bool) RecordListOption {
	return func(c *RecordListOpts) error { x := value; c.Details = &x; return nil }
}
func WithRecordListAllProjects(value bool) RecordListOption {
	return func(c *RecordListOpts) error { c.AllProjects = value; return nil }
}
func WithRecordListLimit(value int) RecordListOption {
	return func(c *RecordListOpts) error { x := value; c.Limit = &x; return nil }
}
func WithRecordListMarker(value string) RecordListOption {
	return func(c *RecordListOpts) error { c.Marker = value; return nil }
}
func WithRecordListMaxItems(value int) RecordListOption {
	return func(c *RecordListOpts) error { c.MaxItems = value; return nil }
}
func WithRecordListPaginated(value bool) RecordListOption {
	return func(c *RecordListOpts) error { x := value; c.Paginated = &x; return nil }
}
func WithRecordListHeaders(values map[string]string) RecordListOption {
	owned := maps.Clone(values)
	return func(c *RecordListOpts) error { c.Headers = maps.Clone(owned); return nil }
}

// WithRecordListServerQuery is an explicit Go extension for an otherwise
// undeclared server filter, such as all_tenants. Pagination keeps owned helpers.
func WithRecordListServerQuery(key string, values ...string) RecordListOption {
	owned := append([]string(nil), values...)
	return func(c *RecordListOpts) error {
		if key == "" || !utf8.ValidString(key) || key == "limit" || key == "marker" {
			return recordInvalid("server extension query must have an undeclared UTF-8 key")
		}
		if c.ServerQuery == nil {
			c.ServerQuery = make(url.Values)
		}
		c.ServerQuery[key] = append([]string(nil), owned...)
		return nil
	}
}

func applyRecordOptions[T any, O ~func(*T) error](ctx context.Context, guard func(context.Context) error, value T, copyValue func(T) T, options []O) (T, error) {
	for _, apply := range options {
		if err := guard(ctx); err != nil {
			return value, err
		}
		if apply == nil {
			return value, recordInvalid("nil record option")
		}
		next := copyValue(value)
		err := apply(&next)
		if err = errors.Join(err, guard(ctx)); err != nil {
			return value, err
		}
		value = copyValue(next)
	}
	return value, guard(ctx)
}
func prepareRecordList(v RecordListOpts) (url.Values, map[string]json.RawMessage, rest.ListControl, error) {
	query := recordQueryCopy(v.ServerQuery)
	filters := make(map[string]json.RawMessage)
	if v.MaxItems < 0 || v.Limit != nil && *v.Limit < 0 {
		return nil, nil, rest.ListControl{}, recordInvalid("negative limit or maximum items")
	}
	for key, raw := range v.Query {
		if !recordKnown(key) && key != "limit" && key != "marker" {
			continue
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, rest.ListControl{}, recordInvalid("query %q must be UTF-8 JSON", key)
		}
		if key == "limit" || key == "marker" {
			values, err := cloudfilter.RequestQueryValues(raw)
			if err != nil {
				return nil, nil, rest.ListControl{}, err
			}
			if len(values) != 0 {
				query[key] = values
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
			return nil, nil, rest.ListControl{}, recordInvalid("marker must be UTF-8")
		}
		query.Set("marker", v.Marker)
	}
	return query, filters, rest.ListControl{MaxItems: v.MaxItems, SinglePage: v.Paginated != nil && !*v.Paginated, LimitHint: true}, nil
}

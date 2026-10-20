package policies

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"maps"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// PolicyRecord keeps the declared Source view separately from remote fields
// and the physical page receipt. Policy Body values are nullable and untyped.
type PolicyRecord = neutronfind.Record
type RecordListOpts struct {
	Limit     *int
	Marker    string
	MaxItems  int
	Paginated *bool
	Filters   map[string]json.RawMessage
	Headers   map[string]string
}
type RecordListOption func(*RecordListOpts) error

func copyRecordList(v RecordListOpts) RecordListOpts {
	if v.Limit != nil {
		x := *v.Limit
		v.Limit = &x
	}
	if v.Paginated != nil {
		x := *v.Paginated
		v.Paginated = &x
	}
	v.Headers = maps.Clone(v.Headers)
	owned := make(map[string]json.RawMessage, len(v.Filters))
	for k, raw := range v.Filters {
		owned[k] = bytes.Clone(raw)
	}
	v.Filters = owned
	return v
}
func WithRecordListOpts(v RecordListOpts) RecordListOption {
	owned := copyRecordList(v)
	return func(c *RecordListOpts) error { *c = copyRecordList(owned); return nil }
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
func WithRecordListHeader(key, value string) RecordListOption {
	return func(c *RecordListOpts) error {
		if c.Headers == nil {
			c.Headers = map[string]string{}
		}
		c.Headers[key] = value
		return nil
	}
}
func policyFieldKnown(key string) bool {
	switch key {
	case "id", "name", "blob", "links", "project_id", "type", "user_id":
		return true
	}
	return false
}

// WithRecordListFilter is a canonical Body filter, applied after each row is
// projected. Unknown keys are ignored before invoking JSON marshaling.
func WithRecordListFilter(key string, value any) RecordListOption {
	if !policyFieldKnown(key) {
		return func(*RecordListOpts) error { return nil }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordListOpts) error {
		if err != nil {
			return err
		}
		if c.Filters == nil {
			c.Filters = map[string]json.RawMessage{}
		}
		c.Filters[key] = bytes.Clone(owned)
		return nil
	}
}
func policyListInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func policyListProfile() neutronfind.Profile {
	fields := []neutronfind.Descriptor{}
	for _, key := range []string{"blob", "links", "project_id", "type", "user_id", "id", "name"} {
		fields = append(fields, neutronfind.Descriptor{Name: key, Wire: key, Missing: "null"})
	}
	return neutronfind.Profile{Kind: "policies", ServiceRole: "identity", Path: "policies", Plural: "policies", Singular: "policy", IDKey: "id", NameKey: "name", Body: fields,
		Query: []neutronfind.QueryPair{{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}}, SourceFindPolicy: true, SourceStatusPolicy: true, SourceLinkPolicy: true, StopFaultRetryCallbacks: true, PythonLoadedEquality: true, VersionedPath: "/v3/policies"}
}

// ListRecords implements the Policy list profile without changing native List
// or All. Only limit/marker are server filters; all known Body keys are local.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*PolicyRecord, error] {
	owned := slices.Clone(options)
	return func(yield func(*PolicyRecord, error) bool) {
		fail := func(err error) { yield(nil, request.Wrap("ListRecords", "policies", cloudread.ContextError(ctx, err))) }
		if err := cloudread.Context(ctx); err != nil {
			fail(err)
			return
		}
		if a == nil {
			fail(policyListInvalid("policy API is required"))
			return
		}
		selected := a.client
		engine, err := neutronfind.New(ctx, selected, policyListProfile(), func(context.Context) error {
			if a.client != selected {
				return policyListInvalid("selected policy API changed")
			}
			return nil
		})
		if err != nil {
			fail(err)
			return
		}
		cfg := copyRecordList(RecordListOpts{})
		for _, apply := range owned {
			if apply == nil {
				fail(policyListInvalid("nil list record option"))
				return
			}
			if err := engine.Check(ctx); err != nil {
				fail(err)
				return
			}
			next := copyRecordList(cfg)
			if err := errors.Join(apply(&next), engine.Check(ctx)); err != nil {
				fail(err)
				return
			}
			cfg = copyRecordList(next)
		}
		if cfg.MaxItems < 0 || cfg.Limit != nil && *cfg.Limit < 0 {
			fail(policyListInvalid("limit and max items must be nonnegative"))
			return
		}
		query := make(map[string]json.RawMessage)
		for key, raw := range cfg.Filters {
			if policyFieldKnown(key) {
				query[key] = bytes.Clone(raw)
			}
		}
		if cfg.Limit != nil {
			query["limit"], err = json.Marshal(*cfg.Limit)
		}
		if cfg.Marker != "" {
			query["marker"], err = json.Marshal(cfg.Marker)
		}
		if cfg.MaxItems > 0 {
			query["max_items"], err = json.Marshal(cfg.MaxItems)
		}
		if cfg.Paginated != nil {
			query["paginated"], err = json.Marshal(*cfg.Paginated)
		}
		if err != nil {
			fail(err)
			return
		}
		for value, err := range engine.List(ctx, neutronfind.Options{Query: query, Headers: cfg.Headers}) {
			if err != nil {
				fail(err)
				return
			}
			if !yield(value, nil) {
				return
			}
		}
	}
}

// AllRecords returns successfully consumed rows even after a later error.
func (a *API) AllRecords(ctx context.Context, options ...RecordListOption) ([]*PolicyRecord, error) {
	out := make([]*PolicyRecord, 0)
	for value, err := range a.ListRecords(ctx, options...) {
		if err != nil {
			return out, err
		}
		out = append(out, value)
	}
	return out, nil
}

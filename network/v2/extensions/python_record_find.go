package extensions

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// RecordFindOpts owns Source kwargs. Omitted IgnoreMissing defaults to true.
// Query preserves JSON values, including arrays and null; unknown kwargs go to
// the member GET but are ignored by the list, following Source Resource.find.
// requires_id, base_path, skip_cache, error_message and resource_response_key
// control the GET stage. Raw headers remains a GET query value and supplies
// HTTP headers only if the lookup falls back to LIST.
type RecordFindOpts struct {
	IgnoreMissing *bool
	Microversion  *string
	ListBasePath  *string
	queryOrder    []string
	Query         map[string]json.RawMessage
	Headers       map[string]string
}
type RecordFindOption func(*RecordFindOpts) error

func internalFindOpts(value RecordFindOpts) neutronfind.Options {
	return neutronfind.Options{IgnoreMissing: value.IgnoreMissing, Microversion: value.Microversion, ListBasePath: value.ListBasePath, Query: value.Query, Headers: value.Headers, Order: value.queryOrder}
}
func copyFindOpts(value RecordFindOpts) RecordFindOpts {
	owned := neutronfind.CloneOptions(internalFindOpts(value))
	value.IgnoreMissing, value.Microversion, value.ListBasePath = owned.IgnoreMissing, owned.Microversion, owned.ListBasePath
	value.Query, value.Headers, value.queryOrder = owned.Query, owned.Headers, owned.Order
	if value.Headers == nil {
		value.Headers = make(map[string]string)
	}
	return value
}

func WithRecordFindOpts(value RecordFindOpts) RecordFindOption {
	owned := copyFindOpts(value)
	return func(c *RecordFindOpts) error { *c = copyFindOpts(owned); return nil }
}
func WithRecordFindIgnoreMissing(value bool) RecordFindOption {
	return func(c *RecordFindOpts) error { x := value; c.IgnoreMissing = &x; return nil }
}
func WithRecordFindQuery(key string, value any) RecordFindOption {
	switch key {
	case "id", "name_or_id", "ignore_missing", "cls", "session", "self", "resource_type", "connection", "_synchronized":
		return func(*RecordFindOpts) error {
			return findInvalid("%q collides with a bound Find/Resource constructor argument", key)
		}
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordFindOpts) error {
		if err != nil {
			return err
		}
		switch key {
		case "list_base_path", "microversion":
			formalKey := key
			if key == "list_base_path" {
				formalKey = "base_path"
			}
			normalized, _, err := neutronfind.NormalizeProxyListOptions(neutronfind.Options{Query: map[string]json.RawMessage{formalKey: owned}})
			if err != nil {
				return err
			}
			if key == "list_base_path" {
				c.ListBasePath = normalized.BasePath
			} else {
				c.Microversion = normalized.Microversion
			}
			delete(c.Query, key)
			c.queryOrder = slices.DeleteFunc(c.queryOrder, func(existing string) bool { return existing == key })
			return nil
		}
		if c.Query == nil {
			c.Query = make(map[string]json.RawMessage)
		}
		c.Query[key] = bytes.Clone(owned)
		if !slices.Contains(c.queryOrder, key) {
			c.queryOrder = append(c.queryOrder, key)
		}
		return nil
	}
}
func WithRecordFindMicroversion(value string) RecordFindOption {
	return func(c *RecordFindOpts) error {
		x := value
		c.Microversion = &x
		delete(c.Query, "microversion")
		c.queryOrder = slices.DeleteFunc(c.queryOrder, func(key string) bool { return key == "microversion" })
		return nil
	}
}
func WithRecordFindHeaders(value map[string]string) RecordFindOption {
	owned := maps.Clone(value)
	return func(c *RecordFindOpts) error { c.Headers = maps.Clone(owned); return nil }
}

// WithRecordFindListBasePath selects only the fallback list route. The member
// GET keeps the default route or its raw base_path override. An empty list
// path keeps the route selected by raw base_path or the Extensions default.
func WithRecordFindListBasePath(path string) RecordFindOption {
	return func(c *RecordFindOpts) error {
		owned := path
		c.ListBasePath = &owned
		delete(c.Query, "list_base_path")
		c.queryOrder = slices.DeleteFunc(c.queryOrder, func(key string) bool { return key == "list_base_path" })
		return nil
	}
}

// FindRecord matches the pinned Source GET-first, complete ID-or-name search.
// It suppresses only clean GET 400/403/404. A physical body/Close/source/cancel
// failure never becomes a fallback or missing-resource success.
func (a *API) FindRecord(ctx context.Context, identity string, options ...RecordFindOption) (*ExtensionRecord, error) {
	const op = "FindRecord"
	wrap := func(err error) error { return request.Wrap(op, "extensions", cloudread.ContextError(ctx, err)) }
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrap(err)
	}
	if a == nil {
		return nil, wrap(findInvalid("extensions API is required"))
	}
	selected := a.client
	engine, err := neutronfind.New(ctx, selected, extensionRecordProfile, func(ctx context.Context) error {
		if a.client != selected {
			return findInvalid("selected extensions client changed")
		}
		return nil
	})
	if err != nil {
		return nil, wrap(err)
	}
	guard := engine.Check
	policy := RecordFindOpts{}
	for _, apply := range slices.Clone(options) {
		if err := guard(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(findInvalid("nil find option"))
		}
		next := copyFindOpts(policy)
		err := apply(&next)
		if err = errors.Join(err, guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		policy = copyFindOpts(next)
	}
	for key, raw := range policy.Query {
		if key != "microversion" && key != "list_base_path" {
			continue
		}
		formalKey := key
		if key == "list_base_path" {
			formalKey = "base_path"
		}
		formal, _, err := neutronfind.NormalizeProxyListOptions(neutronfind.Options{Query: map[string]json.RawMessage{formalKey: raw}})
		if err != nil {
			return nil, wrap(err)
		}
		if key == "list_base_path" {
			policy.ListBasePath = formal.BasePath
		} else {
			policy.Microversion = formal.Microversion
		}
		delete(policy.Query, key)
		policy.queryOrder = slices.DeleteFunc(policy.queryOrder, func(existing string) bool { return existing == key })
	}
	value, err := engine.FindWithFetchControls(ctx, identity, internalFindOpts(policy))
	if value == nil {
		return nil, wrap(err)
	}
	result := &ExtensionRecord{Resource: value.Resource, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode, row: value.Row(), fromList: value.FromList()}
	return result, wrap(err)
}

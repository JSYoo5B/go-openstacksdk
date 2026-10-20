package bgpvpns

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// PortAssociationFindOpts owns Source kwargs. Omitted IgnoreMissing defaults to true.
// Query preserves JSON values, including arrays and null; unknown kwargs go to
// the member GET but are ignored by the list, following Source Resource.find.
// requires_id, base_path, skip_cache, error_message and resource_response_key
// control the GET stage. Raw headers remains a GET query value and supplies
// HTTP headers only if the lookup falls back to LIST.
type PortAssociationFindOpts struct {
	IgnoreMissing *bool
	Microversion  *string
	ListBasePath  *string
	queryOrder    []string
	Query         map[string]json.RawMessage
	Headers       map[string]string
}
type PortAssociationFindOption func(*PortAssociationFindOpts) error

func internalPortAssociationFindOpts(value PortAssociationFindOpts) neutronfind.Options {
	return neutronfind.Options{IgnoreMissing: value.IgnoreMissing, Microversion: value.Microversion, ListBasePath: value.ListBasePath, Query: value.Query, Headers: value.Headers, Order: value.queryOrder}
}
func copyPortAssociationFindOpts(value PortAssociationFindOpts) PortAssociationFindOpts {
	owned := neutronfind.CloneOptions(internalPortAssociationFindOpts(value))
	value.IgnoreMissing, value.Microversion, value.ListBasePath = owned.IgnoreMissing, owned.Microversion, owned.ListBasePath
	value.Query, value.Headers, value.queryOrder = owned.Query, owned.Headers, owned.Order
	if value.Headers == nil {
		value.Headers = make(map[string]string)
	}
	return value
}

func WithPortAssociationFindOpts(value PortAssociationFindOpts) PortAssociationFindOption {
	owned := copyPortAssociationFindOpts(value)
	return func(c *PortAssociationFindOpts) error { *c = copyPortAssociationFindOpts(owned); return nil }
}
func WithPortAssociationFindIgnoreMissing(value bool) PortAssociationFindOption {
	return func(c *PortAssociationFindOpts) error { x := value; c.IgnoreMissing = &x; return nil }
}
func WithPortAssociationFindQuery(key string, value any) PortAssociationFindOption {
	switch key {
	case "bgpvpn_id", "id", "name_or_id", "ignore_missing", "cls", "session", "self", "resource_type", "connection", "_synchronized":
		return func(*PortAssociationFindOpts) error {
			return listRecordInvalid("%q collides with a bound Find/Resource constructor argument", key)
		}
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *PortAssociationFindOpts) error {
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
func WithPortAssociationFindMicroversion(value string) PortAssociationFindOption {
	return func(c *PortAssociationFindOpts) error {
		x := value
		c.Microversion = &x
		delete(c.Query, "microversion")
		c.queryOrder = slices.DeleteFunc(c.queryOrder, func(key string) bool { return key == "microversion" })
		return nil
	}
}
func WithPortAssociationFindHeaders(value map[string]string) PortAssociationFindOption {
	owned := maps.Clone(value)
	return func(c *PortAssociationFindOpts) error { c.Headers = maps.Clone(owned); return nil }
}

// WithPortAssociationFindListBasePath selects only the fallback list route. The member
// GET keeps the default route or its raw base_path override. An empty list
// path keeps the route selected by raw base_path or the parent port-association default.
func WithPortAssociationFindListBasePath(path string) PortAssociationFindOption {
	return func(c *PortAssociationFindOpts) error {
		owned := path
		c.ListBasePath = &owned
		delete(c.Query, "list_base_path")
		c.queryOrder = slices.DeleteFunc(c.queryOrder, func(key string) bool { return key == "list_base_path" })
		return nil
	}
}

// FindPortAssociationRecord matches the pinned Source GET-first, complete ID-or-name search.
// It suppresses only clean GET 400/403/404. A physical body/Close/source/cancel
// failure never becomes a fallback or missing-resource success.
func (a *API) FindPortAssociationRecord(ctx context.Context, parent, identity string, options ...PortAssociationFindOption) (*PortAssociationRecord, error) {
	const op = "FindPortAssociationRecord"
	wrap := func(err error) error {
		return request.Wrap(op, "bgpvpn_port_associations", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrap(err)
	}
	if a == nil {
		return nil, wrap(listRecordInvalid("BGPVPN API is required"))
	}
	if err := portAssociationParent(parent); err != nil {
		return nil, wrap(err)
	}
	selected := a.client
	engine, err := neutronfind.New(ctx, selected, portAssociationProfile(parent), func(ctx context.Context) error {
		if a.client != selected {
			return listRecordInvalid("selected BGPVPN client changed")
		}
		return nil
	})
	if err != nil {
		return nil, wrap(err)
	}
	guard := engine.Check
	policy := PortAssociationFindOpts{}
	for _, apply := range slices.Clone(options) {
		if err := guard(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(listRecordInvalid("nil find option"))
		}
		next := copyPortAssociationFindOpts(policy)
		err := apply(&next)
		if err = errors.Join(err, guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		policy = copyPortAssociationFindOpts(next)
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
	for key := range policy.Query {
		switch key {
		case "bgpvpn_id", "id", "name_or_id", "ignore_missing", "cls", "session", "self", "resource_type", "connection", "_synchronized":
			return nil, wrap(listRecordInvalid("%q collides with a bound Find/Resource argument", key))
		}
	}
	value, err := engine.FindWithFetchControls(ctx, identity, internalPortAssociationFindOpts(policy))
	if value == nil {
		return nil, wrap(err)
	}
	result := &PortAssociationRecord{Resource: value.Resource, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode}
	if result.Resource != nil {
		delete(result.Resource.Body, "bgpvpn_id")
	}
	return result, wrap(errors.Join(err, engine.Check(ctx)))
}

// PortAssociationRecord separates Source attributes from actual response proof.
// bgpvpn_id is a URI parameter and is excluded from the Body attribute view.
type PortAssociationRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

func (r *PortAssociationRecord) Clone() *PortAssociationRecord {
	if r == nil {
		return nil
	}
	out := *r
	out.Resource = r.Resource.Clone()
	out.Wire = r.Wire.Clone()
	out.Envelope = bytes.Clone(r.Envelope)
	out.Header = r.Header.Clone()
	return &out
}
func portAssociationParent(id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." || !utf8.ValidString(id) || strings.IndexFunc(id, unicode.IsControl) >= 0 {
		return listRecordInvalid("a nonblank UTF-8 parent ID without controls is required")
	}
	return nil
}
func portAssociationProfile(parent string) neutronfind.Profile {
	path := "bgpvpn/bgpvpns/" + url.PathEscape(parent) + "/port_associations"
	return neutronfind.Profile{Kind: "bgpvpn_port_associations", Path: path, Singular: "port_association", Plural: "port_associations", IDKey: "id", NameKey: "name", VersionedPath: "/v2.0/" + path, PreserveUnknownBody: true, SourceFindPolicy: true, SourceStatusPolicy: true, SourceLinkPolicy: true, StopFaultRetryCallbacks: true, PythonLoadedEquality: true,
		Body: []neutronfind.Descriptor{{Name: "id", Wire: "id", Missing: "null"}, {Name: "name", Wire: "name", Missing: "null"}, {Name: "project_id", Wire: "project_id", Alias: "tenant_id", Missing: "null"}, {Name: "tenant_id", Wire: "tenant_id", Missing: "null"}, {Name: "port_id", Wire: "port_id", Missing: "null"}, {Name: "advertise_fixed_ips", Wire: "advertise_fixed_ips", Missing: "null"}, {Name: "routes", Wire: "routes", Kind: neutronfind.List, Missing: "null"}}, Query: []neutronfind.QueryPair{{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}}}
}

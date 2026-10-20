package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"maps"
	"net/url"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jmespath"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

type DHCPHostRecord = neutronfind.Record
type DHCPHostListOpts struct {
	Query        map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	order        []string
	basePath     *string
}
type DHCPHostListOption func(*DHCPHostListOpts) error

func dhcpHostPolicy(v DHCPHostListOpts) neutronfind.Options {
	return neutronfind.Options{Query: v.Query, Headers: v.Headers, Microversion: v.Microversion, Order: v.order}
}
func cloneDHCPHostOpts(v DHCPHostListOpts) DHCPHostListOpts {
	own := neutronfind.CloneOptions(dhcpHostPolicy(v))
	v.Query, v.Headers, v.Microversion, v.order = own.Query, own.Headers, own.Microversion, own.Order
	if v.basePath != nil {
		path := *v.basePath
		v.basePath = &path
	}
	return v
}
func WithDHCPHostListOpts(v DHCPHostListOpts) DHCPHostListOption {
	own := cloneDHCPHostOpts(v)
	return func(c *DHCPHostListOpts) error { *c = cloneDHCPHostOpts(own); return nil }
}
func WithDHCPHostListQuery(key string, v any) DHCPHostListOption {
	raw, err := json.Marshal(v)
	own := bytes.Clone(raw)
	return func(c *DHCPHostListOpts) error {
		if err != nil {
			return err
		}
		if c.Query == nil {
			c.Query = map[string]json.RawMessage{}
		}
		c.Query[key] = bytes.Clone(own)
		if !slices.Contains(c.order, key) {
			c.order = append(c.order, key)
		}
		return nil
	}
}
func WithDHCPHostListHeaders(v map[string]string) DHCPHostListOption {
	own := maps.Clone(v)
	return func(c *DHCPHostListOpts) error { c.Headers = maps.Clone(own); return nil }
}
func WithDHCPHostListMicroversion(v string) DHCPHostListOption {
	return func(c *DHCPHostListOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithDHCPHostListMaxItems(v int) DHCPHostListOption { return WithDHCPHostListQuery("max_items", v) }
func WithDHCPHostListPaginated(v bool) DHCPHostListOption {
	return WithDHCPHostListQuery("paginated", v)
}
func WithDHCPHostListJMESPath(v string) DHCPHostListOption {
	return WithDHCPHostListQuery("jmespath_filters", v)
}
func dhcpHostSegment(v string) error {
	if strings.TrimSpace(v) == "" || v == "." || v == ".." || !utf8.ValidString(v) || strings.IndexFunc(v, unicode.IsControl) >= 0 {
		return dhcpNetworkInvalid("parent identity must be nonblank UTF-8 literal text without controls or dot segments")
	}
	return nil
}

var dhcpHostsProfile = neutronfind.Profile{
	Kind: "network_hosting_dhcp_agents", Path: "agents", Singular: "agent", Plural: "agents", IDKey: "id", NameKey: "name", PreserveUnknownBody: true,
	SourceFindPolicy: true, SourceStatusPolicy: true, SourceLinkPolicy: true, PythonLoadedEquality: true, StopFaultRetryCallbacks: true,
	Body: []neutronfind.Descriptor{
		{Name: "agent_type", Wire: "agent_type", Missing: "null"}, {Name: "availability_zone", Wire: "availability_zone", Missing: "null"}, {Name: "binary", Wire: "binary", Missing: "null"},
		{Name: "configuration", Wire: "configurations", Missing: "null"}, {Name: "created_at", Wire: "created_at", Missing: "null"}, {Name: "description", Wire: "description", Missing: "null"}, {Name: "last_heartbeat_at", Wire: "heartbeat_timestamp", Missing: "null"},
		{Name: "host", Wire: "host", Missing: "null"}, {Name: "is_admin_state_up", Wire: "admin_state_up", Kind: neutronfind.Boolean, Missing: "null"}, {Name: "is_alive", Wire: "alive", Kind: neutronfind.Boolean, Missing: "null"}, {Name: "resources_synced", Wire: "resources_synced", Kind: neutronfind.Boolean, Missing: "null"},
		{Name: "started_at", Wire: "started_at", Missing: "null"}, {Name: "topic", Wire: "topic", Missing: "null"}, {Name: "ha_state", Wire: "ha_state", Missing: "null"}, {Name: "ha_chassis_priority", Wire: "ha_chassis_priority", Kind: neutronfind.Integer, Missing: "null"}, {Name: "id", Wire: "id", Missing: "null"}, {Name: "name", Wire: "name", Missing: "null"},
	},
	Query: []neutronfind.QueryPair{{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}, {Name: "agent_type", Wire: "agent_type"}, {Name: "availability_zone", Wire: "availability_zone"}, {Name: "binary", Wire: "binary"}, {Name: "description", Wire: "description"}, {Name: "fields", Wire: "fields"}, {Name: "host", Wire: "host"}, {Name: "topic", Wire: "topic"}, {Name: "is_admin_state_up", Wire: "admin_state_up"}, {Name: "is_alive", Wire: "alive"}},
}

func (a *API) prepareDHCPHosts(ctx context.Context, networkID string, options []DHCPHostListOption) (*neutronfind.Engine, DHCPHostListOpts, json.RawMessage, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, DHCPHostListOpts{}, nil, err
	}
	if a == nil {
		return nil, DHCPHostListOpts{}, nil, dhcpNetworkInvalid("agents API is required")
	}
	if err := dhcpHostSegment(networkID); err != nil {
		return nil, DHCPHostListOpts{}, nil, err
	}
	selected := a.client
	profile := dhcpHostsProfile
	escaped := url.PathEscape(networkID)
	profile.Path = "networks/" + escaped + "/dhcp-agents"
	profile.VersionedPath = "/v2.0/" + profile.Path
	engine, err := neutronfind.New(ctx, selected, profile, func(context.Context) error {
		if a.client != selected {
			return dhcpNetworkInvalid("selected agents client changed")
		}
		return nil
	})
	if err != nil {
		return nil, DHCPHostListOpts{}, nil, err
	}
	policy := DHCPHostListOpts{}
	for _, apply := range slices.Clone(options) {
		if err := engine.Check(ctx); err != nil {
			return nil, policy, nil, err
		}
		if apply == nil {
			return nil, policy, nil, dhcpNetworkInvalid("nil DHCP host list option")
		}
		next := cloneDHCPHostOpts(policy)
		if err := errors.Join(apply(&next), engine.Check(ctx)); err != nil {
			return nil, policy, nil, err
		}
		policy = cloneDHCPHostOpts(next)
	}
	if _, present := policy.Query["network_id"]; present {
		return nil, policy, nil, dhcpNetworkInvalid("network_id duplicates the fixed parent argument")
	}
	normalized, expression, err := neutronfind.NormalizeProxyListOptions(dhcpHostPolicy(policy))
	if err != nil {
		return nil, policy, nil, err
	}
	if _, present := normalized.Query["network_id"]; present {
		return nil, policy, nil, dhcpNetworkInvalid("restored network_id cannot replace the selected parent")
	}
	if normalized.BasePath != nil {
		path := strings.ReplaceAll(*normalized.BasePath, "%(network_id)s", escaped)
		if strings.Contains(path, "%(") {
			return nil, policy, nil, dhcpNetworkInvalid("unsupported DHCP URI template")
		}
		normalized.BasePath = &path
	}
	// Normalize the safe selected route through the common list engine, keeping
	// formal base_path separate from ordinary query and response Body fields.
	policy.Query, policy.Headers, policy.Microversion, policy.order = normalized.Query, normalized.Headers, normalized.Microversion, normalized.Order
	policy.basePath = normalized.BasePath

	return engine, policy, expression, nil
}
func dhcpHostsListPolicy(p DHCPHostListOpts) neutronfind.Options {
	out := dhcpHostPolicy(p)
	out.BasePath = p.basePath
	return neutronfind.CloneOptions(out)
}

func dhcpHostsExpression(raw json.RawMessage) (*string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	truth, err := cloudfilter.PythonJSONTruthy(raw)
	if err != nil {
		return nil, err
	}
	if !truth || bytes.TrimSpace(raw)[0] != '"' {
		return nil, nil
	}
	var text string
	err = json.Unmarshal(raw, &text)
	return &text, err
}
func dhcpHostsWrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "network_hosting_dhcp_agents", cloudread.ContextError(ctx, err))
}
func (a *API) ListDHCPHostRecords(ctx context.Context, networkID string, options ...DHCPHostListOption) iter.Seq2[*DHCPHostRecord, error] {
	own := slices.Clone(options)
	return func(yield func(*DHCPHostRecord, error) bool) {
		e, p, raw, err := a.prepareDHCPHosts(ctx, networkID, own)
		if err == nil {
			var expr *string
			expr, err = dhcpHostsExpression(raw)
			if err == nil && expr != nil {
				err = dhcpNetworkInvalid("JMESPath changes the return type; use ProxyDHCPHostRecords")
			}
		}
		if err != nil {
			yield(nil, dhcpHostsWrap(ctx, "ListDHCPHostRecords", err))
			return
		}
		for row, err := range e.List(ctx, dhcpHostsListPolicy(p)) {
			if !yield(row, dhcpHostsWrap(ctx, "ListDHCPHostRecords", err)) {
				return
			}
		}
	}
}
func (a *API) AllDHCPHostRecords(ctx context.Context, networkID string, options ...DHCPHostListOption) ([]*DHCPHostRecord, error) {
	out := make([]*DHCPHostRecord, 0)
	for row, err := range a.ListDHCPHostRecords(ctx, networkID, options...) {
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, nil
}

type DHCPHostGenerator struct {
	Records iter.Seq2[*DHCPHostRecord, error]
}

func (g *DHCPHostGenerator) JMESPathIdentity() any { return g }
func (*DHCPHostGenerator) MarshalJSON() ([]byte, error) {
	return nil, dhcpNetworkInvalid("unconsumed DHCP host generator is not JSON")
}

type DHCPHostProxyResult struct {
	Value any
	JSON  json.RawMessage
}

func (a *API) ProxyDHCPHostRecords(ctx context.Context, networkID string, options ...DHCPHostListOption) (*DHCPHostProxyResult, error) {
	e, p, raw, err := a.prepareDHCPHosts(ctx, networkID, slices.Clone(options))
	if err != nil {
		return nil, dhcpHostsWrap(ctx, "ProxyDHCPHostRecords", err)
	}
	text, err := dhcpHostsExpression(raw)
	if err != nil {
		return nil, dhcpHostsWrap(ctx, "ProxyDHCPHostRecords", err)
	}
	generator := &DHCPHostGenerator{Records: e.List(ctx, dhcpHostsListPolicy(p))}
	var value any = generator
	if text != nil {
		value, err = jmespath.Search(*text, generator)
	}
	if err = errors.Join(err, e.Check(ctx)); err != nil {
		return nil, dhcpHostsWrap(ctx, "ProxyDHCPHostRecords", err)
	}
	result := &DHCPHostProxyResult{Value: value}
	if raw, cause := json.Marshal(value); cause == nil {
		result.JSON = bytes.Clone(raw)
	}
	return result, nil
}

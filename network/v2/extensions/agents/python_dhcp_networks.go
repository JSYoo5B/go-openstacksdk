package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"maps"
	"math/big"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	networks "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
)

// DHCPNetworkRecordsOpts carries dhcp_agent_hosting_networks(agent, **query).
// Query holds the Python keyword arguments as JSON values: Network query
// parameters (Python or wire names) go to the server, other Network Body
// attributes filter rows locally, and unknown keys are dropped like Python.
// MaxItems is Python max_items and SinglePage is paginated=False.
type DHCPNetworkRecordsOpts struct {
	Query      map[string]json.RawMessage
	MaxItems   int
	SinglePage bool
}
type DHCPNetworkRecordsOption = request.Option[DHCPNetworkRecordsOpts]

func WithDHCPNetworkRecordsOptions(value DHCPNetworkRecordsOpts) DHCPNetworkRecordsOption {
	value = cloneDHCPNetworkRecordsOpts(value)
	return func(c *request.Config[DHCPNetworkRecordsOpts]) error {
		c.Options = cloneDHCPNetworkRecordsOpts(value)
		return nil
	}
}

// WithDHCPNetworkRecordsQuery adds one Python keyword argument. The value is
// encoded with encoding/json when the option is built; a list becomes a
// repeated query key and null omits a server query key, as in requests.
func WithDHCPNetworkRecordsQuery(key string, value any) DHCPNetworkRecordsOption {
	encoded, err := json.Marshal(value)
	return func(c *request.Config[DHCPNetworkRecordsOpts]) error {
		if err != nil {
			return fmt.Errorf("%w: query %q: %v", resource.ErrInvalidOption, key, err)
		}
		if c.Options.Query == nil {
			c.Options.Query = make(map[string]json.RawMessage)
		}
		c.Options.Query[key] = bytes.Clone(encoded)
		return nil
	}
}
func WithDHCPNetworkRecordsMaxItems(value int) DHCPNetworkRecordsOption {
	return func(c *request.Config[DHCPNetworkRecordsOpts]) error { c.Options.MaxItems = value; return nil }
}

// WithDHCPNetworkRecordsPaginated(false) reads only the first page.
func WithDHCPNetworkRecordsPaginated(value bool) DHCPNetworkRecordsOption {
	return func(c *request.Config[DHCPNetworkRecordsOpts]) error { c.Options.SinglePage = !value; return nil }
}
func WithDHCPNetworkRecordsHeader(key, value string) DHCPNetworkRecordsOption {
	return request.WithHeader[DHCPNetworkRecordsOpts](key, value)
}

func cloneDHCPNetworkRecordsOpts(v DHCPNetworkRecordsOpts) DHCPNetworkRecordsOpts {
	if v.Query != nil {
		query := make(map[string]json.RawMessage, len(v.Query))
		for key, raw := range v.Query {
			query[key] = bytes.Clone(raw)
		}
		v.Query = query
	}
	return v
}

func ownDHCPNetworkRecordsConfig(c *request.Config[DHCPNetworkRecordsOpts]) {
	cloudread.OwnReadConfig(c)
	c.Options = cloneDHCPNetworkRecordsOpts(c.Options)
}

// DHCPNetworkRecord is one DHCPAgentHostingNetwork row. Network is the native
// typed decode, Fields keeps the row without Python's discarded "self" key,
// and AgentID is the URI attribute Python adds to every row.
type DHCPNetworkRecord struct {
	Network    *networks.Network          `json:"-"`
	Fields     map[string]json.RawMessage `json:"-"`
	AgentID    string                     `json:"-"`
	Header     http.Header                `json:"-"`
	StatusCode int                        `json:"-"`
	meta       resource.Metadata
}

// dhcpNetworkQuery is Network._query_mapping: Python name to wire name.
var dhcpNetworkQuery = map[string]string{
	"description": "description", "fields": "fields", "name": "name", "status": "status", "project_id": "project_id",
	"sort_key": "sort_key", "sort_dir": "sort_dir", "id": "id", "limit": "limit", "marker": "marker",
	"ipv4_address_scope_id": "ipv4_address_scope", "ipv6_address_scope_id": "ipv6_address_scope",
	"is_admin_state_up": "admin_state_up", "is_port_security_enabled": "port_security_enabled",
	"is_router_external": "router:external", "is_shared": "shared",
	"provider_network_type": "provider:network_type", "provider_physical_network": "provider:physical_network",
	"provider_segmentation_id": "provider:segmentation_id",
	"tags":                     "tags", "any_tags": "tags-any", "not_tags": "not-tags", "not_any_tags": "not-tags-any",
}

// dhcpNetworkLocal lists Network Body attributes outside the query mapping,
// which Resource.list compares locally: Python name to wire name.
var dhcpNetworkLocal = map[string]string{
	"availability_zone_hints": "availability_zone_hints", "availability_zones": "availability_zones",
	"created_at": "created_at", "dns_domain": "dns_domain", "is_default": "is_default", "mtu": "mtu",
	"pvlan": "pvlan", "qos_policy_id": "qos_policy_id", "segments": "segments", "subnet_ids": "subnets",
	"updated_at": "updated_at", "is_vlan_transparent": "vlan_transparent", "is_vlan_qinq": "vlan_qinq",
	"revision_number": "revision_number",
}

func dhcpNetworkInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

// dhcpNetworkPrepare splits the keyword arguments like Resource.list.
func dhcpNetworkPrepare(query map[string]json.RawMessage) (url.Values, map[string]json.RawMessage, error) {
	server := make(url.Values)
	filters := make(map[string]json.RawMessage)
	for key, raw := range query {
		if strings.TrimSpace(key) == "" || !utf8.ValidString(key) || strings.IndexFunc(key, unicode.IsControl) >= 0 {
			return nil, nil, dhcpNetworkInvalid("query key requires nonblank text")
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, dhcpNetworkInvalid("query %q requires complete UTF-8 JSON", key)
		}
		switch key {
		case "agent_id", "paginated", "max_items", "base_path", "jmespath_filters", "microversion", "headers", "allow_unknown_params":
			return nil, nil, dhcpNetworkInvalid("%q is a call control, not a query argument", key)
		}
		if wire, local := dhcpNetworkLocal[key]; local {
			filters[wire] = bytes.Clone(raw)
		}
	}
	for name, wire := range dhcpNetworkQuery {
		// _transpose prefers the Python name over the wire name.
		raw, exists := query[name]
		if !exists {
			raw, exists = query[wire]
		}
		if !exists {
			continue
		}
		values, err := cloudfilter.RequestQueryValues(raw)
		if err != nil {
			return nil, nil, err
		}
		if len(values) > 0 {
			server[wire] = values
		}
	}
	return server, filters, nil
}

// ListDHCPNetworkRecords implements dhcp_agent_hosting_networks. It sends GET
// agents/{id}/dhcp-networks with the Network query parameters, follows
// networks_links (or a Link header), continues with marker=<last id> while a
// limit is present until an empty page, applies Python's local Body filters
// after max_items counts every row, and accepts every status below 400 whose
// body is JSON.
func (a *API) ListDHCPNetworkRecords(ctx context.Context, agentID string, options ...DHCPNetworkRecordsOption) iter.Seq2[*DHCPNetworkRecord, error] {
	owned := slices.Clone(options)
	return func(yield func(*DHCPNetworkRecord, error) bool) {
		const op = "ListDHCPNetworkRecords"
		fail := func(err error) { yield(nil, request.Wrap(op, "agents", cloudread.ContextError(ctx, err))) }
		if err := cloudread.Context(ctx); err != nil {
			fail(err)
			return
		}
		if a == nil {
			fail(dhcpNetworkInvalid("agent API is required"))
			return
		}
		if strings.TrimSpace(agentID) == "" || agentID == "." || agentID == ".." {
			fail(dhcpNetworkInvalid("agent ID is required"))
			return
		}
		source, err := cloudread.Capture(ctx, a.client, "network")
		if err != nil {
			fail(err)
			return
		}
		config, err := cloudread.ApplyReadOptions(ctx, DHCPNetworkRecordsOpts{}, owned, ownDHCPNetworkRecordsConfig, source.Guard)
		if err == nil {
			err = request.ValidateCapabilities(config, false, false, true)
		}
		if err == nil && config.Options.MaxItems < 0 {
			err = dhcpNetworkInvalid("max_items must not be negative")
		}
		var query url.Values
		var filters map[string]json.RawMessage
		if err == nil {
			query, filters, err = dhcpNetworkPrepare(config.Options.Query)
		}
		if err == nil {
			err = source.WithPolicy(ctx, nil, config.Headers)
		}
		if err != nil {
			fail(err)
			return
		}
		codes := make([]int, 0, 200)
		for code := 200; code < 400; code++ {
			codes = append(codes, code)
		}
		escaped := url.PathEscape(agentID)
		spec := rest.CollectionSpec[DHCPNetworkRecord]{
			Client: &source.Client, Path: "agents/" + escaped + "/dhcp-networks", Kind: "agents", PluralKey: "networks",
			Metadata:    func(v *DHCPNetworkRecord) *resource.Metadata { return &v.meta },
			Validate:    source.Guard,
			SourceGuard: source.Guard,
			ListCodes:   codes,
			ValidateItem: func(v *DHCPNetworkRecord) error {
				v.Fields = maps.Clone(v.meta.Body)
				delete(v.Fields, "self")
				v.AgentID, v.Header, v.StatusCode = agentID, v.meta.Header.Clone(), v.meta.StatusCode
				typed := new(networks.Network)
				encoded, err := json.Marshal(v.Fields)
				if err == nil {
					err = json.Unmarshal(encoded, typed)
				}
				v.Network = typed
				return errors.Join(err, source.Guard(ctx))
			},
			Paging: rest.PagePolicy[DHCPNetworkRecord]{
				HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, NextTruthy: cloudfilter.PythonJSONTruthy,
				MarkerFallback: true, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true,
				MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true,
				VersionedPath: "/v2.0/agents/" + escaped + "/dhcp-networks",
				Marker: func(v *DHCPNetworkRecord) (string, error) {
					values, err := cloudfilter.RequestQueryValues(v.meta.Body["id"])
					if err != nil || len(values) != 1 || strings.TrimSpace(values[0]) == "" {
						return "", errors.Join(dhcpNetworkInvalid("pagination marker requires the row id"), err)
					}
					return values[0], nil
				},
			},
		}
		control := rest.ListControl{MaxItems: config.Options.MaxItems, SinglePage: config.Options.SinglePage, LimitHint: true}
		for value, err := range rest.ListWithControl(ctx, spec, query, control) {
			if err != nil {
				fail(err)
				return
			}
			matched := true
			for wire, filter := range filters {
				ok, err := dhcpNetworkMatch(value.Fields[wire], filter)
				if err != nil {
					fail(err)
					return
				}
				if !ok {
					matched = false
					break
				}
			}
			if matched && !yield(value, nil) {
				return
			}
		}
	}
}

// dhcpNetworkMatch is Resource.list's client filter: a dictionary filter
// matches a mapping containing its keys recursively, anything else uses
// Python equality, where a missing attribute is None.
func dhcpNetworkMatch(actual, filter json.RawMessage) (bool, error) {
	if len(actual) == 0 {
		actual = json.RawMessage("null")
	}
	want, err := dhcpNetworkDecode(filter)
	if err != nil {
		return false, err
	}
	got, err := dhcpNetworkDecode(actual)
	if err != nil {
		return false, err
	}
	if wantMap, ok := want.(map[string]any); ok {
		return dhcpNetworkDictFilter(wantMap, got)
	}
	return dhcpNetworkEqual(got, want), nil
}

func dhcpNetworkDecode(raw json.RawMessage) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

func dhcpNetworkDictFilter(filter map[string]any, actual any) (bool, error) {
	if actual == nil || actual == false {
		return false, nil
	}
	mapping, ok := actual.(map[string]any)
	if !ok {
		if text, isText := actual.(string); isText && text == "" {
			return false, nil
		}
		if items, isList := actual.([]any); isList && len(items) == 0 {
			return false, nil
		}
		return false, dhcpNetworkInvalid("dictionary filter requires a mapping attribute")
	}
	if len(mapping) == 0 {
		return false, nil
	}
	for key, want := range filter {
		if nested, ok := want.(map[string]any); ok {
			matched, err := dhcpNetworkDictFilter(nested, mapping[key])
			if err != nil || !matched {
				return false, err
			}
			continue
		}
		if !dhcpNetworkEqual(mapping[key], want) {
			return false, nil
		}
	}
	return true, nil
}

// dhcpNetworkEqual follows Python == for JSON values, including True == 1.
func dhcpNetworkEqual(a, b any) bool {
	if x, ok := dhcpNetworkNumber(a); ok {
		y, ok := dhcpNetworkNumber(b)
		return ok && x.Cmp(y) == 0
	}
	switch x := a.(type) {
	case nil:
		return b == nil
	case string:
		y, ok := b.(string)
		return ok && x == y
	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for i := range x {
			if !dhcpNetworkEqual(x[i], y[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		y, ok := b.(map[string]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for key, value := range x {
			other, exists := y[key]
			if !exists || !dhcpNetworkEqual(value, other) {
				return false
			}
		}
		return true
	}
	return false
}

func dhcpNetworkNumber(value any) (*big.Rat, bool) {
	switch v := value.(type) {
	case bool:
		if v {
			return big.NewRat(1, 1), true
		}
		return new(big.Rat), true
	case json.Number:
		number, ok := new(big.Rat).SetString(v.String())
		return number, ok
	}
	return nil, false
}

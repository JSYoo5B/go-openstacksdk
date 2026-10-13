package network

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

var ErrUnavailableSecurityGroups = errors.New("OpenStack security groups feature unavailable")

// SecurityGroupReadDependencies is a concrete SDK-owned backend policy. Source
// nil defaults to neutron; explicit empty/unrecognized sources disable the
// feature. NetworkAvailable nil derives from the supplied Neutron facade.
// Neutron absence selects the legacy Nova backend only when Nova is configured.
type SecurityGroupReadDependencies struct {
	Neutron          *Service
	Nova             *gophercloud.ServiceClient
	Source           *string
	NetworkAvailable *bool
	UseDirectGet     bool
	Strict           bool
	CurrentLocation  *resource.CloudLocation
}
type SecurityGroupCloud struct{ dependencies SecurityGroupReadDependencies }

func NewSecurityGroupCloud(d SecurityGroupReadDependencies) *SecurityGroupCloud {
	if d.Source != nil {
		x := *d.Source
		d.Source = &x
	}
	if d.NetworkAvailable != nil {
		x := *d.NetworkAvailable
		d.NetworkAvailable = &x
	}
	if d.CurrentLocation != nil {
		x := d.CurrentLocation.Clone()
		d.CurrentLocation = &x
	}
	return &SecurityGroupCloud{dependencies: d}
}

type SecurityGroupReadOpts struct {
	Filters         *json.RawMessage
	Headers         map[string]string
	Microversion    *string
	CurrentLocation *resource.CloudLocation
}
type SecurityGroupReadOption func(*SecurityGroupReadOpts) error

func copySecurityGroupRead(c SecurityGroupReadOpts) SecurityGroupReadOpts {
	if c.Filters != nil {
		x := json.RawMessage(bytes.Clone(*c.Filters))
		c.Filters = &x
	}
	c.Headers = maps.Clone(c.Headers)
	if c.Headers == nil {
		c.Headers = map[string]string{}
	}
	if c.Microversion != nil {
		x := *c.Microversion
		c.Microversion = &x
	}
	if c.CurrentLocation != nil {
		x := c.CurrentLocation.Clone()
		c.CurrentLocation = &x
	}
	return c
}
func WithSecurityGroupReadOpts(c SecurityGroupReadOpts) SecurityGroupReadOption {
	owned := copySecurityGroupRead(c)
	return func(c *SecurityGroupReadOpts) error { *c = copySecurityGroupRead(owned); return nil }
}
func WithSecurityGroupReadFiltersJSON(raw json.RawMessage) SecurityGroupReadOption {
	owned := bytes.Clone(raw)
	return func(c *SecurityGroupReadOpts) error {
		c.Filters = nil
		if owned != nil {
			x := json.RawMessage(bytes.Clone(owned))
			c.Filters = &x
		}
		return nil
	}
}
func WithSecurityGroupReadExpression(v string) SecurityGroupReadOption {
	raw, _ := json.Marshal(v)
	return WithSecurityGroupReadFiltersJSON(raw)
}
func WithSecurityGroupReadHeaders(v map[string]string) SecurityGroupReadOption {
	owned := maps.Clone(v)
	return func(c *SecurityGroupReadOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithSecurityGroupReadMicroversion(v string) SecurityGroupReadOption {
	return func(c *SecurityGroupReadOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithSecurityGroupReadCurrentLocation(v resource.CloudLocation) SecurityGroupReadOption {
	owned := v.Clone()
	return func(c *SecurityGroupReadOpts) error { x := owned.Clone(); c.CurrentLocation = &x; return nil }
}

// Groups/Group contain only actual HTTP or explicitly supplied existing records.
// Value is arbitrary selected JSON, including scalar/null expression results.
// Inventory/LastGroup and NovaResponse retain actual response evidence on errors.
type SecurityGroupReadResponse struct {
	Body       json.RawMessage
	Header     http.Header
	StatusCode int
}

type SecurityGroupReadResult struct {
	Backend      string
	Groups       []*groups.SecurityGroupRecord
	Group        *groups.SecurityGroupRecord
	Inventory    []*groups.SecurityGroupRecord
	LastGroup    *groups.SecurityGroupRecord
	NovaResponse *SecurityGroupReadResponse
	Values       []any
	Value        json.RawMessage
	Expression   bool
}

func securityGroupReadRecord(v *groups.SecurityGroupRecord) *groups.SecurityGroupRecord {
	if v == nil {
		return nil
	}
	c := *v
	c.Resource = v.Resource.Clone()
	c.Wire = v.Wire.Clone()
	c.Envelope = bytes.Clone(v.Envelope)
	c.Header = v.Header.Clone()
	return &c
}
func securityGroupReadProof(v *groups.SecurityGroupRecord, e error) error {
	if v == nil {
		return e
	}
	return (&rest.Response{Body: bytes.Clone(v.Envelope), Header: v.Header.Clone(), StatusCode: v.StatusCode}).Fail(e)
}
func securityGroupReadWrap(ctx context.Context, op string, e error) error {
	return request.Wrap(op, "security groups", cloudread.ContextError(ctx, e))
}

type securityGroupReadState struct {
	source  *cloudread.Source
	api     *groups.API
	guard   func(context.Context) error
	backend string
	cfg     SecurityGroupReadOpts
	strict  bool
}

func (c *SecurityGroupCloud) start(ctx context.Context, options []SecurityGroupReadOption) (*securityGroupReadState, error) {
	if e := cloudread.Context(ctx); e != nil {
		return nil, e
	}
	if c == nil {
		return nil, fmt.Errorf("%w: security group cloud required", resource.ErrInvalidOption)
	}
	d := c.dependencies
	sourceName := "neutron"
	if d.Source != nil {
		sourceName = *d.Source
	}
	if strings.ToLower(sourceName) != "neutron" && strings.ToLower(sourceName) != "nova" {
		return nil, ErrUnavailableSecurityGroups
	}
	available := d.Neutron != nil
	if d.NetworkAvailable != nil {
		available = *d.NetworkAvailable
	}
	state := &securityGroupReadState{backend: "nova", strict: d.Strict}
	var selected *gophercloud.ServiceClient
	if available && sourceName == "neutron" {
		state.backend = "neutron"
		if d.Neutron == nil || d.Neutron.API == nil || d.Neutron.client == nil || d.Neutron.API.RawClient() != d.Neutron.client {
			return nil, fmt.Errorf("%w: selected Neutron service required", resource.ErrInvalidOption)
		}
		selected = d.Neutron.client
		originalAPI := d.Neutron.API
		state.guard = func(ctx context.Context) error {
			if d.Neutron.client != selected || d.Neutron.API != originalAPI || originalAPI.RawClient() != selected {
				return fmt.Errorf("%w: selected Neutron service/API changed", resource.ErrInvalidOption)
			}
			return state.source.Guard(ctx)
		}
	} else {
		selected = d.Nova
		state.guard = func(ctx context.Context) error { return state.source.Guard(ctx) }
	}
	role := "compute"
	if state.backend == "neutron" {
		role = "network"
	}
	source, e := cloudread.Capture(ctx, selected, role)
	if e != nil {
		return nil, e
	}
	state.source = source
	cfg := copySecurityGroupRead(SecurityGroupReadOpts{CurrentLocation: d.CurrentLocation})
	for _, apply := range slices.Clone(options) {
		if e := errors.Join(state.guard(ctx), cloudread.Context(ctx)); e != nil {
			return nil, e
		}
		if apply == nil {
			return nil, fmt.Errorf("%w: nil security group read option", resource.ErrInvalidOption)
		}
		next := copySecurityGroupRead(cfg)
		if e := errors.Join(apply(&next), state.guard(ctx), cloudread.Context(ctx)); e != nil {
			return nil, e
		}
		cfg = copySecurityGroupRead(next)
	}
	if cfg.CurrentLocation == nil && d.CurrentLocation != nil {
		x := d.CurrentLocation.Clone()
		cfg.CurrentLocation = &x
	}
	state.cfg = cfg
	if state.backend == "neutron" {
		state.api = groups.New(&state.source.Client)
	}
	return state, errors.Join(state.guard(ctx), cloudread.Context(ctx))
}
func (s *Service) SearchSecurityGroups(ctx context.Context, nameOrID string, options ...SecurityGroupReadOption) (*SecurityGroupReadResult, error) {
	return NewSecurityGroupCloud(SecurityGroupReadDependencies{Neutron: s}).SearchSecurityGroups(ctx, nameOrID, options...)
}
func (s *Service) GetSecurityGroup(ctx context.Context, reference any, options ...SecurityGroupReadOption) (*SecurityGroupReadResult, error) {
	return NewSecurityGroupCloud(SecurityGroupReadDependencies{Neutron: s}).GetSecurityGroup(ctx, reference, options...)
}
func securityGroupReadIterable(value any) ([]any, error) {
	switch v := value.(type) {
	case []any:
		return slices.Clone(v), nil
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		result := make([]any, len(keys))
		for i, k := range keys {
			result[i] = k
		}
		return result, nil
	case string:
		result := make([]any, 0)
		for _, r := range v {
			result = append(result, string(r))
		}
		return result, nil
	default:
		return nil, fmt.Errorf("Source list() cannot iterate %T", value)
	}
}
func (s *securityGroupReadState) bindLocation(record *groups.SecurityGroupRecord) error {
	if record == nil || record.Resource == nil {
		return fmt.Errorf("%w: missing SecurityGroup Resource", resource.ErrInvalidOption)
	}
	if s.cfg.CurrentLocation != nil {
		raw, e := cloudNetworkReadLocation(record.Resource.Body, s.cfg.CurrentLocation)
		if e != nil {
			return e
		}
		record.Resource.Body["location"] = raw
	}
	return nil
}
func (s *securityGroupReadState) bindGenerators(values []any, ctx context.Context) {
	visited := map[*groups.RecordListGenerator]bool{}
	var bind func(any)
	bind = func(value any) {
		switch v := value.(type) {
		case *groups.RecordListGenerator:
			if visited[v] {
				return
			}
			visited[v] = true
			seq := v.Records
			v.Records = func(yield func(*groups.SecurityGroupRecord, error) bool) {
				for row, e := range seq {
					if e == nil {
						e = s.bindLocation(row)
					}
					e = errors.Join(e, s.guard(ctx), cloudread.Context(ctx))
					if !yield(row, e) || e != nil {
						return
					}
				}
			}
		case []any:
			for _, v := range v {
				bind(v)
			}
		case map[string]any:
			for _, v := range v {
				bind(v)
			}
		}
	}
	for _, v := range values {
		bind(v)
	}
}
func (s *securityGroupReadState) list(ctx context.Context, result *SecurityGroupReadResult) error {
	ctx = rest.WithOperationGuard(ctx, s.guard)
	var pushdown []cloudfilter.JSONMember
	if s.cfg.Filters != nil {
		raw := bytes.TrimSpace(*s.cfg.Filters)
		if len(raw) == 0 || !json.Valid(raw) {
			return fmt.Errorf("%w: complete filter JSON required", resource.ErrInvalidOption)
		}
		if raw[0] == '{' {
			var e error
			pushdown, e = cloudfilter.ObjectMembers(raw)
			if e != nil {
				return e
			}
		}
	}
	if s.backend == "nova" {
		return s.nova(ctx, result, pushdown, "")
	}
	leaf := make([]groups.RecordListOption, 0)
	for _, m := range pushdown {
		leaf = append(leaf, groups.WithRecordProxyQuery(m.Key, json.RawMessage(bytes.Clone(m.Value))))
	}
	leaf = append(leaf, groups.WithRecordListHeaders(s.cfg.Headers))
	if s.cfg.Microversion != nil {
		leaf = append(leaf, groups.WithRecordListMicroversion(*s.cfg.Microversion))
	}
	proxy, e := s.api.ProxyListRecords(ctx, leaf...)
	if e != nil {
		return e
	}
	result.Values = make([]any, 0)
	if g, ok := proxy.Value.(*groups.RecordListGenerator); ok {
		for row, e := range g.Records {
			if row != nil {
				result.LastGroup = securityGroupReadRecord(row)
			}
			if e = errors.Join(e, s.guard(ctx), cloudread.Context(ctx)); e != nil {
				return e
			}
			if e := s.bindLocation(row); e != nil {
				return securityGroupReadProof(row, e)
			}
			result.Inventory = append(result.Inventory, securityGroupReadRecord(row))
			result.Values = append(result.Values, securityGroupReadRecord(row))
		}
	} else {
		result.Values, e = securityGroupReadIterable(proxy.Value)
		if e != nil {
			return e
		}
		s.bindGenerators(result.Values, ctx)
	}
	return errors.Join(s.guard(ctx), cloudread.Context(ctx))
}
func (c *SecurityGroupCloud) SearchSecurityGroups(ctx context.Context, nameOrID string, options ...SecurityGroupReadOption) (*SecurityGroupReadResult, error) {
	const op = "SearchSecurityGroups"
	state, e := c.start(ctx, options)
	if e != nil {
		return nil, securityGroupReadWrap(ctx, op, e)
	}
	result := &SecurityGroupReadResult{Backend: state.backend, Inventory: make([]*groups.SecurityGroupRecord, 0)}
	fail := func(e error) (*SecurityGroupReadResult, error) {
		result.Value = nil
		result.Values = nil
		result.Groups = nil
		return result, securityGroupReadWrap(ctx, op, e)
	}
	if e := state.list(ctx, result); e != nil {
		return fail(e)
	}
	values := make([]any, 0, len(result.Values))
	filterTruthy := false
	if state.cfg.Filters != nil {
		filterTruthy, e = cloudfilter.PythonJSONTruthy(*state.cfg.Filters)
		if e != nil {
			return fail(e)
		}
	}
	for _, value := range result.Values {
		if nameOrID == "" && filterTruthy {
			if g, ok := value.(*groups.RecordListGenerator); ok {
				for row, e := range g.Records {
					if row != nil {
						result.LastGroup = securityGroupReadRecord(row)
					}
					if e = errors.Join(e, state.guard(ctx), cloudread.Context(ctx)); e != nil {
						return fail(e)
					}
					result.Inventory = append(result.Inventory, securityGroupReadRecord(row))
				}
				return fail(securityGroupReadProof(result.LastGroup, fmt.Errorf("Source filter key is not an attribute of an opaque generator")))
			}
			if _, actual := value.(*groups.SecurityGroupRecord); !actual {
				raw, e := json.Marshal(value)
				if e != nil {
					return fail(e)
				}
				truthy, e := cloudfilter.PythonJSONTruthy(raw)
				if e != nil {
					return fail(e)
				}
				if !truthy {
					continue
				}
			}
		}
		values = append(values, value)
	}
	rows := make([]json.RawMessage, len(values))
	for i, value := range values {
		if row, ok := value.(*groups.SecurityGroupRecord); ok {
			rows[i], e = json.Marshal(row.Resource.Body)
		} else {
			rows[i], e = json.Marshal(value)
		}
		if e != nil {
			return fail(securityGroupReadProof(result.LastGroup, e))
		}
	}
	selected, e := cloudfilter.SelectLoaded(rows, nameOrID, state.cfg.Filters, func() error { return errors.Join(state.guard(ctx), cloudread.Context(ctx)) })
	if e != nil {
		return fail(securityGroupReadProof(result.LastGroup, e))
	}
	result.Value = bytes.Clone(selected.Value)
	result.Expression = selected.Expression
	if !selected.Expression {
		result.Values = make([]any, len(selected.Indices))
		result.Groups = make([]*groups.SecurityGroupRecord, 0)
		for i, index := range selected.Indices {
			value := values[index]
			result.Values[i] = value
			if record, ok := value.(*groups.SecurityGroupRecord); ok {
				result.Groups = append(result.Groups, securityGroupReadRecord(record))
			}
		}
	} else {
		result.Values = nil
	}
	if e := errors.Join(state.guard(ctx), cloudread.Context(ctx)); e != nil {
		return fail(e)
	}
	return result, nil
}
func securityGroupUUIDLike(value string) bool {
	text := strings.ReplaceAll(strings.ReplaceAll(value, "urn:", ""), "uuid:", "")
	text = strings.Trim(text, "{}")
	text = strings.ReplaceAll(text, "-", "")
	if len(text) != 32 {
		return false
	}
	_, e := hex.DecodeString(text)
	return e == nil
}

// GetSecurityGroup accepts a string or finite JSON reference. A JSON object
// containing id, RawResource or existing SecurityGroupRecord is returned without
// discovery/I/O, matching Source _get_entity's already-fetched-object bypass.
func (c *SecurityGroupCloud) GetSecurityGroup(ctx context.Context, reference any, options ...SecurityGroupReadOption) (*SecurityGroupReadResult, error) {
	const op = "GetSecurityGroup"
	wrap := func(e error) error { return securityGroupReadWrap(ctx, op, e) }
	if e := cloudread.Context(ctx); e != nil {
		return nil, wrap(e)
	}
	if existing, handled, e := ExistingSecurityGroupRead(reference); e != nil {
		return nil, wrap(e)
	} else if handled {
		return existing, nil
	}
	if record, ok := reference.(*resource.RawResource); ok && record != nil {
		reference = record.Body
	}
	var identity string
	if text, ok := reference.(string); ok {
		identity = text
	} else {
		var raw json.RawMessage
		var e error
		if x, ok := reference.(json.RawMessage); ok {
			raw = bytes.Clone(x)
		} else {
			raw, e = json.Marshal(reference)
		}
		if e != nil {
			return nil, wrap(e)
		}

		truthy, e := cloudfilter.PythonJSONTruthy(raw)
		if e != nil {
			return nil, wrap(e)
		}
		if truthy {
			identity, e = cloudfilter.PythonString(raw)
			if e != nil {
				return nil, wrap(e)
			}
		}
	}
	if c != nil && c.dependencies.UseDirectGet && securityGroupUUIDLike(identity) {
		state, e := c.start(ctx, options)
		if e != nil {
			return nil, wrap(e)
		}
		result := &SecurityGroupReadResult{Backend: state.backend, Inventory: make([]*groups.SecurityGroupRecord, 0)}
		ctx = rest.WithOperationGuard(ctx, state.guard)
		if state.backend == "nova" {
			e = state.nova(ctx, result, nil, identity)
			if e == nil && len(result.Inventory) > 0 {
				result.Group = securityGroupReadRecord(result.Inventory[0])
				result.Value, _ = json.Marshal(result.Group.Resource.Body)
			}
		} else {
			leaf := []groups.RecordListOption{groups.WithRecordListHeaders(state.cfg.Headers)}
			if state.cfg.Microversion != nil {
				leaf = append(leaf, groups.WithRecordListMicroversion(*state.cfg.Microversion))
			}
			result.LastGroup, e = state.api.GetSourceRecord(ctx, identity, leaf...)
			if e == nil {
				e = state.bindLocation(result.LastGroup)
			}
			if e == nil {
				result.Group = securityGroupReadRecord(result.LastGroup)
				result.Inventory = append(result.Inventory, securityGroupReadRecord(result.LastGroup))
				result.Value, _ = json.Marshal(result.Group.Resource.Body)
			}
		}
		if e = errors.Join(e, state.guard(ctx), cloudread.Context(ctx)); e != nil {
			result.Group = nil
			result.Value = nil
			return result, wrap(e)
		}
		return result, nil
	}
	result, e := c.SearchSecurityGroups(ctx, identity, options...)
	if e != nil {
		return result, wrap(e)
	}
	truthy, e := cloudfilter.PythonJSONTruthy(result.Value)
	if e != nil {
		return result, wrap(e)
	}
	if !truthy {
		result.Value = nil
		result.Groups = nil
		result.Values = nil
		return result, nil
	}
	first, e := cloudfilter.First(result.Value)
	if e != nil {
		result.Value = nil
		result.Groups = nil
		result.Values = nil
		var multiple *cloudfilter.MultipleError
		if errors.As(e, &multiple) {
			e = errors.Join(resource.ErrAmbiguous, e)
		}
		return result, wrap(securityGroupReadProof(result.LastGroup, e))
	}
	result.Value = bytes.Clone(first)
	if !result.Expression && len(result.Groups) == 1 {
		result.Group = securityGroupReadRecord(result.Groups[0])
	}
	result.Groups = nil
	result.Values = nil
	return result, nil
}
func (s *securityGroupReadState) nova(ctx context.Context, result *SecurityGroupReadResult, filters []cloudfilter.JSONMember, id string) error {
	if e := s.source.WithPolicy(ctx, s.cfg.Microversion, s.cfg.Headers); e != nil {
		return e
	}
	target := s.source.Client.ServiceURL("os-security-groups")
	if id != "" {
		target = s.source.Client.ServiceURL("os-security-groups", url.PathEscape(id))
	}
	q := url.Values{}
	for _, m := range filters {
		values, e := cloudfilter.RequestQueryValues(m.Value)
		if e != nil {
			return e
		}
		if len(values) > 0 {
			q[m.Key] = values
		}
	}
	if len(q) > 0 {
		target += "?" + q.Encode()
	}
	codes := make([]int, 300)
	for i := range codes {
		codes[i] = 100 + i
	}
	rejections := make([]int, 200)
	for i := range rejections {
		rejections[i] = 400 + i
	}
	response, e := rest.DoJSONGuardedRejectionsHeaders(ctx, &s.source.Client, s.guard, http.MethodGet, target, nil, s.source.Client.MoreHeaders, rest.RejectionPolicy{Codes: rejections, PreserveCleanRetry: true, StopFaultRetryCallbacks: true}, codes...)
	if response != nil {
		result.NovaResponse = &SecurityGroupReadResponse{Body: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	}
	if e != nil {
		return e
	}
	var envelope map[string]json.RawMessage
	if e := json.Unmarshal(response.Body, &envelope); e != nil || envelope == nil {
		return response.Fail(errors.Join(fmt.Errorf("Source Nova response must be a mapping"), e))
	}
	var rows []json.RawMessage
	if id != "" {
		raw, ok := envelope["security_group"]
		if !ok {
			return response.Fail(fmt.Errorf("missing security_group"))
		}
		rows = []json.RawMessage{raw}
	} else {
		raw, ok := envelope["security_groups"]
		if !ok {
			rows = []json.RawMessage{}
		} else {
			var e error
			rows, e = securityGroupNovaRows(raw)
			if e != nil {
				return response.Fail(e)
			}
		}
	}
	result.Values = make([]any, 0, len(rows))
	for _, raw := range rows {
		if e := errors.Join(s.guard(ctx), cloudread.Context(ctx)); e != nil {
			return response.Fail(e)
		}
		wire := &resource.RawResource{}
		if e := json.Unmarshal(raw, wire); e != nil {
			return response.Fail(e)
		}
		wire.Header = response.Header.Clone()
		wire.StatusCode = response.StatusCode
		record := &groups.SecurityGroupRecord{Wire: wire, Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
		result.LastGroup = securityGroupReadRecord(record)
		body, e := s.normalizeNovaGroup(wire.Body)
		if e != nil {
			return response.Fail(e)
		}
		record.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: body, Header: response.Header.Clone(), StatusCode: response.StatusCode}}
		result.LastGroup = securityGroupReadRecord(record)
		result.Inventory = append(result.Inventory, securityGroupReadRecord(record))
		result.Values = append(result.Values, securityGroupReadRecord(record))
	}
	return errors.Join(s.guard(ctx), cloudread.Context(ctx))
}
func securityGroupPop(fields map[string]json.RawMessage, key string, fallback json.RawMessage) json.RawMessage {
	value, ok := fields[key]
	delete(fields, key)
	if ok {
		return bytes.Clone(value)
	}
	return bytes.Clone(fallback)
}
func securityGroupFields(fields map[string]json.RawMessage) map[string]json.RawMessage {
	result := make(map[string]json.RawMessage, len(fields))
	for k, v := range fields {
		result[k] = bytes.Clone(v)
	}
	return result
}
func securityGroupRequired(fields map[string]json.RawMessage, key string) (json.RawMessage, error) {
	raw, ok := fields[key]
	if !ok {
		return nil, fmt.Errorf("Source Nova security group missing %s", key)
	}
	delete(fields, key)
	return bytes.Clone(raw), nil
}
func (s *securityGroupReadState) novaLocation(project json.RawMessage) (json.RawMessage, error) {
	location := s.cfg.CurrentLocation
	if location == nil {
		location = &resource.CloudLocation{}
	}
	return cloudNetworkReadLocation(map[string]json.RawMessage{"project_id": project}, location)
}
func (s *securityGroupReadState) normalizeNovaGroup(original map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	fields := securityGroupFields(original)
	for _, key := range []string{"links", "NAME_ATTR", "HUMAN_ID", "human_id", "request_ids", "x_openstack_request_ids"} {
		delete(fields, key)
	}
	rulesRaw := securityGroupPop(fields, "security_group_rules", securityGroupPop(fields, "rules", json.RawMessage("[]")))
	rawRules, e := securityGroupNovaRows(rulesRaw)
	if e != nil {
		return nil, e
	}
	normalized := make([]map[string]json.RawMessage, 0, len(rawRules))
	for _, raw := range rawRules {
		var rule map[string]json.RawMessage
		if e := json.Unmarshal(raw, &rule); e != nil || rule == nil {
			return nil, fmt.Errorf("Source Nova rule must be a mapping")
		}
		value, e := s.normalizeNovaRule(rule)
		if e != nil {
			return nil, e
		}
		normalized = append(normalized, value)
	}
	project := securityGroupPop(fields, "project_id", securityGroupPop(fields, "tenant_id", json.RawMessage(`""`)))
	location, e := s.novaLocation(project)
	if e != nil {
		return nil, e
	}
	result := map[string]json.RawMessage{"location": location}
	for _, key := range []string{"id", "name", "description"} {
		raw, e := securityGroupRequired(fields, key)
		if e != nil {
			return nil, e
		}
		result[key] = raw
	}
	result["security_group_rules"], _ = json.Marshal(normalized)
	result["properties"], _ = json.Marshal(fields)
	if !s.strict {
		result["project_id"] = bytes.Clone(project)
		result["tenant_id"] = bytes.Clone(project)
		for k, v := range fields {
			if _, present := result[k]; !present {
				result[k] = bytes.Clone(v)
			}
		}
	}
	return result, nil
}
func securityGroupNovaInt(raw json.RawMessage) (json.RawMessage, error) {
	trim := bytes.TrimSpace(raw)
	if bytes.Equal(trim, []byte("true")) {
		return json.RawMessage("1"), nil
	}
	if bytes.Equal(trim, []byte("false")) {
		return json.RawMessage("0"), nil
	}
	if len(trim) > 0 && trim[0] == '"' {
		var value string
		if e := json.Unmarshal(raw, &value); e != nil {
			return nil, e
		}
		integer, e := jsonfilter.PythonIntegerString(value)
		if e != nil {
			return nil, e
		}
		return json.RawMessage(integer.String()), nil
	}
	if len(trim) == 0 || trim[0] == '{' || trim[0] == '[' || bytes.Equal(trim, []byte("null")) {
		return nil, fmt.Errorf("Source int() cannot convert Nova rule minimum")
	}
	return jsonfilter.DescriptorIntegerJSON(raw)
}
func (s *securityGroupReadState) normalizeNovaRule(original map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	fields := securityGroupFields(original)
	result := map[string]json.RawMessage{}
	id, e := securityGroupRequired(fields, "id")
	if e != nil {
		return nil, e
	}
	result["id"] = id
	result["direction"] = securityGroupPop(fields, "direction", json.RawMessage(`"ingress"`))
	result["ethertype"] = securityGroupPop(fields, "ethertype", json.RawMessage(`"IPv4"`))
	minimum := securityGroupPop(fields, "from_port", json.RawMessage("null"))
	if raw, ok := fields["port_range_min"]; ok {
		minimum = bytes.Clone(raw)
	}
	equal, e := jsonfilter.EqualPythonLoadedJSON(minimum, json.RawMessage("-1"))
	if e != nil {
		return nil, e
	}
	if equal {
		minimum = json.RawMessage("null")
	}
	if !bytes.Equal(bytes.TrimSpace(minimum), []byte("null")) {
		minimum, e = securityGroupNovaInt(minimum)
		if e != nil {
			return nil, e
		}
	}
	result["port_range_min"] = minimum
	maximum := securityGroupPop(fields, "port_range_max", securityGroupPop(fields, "to_port", json.RawMessage("null")))
	equal, e = jsonfilter.EqualPythonLoadedJSON(maximum, json.RawMessage("-1"))
	if e != nil {
		return nil, e
	}
	if equal {
		maximum = json.RawMessage("null")
	}
	result["port_range_max"] = maximum
	result["protocol"] = securityGroupPop(fields, "protocol", securityGroupPop(fields, "ip_protocol", json.RawMessage("null")))
	ipRange := securityGroupPop(fields, "ip_range", json.RawMessage("{}"))
	var ip map[string]json.RawMessage
	if e := json.Unmarshal(ipRange, &ip); e != nil || ip == nil {
		return nil, fmt.Errorf("Source Nova ip_range must provide get()")
	}
	cidr := ip["cidr"]
	if cidr == nil {
		cidr = json.RawMessage("null")
	}
	result["remote_ip_prefix"] = securityGroupPop(fields, "remote_ip_prefix", cidr)
	result["security_group_id"] = securityGroupPop(fields, "security_group_id", securityGroupPop(fields, "parent_group_id", json.RawMessage("null")))
	result["remote_group_id"] = securityGroupPop(fields, "remote_group_id", json.RawMessage("null"))
	project := securityGroupPop(fields, "project_id", securityGroupPop(fields, "tenant_id", json.RawMessage(`""`)))
	result["location"], e = s.novaLocation(project)
	if e != nil {
		return nil, e
	}
	result["properties"], _ = json.Marshal(fields)
	if !s.strict {
		result["project_id"] = bytes.Clone(project)
		result["tenant_id"] = bytes.Clone(project)
		for k, v := range fields {
			if _, present := result[k]; !present {
				result[k] = bytes.Clone(v)
			}
		}
	}
	return result, nil
}

func securityGroupNovaRows(raw json.RawMessage) ([]json.RawMessage, error) {
	token := bytes.TrimSpace(raw)
	if len(token) == 0 {
		return nil, fmt.Errorf("Source Nova sequence must be JSON")
	}
	if token[0] == '[' {
		var values []json.RawMessage
		e := json.Unmarshal(raw, &values)
		return values, e
	}
	if token[0] == '{' || token[0] == '"' {
		truthy, e := cloudfilter.PythonJSONTruthy(raw)
		if e != nil {
			return nil, e
		}
		if !truthy {
			return []json.RawMessage{}, nil
		}
		return nil, fmt.Errorf("Source Nova sequence iterates non-mapping strings")
	}
	return nil, fmt.Errorf("Source Nova value is not iterable")
}

// ExistingSecurityGroupRead implements Source _get_entity's existing-object
// shortcut without service discovery, backend selection or option callbacks.
// Falsey id values still count as present. It owns every exported carrier.
func ExistingSecurityGroupRead(reference any) (*SecurityGroupReadResult, bool, error) {
	if record, ok := reference.(*groups.SecurityGroupRecord); ok && record != nil && record.Resource != nil {
		raw, e := json.Marshal(record.Resource.Body)
		if e != nil {
			return nil, true, e
		}
		return &SecurityGroupReadResult{Backend: "existing", Group: securityGroupReadRecord(record), Value: bytes.Clone(raw)}, true, nil
	}
	if record, ok := reference.(*resource.RawResource); ok && record != nil {
		reference = record.Body
	}
	var raw json.RawMessage
	switch value := reference.(type) {
	case json.RawMessage:
		raw = bytes.Clone(value)
	case map[string]json.RawMessage:
		var e error
		raw, e = json.Marshal(value)
		if e != nil {
			return nil, true, e
		}
	case map[string]any:
		var e error
		raw, e = json.Marshal(value)
		if e != nil {
			return nil, true, e
		}
	default:
		return nil, false, nil
	}
	var fields map[string]json.RawMessage
	if e := json.Unmarshal(raw, &fields); e != nil {
		return nil, false, nil
	}
	if fields != nil {
		if _, present := fields["id"]; present {
			return &SecurityGroupReadResult{Backend: "existing", Value: bytes.Clone(raw)}, true, nil
		}
	}
	return nil, false, nil
}

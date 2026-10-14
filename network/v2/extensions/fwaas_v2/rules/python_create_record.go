package rules

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// CreateRecordOpts carries the declared Python FirewallRule attributes for
// create_firewall_rule. An omitted field is not sent, request.Null sends JSON
// null (protocol null means any) and request.Present keeps empty strings and
// false. Nothing is required, so the no-argument call sends an empty object
// and Neutron applies its defaults.
type CreateRecordOpts struct {
	Action               request.Optional[string] `json:"action,omitzero"`
	Description          request.Optional[string] `json:"description,omitzero"`
	DestinationIPAddress request.Optional[string] `json:"destination_ip_address,omitzero"`
	DestinationPort      request.Optional[string] `json:"destination_port,omitzero"`
	Enabled              request.Optional[bool]   `json:"enabled,omitzero"`
	IPVersion            request.Optional[int]    `json:"ip_version,omitzero"`
	Name                 request.Optional[string] `json:"name,omitzero"`
	ProjectID            request.Optional[string] `json:"project_id,omitzero"`
	Protocol             request.Optional[string] `json:"protocol,omitzero"`
	Shared               request.Optional[bool]   `json:"shared,omitzero"`
	SourceIPAddress      request.Optional[string] `json:"source_ip_address,omitzero"`
	SourcePort           request.Optional[string] `json:"source_port,omitzero"`
	FirewallPolicyID     request.Optional[string] `json:"firewall_policy_id,omitzero"`
}

type CreateRecordOption = request.Option[CreateRecordOpts]

func WithCreateRecordOptions(value CreateRecordOpts) CreateRecordOption {
	return request.WithOptions(value)
}

// WithCreateRecordField adds another attribute inside the firewall_rule
// envelope, such as source_firewall_group_id. FirewallRule allows unknown body
// attributes. A declared key may be given here with another JSON type, such
// as protocol 6, only when CreateRecordOpts leaves that key omitted.
func WithCreateRecordField(key string, value any) CreateRecordOption {
	return request.WithField[CreateRecordOpts](key, value)
}
func WithCreateRecordHeader(key, value string) CreateRecordOption {
	return request.WithHeader[CreateRecordOpts](key, value)
}

// RuleRecord is the created Python FirewallRule view. Rule is the native typed
// decode of the response object and Fields keeps that object unchanged.
type RuleRecord struct {
	Rule       *Rule
	Fields     map[string]json.RawMessage
	Header     http.Header
	StatusCode int
}

// CreateRecord implements create_firewall_rule: Resource.create sends POST
// fwaas/firewall_rules with only the supplied attributes in
// {"firewall_rule": ...} and accepts every status below 400.
func (a *API) CreateRecord(ctx context.Context, options ...CreateRecordOption) (*RuleRecord, error) {
	const op = "CreateRecord"
	fail := func(value *RuleRecord, err error) (*RuleRecord, error) {
		return value, request.Wrap(op, "rules", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return fail(nil, err)
	}
	if a == nil {
		return fail(nil, fmt.Errorf("%w: firewall rule API is required", resource.ErrInvalidOption))
	}
	source, err := cloudread.Capture(ctx, a.client, "network")
	if err != nil {
		return fail(nil, err)
	}
	config, err := cloudread.ApplyReadOptions(ctx, CreateRecordOpts{}, options, cloudread.OwnReadConfig[CreateRecordOpts], source.Guard)
	if err == nil {
		err = request.ValidateCapabilities(config, true, false, true)
	}
	var attrs map[string]any
	if err == nil {
		attrs, err = pythonFirewallRuleAttrs(config)
	}
	if err == nil {
		err = source.WithPolicy(ctx, nil, config.Headers)
	}
	if err != nil {
		return fail(nil, err)
	}
	codes := make([]int, 0, 200)
	for code := 200; code < 400; code++ {
		codes = append(codes, code)
	}
	target := source.Client.ServiceURL("fwaas", "firewall_rules")
	response, err := rest.DoJSONGuarded(ctx, &source.Client, source.Guard, http.MethodPost, target, map[string]any{"firewall_rule": attrs}, nil, codes...)
	if err != nil {
		return fail(nil, err)
	}
	value, err := pythonFirewallRuleRecord(response)
	if err = errors.Join(err, source.Guard(ctx)); err != nil {
		return fail(value, response.Fail(err))
	}
	return value, nil
}

// pythonFirewallRuleAttrs merges the supplied typed attributes and extension
// fields; an extension cannot repeat a supplied typed key.
func pythonFirewallRuleAttrs(config request.Config[CreateRecordOpts]) (map[string]any, error) {
	encoded, err := json.Marshal(config.Options)
	if err != nil {
		return nil, err
	}
	var typed map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &typed); err != nil {
		return nil, err
	}
	attrs := make(map[string]any, len(typed)+len(config.Fields))
	for key, value := range typed {
		attrs[key] = value
	}
	for key, value := range config.Fields {
		if _, exists := attrs[key]; exists {
			return nil, fmt.Errorf("%w: extension %q repeats a supplied attribute", resource.ErrInvalidOption, key)
		}
		attrs[key] = json.RawMessage(bytes.Clone(value))
	}
	return attrs, nil
}

// pythonFirewallRuleRecord follows _translate_response: the firewall_rule
// envelope is unwrapped when present and an empty or non-JSON body leaves no
// fields.
func pythonFirewallRuleRecord(response *rest.Response) (*RuleRecord, error) {
	value := &RuleRecord{Header: response.Header.Clone(), StatusCode: response.StatusCode}
	trimmed := bytes.TrimSpace(response.Body)
	if len(trimmed) == 0 || !json.Valid(trimmed) {
		return value, nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &fields); err != nil || fields == nil {
		return value, fmt.Errorf("%w: firewall rule response must be a JSON object", resource.ErrInvalidOption)
	}
	object := trimmed
	if nested, exists := fields["firewall_rule"]; exists {
		fields = nil
		if err := json.Unmarshal(nested, &fields); err != nil || fields == nil {
			return value, fmt.Errorf("%w: firewall rule envelope must be a JSON object", resource.ErrInvalidOption)
		}
		object = nested
	}
	typed := new(Rule)
	if err := json.Unmarshal(object, typed); err != nil {
		return value, err
	}
	value.Rule, value.Fields = typed, fields
	return value, nil
}

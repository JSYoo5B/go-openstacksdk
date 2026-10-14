package policies

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// InsertRuleRecordOpts carries insert_rule_into_policy's positions. A nil
// position is sent as JSON null, as Python always sends both keys; an empty
// string is sent as given.
type InsertRuleRecordOpts struct {
	InsertAfter  *string `json:"insert_after"`
	InsertBefore *string `json:"insert_before"`
}
type InsertRuleRecordOption = request.Option[InsertRuleRecordOpts]

func WithInsertRuleRecordOptions(value InsertRuleRecordOpts) InsertRuleRecordOption {
	value = cloneInsertRuleRecordOpts(value)
	return func(c *request.Config[InsertRuleRecordOpts]) error {
		c.Options = cloneInsertRuleRecordOpts(value)
		return nil
	}
}
func WithInsertRuleRecordAfter(ruleID string) InsertRuleRecordOption {
	return func(c *request.Config[InsertRuleRecordOpts]) error { c.Options.InsertAfter = &ruleID; return nil }
}
func WithInsertRuleRecordBefore(ruleID string) InsertRuleRecordOption {
	return func(c *request.Config[InsertRuleRecordOpts]) error { c.Options.InsertBefore = &ruleID; return nil }
}
func WithInsertRuleRecordHeader(key, value string) InsertRuleRecordOption {
	return request.WithHeader[InsertRuleRecordOpts](key, value)
}

func cloneInsertRuleRecordOpts(v InsertRuleRecordOpts) InsertRuleRecordOpts {
	clone := func(value *string) *string {
		if value == nil {
			return nil
		}
		copy := *value
		return &copy
	}
	return InsertRuleRecordOpts{InsertAfter: clone(v.InsertAfter), InsertBefore: clone(v.InsertBefore)}
}

func ownInsertRuleRecordConfig(c *request.Config[InsertRuleRecordOpts]) {
	cloudread.OwnReadConfig(c)
	c.Options = cloneInsertRuleRecordOpts(c.Options)
}

// PolicyRecord is the updated Python FirewallPolicy view. Policy is the
// native typed decode and Fields keeps the response object unchanged.
type PolicyRecord struct {
	Policy     *Policy
	Fields     map[string]json.RawMessage
	Header     http.Header
	StatusCode int
}

// InsertRuleRecord implements insert_rule_into_policy: FirewallPolicy
// .insert_rule sends PUT fwaas/firewall_policies/{id}/insert_rule with the
// unwrapped body {"firewall_rule_id", "insert_after", "insert_before"}, both
// positions null unless given. Like _put_request it accepts every status below
// 400 whose body is a JSON object and returns the unwrapped policy. Neutron
// validates the positions, so both or neither may be sent.
func (a *API) InsertRuleRecord(ctx context.Context, policyID, ruleID string, options ...InsertRuleRecordOption) (*PolicyRecord, error) {
	const op = "InsertRuleRecord"
	fail := func(value *PolicyRecord, err error) (*PolicyRecord, error) {
		return value, request.Wrap(op, "policies", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return fail(nil, err)
	}
	if a == nil {
		return fail(nil, fmt.Errorf("%w: firewall policy API is required", resource.ErrInvalidOption))
	}
	if strings.TrimSpace(policyID) == "" || policyID == "." || policyID == ".." {
		return fail(nil, fmt.Errorf("%w: firewall policy ID is required", resource.ErrInvalidOption))
	}
	source, err := cloudread.Capture(ctx, a.client, "network")
	if err != nil {
		return fail(nil, err)
	}
	config, err := cloudread.ApplyReadOptions(ctx, InsertRuleRecordOpts{}, options, ownInsertRuleRecordConfig, source.Guard)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	if err == nil {
		err = source.WithPolicy(ctx, nil, config.Headers)
	}
	if err != nil {
		return fail(nil, err)
	}
	body := map[string]any{"firewall_rule_id": ruleID, "insert_after": config.Options.InsertAfter, "insert_before": config.Options.InsertBefore}
	target := source.Client.ServiceURL("fwaas", "firewall_policies", url.PathEscape(policyID), "insert_rule")
	codes := make([]int, 0, 200)
	for code := 200; code < 400; code++ {
		codes = append(codes, code)
	}
	response, err := rest.DoJSONGuarded(ctx, &source.Client, source.Guard, http.MethodPut, target, body, nil, codes...)
	if err != nil {
		return fail(nil, err)
	}
	value, err := pythonPolicyRecord(response)
	if err = errors.Join(err, source.Guard(ctx)); err != nil {
		return fail(value, response.Fail(err))
	}
	return value, nil
}

// pythonPolicyRecord follows _put_request: resp.json() must succeed and the
// object updates the policy attributes without an envelope.
func pythonPolicyRecord(response *rest.Response) (*PolicyRecord, error) {
	value := &PolicyRecord{Header: response.Header.Clone(), StatusCode: response.StatusCode}
	trimmed := bytes.TrimSpace(response.Body)
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &fields); err != nil || fields == nil {
		return value, fmt.Errorf("%w: firewall policy response must be a JSON object", resource.ErrInvalidOption)
	}
	typed := new(Policy)
	if err := json.Unmarshal(trimmed, typed); err != nil {
		return value, err
	}
	value.Policy, value.Fields = typed, fields
	return value, nil
}

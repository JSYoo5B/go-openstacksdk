package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/responsejson"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// RouterAssignmentOpts owns the optional Source priority. Null and omission
// both produce an empty object; all other finite JSON values are preserved.
type RouterAssignmentOpts struct {
	Priority     json.RawMessage
	Headers      map[string]string
	Microversion *string
}
type RouterAssignmentOption func(*RouterAssignmentOpts) error

func cloneRouterAssignmentOpts(v RouterAssignmentOpts) RouterAssignmentOpts {
	v.Priority = bytes.Clone(v.Priority)
	v.Headers = maps.Clone(v.Headers)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	return v
}
func WithRouterAssignmentOpts(v RouterAssignmentOpts) RouterAssignmentOption {
	own := cloneRouterAssignmentOpts(v)
	return func(c *RouterAssignmentOpts) error { *c = cloneRouterAssignmentOpts(own); return nil }
}
func WithRouterAssignmentPriority(v int) RouterAssignmentOption {
	raw, _ := json.Marshal(v)
	return WithRouterAssignmentPriorityJSON(raw)
}
func WithRouterAssignmentPriorityJSON(v json.RawMessage) RouterAssignmentOption {
	own := bytes.Clone(v)
	return func(c *RouterAssignmentOpts) error { c.Priority = bytes.Clone(own); return nil }
}
func WithRouterAssignmentHeaders(v map[string]string) RouterAssignmentOption {
	own := maps.Clone(v)
	return func(c *RouterAssignmentOpts) error { c.Headers = maps.Clone(own); return nil }
}
func WithRouterAssignmentMicroversion(v string) RouterAssignmentOption {
	return func(c *RouterAssignmentOpts) error { x := v; c.Microversion = &x; return nil }
}

// RouterAssignmentResult preserves decoded JSON separately from actual bytes.
// A JSON response with an error HTTP status remains a value, following the
// pinned Agent method's direct resp.json() without raise_from_response.
type RouterAssignmentResult struct {
	Value, Envelope json.RawMessage
	Header          http.Header
	StatusCode      int
}

func (v *RouterAssignmentResult) Clone() *RouterAssignmentResult {
	if v == nil {
		return nil
	}
	out := *v
	out.Value, out.Envelope = bytes.Clone(v.Value), bytes.Clone(v.Envelope)
	out.Header = v.Header.Clone()
	return &out
}

// UpdateRouterInAgent performs one PUT without preliminary Agent/Router GETs.
// Priority range validation belongs to Neutron, as in the pinned Source method.
func (a *API) UpdateRouterInAgent(ctx context.Context, agentID, routerID string, options ...RouterAssignmentOption) (*RouterAssignmentResult, error) {
	wrap := func(err error) error {
		return request.Wrap("UpdateRouterInAgent", "agents", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrap(err)
	}
	if a == nil {
		return nil, wrap(dhcpNetworkInvalid("agents API is required"))
	}
	if err := dhcpHostSegment(agentID); err != nil {
		return nil, wrap(err)
	}
	if err := dhcpHostSegment(routerID); err != nil {
		return nil, wrap(err)
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "network")
	if err != nil {
		return nil, wrap(err)
	}
	var observed error
	guard := func(ctx context.Context) error {
		if observed == nil && a.client != selected {
			observed = dhcpNetworkInvalid("selected agents client changed")
		}
		return errors.Join(observed, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	policy := RouterAssignmentOpts{}
	for _, apply := range slices.Clone(options) {
		if err := guard(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(dhcpNetworkInvalid("nil router assignment option"))
		}
		next := cloneRouterAssignmentOpts(policy)
		if err := errors.Join(apply(&next), guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		policy = cloneRouterAssignmentOpts(next)
	}
	body := map[string]json.RawMessage{}
	if len(policy.Priority) != 0 {
		if !utf8.Valid(policy.Priority) || !json.Valid(policy.Priority) {
			return nil, wrap(dhcpNetworkInvalid("priority requires complete UTF-8 JSON"))
		}
		if !bytes.Equal(bytes.TrimSpace(policy.Priority), []byte("null")) {
			body["ha_chassis_priority"] = bytes.Clone(policy.Priority)
		}
	}
	if policy.Microversion != nil {
		source.Client.MoreHeaders = maps.Clone(source.Client.MoreHeaders)
		for key := range source.Client.MoreHeaders {
			if strings.EqualFold(key, "OpenStack-API-Version") {
				delete(source.Client.MoreHeaders, key)
			}
		}
	}
	if err := source.WithPolicy(ctx, policy.Microversion, policy.Headers); err != nil {
		return nil, wrap(err)
	}
	codes := make([]int, 900)
	for i := range codes {
		codes[i] = i + 100
	}
	headers := maps.Clone(source.Client.MoreHeaders)
	if headers == nil {
		headers = map[string]string{}
	}
	if source.Client.Microversion != "" {
		headers["OpenStack-API-Version"] = "network " + source.Client.Microversion
	}
	response, err := rest.DoJSONGuardedRejectionsHeaderPolicy(ctx, &source.Client, guard, http.MethodPut, source.Client.ServiceURL("agents", url.PathEscape(agentID), "l3-routers", url.PathEscape(routerID)), body, rest.RequestHeaderPolicy{Values: headers, Absent: []string{"Authorization", "X-Service-Token", "Cookie", "Host", "X-OpenStack-Nova-API-Version", "X-OpenStack-Volume-API-Version"}}, rest.RejectionPolicy{StopFaultRetryCallbacks: true}, codes...)
	var result *RouterAssignmentResult
	if response != nil {
		result = &RouterAssignmentResult{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	}
	if err != nil {
		return result, wrap(errors.Join(err, guard(ctx)))
	}
	normalized, err := responsejson.Normalize(response.Body, response.Header)
	if err == nil && (!utf8.Valid(normalized) || !json.Valid(normalized)) {
		err = dhcpNetworkInvalid("router assignment response must contain complete UTF-8 JSON")
	}
	if err = errors.Join(err, guard(ctx)); err != nil {
		return result, wrap(response.Fail(err))
	}
	result.Value = bytes.Clone(normalized)
	return result, nil
}

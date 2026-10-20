package subnetpools

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

	"github.com/JSYoo5B/go-openstacksdk/internal/responsejson"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// OnboardNetworkOpts owns transport policy for one Source subnet-onboard PUT.
type OnboardNetworkOpts struct {
	Headers      map[string]string
	Microversion *string
}
type OnboardNetworkOption func(*OnboardNetworkOpts) error

func cloneOnboardNetworkOpts(v OnboardNetworkOpts) OnboardNetworkOpts {
	v.Headers = maps.Clone(v.Headers)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	return v
}
func WithOnboardNetworkOpts(v OnboardNetworkOpts) OnboardNetworkOption {
	own := cloneOnboardNetworkOpts(v)
	return func(c *OnboardNetworkOpts) error { *c = cloneOnboardNetworkOpts(own); return nil }
}
func WithOnboardNetworkHeaders(v map[string]string) OnboardNetworkOption {
	own := maps.Clone(v)
	return func(c *OnboardNetworkOpts) error { c.Headers = maps.Clone(own); return nil }
}
func WithOnboardNetworkMicroversion(v string) OnboardNetworkOption {
	return func(c *OnboardNetworkOpts) error { x := v; c.Microversion = &x; return nil }
}

// OnboardNetworkResult owns arbitrary decoded JSON and physical HTTP evidence.
// Source's cast(dict, resp.json()) does not impose an object shape at runtime.
type OnboardNetworkResult struct {
	Value, Envelope json.RawMessage
	Header          http.Header
	StatusCode      int
}

func (r *OnboardNetworkResult) Clone() *OnboardNetworkResult {
	if r == nil {
		return nil
	}
	out := *r
	out.Value = bytes.Clone(r.Value)
	out.Envelope = bytes.Clone(r.Envelope)
	out.Header = r.Header.Clone()
	return &out
}

// OnboardNetworkSubnetsRecord onboards all subnets of a literal network ID.
// It sends one PUT with an unwrapped network_id body, without preliminary GETs.
// Neutron owns eligibility validation; even an empty network ID is forwarded.
func (a *API) OnboardNetworkSubnetsRecord(ctx context.Context, poolID, networkID string, options ...OnboardNetworkOption) (*OnboardNetworkResult, error) {
	const op = "OnboardNetworkSubnetsRecord"
	p, err := a.preparePoolRecord(ctx)
	if err == nil {
		err = poolRecordID(poolID)
	}
	if err == nil && !utf8.ValidString(networkID) {
		err = poolRecordInvalid("network ID must contain valid UTF-8")
	}
	if err != nil {
		return nil, poolRecordWrap(ctx, op, err)
	}
	policy := OnboardNetworkOpts{}
	for _, apply := range slices.Clone(options) {
		if err = p.guard(ctx); err != nil {
			return nil, poolRecordWrap(ctx, op, err)
		}
		if apply == nil {
			return nil, poolRecordWrap(ctx, op, poolRecordInvalid("nil onboard option"))
		}
		next := cloneOnboardNetworkOpts(policy)
		if err = errors.Join(apply(&next), p.guard(ctx)); err != nil {
			return nil, poolRecordWrap(ctx, op, err)
		}
		policy = cloneOnboardNetworkOpts(next)
	}
	if policy.Microversion != nil {
		p.source.Client.MoreHeaders = maps.Clone(p.source.Client.MoreHeaders)
		for key := range p.source.Client.MoreHeaders {
			if strings.EqualFold(key, "OpenStack-API-Version") {
				delete(p.source.Client.MoreHeaders, key)
			}
		}
	}
	if err = p.withPolicy(ctx, policy.Microversion, policy.Headers); err != nil {
		return nil, poolRecordWrap(ctx, op, err)
	}
	headerPolicy := poolRecordHeaderPolicy(p.source.Client.MoreHeaders, p.source.Client.Microversion, true)
	for key, value := range p.source.Client.MoreHeaders {
		if headerPolicy.Values == nil {
			headerPolicy.Values = map[string]string{}
		}
		headerPolicy.Values[key] = value
	}
	target := p.source.Client.ServiceURL("subnetpools", url.PathEscape(poolID), "onboard_network_subnets")
	response, err := rest.DoJSONGuardedRejectionsHeaderPolicy(ctx, &p.source.Client, p.guard, http.MethodPut, target, map[string]string{"network_id": networkID}, headerPolicy, rest.RejectionPolicy{Codes: poolRecordCodes(400, 600), PreserveCleanRetry: true, StopFaultRetryCallbacks: true}, poolRecordCodes(100, 400)...)
	var result *OnboardNetworkResult
	if response != nil {
		result = &OnboardNetworkResult{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	}
	if err != nil {
		return result, poolRecordWrap(ctx, op, errors.Join(err, p.guard(ctx)))
	}
	normalized, err := responsejson.Normalize(response.Body, response.Header)
	if err == nil && (!utf8.Valid(normalized) || !json.Valid(normalized)) {
		err = poolRecordInvalid("onboarding response must contain complete UTF-8 JSON")
	}
	if err = errors.Join(err, p.guard(ctx)); err != nil {
		return result, poolRecordWrap(ctx, op, response.Fail(err))
	}
	result.Value = bytes.Clone(normalized)
	return result, nil
}

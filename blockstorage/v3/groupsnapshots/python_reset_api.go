package groupsnapshots

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

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/gophercloud/gophercloud/v2"
)

// ResetStatusRecordResult carries the literal requested reference and the
// actual action HTTP receipt. Python returns None; no GroupSnapshot is
// fabricated. Preparation errors can return Discovery without a body.
type ResetStatusRecordResult struct {
	Reference  string
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
	Discovery  []DiscoveryObservation
}

// RecordResetStatusOpts carries an explicit microversion and headers, both
// Go extensions. Without a microversion the call follows GroupSnapshot._action:
// the configured one or min(3.29, server maximum).
type RecordResetStatusOpts struct {
	Microversion *string
	Headers      map[string]string
}
type RecordResetStatusOption func(*RecordResetStatusOpts) error

func copyResetStatus(v RecordResetStatusOpts) RecordResetStatusOpts {
	v.Headers = maps.Clone(v.Headers)
	v.Microversion = cloneString(v.Microversion)
	return v
}
func WithRecordResetStatusMicroversion(v string) RecordResetStatusOption {
	return func(c *RecordResetStatusOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithRecordResetStatusHeaders(v map[string]string) RecordResetStatusOption {
	owned := maps.Clone(v)
	return func(c *RecordResetStatusOpts) error { c.Headers = maps.Clone(owned); return nil }
}

// ResetStatusRecord implements reset_group_snapshot_status and its deprecated
// alias reset_group_snapshot_state: POST group_snapshots/{id}/action with
// {"reset_status": {"status": status}} and the empty Accept header that
// GroupSnapshot._action sends. Every status below 400 is accepted.
func (a *API) ResetStatusRecord(ctx context.Context, id, status string, options ...RecordResetStatusOption) (*ResetStatusRecordResult, error) {
	const op = "ResetStatusRecord"
	p, err := a.capture(ctx)
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	if err := recordID(id); err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	cfg, err := applyRecord(p, RecordResetStatusOpts{}, copyResetStatus, slices.Clone(options))
	if err == nil {
		err = p.prepare(cfg.Microversion, cfg.Headers, nil, true)
	}
	if err != nil {
		var value *ResetStatusRecordResult
		if len(p.discovery) != 0 {
			value = &ResetStatusRecordResult{Reference: id, Discovery: recordDiscovery(p.discovery)}
		}
		return value, recordWrap(ctx, op, errors.Join(err, p.guard(ctx)))
	}
	// The Python action header replaces any Accept from the selected client.
	for key := range p.source.Client.MoreHeaders {
		if strings.EqualFold(key, "Accept") {
			delete(p.source.Client.MoreHeaders, key)
		}
	}
	body := map[string]any{"reset_status": map[string]string{"status": status}}
	response, err := p.resetRequest(p.source.Client.ServiceURL("group_snapshots", url.PathEscape(id), "action"), body)
	value := &ResetStatusRecordResult{Reference: id, Discovery: recordDiscovery(p.discovery)}
	if response != nil {
		value.Envelope, value.Header, value.StatusCode = bytes.Clone(response.Body), response.Header.Clone(), response.StatusCode
	}
	return value, recordWrap(ctx, op, err)
}

// resetRequest is request with the fixed empty Accept header.
func (p *recordPrepared) resetRequest(target string, body any) (*rest.Response, error) {
	policy := rest.RequestHeaderPolicy{Values: maps.Clone(p.source.Client.MoreHeaders), Absent: []string{"Authorization", "X-Service-Token", "Host", "Cookie", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "TE", "Upgrade", "X-OpenStack-Nova-API-Version"}}
	if policy.Values == nil {
		policy.Values = make(map[string]string)
	}
	if selected := p.source.Client.Microversion; selected != "" {
		policy.Values["OpenStack-API-Version"] = "volume " + selected
		policy.Values["X-OpenStack-Volume-API-Version"] = selected
	} else {
		policy.Absent = append(policy.Absent, "OpenStack-API-Version", "X-OpenStack-Volume-API-Version")
	}
	policy.Values["Accept"] = ""
	policy.Values["Content-Type"] = "application/json"
	response, err := rest.DoJSONGuardedRejectionsHeaderPolicy(p.ctx, &p.source.Client, p.guard, http.MethodPost, target, body, policy, rest.RejectionPolicy{Codes: recordCodes(400, 600), PreserveCleanRetry: true, StopFaultRetryCallbacks: true}, recordCodes(100, 400)...)
	if response == nil && err != nil {
		var native gophercloud.ErrUnexpectedResponseCode
		if errors.As(err, &native) {
			response = &rest.Response{Body: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual}
		}
	}
	return response, errors.Join(err, p.guard(p.ctx))
}

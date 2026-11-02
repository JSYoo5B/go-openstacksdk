package groups

import (
	"context"
	"errors"
	"maps"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// ResetStatusRecordResult carries the literal requested reference and the
// actual action HTTP receipt. Python returns None; no Group is fabricated.
// Preparation errors can return Discovery with StatusCode 0 and no body.
type ResetStatusRecordResult = ReplicationRecordResult

// RecordResetStatusOpts carries an explicit microversion and headers, both
// Go extensions. Without a microversion the call follows
// Group._get_microversion: the configured one or min(3.38, server maximum).
type RecordResetStatusOpts struct {
	Microversion *string
	Headers      map[string]string
}

// RecordResetStatusOption is one option of ResetStatusRecord: request.Custom
// over *RecordResetStatusOpts, request.Microversion or request.Headers
// (replaces the headers). Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type RecordResetStatusOption = request.WriteCall

var resetStatusCalls = calls.Spec[RecordResetStatusOpts]{
	Headers:      calls.Value(func(o *RecordResetStatusOpts) *map[string]string { return &o.Headers }),
	Microversion: calls.Pointer(func(o *RecordResetStatusOpts) **string { return &o.Microversion }),
}

func copyResetStatus(v RecordResetStatusOpts) RecordResetStatusOpts {
	v.Headers = maps.Clone(v.Headers)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	return v
}

// ResetStatusRecord implements reset_group_status and its deprecated alias
// reset_group_state: POST groups/{id}/action with
// {"reset_status": {"status": status}}. Every status below 400 is accepted.
func (a *API) ResetStatusRecord(ctx context.Context, id, status string, options ...RecordResetStatusOption) (*ResetStatusRecordResult, error) {
	const op = "ResetStatusRecord"
	p, err := a.capture(ctx)
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	if err = recordID(id); err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	cfg, err := applyRecord(p, RecordResetStatusOpts{}, copyResetStatus, calls.Translate(options, resetStatusCalls))
	if err == nil {
		err = p.policy(cfg.Microversion, cfg.Headers)
	}
	if err == nil {
		err = p.negotiate()
	}
	if err != nil {
		value, cause := p.replicationPreparation(id, errors.Join(err, p.guard(ctx)))
		return value, recordWrap(ctx, op, cause)
	}
	body := map[string]any{"reset_status": map[string]string{"status": status}}
	response, err := p.exchange(http.MethodPost, p.source.Client.ServiceURL("groups", url.PathEscape(id), "action"), body)
	return p.replicationResult(id, response, err), recordWrap(ctx, op, errors.Join(err, p.guard(ctx)))
}

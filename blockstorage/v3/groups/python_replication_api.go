package groups

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cinderrequest"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// ReplicationRecordResult carries the literal requested reference and actual
// action HTTP receipt. It has no fabricated Group Resource or completed state.
// Preparation errors can return Discovery with StatusCode0 and no action body.
type ReplicationRecordResult struct {
	Reference  string
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
	Discovery  []DiscoveryObservation
}

// Native authentication failure preserves two independent public fields and
// does not unwrap ErrOriginal. Receipt extraction reads the actual original
// rejection without replacing the native wrapper or its ErrReauth cause.
func replicationNativeOriginal(err error, native *gophercloud.ErrUnexpectedResponseCode) bool {
	var authentication *gophercloud.ErrUnableToReauthenticate
	if errors.As(err, &authentication) {
		err = authentication.ErrOriginal
	}
	return errors.As(err, native)
}

func (p *recordPrepared) replicationResult(id string, response *rest.Response, err error) *ReplicationRecordResult {
	if response == nil {
		var native gophercloud.ErrUnexpectedResponseCode
		if replicationNativeOriginal(err, &native) {
			response = &rest.Response{Body: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual}
		}
	}
	value := &ReplicationRecordResult{Reference: id, Discovery: p.proof(&GroupRecord{}).Discovery}
	if response != nil {
		value.Envelope = bytes.Clone(response.Body)
		value.Header, value.StatusCode = response.Header.Clone(), response.StatusCode
	}
	return value
}
func (p *recordPrepared) replicationPreparation(id string, err error) (*ReplicationRecordResult, error) {
	if len(p.discovery) == 0 {
		return nil, err
	}
	return &ReplicationRecordResult{Reference: id, Discovery: p.proof(&GroupRecord{}).Discovery}, err
}
func (p *recordPrepared) prepareReplication(version *string, headers map[string]string) error {
	for key := range headers {
		if strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Volume-API-Version") || strings.EqualFold(key, "X-OpenStack-Nova-API-Version") {
			return recordInvalid("replication version header %q is SDK owned", key)
		}
	}
	// Capture already checked the original selected aliases. Remove only their
	// private copies, so a typed override can retain the original sticky guard.
	for key := range p.source.Client.MoreHeaders {
		if strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Volume-API-Version") {
			delete(p.source.Client.MoreHeaders, key)
		}
	}
	if err := p.policy(version, headers); err != nil {
		return err
	}
	if selected := p.source.Client.Microversion; selected != "" {
		match, err := microversions.Matches(selected, "3.38")
		if err != nil {
			return recordInvalid("replication selected microversion: %v", err)
		}
		if !match {
			return fmt.Errorf("%w: group replication requires Cinder3.38 or later, selected %q", resource.ErrUnsupported, selected)
		}
		return p.guard(p.ctx)
	}
	selected, pages, err := cinderrequest.NegotiateRequired(p.ctx, p.source, "3.38", "3.38")
	for _, page := range pages {
		p.discovery = append(p.discovery, DiscoveryObservation{Envelope: bytes.Clone(page.Body), Header: page.Header.Clone(), StatusCode: page.StatusCode})
	}
	if err != nil {
		// The common reader retains native rejection evidence separately from
		// accepted discovery pages. This carrier remains discovery, never POST.
		if len(pages) == 0 {
			var native gophercloud.ErrUnexpectedResponseCode
			if replicationNativeOriginal(err, &native) {
				p.discovery = append(p.discovery, DiscoveryObservation{Envelope: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual})
			}
		}
		return errors.Join(err, p.guard(p.ctx))
	}
	return errors.Join(p.source.WithPolicy(p.ctx, &selected, nil), p.guard(p.ctx))
}
func (p *recordPrepared) exchangeReplication(id string, body map[string]any) (*ReplicationRecordResult, error) {
	response, err := p.exchange(http.MethodPost, p.source.Client.ServiceURL("groups", url.PathEscape(id), "action"), body)
	return p.replicationResult(id, response, err), errors.Join(err, p.guard(p.ctx))
}
func (a *API) replicationRecord(ctx context.Context, id, op, action string, options []RecordReplicationOption) (*ReplicationRecordResult, error) {
	p, err := a.capture(ctx)
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	if err = recordID(id); err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	cfg, err := applyRecord(p, RecordReplicationOpts{}, copyReplication, slices.Clone(options))
	if err == nil {
		err = p.prepareReplication(cfg.Microversion, cfg.Headers)
	}
	if err != nil {
		value, cause := p.replicationPreparation(id, errors.Join(err, p.guard(ctx)))
		return value, recordWrap(ctx, op, cause)
	}
	var actionValue any = map[string]any{}
	if cfg.PinnedSourceWire {
		actionValue = nil
	}
	value, err := p.exchangeReplication(id, map[string]any{action: actionValue})
	return value, recordWrap(ctx, op, err)
}

// EnableReplicationRecord sends the stock schema-compatible empty action.
// PinnedSourceWire explicitly replays the fixed Python null-value divergence.
func (a *API) EnableReplicationRecord(ctx context.Context, id string, options ...RecordReplicationOption) (*ReplicationRecordResult, error) {
	return a.replicationRecord(ctx, id, "EnableReplicationRecord", "enable_replication", options)
}
func (a *API) DisableReplicationRecord(ctx context.Context, id string, options ...RecordReplicationOption) (*ReplicationRecordResult, error) {
	return a.replicationRecord(ctx, id, "DisableReplicationRecord", "disable_replication", options)
}

// FailoverReplicationRecord defaults to false/null and the stock correct key.
// PinnedSourceWire replays Source's literal modify_body_for_action key and can
// therefore receive the stock400; it never reports that body as SDK parity.
func (a *API) FailoverReplicationRecord(ctx context.Context, id string, options ...RecordFailoverOption) (*ReplicationRecordResult, error) {
	const op = "FailoverReplicationRecord"
	p, err := a.capture(ctx)
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	if err = recordID(id); err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	cfg, err := applyRecord(p, RecordFailoverOpts{}, copyFailover, slices.Clone(options))
	if err == nil && cfg.SecondaryBackendID != nil && !utf8.ValidString(*cfg.SecondaryBackendID) {
		err = recordInvalid("replication backend ID must be UTF-8")
	}
	if err == nil {
		err = p.prepareReplication(cfg.Microversion, cfg.Headers)
	}
	if err != nil {
		value, cause := p.replicationPreparation(id, errors.Join(err, p.guard(ctx)))
		return value, recordWrap(ctx, op, cause)
	}
	allow := cfg.AllowedAttachedVolume != nil && *cfg.AllowedAttachedVolume
	var backend any
	if cfg.SecondaryBackendID != nil {
		backend = *cfg.SecondaryBackendID
	}
	action := "failover_replication"
	if cfg.PinnedSourceWire {
		action = "modify_body_for_action"
	}
	value, err := p.exchangeReplication(id, map[string]any{action: map[string]any{"allow_attached_volume": allow, "secondary_backend_id": backend}})
	return value, recordWrap(ctx, op, err)
}

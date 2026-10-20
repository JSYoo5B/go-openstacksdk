package snapshots

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
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

type ManageRecordOpts struct {
	Name, Description *string
	Metadata          json.RawMessage
	Headers           map[string]string
}
type ManageRecordOption func(*ManageRecordOpts) error

func copyManageOpts(v ManageRecordOpts) ManageRecordOpts {
	v.Metadata = bytes.Clone(v.Metadata)
	v.Headers = maps.Clone(v.Headers)
	if v.Name != nil {
		x := *v.Name
		v.Name = &x
	}
	if v.Description != nil {
		x := *v.Description
		v.Description = &x
	}
	return v
}
func WithManageRecordOpts(v ManageRecordOpts) ManageRecordOption {
	own := copyManageOpts(v)
	return func(c *ManageRecordOpts) error { *c = copyManageOpts(own); return nil }
}
func WithManageRecordName(v string) ManageRecordOption {
	return func(c *ManageRecordOpts) error { x := v; c.Name = &x; return nil }
}
func WithManageRecordDescription(v string) ManageRecordOption {
	return func(c *ManageRecordOpts) error { x := v; c.Description = &x; return nil }
}
func WithManageRecordMetadata(v json.RawMessage) ManageRecordOption {
	own := bytes.Clone(v)
	return func(c *ManageRecordOpts) error { c.Metadata = bytes.Clone(own); return nil }
}
func WithManageRecordHeaders(v map[string]string) ManageRecordOption {
	own := maps.Clone(v)
	return func(c *ManageRecordOpts) error { c.Headers = maps.Clone(own); return nil }
}

type UnmanageRecordOpts struct{ Headers map[string]string }
type UnmanageRecordOption func(*UnmanageRecordOpts) error

func WithUnmanageRecordHeaders(v map[string]string) UnmanageRecordOption {
	own := maps.Clone(v)
	return func(c *UnmanageRecordOpts) error { c.Headers = maps.Clone(own); return nil }
}

// ManagementReceipt is actual opaque action evidence; accepted means submitted,
// not proof that backend provisioning has already changed.
type ManagementReceipt struct {
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
}
type manageSource struct {
	source *cloudread.Source
	guard  func(context.Context) error
}

func (a *API) prepareManage(ctx context.Context) (*manageSource, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, findInvalid("snapshot API is required")
	}
	selected := a.client
	s, err := cloudread.Capture(ctx, selected, "volume")
	if err != nil {
		return nil, err
	}
	var observed error
	guard := func(ctx context.Context) error {
		if observed == nil && a.client != selected {
			observed = findInvalid("selected snapshot client changed")
		}
		return errors.Join(observed, s.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	return &manageSource{s, guard}, guard(ctx)
}
func managementCodes() (accepted, rejected []int) {
	for n := 100; n < 400; n++ {
		accepted = append(accepted, n)
	}
	for n := 400; n < 600; n++ {
		rejected = append(rejected, n)
	}
	return
}
func (p *manageSource) fixedPolicy(ctx context.Context, headers map[string]string) error {
	p.source.Client.MoreHeaders = maps.Clone(p.source.Client.MoreHeaders)
	for key := range p.source.Client.MoreHeaders {
		if strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Volume-API-Version") {
			delete(p.source.Client.MoreHeaders, key)
		}
	}
	version := "3.65"
	return errors.Join(p.source.WithPolicy(ctx, &version, headers), p.guard(ctx))
}

// managementHeaders freezes selected application/version headers across native
// retries; authentication remains live and owned by the provider transport.
func (p *manageSource) managementHeaders() rest.RequestHeaderPolicy {
	values := maps.Clone(p.source.Client.MoreHeaders)
	if values == nil {
		values = map[string]string{}
	}
	values["OpenStack-API-Version"] = "volume 3.65"
	values["X-OpenStack-Volume-API-Version"] = "3.65"
	absent := []string{"Authorization", "X-Service-Token", "Cookie", "Host", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "TE", "Upgrade"}
	for _, name := range []string{"If-Match", "If-None-Match", "If-Modified-Since", "If-Unmodified-Since"} {
		found := false
		for key := range values {
			found = found || strings.EqualFold(key, name)
		}
		if !found {
			absent = append(absent, name)
		}
	}
	return rest.RequestHeaderPolicy{Values: values, Absent: absent}
}

// ManageRecord submits existing backend storage as a snapshot. It advertises
// Source's fixed 3.65 header even when capability checks select the legacy URL.
// Every body field is present; omitted name/description/metadata remain null.
func (a *API) ManageRecord(ctx context.Context, volumeID string, reference json.RawMessage, options ...ManageRecordOption) (*SnapshotRecord, error) {
	wrap := func(err error) error {
		return request.Wrap("ManageRecord", "snapshots", cloudread.ContextError(ctx, err))
	}
	p, err := a.prepareManage(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	ref := bytes.Clone(reference)
	if ref == nil {
		ref = json.RawMessage("null")
	}
	policy := ManageRecordOpts{}
	for _, apply := range slices.Clone(options) {
		if err := p.guard(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(findInvalid("nil manage option"))
		}
		next := copyManageOpts(policy)
		if err := errors.Join(apply(&next), p.guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		policy = copyManageOpts(next)
	}
	metadata := policy.Metadata
	if metadata == nil {
		metadata = json.RawMessage("null")
	}
	if !utf8.ValidString(volumeID) || !utf8.Valid(ref) || !json.Valid(ref) || !utf8.Valid(metadata) || !json.Valid(metadata) {
		return nil, wrap(findInvalid("manage inputs must contain UTF-8 JSON"))
	}
	if policy.Name != nil && !utf8.ValidString(*policy.Name) || policy.Description != nil && !utf8.ValidString(*policy.Description) {
		return nil, wrap(findInvalid("manage optional text must be UTF-8"))
	}
	// Own/validate headers before discovery; discovery evidence is independent.
	if err := p.source.WithPolicy(ctx, nil, policy.Headers); err != nil {
		return nil, wrap(err)
	}
	discoveryProfile := microversions.CinderProfile
	discoveryProfile.StopFaultRetryCallbacks = true
	discoveryCtx := rest.WithOperationGuard(ctx, func(context.Context) error { return p.guard(ctx) })
	discovered, err := microversions.Read(discoveryCtx, p.source, discoveryProfile)
	var observations []RecordObservation
	for _, r := range discovered.Responses {
		observations = append(observations, RecordObservation{Envelope: bytes.Clone(r.Body), Header: r.Header.Clone(), StatusCode: r.StatusCode})
	}
	if err = errors.Join(err, p.guard(ctx)); err != nil {
		return &SnapshotRecord{Discovery: observations}, wrap(err)
	}
	supported, err := microversions.Supports(discovered.Maximum, discovered.Minimum, p.source.Client.Microversion, "3.8")
	if err != nil {
		return &SnapshotRecord{Discovery: observations}, wrap(err)
	}
	path := "os-snapshot-manage"
	if supported {
		path = "manageable_snapshots"
	}
	if err := p.fixedPolicy(ctx, policy.Headers); err != nil {
		return &SnapshotRecord{Discovery: observations}, wrap(err)
	}
	payload := map[string]any{"snapshot": map[string]any{"volume_id": volumeID, "ref": ref, "name": policy.Name, "description": policy.Description, "metadata": metadata}}
	accepted, rejected := managementCodes()
	response, err := rest.DoJSONGuardedRejectionsHeaderPolicy(ctx, &p.source.Client, p.guard, http.MethodPost, p.source.Client.ServiceURL(path), payload, p.managementHeaders(), rest.RejectionPolicy{Codes: rejected, PreserveCleanRetry: true, StopFaultRetryCallbacks: true}, accepted...)
	if err != nil {
		row := &SnapshotRecord{Discovery: observations}
		if response != nil {
			row.Envelope, row.Header, row.StatusCode = bytes.Clone(response.Body), response.Header.Clone(), response.StatusCode
		}
		return row, wrap(errors.Join(err, p.guard(ctx)))
	}
	row, err := findResponse(response, "")
	if row != nil {
		row.Discovery = observations
		if row.Resource != nil && (row.Wire == nil || row.Wire.Body["id"] == nil) {
			row.Resource.Body["id"] = json.RawMessage("null")
		}
	}
	if err = errors.Join(err, p.guard(ctx)); err != nil {
		if row != nil {
			row.Resource = nil
		}
		err = response.Fail(err)
	}
	return row, wrap(err)
}

// UnmanageRecord submits the fixed Source action. Missing snapshots are errors.
// Opaque success is returned independently from a resource or cache mutation.
func (a *API) UnmanageRecord(ctx context.Context, id string, options ...UnmanageRecordOption) (*ManagementReceipt, error) {
	wrap := func(err error) error {
		return request.Wrap("UnmanageRecord", "snapshots", cloudread.ContextError(ctx, err))
	}
	p, err := a.prepareManage(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	if err := findSegment(id); err != nil {
		return nil, wrap(err)
	}
	policy := UnmanageRecordOpts{}
	for _, apply := range slices.Clone(options) {
		if err := p.guard(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(findInvalid("nil unmanage option"))
		}
		next := UnmanageRecordOpts{Headers: maps.Clone(policy.Headers)}
		if err := errors.Join(apply(&next), p.guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		policy.Headers = maps.Clone(next.Headers)
	}
	if err := p.fixedPolicy(ctx, policy.Headers); err != nil {
		return nil, wrap(err)
	}
	accepted, rejected := managementCodes()
	response, err := rest.DoJSONGuardedRejectionsHeaderPolicy(ctx, &p.source.Client, p.guard, http.MethodPost, p.source.Client.ServiceURL("snapshots", url.PathEscape(id), "action"), map[string]any{"os-unmanage": nil}, p.managementHeaders(), rest.RejectionPolicy{Codes: rejected, PreserveCleanRetry: true, StopFaultRetryCallbacks: true}, accepted...)
	var receipt *ManagementReceipt
	if response != nil {
		receipt = &ManagementReceipt{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	}
	return receipt, wrap(errors.Join(err, p.guard(ctx)))
}

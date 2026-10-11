package groupsnapshots

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudlocation"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudsnapshot"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"iter"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }
func (a *API) RawClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}

type recordPrepared struct {
	ctx       context.Context
	source    *cloudread.Source
	guard     func(context.Context) error
	location  resource.CloudLocation
	discovery []DiscoveryObservation
}

func (a *API) capture(ctx context.Context) (*recordPrepared, error) {
	if a == nil {
		return nil, recordInvalid("group snapshot API required")
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "volume")
	if err != nil {
		return nil, err
	}
	guard := func(ctx context.Context) error {
		var changed error
		if a.client != selected {
			changed = recordInvalid("selected group snapshot client changed")
		}
		return errors.Join(changed, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	return &recordPrepared{ctx: ctx, source: source, guard: guard}, guard(ctx)
}
func applyRecord[T any, O ~func(*T) error](p *recordPrepared, zero T, copy func(T) T, options []O) (T, error) {
	value := copy(zero)
	for _, apply := range options {
		if err := p.guard(p.ctx); err != nil {
			return value, err
		}
		if apply == nil {
			return value, recordInvalid("nil group snapshot option")
		}
		next := copy(value)
		if err := errors.Join(apply(&next), p.guard(p.ctx)); err != nil {
			return value, err
		}
		value = copy(next)
	}
	return value, p.guard(p.ctx)
}
func recordID(id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." || !utf8.ValidString(id) {
		return recordInvalid("nonblank literal UTF-8 group snapshot ID required")
	}
	for _, char := range id {
		if unicode.IsControl(char) {
			return recordInvalid("group snapshot ID contains controls")
		}
	}
	return nil
}
func recordCodes(first, end int) []int {
	out := make([]int, end-first)
	for i := range out {
		out[i] = first + i
	}
	return out
}
func recordWrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "group_snapshots", cloudread.ContextError(ctx, err))
}
func (p *recordPrepared) prepare(version *string, headers map[string]string, location *resource.CloudLocation, jsonBody bool) error {
	if err := errors.Join(p.source.WithPolicy(p.ctx, version, headers), p.guard(p.ctx)); err != nil {
		return err
	}
	// JSON framing is owned before any discovery request is attempted.
	if jsonBody {
		if value, exists := p.source.Client.MoreHeaders["Content-Type"]; exists && value != "application/json" {
			return recordInvalid("group snapshot JSON content type is SDK owned")
		}
	}
	if location != nil {
		p.location = location.Clone()
	} else {
		var err error
		p.location.Project.ID, err = cloudlocation.ProjectID(p.source.Client.ProviderClient)
		if err = errors.Join(err, p.guard(p.ctx)); err != nil {
			return err
		}
	}
	if _, err := p.location.ForResource(nil, nil); err != nil {
		return errors.Join(err, p.guard(p.ctx))
	}
	if version == nil && p.source.Client.Microversion == "" {
		found, err := microversions.Read(p.ctx, p.source, microversions.CinderProfile)
		for _, response := range found.Responses {
			p.discovery = append(p.discovery, DiscoveryObservation{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode})
		}
		if err = errors.Join(err, p.guard(p.ctx)); err != nil {
			return err
		}
		selected, err := microversions.SelectVersion(found.Maximum, found.Minimum, "3.29")
		if err != nil {
			cause := recordInvalid("group snapshot version selection: %v", err)
			if len(found.Responses) > 0 {
				return found.Responses[len(found.Responses)-1].Fail(cause)
			}
			return cause
		}
		if err := p.source.WithPolicy(p.ctx, &selected, nil); err != nil {
			return err
		}
	} else if p.source.Client.Microversion != "" {
		if _, err := microversions.ParseVersion(p.source.Client.Microversion); err != nil {
			return recordInvalid("invalid selected group snapshot microversion: %v", err)
		}
	}
	for key := range p.source.Client.MoreHeaders {
		if strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Volume-API-Version") {
			delete(p.source.Client.MoreHeaders, key)
		}
	}
	return p.guard(p.ctx)
}
func (p *recordPrepared) proof(value *GroupSnapshotRecord) *GroupSnapshotRecord {
	if value != nil {
		value.Discovery = recordDiscovery(p.discovery)
	}
	return value
}
func (p *recordPrepared) request(method, target string, body any) (*rest.Response, error) {
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
	if body != nil {
		if value, exists := policy.Values["Content-Type"]; exists && value != "application/json" {
			return nil, recordInvalid("group snapshot JSON content type is SDK owned")
		}
		policy.Values["Content-Type"] = "application/json"
	}
	response, err := rest.DoJSONGuardedRejectionsHeaderPolicy(p.ctx, &p.source.Client, p.guard, method, target, body, policy, rest.RejectionPolicy{Codes: recordCodes(400, 600), PreserveCleanRetry: true, StopFaultRetryCallbacks: true}, recordCodes(100, 400)...)
	if response == nil && err != nil {
		var native gophercloud.ErrUnexpectedResponseCode
		if errors.As(err, &native) {
			response = &rest.Response{Body: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual}
		}
	}
	return response, errors.Join(err, p.guard(p.ctx))
}
func (p *recordPrepared) response(r *rest.Response, seed map[string]json.RawMessage, err error) (*GroupSnapshotRecord, error) {
	if err != nil {
		return p.proof(recordReceipt(r)), err
	}
	value, err := recordResponse(r, seed, p.location)
	if err = errors.Join(err, p.guard(p.ctx)); err != nil {
		err = r.Fail(err)
	}
	return p.proof(value), err
}

// GetRecord fetches one literal ID. Optional seed is a declared view, never a route or HTTP receipt.
func (a *API) GetRecord(ctx context.Context, id string, options ...RecordReadOption) (*GroupSnapshotRecord, error) {
	const op = "GetRecord"
	p, err := a.capture(ctx)
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	if err := recordID(id); err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	cfg, err := applyRecord(p, RecordReadOpts{}, copyRead, slices.Clone(options))
	seed := make(map[string]json.RawMessage)
	if err == nil && cfg.Seed != nil {
		if cfg.Seed.Resource == nil {
			err = recordInvalid("group snapshot seed requires declared Resource")
		} else {
			for key, raw := range cfg.Seed.Resource.Body {
				if recordKnown(key) {
					if !utf8.Valid(raw) || !json.Valid(raw) {
						err = recordInvalid("invalid group snapshot seed %q", key)
						break
					}
					seed[key] = bytes.Clone(raw)
				}
			}
		}
	}
	seed["id"], _ = json.Marshal(id)
	if err != nil {
		return nil, recordWrap(ctx, op, errors.Join(err, p.guard(ctx)))
	}
	value, err := p.getRecord(id, seed, nil, cfg)
	return value, recordWrap(ctx, op, err)
}

// getRecord shares the existing guarded transport and projector with FindRecord.
// Its caller owns the constructor seed and any query; it never captures a new source.
func (p *recordPrepared) getRecord(id string, seed map[string]json.RawMessage, query url.Values, cfg RecordReadOpts) (*GroupSnapshotRecord, error) {
	if err := errors.Join(p.prepare(cfg.Microversion, cfg.Headers, cfg.Location, false), p.guard(p.ctx)); err != nil {
		return nil, err
	}
	target := p.source.Client.ServiceURL("group_snapshots", url.PathEscape(id))
	if len(query) != 0 {
		target += "?" + query.Encode()
	}
	r, err := p.request(http.MethodGet, target, nil)
	return p.response(r, seed, err)
}

// CreateRecord posts only the pinned declared attrs. Unknown attrs are ignored;
// nil is an explicit empty group_snapshot object and no prefetch is performed.
func (a *API) CreateRecord(ctx context.Context, attrs map[string]any, options ...RecordCreateOption) (*GroupSnapshotRecord, error) {
	const op = "CreateRecord"
	p, err := a.capture(ctx)
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	selected := make(map[string]any)
	for key, value := range attrs {
		if key == "self" || key == "cls" || key == "connection" || key == "_synchronized" {
			return nil, recordWrap(ctx, op, recordInvalid("group snapshot constructor collides with %q", key))
		}
		if recordKnown(key) {
			selected[key] = value
		}
	}
	raw, marshalErr := json.Marshal(selected)
	if err = errors.Join(marshalErr, p.guard(ctx)); err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	cfg, err := applyRecord(p, RecordCreateOpts{}, copyCreate, slices.Clone(options))
	if err == nil {
		err = p.prepare(cfg.Microversion, cfg.Headers, cfg.Location, true)
	}
	if err != nil {
		return nil, recordWrap(ctx, op, errors.Join(err, p.guard(ctx)))
	}
	// GroupSnapshot.new has no allow_patch: even an explicitly supplied ID
	// remains dirty in Source's POST body. Server schema decides validity.
	r, err := p.request(http.MethodPost, p.source.Client.ServiceURL("group_snapshots"), map[string]any{"group_snapshot": recordClone(fields)})
	value, err := p.response(r, fields, err)
	return value, recordWrap(ctx, op, err)
}

// DeleteRecord ignores only a clean native404 by default and retains its receipt.
func (a *API) DeleteRecord(ctx context.Context, id string, options ...RecordDeleteOption) (*DeleteRecordResult, error) {
	const op = "DeleteRecord"
	p, err := a.capture(ctx)
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	if err := recordID(id); err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	cfg, err := applyRecord(p, RecordDeleteOpts{}, copyDelete, slices.Clone(options))
	if err == nil {
		err = p.prepare(cfg.Microversion, cfg.Headers, nil, false)
	}
	if err != nil {
		return nil, recordWrap(ctx, op, errors.Join(err, p.guard(ctx)))
	}
	r, err := p.request(http.MethodDelete, p.source.Client.ServiceURL("group_snapshots", url.PathEscape(id)), nil)
	var value *DeleteRecordResult
	if r != nil {
		value = &DeleteRecordResult{Envelope: bytes.Clone(r.Body), Header: r.Header.Clone(), StatusCode: r.StatusCode, Discovery: recordDiscovery(p.discovery)}
	}
	if (cfg.IgnoreMissing == nil || *cfg.IgnoreMissing) && err != nil {
		if ignored := resource.IgnoreMissing(err); ignored == nil {
			if value != nil {
				value.IgnoredMissing = true
			}
			err = nil
		}
	}
	return value, recordWrap(ctx, op, err)
}

// ListRecords lazily lists /group_snapshots/detail by default, with Source
// server queries, declared Body filters and shared guarded page traversal.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*GroupSnapshotRecord, error] {
	owned := slices.Clone(options)
	return func(yield func(*GroupSnapshotRecord, error) bool) {
		const op = "ListRecords"
		fail := func(err error) { yield(nil, recordWrap(ctx, op, err)) }
		p, err := a.capture(ctx)
		if err != nil {
			fail(err)
			return
		}
		cfg, err := applyRecord(p, RecordListOpts{}, copyList, owned)
		if err != nil {
			fail(errors.Join(err, p.guard(ctx)))
			return
		}
		for value, err := range p.listRecords(cfg) {
			if err != nil {
				fail(err)
				return
			}
			if !yield(value, nil) {
				return
			}
		}
	}
}

// listRecords reuses the source-owned pager and raw/declared record projection.
// It receives already-owned options, without replaying caller callbacks or capture.
func (p *recordPrepared) listRecords(cfg RecordListOpts) iter.Seq2[*GroupSnapshotRecord, error] {
	cfg = copyList(cfg)
	return func(yield func(*GroupSnapshotRecord, error) bool) {
		fail := func(err error) { yield(nil, err) }
		query, filters, control, err := prepareList(cfg)
		if err == nil {
			err = p.prepare(cfg.Microversion, cfg.Headers, cfg.Location, false)
		}
		if err != nil {
			fail(errors.Join(err, p.guard(p.ctx)))
			return
		}
		// Resource.list owns JSON Accept over an adapter/source default.
		// Only the explicit method header option may override that default.
		p.source.Client.MoreHeaders["Accept"] = "application/json"
		for key, value := range cfg.Headers {
			if strings.EqualFold(key, "Accept") {
				p.source.Client.MoreHeaders["Accept"] = value
			}
		}
		path := "group_snapshots/detail"
		if cfg.Details != nil && !*cfg.Details {
			path = "group_snapshots"
		}
		var page *rest.Response
		spec := rest.CollectionSpec[GroupSnapshotRecord]{Client: &p.source.Client, Path: path, Kind: "group_snapshots", PluralKey: "group_snapshots", Metadata: recordMetadata, Validate: p.guard, SourceGuard: p.guard, ListCodes: recordCodes(100, 400), ReadPage: func(_ context.Context, target string, _ ...int) (*rest.Response, error) {
			return p.request(http.MethodGet, target, nil)
		}, ValidateResponse: func(r *rest.Response) error { page = r; return p.guard(p.ctx) }, ValidateItem: func(v *GroupSnapshotRecord) error {
			v.Envelope = bytes.Clone(page.Body)
			v.Header, v.StatusCode = v.Wire.Header.Clone(), v.Wire.StatusCode
			p.proof(v)
			return errors.Join(recordProject(v, nil, p.location), p.guard(p.ctx))
		}, Paging: rest.PagePolicy[GroupSnapshotRecord]{HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, MarkerFallback: true, Marker: recordMarker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true, MarkerOrOffsetPagination: true}}
		for value, err := range rest.ListWithControl(p.ctx, spec, query, control) {
			if err != nil {
				fail(err)
				return
			}
			raw, err := json.Marshal(value.Resource.Body)
			if err != nil {
				fail(page.Fail(err))
				return
			}
			matched, cause := cloudsnapshot.MatchLocal(p.ctx, raw, filters, func() error { return p.guard(p.ctx) })
			if cause != nil {
				fail(page.Fail(cause))
				return
			}
			if matched && !yield(value, nil) {
				return
			}
		}
	}
}

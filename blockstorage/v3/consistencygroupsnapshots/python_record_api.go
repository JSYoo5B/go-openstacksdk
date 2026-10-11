package consistencygroupsnapshots

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
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

// API owns the fixed legacy /cgsnapshots resource and named Source defaults.
type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }
func (a *API) RawClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}

type recordPrepared struct {
	ctx      context.Context
	source   *cloudread.Source
	guard    func(context.Context) error
	location resource.CloudLocation
}

func (a *API) capture(ctx context.Context) (*recordPrepared, error) {
	if a == nil {
		return nil, recordInvalid("consistency group snapshot API required")
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "volume")
	if err != nil {
		return nil, err
	}
	guard := func(ctx context.Context) error {
		var changed error
		if a.client != selected {
			changed = recordInvalid("selected consistency group snapshot client changed")
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
			return value, recordInvalid("nil consistency group snapshot option")
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
		return recordInvalid("nonblank literal UTF-8 consistency group snapshot ID required")
	}
	for _, char := range id {
		if unicode.IsControl(char) {
			return recordInvalid("consistency group snapshot ID contains controls")
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
	return request.Wrap(op, "cgsnapshots", cloudread.ContextError(ctx, err))
}
func (p *recordPrepared) prepare(version *string, headers map[string]string, location *resource.CloudLocation, jsonBody bool) error {
	for key := range p.source.Client.MoreHeaders {
		if strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Volume-API-Version") {
			delete(p.source.Client.MoreHeaders, key)
		}
	}
	if err := errors.Join(p.source.WithPolicy(p.ctx, version, headers), p.guard(p.ctx)); err != nil {
		return err
	}
	// JSON framing is owned before the member request is attempted.
	if jsonBody {
		if value, exists := p.source.Client.MoreHeaders["Content-Type"]; exists && value != "application/json" {
			return recordInvalid("consistency group snapshot JSON content type is SDK owned")
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
	// Source inherits _max_microversion=None: no discovery is performed.
	// A selected default/explicit version is retained without a class ceiling.
	if p.source.Client.Microversion != "" {
		if _, err := microversions.ParseVersion(p.source.Client.Microversion); err != nil {
			return recordInvalid("invalid selected consistency group snapshot microversion: %v", err)
		}
	}
	for key := range p.source.Client.MoreHeaders {
		if strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Volume-API-Version") {
			delete(p.source.Client.MoreHeaders, key)
		}
	}
	return p.guard(p.ctx)
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
			return nil, recordInvalid("consistency group snapshot JSON content type is SDK owned")
		}
		policy.Values["Content-Type"] = "application/json"
	}
	response, err := rest.DoJSONGuardedRejectionsHeaderPolicy(p.ctx, &p.source.Client, p.guard, method, target, body, policy, rest.RejectionPolicy{Codes: recordCodes(400, 600), PreserveCleanRetry: true, StopFaultRetryCallbacks: true}, recordCodes(100, 400)...)
	if response == nil && err != nil {
		response = recordNativeResponse(err)
	}
	return response, errors.Join(err, p.guard(p.ctx))
}

// recordNativeResponse copies only actual native status/body/header evidence.
// Native authentication failure intentionally does not unwrap ErrOriginal;
// inspecting that public field retains its pointer/error contract unchanged.
func recordNativeResponse(err error) *rest.Response {
	var authentication *gophercloud.ErrUnableToReauthenticate
	if errors.As(err, &authentication) {
		err = authentication.ErrOriginal
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) {
		return nil
	}
	return &rest.Response{Body: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual}
}

func (p *recordPrepared) response(r *rest.Response, seed map[string]json.RawMessage, err error) (*ConsistencyGroupSnapshotRecord, error) {
	if err != nil {
		return recordReceipt(r), err
	}
	value, err := recordResponse(r, seed, p.location)
	if err = errors.Join(err, p.guard(p.ctx)); err != nil {
		err = r.Fail(err)
	}
	return value, err
}

// GetRecord fetches one literal ID. Optional seed is a declared view, never a route or HTTP receipt.
func (a *API) GetRecord(ctx context.Context, id string, options ...RecordReadOption) (*ConsistencyGroupSnapshotRecord, error) {
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
			err = recordInvalid("consistency group snapshot seed requires declared Resource")
		} else {
			for key, raw := range cfg.Seed.Resource.Body {
				if recordKnown(key) {
					if !utf8.Valid(raw) || !json.Valid(raw) {
						err = recordInvalid("invalid consistency group snapshot seed %q", key)
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
func (p *recordPrepared) getRecord(id string, seed map[string]json.RawMessage, query url.Values, cfg RecordReadOpts) (*ConsistencyGroupSnapshotRecord, error) {
	if err := errors.Join(p.prepare(cfg.Microversion, cfg.Headers, cfg.Location, false), p.guard(p.ctx)); err != nil {
		return nil, err
	}
	return p.getPrepared(id, seed, query)
}

// getPrepared exchanges only a member GET. Find performs policy preparation
// before its identity engine so preparation is never member absence.
func (p *recordPrepared) getPrepared(id string, seed map[string]json.RawMessage, query url.Values) (*ConsistencyGroupSnapshotRecord, error) {
	target := p.source.Client.ServiceURL("cgsnapshots", url.PathEscape(id))
	if len(query) != 0 {
		target += "?" + query.Encode()
	}
	r, err := p.request(http.MethodGet, target, nil)
	return p.response(r, seed, err)
}

// CreateRecord posts only the pinned declared attrs. Unknown attrs are ignored;
// nil is an explicit empty cgsnapshot object and no prefetch is performed.
func (a *API) CreateRecord(ctx context.Context, attrs map[string]any, options ...RecordCreateOption) (*ConsistencyGroupSnapshotRecord, error) {
	const op = "CreateRecord"
	p, err := a.capture(ctx)
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	selected := make(map[string]any)
	for key, value := range attrs {
		if key == "self" || key == "cls" || key == "connection" || key == "_synchronized" {
			return nil, recordWrap(ctx, op, recordInvalid("consistency group snapshot constructor collides with %q", key))
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
	// ConsistencyGroupSnapshot.new has no allow_patch: even an explicitly supplied ID
	// remains dirty in Source's POST body. Server schema decides validity.
	r, err := p.request(http.MethodPost, p.source.Client.ServiceURL("cgsnapshots"), map[string]any{"cgsnapshot": recordClone(fields)})
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
	r, err := p.request(http.MethodDelete, p.source.Client.ServiceURL("cgsnapshots", url.PathEscape(id)), nil)
	var value *DeleteRecordResult
	if r != nil {
		value = &DeleteRecordResult{Envelope: bytes.Clone(r.Body), Header: r.Header.Clone(), StatusCode: r.StatusCode}
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

// ListRecords lazily lists /cgsnapshots/detail by default, with Source
// server queries, declared Body filters and shared guarded page traversal.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*ConsistencyGroupSnapshotRecord, error] {
	owned := slices.Clone(options)
	return func(yield func(*ConsistencyGroupSnapshotRecord, error) bool) {
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
func (p *recordPrepared) listRecords(cfg RecordListOpts) iter.Seq2[*ConsistencyGroupSnapshotRecord, error] {
	cfg = copyList(cfg)
	return func(yield func(*ConsistencyGroupSnapshotRecord, error) bool) {
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
		path := "cgsnapshots/detail"
		if cfg.Details != nil && !*cfg.Details {
			path = "cgsnapshots"
		}
		var page *rest.Response
		spec := rest.CollectionSpec[ConsistencyGroupSnapshotRecord]{Client: &p.source.Client, Path: path, Kind: "cgsnapshots", PluralKey: "cgsnapshots", Metadata: recordMetadata, Validate: p.guard, SourceGuard: p.guard, ListCodes: recordCodes(100, 400), ReadPage: func(_ context.Context, target string, _ ...int) (*rest.Response, error) {
			return p.request(http.MethodGet, target, nil)
		}, ValidateResponse: func(r *rest.Response) error { page = r; return p.guard(p.ctx) }, ValidateItem: func(v *ConsistencyGroupSnapshotRecord) error {
			v.Envelope = bytes.Clone(page.Body)
			v.Header, v.StatusCode = v.Wire.Header.Clone(), v.Wire.StatusCode
			return errors.Join(recordProject(v, nil, p.location), p.guard(p.ctx))
		}, Paging: rest.PagePolicy[ConsistencyGroupSnapshotRecord]{HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, NextTruthy: cloudfilter.PythonJSONTruthy, MarkerFallback: true, Marker: recordMarker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true, MarkerOrOffsetPagination: true}}
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

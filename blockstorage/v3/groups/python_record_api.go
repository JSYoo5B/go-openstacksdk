package groups

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cinderrequest"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// API shares the selected authenticated Cinder client without taking ownership.
// Generic groups are separate from volumes and administrator group types.
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
	discovery []DiscoveryObservation
}

func (a *API) capture(ctx context.Context) (*recordPrepared, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, recordInvalid("group API required")
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "volume")
	if err != nil {
		return nil, err
	}
	if source.Client.Microversion != "" {
		if _, err := microversions.ParseVersion(source.Client.Microversion); err != nil {
			return nil, recordInvalid("invalid selected group microversion: %v", err)
		}
	}
	guard := func(ctx context.Context) error {
		var changed error
		if a.client != selected {
			changed = recordInvalid("selected Cinder group client changed")
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
			return value, recordInvalid("nil group option")
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
		return recordInvalid("nonblank UTF-8 group identity required")
	}
	for _, char := range id {
		if unicode.IsControl(char) {
			return recordInvalid("group identity has controls")
		}
	}
	return nil
}
func recordCodes(first, end int) []int {
	values := make([]int, end-first)
	for i := range values {
		values[i] = first + i
	}
	return values
}
func recordWrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "groups", cloudread.ContextError(ctx, err))
}
func (p *recordPrepared) policy(version *string, headers map[string]string) error {
	if version != nil && *version != "" {
		if _, err := microversions.ParseVersion(*version); err != nil {
			return recordInvalid("invalid group option microversion: %v", err)
		}
	}
	return errors.Join(p.source.WithPolicy(p.ctx, version, headers), p.guard(p.ctx))
}
func (p *recordPrepared) negotiate() error {
	version, pages, err := cinderrequest.Negotiate(p.ctx, p.source, "3.38")
	for _, r := range pages {
		p.discovery = append(p.discovery, DiscoveryObservation{Envelope: bytes.Clone(r.Body), Header: r.Header.Clone(), StatusCode: r.StatusCode})
	}
	if err != nil {
		return err
	}
	return errors.Join(p.source.WithPolicy(p.ctx, &version, nil), p.guard(p.ctx))
}
func (p *recordPrepared) proof(v *GroupRecord) *GroupRecord {
	if v == nil {
		return nil
	}
	v.Discovery = make([]DiscoveryObservation, len(p.discovery))
	for i, r := range p.discovery {
		v.Discovery[i] = DiscoveryObservation{Envelope: bytes.Clone(r.Envelope), Header: r.Header.Clone(), StatusCode: r.StatusCode}
	}
	return v
}
func (p *recordPrepared) exchange(method, target string, body any) (*rest.Response, error) {
	policy := rest.RequestHeaderPolicy{Values: make(map[string]string), Absent: []string{
		"Authorization", "X-Service-Token", "Host", "Cookie", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "TE", "Upgrade", "X-OpenStack-Nova-API-Version",
	}}
	if version := p.source.Client.Microversion; version != "" {
		policy.Values["OpenStack-API-Version"] = "volume " + version
		policy.Values["X-OpenStack-Volume-API-Version"] = version
	} else {
		policy.Absent = append(policy.Absent, "OpenStack-API-Version", "X-OpenStack-Volume-API-Version")
	}
	if body != nil {
		value, supplied := p.source.Client.MoreHeaders["Content-Type"]
		if !supplied {
			value = "application/json"
		}
		policy.Values["Content-Type"] = value
	}
	return rest.DoJSONGuardedRejectionsHeaderPolicy(p.ctx, &p.source.Client, p.guard, method, target, body, policy, rest.RejectionPolicy{Codes: recordCodes(400, 600), PreserveCleanRetry: true}, recordCodes(100, 400)...)
}
func (p *recordPrepared) response(r *rest.Response, seed map[string]json.RawMessage, err error) (*GroupRecord, error) {
	if err != nil {
		return p.proof(recordReceipt(r)), err
	}
	value, err := recordResponse(r, seed)
	if err = errors.Join(err, p.guard(p.ctx)); err != nil {
		err = r.Fail(err)
	}
	return p.proof(value), err
}
func mutationFields(raw map[string]json.RawMessage, update bool) (map[string]json.RawMessage, error) {
	body := make(map[string]json.RawMessage)
	for key, value := range raw {
		if recordReserved(key) {
			return nil, recordInvalid("Resource control %q is SDK owned", key)
		}
		if !recordKnown(key) || update && key == "id" {
			continue
		}
		if !utf8.Valid(value) || !json.Valid(value) {
			return nil, recordInvalid("group attribute %q must be UTF-8 JSON", key)
		}
		body[key] = bytes.Clone(value)
	}
	return body, nil
}

// GetRecord fetches one literal ID; list_volume is absent unless selected.
// An explicit selected microversion is retained, even above the Source cap.
func (a *API) GetRecord(ctx context.Context, id string, options ...RecordGetOption) (*GroupRecord, error) {
	const op = "GetRecord"
	p, err := a.capture(ctx)
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	if err = recordID(id); err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	v, err := applyRecord(p, RecordGetOpts{}, copyGet, slices.Clone(options))
	var seed map[string]json.RawMessage
	if err == nil {
		if _, exists := v.Fields["id"]; exists {
			err = recordInvalid("group get id seed cannot override the literal identity")
		}
	}
	if err == nil {
		seed, err = mutationFields(v.Fields, false)
	}
	if err == nil {
		err = p.policy(v.Microversion, v.Headers)
	}
	if err == nil {
		err = p.negotiate()
	}
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	target := p.source.Client.ServiceURL("groups", url.PathEscape(id))
	if v.ListVolume != nil {
		text := "False"
		if *v.ListVolume {
			text = "True"
		}
		target += "?list_volume=" + text
	}
	r, err := p.exchange(http.MethodGet, target, nil)
	rawID, _ := json.Marshal(id)
	seed["id"] = rawID
	value, err := p.response(r, seed, err)
	return value, recordWrap(ctx, op, err)
}

// CreateRecord posts known Source attrs and preserves explicit null/zero/false.
func (a *API) CreateRecord(ctx context.Context, options ...RecordCreateOption) (*GroupRecord, error) {
	const op = "CreateRecord"
	p, err := a.capture(ctx)
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	v, err := applyRecord(p, RecordCreateOpts{}, copyCreate, slices.Clone(options))
	var fields map[string]json.RawMessage
	if err == nil {
		fields, err = mutationFields(v.Fields, false)
	}
	if err == nil {
		err = p.policy(v.Microversion, v.Headers)
	}
	if err == nil {
		err = p.negotiate()
	}
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	r, err := p.exchange(http.MethodPost, p.source.Client.ServiceURL("groups"), map[string]any{"group": fields})
	value, err := p.response(r, fields, err)
	return value, recordWrap(ctx, op, err)
}

// CreateFromSourceRecord requires both nullable source-ID options. Unlike
// create_group, it posts create-from-src to the collection action route and
// seeds no response attributes when the accepted body omits them.
func (a *API) CreateFromSourceRecord(ctx context.Context, options ...RecordSourceOption) (*GroupRecord, error) {
	const op = "CreateFromSourceRecord"
	p, err := a.capture(ctx)
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	v, err := applyRecord(p, RecordSourceOpts{}, copySource, slices.Clone(options))
	fields := map[string]json.RawMessage{"group_snapshot_id": v.GroupSnapshotID, "source_group_id": v.SourceGroupID, "name": v.Name, "description": v.Description}
	if err == nil {
		if len(v.GroupSnapshotID) == 0 || len(v.SourceGroupID) == 0 {
			err = recordInvalid("both nullable source IDs must be supplied")
		}
		for _, key := range []string{"name", "description"} {
			if len(fields[key]) == 0 {
				fields[key] = json.RawMessage("null")
			}
		}
		for key, raw := range fields {
			if !utf8.Valid(raw) || !json.Valid(raw) {
				err = errors.Join(err, recordInvalid("source attribute %q must be UTF-8 JSON", key))
			}
		}
	}
	if err == nil {
		err = p.policy(v.Microversion, v.Headers)
	}
	if err == nil {
		err = p.negotiate()
	}
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	r, err := p.exchange(http.MethodPost, p.source.Client.ServiceURL("groups", "action"), map[string]any{"create-from-src": fields})
	value, err := p.response(r, nil, err)
	return value, recordWrap(ctx, op, err)
}

// UpdateRecord sends only known dirty attrs. No attrs, unknown-only attrs and
// an ID-only update return an owned local seed without HTTP or discovery.
func (a *API) UpdateRecord(ctx context.Context, id string, options ...RecordUpdateOption) (*GroupRecord, error) {
	const op = "UpdateRecord"
	p, err := a.capture(ctx)
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	if err = recordID(id); err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	v, err := applyRecord(p, RecordUpdateOpts{}, copyUpdate, slices.Clone(options))
	var fields map[string]json.RawMessage
	if err == nil {
		fields, err = mutationFields(v.Fields, true)
	}
	if err == nil {
		err = p.policy(v.Microversion, v.Headers)
	}
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	rawID, _ := json.Marshal(id)
	seed := recordClone(fields)
	seed["id"] = rawID
	if len(fields) == 0 {
		value := &GroupRecord{}
		err = errors.Join(recordProject(value, seed), p.guard(ctx))
		return value, recordWrap(ctx, op, err)
	}
	if err = p.negotiate(); err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	r, err := p.exchange(http.MethodPut, p.source.Client.ServiceURL("groups", url.PathEscape(id)), map[string]any{"group": fields})
	value, err := p.response(r, seed, err)
	return value, recordWrap(ctx, op, err)
}

// DeleteRecord uses POST /groups/{id}/action, never a DELETE request. It sends
// delete-volumes:false by default and ignores only a clean native404.
func (a *API) DeleteRecord(ctx context.Context, id string, options ...RecordDeleteOption) error {
	const op = "DeleteRecord"
	p, err := a.capture(ctx)
	if err != nil {
		return recordWrap(ctx, op, err)
	}
	if err = recordID(id); err != nil {
		return recordWrap(ctx, op, err)
	}
	v, err := applyRecord(p, RecordDeleteOpts{}, copyDelete, slices.Clone(options))
	if err == nil {
		err = p.policy(v.Microversion, v.Headers)
	}
	if err == nil {
		err = p.negotiate()
	}
	if err != nil {
		return recordWrap(ctx, op, err)
	}
	deleteVolumes := v.DeleteVolumes != nil && *v.DeleteVolumes
	_, err = p.exchange(http.MethodPost, p.source.Client.ServiceURL("groups", url.PathEscape(id), "action"), map[string]any{"delete": map[string]bool{"delete-volumes": deleteVolumes}})
	err = errors.Join(err, p.guard(ctx))
	if v.IgnoreMissing == nil || *v.IgnoreMissing {
		err = resource.IgnoreMissing(err)
	}
	return recordWrap(ctx, op, err)
}

// ListRecords lazily applies declared query aliases and local Body filters.
// details defaults to true; raw-row limits run before local filtering.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*GroupRecord, error] {
	owned := slices.Clone(options)
	return func(yield func(*GroupRecord, error) bool) {
		const op = "ListRecords"
		fail := func(err error) { yield(nil, recordWrap(ctx, op, err)) }
		p, err := a.capture(ctx)
		if err != nil {
			fail(err)
			return
		}
		v, err := applyRecord(p, RecordListOpts{}, copyList, owned)
		var query url.Values
		var filters map[string]json.RawMessage
		var control rest.ListControl
		if err == nil {
			query, filters, control, err = prepareList(v)
		}
		if err == nil {
			headers := v.Headers
			if headers == nil {
				headers = make(map[string]string)
			}
			hasAccept := false
			for key := range headers {
				if strings.EqualFold(key, "Accept") {
					hasAccept = true
					break
				}
			}
			if !hasAccept {
				headers["Accept"] = "application/json"
			}
			err = p.policy(v.Microversion, headers)
		}
		if err == nil {
			err = p.negotiate()
		}
		if err != nil {
			fail(err)
			return
		}
		path := "groups/detail"
		if v.Details != nil && !*v.Details {
			path = "groups"
		}
		parsed, _ := url.Parse(p.source.Client.ServiceURL(path))
		var page *rest.Response
		spec := rest.CollectionSpec[GroupRecord]{Client: &p.source.Client, Path: path, Kind: "groups", PluralKey: "groups", Metadata: recordMetadata, Validate: p.guard, SourceGuard: p.guard, ListCodes: recordCodes(100, 400), ReadPage: func(_ context.Context, target string, _ ...int) (*rest.Response, error) {
			return p.exchange(http.MethodGet, target, nil)
		}, ValidateResponse: func(r *rest.Response) error { page = r; return p.guard(ctx) }, ValidateItem: func(value *GroupRecord) error {
			value.Envelope = bytes.Clone(page.Body)
			value.Header, value.StatusCode = value.Wire.Header.Clone(), value.Wire.StatusCode
			p.proof(value)
			return errors.Join(recordProject(value, value.fields), p.guard(ctx))
		}, Paging: rest.PagePolicy[GroupRecord]{VersionedPath: parsed.EscapedPath(), HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, MarkerOrOffsetPagination: true, MarkerFallback: true, Marker: recordMarker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true}}
		keys := make([]string, 0, len(filters))
		for key := range filters {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for value, err := range rest.ListWithControl(ctx, spec, query, control) {
			if err != nil {
				fail(err)
				return
			}
			matched := true
			for _, key := range keys {
				match, cause := recordMatch(value.Resource.Body[key], filters[key])
				if cause = errors.Join(cause, p.guard(ctx)); cause != nil {
					fail(page.Fail(cause))
					return
				}
				if !match {
					matched = false
					break
				}
			}
			if matched && !yield(value, nil) {
				return
			}
		}
	}
}

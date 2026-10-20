package consistencygroups

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

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

// API shares the selected authenticated Cinder client without taking ownership.
// Legacy routes are independent of generic volume groups and group types.
type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }
func (a *API) RawClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}

type recordPrepared struct {
	ctx    context.Context
	source *cloudread.Source
	guard  func(context.Context) error
}

func (a *API) capture(ctx context.Context) (*recordPrepared, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, recordInvalid("consistency group API required")
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "volume")
	if err != nil {
		return nil, err
	}
	if source.Client.Microversion != "" {
		if _, err := microversions.ParseVersion(source.Client.Microversion); err != nil {
			return nil, recordInvalid("invalid selected consistency group microversion: %v", err)
		}
	}
	guard := func(ctx context.Context) error {
		var changed error
		if a.client != selected {
			changed = recordInvalid("selected Cinder consistency group client changed")
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
			return value, recordInvalid("nil consistency group option")
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
		return recordInvalid("nonblank UTF-8 consistency group identity required")
	}
	for _, char := range id {
		if unicode.IsControl(char) {
			return recordInvalid("consistency group identity has controls")
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
	return request.Wrap(op, "consistencygroups", cloudread.ContextError(ctx, err))
}
func (p *recordPrepared) policy(version *string, headers map[string]string) error {
	if version != nil && *version != "" {
		if _, err := microversions.ParseVersion(*version); err != nil {
			return recordInvalid("invalid option microversion: %v", err)
		}
	}
	for key := range headers {
		if strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Volume-API-Version") || strings.EqualFold(key, "X-OpenStack-Nova-API-Version") {
			return recordInvalid("version header %q is SDK owned", key)
		}
	}
	if version != nil {
		for key := range p.source.Client.MoreHeaders {
			if strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Volume-API-Version") {
				delete(p.source.Client.MoreHeaders, key)
			}
		}
	}
	return errors.Join(p.source.WithPolicy(p.ctx, version, headers), p.guard(p.ctx))
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
	return rest.DoJSONGuardedRejectionsHeaderPolicy(p.ctx, &p.source.Client, p.guard, method, target, body, policy, rest.RejectionPolicy{Codes: recordCodes(400, 600), PreserveCleanRetry: true, StopFaultRetryCallbacks: true}, recordCodes(100, 400)...)
}
func recordNativeOriginal(err error, native *gophercloud.ErrUnexpectedResponseCode) bool {
	var authentication *gophercloud.ErrUnableToReauthenticate
	if errors.As(err, &authentication) {
		err = authentication.ErrOriginal
	}
	return errors.As(err, native)
}
func (p *recordPrepared) response(r *rest.Response, seed map[string]json.RawMessage, err error) (*ConsistencyGroupRecord, error) {
	if err != nil {
		if r == nil {
			var native gophercloud.ErrUnexpectedResponseCode
			if recordNativeOriginal(err, &native) {
				r = &rest.Response{Body: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual}
			}
		}
		return recordReceipt(r), err
	}
	value, err := recordResponse(r, seed)
	if err = errors.Join(err, p.guard(p.ctx)); err != nil {
		err = r.Fail(err)
	}
	return value, err
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
			return nil, recordInvalid("consistency group attribute %q must be UTF-8 JSON", key)
		}
		body[key] = bytes.Clone(value)
	}
	return body, nil
}

// GetRecord fetches a literal ID without a list query or default discovery.
// Selected versions are retained without a class ceiling, matching Source.
func (a *API) GetRecord(ctx context.Context, id string, options ...RecordGetOption) (*ConsistencyGroupRecord, error) {
	const op = "GetRecord"
	p, err := a.capture(ctx)
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	if err = recordID(id); err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	v, err := applyRecord(p, RecordGetOpts{}, copyGet, slices.Clone(options))
	if err == nil {
		err = p.policy(v.Microversion, v.Headers)
	}
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	value, err := p.getPrepared(id)
	return value, recordWrap(ctx, op, err)
}
func (p *recordPrepared) getPrepared(id string) (*ConsistencyGroupRecord, error) {
	r, err := p.exchange(http.MethodGet, p.source.Client.ServiceURL("consistencygroups", url.PathEscape(id)), nil)
	rawID, _ := json.Marshal(id)
	return p.response(r, map[string]json.RawMessage{"id": rawID}, err)
}

// CreateRecord posts known Source attrs and preserves explicit null/zero/false.
func (a *API) CreateRecord(ctx context.Context, options ...RecordCreateOption) (*ConsistencyGroupRecord, error) {
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
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	r, err := p.exchange(http.MethodPost, p.source.Client.ServiceURL("consistencygroups"), map[string]any{"consistencygroup": fields})
	value, err := p.response(r, fields, err)
	return value, recordWrap(ctx, op, err)
}

// CreateFromSourceRecord retains four null defaults. Source does not preflight
// whether exactly one source is present; stock rejections retain actual proof.
func (a *API) CreateFromSourceRecord(ctx context.Context, options ...RecordSourceOption) (*ConsistencyGroupRecord, error) {
	const op = "CreateFromSourceRecord"
	p, err := a.capture(ctx)
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	v, err := applyRecord(p, RecordSourceOpts{}, copySource, slices.Clone(options))
	for _, text := range []*string{v.ConsistencyGroupSnapshotID, v.ConsistencyGroupID, v.Name, v.Description} {
		if text != nil && !utf8.ValidString(*text) {
			err = errors.Join(err, recordInvalid("source text must be UTF-8"))
		}
	}
	if err == nil {
		err = p.policy(v.Microversion, v.Headers)
	}
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	var snapshotID, groupID any
	if v.ConsistencyGroupSnapshotID != nil && *v.ConsistencyGroupSnapshotID != "" {
		snapshotID = *v.ConsistencyGroupSnapshotID
	}
	if v.ConsistencyGroupID != nil && *v.ConsistencyGroupID != "" {
		groupID = *v.ConsistencyGroupID
	}
	fields := map[string]any{"cgsnapshot_id": snapshotID, "source_cgid": groupID, "name": v.Name, "description": v.Description}
	r, err := p.exchange(http.MethodPost, p.source.Client.ServiceURL("consistencygroups", "create_from_src"), map[string]any{"consistencygroup-from-src": fields})
	returnValue, err := p.response(r, nil, err)
	return returnValue, recordWrap(ctx, op, err)
}

// UpdateRecord sends only known dirty attrs. No attrs, unknown-only attrs and
// an ID-only update return an owned local seed without HTTP or discovery.
func (a *API) UpdateRecord(ctx context.Context, id string, options ...RecordUpdateOption) (*ConsistencyGroupRecord, error) {
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
		value := &ConsistencyGroupRecord{}
		err = errors.Join(recordProject(value, seed), p.guard(ctx))
		return value, recordWrap(ctx, op, err)
	}
	r, err := p.exchange(http.MethodPut, p.source.Client.ServiceURL("consistencygroups", url.PathEscape(id)), map[string]any{"consistencygroup": fields})
	value, err := p.response(r, seed, err)
	return value, recordWrap(ctx, op, err)
}

// DeleteRecord corrects Source's top-level spelling for stock Cinder.
// PinnedSourceWire opts into the original consistency_group body. Missing is
// never ignored: the named Source operation has no ignore_missing argument.
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
	if err != nil {
		return recordWrap(ctx, op, err)
	}
	key := "consistencygroup"
	if v.PinnedSourceWire {
		key = "consistency_group"
	}
	force := v.Force != nil && *v.Force
	_, err = p.exchange(http.MethodPost, p.source.Client.ServiceURL("consistencygroups", url.PathEscape(id), "delete"), map[string]any{key: map[string]bool{"force": force}})
	return recordWrap(ctx, op, errors.Join(err, p.guard(ctx)))
}

// ListRecords lazily applies declared query aliases and local Body filters.
// details defaults to true; raw-row limits run before local filtering.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*ConsistencyGroupRecord, error] {
	owned := slices.Clone(options)
	return func(yield func(*ConsistencyGroupRecord, error) bool) {
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
			err = p.prepareListPolicy(v)
		}
		if err != nil {
			fail(err)
			return
		}
		for value, err := range p.listPrepared(v, query, filters, control, nil) {
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

func (p *recordPrepared) prepareListPolicy(v RecordListOpts) error {
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
	if err := p.policy(v.Microversion, headers); err != nil {
		return err
	}
	return p.guard(p.ctx)
}

// listPrepared reuses frozen record/pager behavior with a fully prepared
// source and validated query/filter/control. Optional observation is an actual
// physical response, never a synthetic record or Find-specific type.
func (p *recordPrepared) listPrepared(v RecordListOpts, query url.Values, filters map[string]json.RawMessage, control rest.ListControl, observe func(*rest.Response)) iter.Seq2[*ConsistencyGroupRecord, error] {
	return func(yield func(*ConsistencyGroupRecord, error) bool) {
		ctx := p.ctx
		fail := func(err error) { yield(nil, err) }
		path := "consistencygroups/detail"
		if v.Details != nil && !*v.Details {
			path = "consistencygroups"
		}
		parsed, _ := url.Parse(p.source.Client.ServiceURL(path))
		var page *rest.Response
		spec := rest.CollectionSpec[ConsistencyGroupRecord]{Client: &p.source.Client, Path: path, Kind: "consistencygroups", PluralKey: "consistencygroups", Metadata: recordMetadata, Validate: p.guard, SourceGuard: p.guard, ListCodes: recordCodes(100, 400), ReadPage: func(_ context.Context, target string, _ ...int) (*rest.Response, error) {
			return p.exchange(http.MethodGet, target, nil)
		}, ValidateResponse: func(r *rest.Response) error {
			page = r
			if observe != nil {
				observe(r)
			}
			return p.guard(ctx)
		}, ValidateItem: func(value *ConsistencyGroupRecord) error {
			value.Envelope = bytes.Clone(page.Body)
			value.Header, value.StatusCode = value.Wire.Header.Clone(), value.Wire.StatusCode
			return errors.Join(recordProject(value, value.fields), p.guard(ctx))
		}, Paging: rest.PagePolicy[ConsistencyGroupRecord]{VersionedPath: parsed.EscapedPath(), LinkKeys: []string{"links", "consistencygroups_links", "consistencygroup_links"}, HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, NextTruthy: cloudfilter.PythonJSONTruthy, MarkerOrOffsetPagination: true, MarkerFallback: true, Marker: recordMarker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true}}
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

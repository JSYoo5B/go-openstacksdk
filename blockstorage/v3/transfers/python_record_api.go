package transfers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type preparedRecord struct {
	ctx       context.Context
	source    *cloudread.Source
	guard     func(context.Context) error
	path      string
	discovery []DiscoveryObservation
}

func recordCodes() []int {
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = 200 + i
	}
	return codes
}
func recordID(id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." || !utf8.ValidString(id) {
		return recordInvalid("transfer ID must be a nonblank UTF-8 segment")
	}
	for _, c := range id {
		if unicode.IsControl(c) {
			return recordInvalid("transfer ID has controls")
		}
	}
	return nil
}
func recordWrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "transfers", cloudread.ContextError(ctx, err))
}
func (a *API) captureRecord(ctx context.Context) (*preparedRecord, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, recordInvalid("transfer API is required")
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "volume")
	if err != nil {
		return nil, err
	}
	if source.Client.Microversion != "" {
		if _, err := microversions.ParseVersion(source.Client.Microversion); err != nil {
			return nil, recordInvalid("invalid selected microversion: %v", err)
		}
	}
	guard := func(ctx context.Context) error {
		var replaced error
		if a.client != selected {
			replaced = recordInvalid("selected transfer client changed")
		}
		return errors.Join(replaced, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	return &preparedRecord{ctx: ctx, source: source, guard: guard}, guard(ctx)
}

// Discovery determines support independently of the selected default version,
// as Source supports_microversion does. An absent default selects at most3.55;
// an explicit default is retained, including defaults above the Resource cap.
func (p *preparedRecord) prepare(headers map[string]string) error {
	if err := errors.Join(p.source.WithPolicy(p.ctx, nil, headers), p.guard(p.ctx)); err != nil {
		return err
	}
	selected := p.source.Client.Microversion
	discovered, err := microversions.Read(p.ctx, p.source, microversions.CinderProfile)
	for _, response := range discovered.Responses {
		p.discovery = append(p.discovery, DiscoveryObservation{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode})
	}
	if err != nil {
		return err
	}
	supported, err := microversions.Supports(discovered.Maximum, discovered.Minimum, selected, "3.55")
	if err != nil {
		return recordInvalid("transfer support: %v", err)
	}
	p.path = "os-volume-transfer"
	if supported {
		p.path = "volume-transfers"
	}
	version := selected
	if version == "" {
		version, err = microversions.SelectVersion(discovered.Maximum, discovered.Minimum, "3.55")
		if err != nil {
			return recordInvalid("transfer microversion: %v", err)
		}
		if err = p.source.WithPolicy(p.ctx, &version, nil); err != nil {
			return err
		}
	}
	return p.guard(p.ctx)
}
func (p *preparedRecord) proof(v *TransferRecord) *TransferRecord {
	if v == nil {
		return nil
	}
	v.Discovery = make([]DiscoveryObservation, len(p.discovery))
	for i, observed := range p.discovery {
		v.Discovery[i] = DiscoveryObservation{Envelope: bytes.Clone(observed.Envelope), Header: observed.Header.Clone(), StatusCode: observed.StatusCode}
	}
	return v
}
func (p *preparedRecord) request(method, endpoint string, body any, rejected ...int) (*rest.Response, error) {
	policy := rest.RequestHeaderPolicy{Values: make(map[string]string)}
	if version := p.source.Client.Microversion; version != "" {
		policy.Values["OpenStack-API-Version"] = "volume " + version
		policy.Values["X-OpenStack-Volume-API-Version"] = version
	} else {
		policy.Absent = []string{"OpenStack-API-Version", "X-OpenStack-Volume-API-Version"}
	}
	return rest.DoJSONGuardedRejectionsHeaderPolicy(p.ctx, &p.source.Client, p.guard, method, endpoint, body, policy, rest.RejectionPolicy{Codes: rejected, PreserveCleanRetry: true}, recordCodes()...)
}
func (p *preparedRecord) response(r *rest.Response, fields map[string]json.RawMessage, err error) (*TransferRecord, error) {
	if err != nil {
		if r == nil {
			return nil, err
		}
		return p.proof(&TransferRecord{Envelope: bytes.Clone(r.Body), Header: r.Header.Clone(), StatusCode: r.StatusCode}), err
	}
	value, err := recordFromResponse(r, fields)
	if err = errors.Join(err, p.guard(p.ctx)); err != nil {
		err = r.Fail(err)
	}
	return p.proof(value), err
}
func (p *preparedRecord) get(id string) (*TransferRecord, error) {
	if err := recordID(id); err != nil {
		return nil, err
	}
	r, err := p.request(http.MethodGet, p.source.Client.ServiceURL(p.path, url.PathEscape(id)), nil, 400, 403, 404)
	rawID, _ := json.Marshal(id)
	return p.response(r, map[string]json.RawMessage{"id": rawID}, err)
}

// GetRecord uses the same discovered route as create, delete and accept.
func (a *API) GetRecord(ctx context.Context, id string, options ...RecordReadOption) (*TransferRecord, error) {
	const op = "GetRecord"
	p, err := a.captureRecord(ctx)
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	if err = recordID(id); err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	policy, err := applyRecordOptions(ctx, p.guard, RecordReadOpts{}, copyRecordRead, slices.Clone(options))
	if err == nil {
		err = p.prepare(policy.Headers)
	}
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	value, err := p.get(id)
	return value, recordWrap(ctx, op, err)
}

// CreateRecord sends Source declared attrs with omission/null/false/empty values
// intact. It discovers 3.55 support, selecting the modern or legacy route.
func (a *API) CreateRecord(ctx context.Context, options ...RecordCreateOption) (*TransferRecord, error) {
	const op = "CreateRecord"
	p, err := a.captureRecord(ctx)
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	policy, err := applyRecordOptions(ctx, p.guard, RecordCreateOpts{}, copyRecordCreate, slices.Clone(options))
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	fields := make(map[string]json.RawMessage)
	for key, raw := range policy.Fields {
		if !recordKnown(key) {
			continue
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, recordWrap(ctx, op, recordInvalid("attribute %q must be UTF-8 JSON", key))
		}
		fields[key] = bytes.Clone(raw)
	}
	body := cloneRecordFields(fields)
	for key, raw := range policy.Extensions {
		if key == "" || !utf8.ValidString(key) || recordKnown(key) || !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, recordWrap(ctx, op, recordInvalid("invalid transfer extension %q", key))
		}
		body[key] = bytes.Clone(raw)
	}
	if err = p.prepare(policy.Headers); err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	if p.path == "os-volume-transfer" && policy.SuppressLegacySnapshots {
		delete(body, "no_snapshots")
	}
	r, err := p.request(http.MethodPost, p.source.Client.ServiceURL(p.path), map[string]any{"transfer": body})
	value, err := p.response(r, fields, err)
	return value, recordWrap(ctx, op, err)
}

// AcceptRecord sends auth_key:null when no option is supplied, matching the
// Source Resource method. Proxy callers supply WithRecordAcceptAuthKey(key).
func (a *API) AcceptRecord(ctx context.Context, id string, options ...RecordAcceptOption) (*TransferRecord, error) {
	const op = "AcceptRecord"
	p, err := a.captureRecord(ctx)
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	if err = recordID(id); err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	policy, err := applyRecordOptions(ctx, p.guard, RecordAcceptOpts{}, copyRecordAccept, slices.Clone(options))
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	authKey := policy.AuthKey
	if len(authKey) == 0 {
		authKey = json.RawMessage("null")
	}
	if !utf8.Valid(authKey) || !json.Valid(authKey) {
		return nil, recordWrap(ctx, op, recordInvalid("auth_key must be UTF-8 JSON"))
	}
	if err = p.prepare(policy.Headers); err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	r, err := p.request(http.MethodPost, p.source.Client.ServiceURL(p.path, url.PathEscape(id), "accept"), map[string]any{"accept": map[string]json.RawMessage{"auth_key": authKey}})
	// Source creates a new Transfer for accept, without seeding the input ID.
	value, err := p.response(r, nil, err)
	return value, recordWrap(ctx, op, err)
}

// DeleteRecord defaults to ignoring only a clean404 and retains mixed causes.
func (a *API) DeleteRecord(ctx context.Context, id string, options ...RecordDeleteOption) error {
	const op = "DeleteRecord"
	p, err := a.captureRecord(ctx)
	if err != nil {
		return recordWrap(ctx, op, err)
	}
	if err = recordID(id); err != nil {
		return recordWrap(ctx, op, err)
	}
	policy, err := applyRecordOptions(ctx, p.guard, RecordDeleteOpts{}, copyRecordDelete, slices.Clone(options))
	if err == nil {
		err = p.prepare(policy.Headers)
	}
	if err != nil {
		return recordWrap(ctx, op, err)
	}
	_, err = p.request(http.MethodDelete, p.source.Client.ServiceURL(p.path, url.PathEscape(id)), nil, 404)
	if policy.IgnoreMissing == nil || *policy.IgnoreMissing {
		err = resource.IgnoreMissing(err)
	}
	return recordWrap(ctx, op, err)
}

func (p *preparedRecord) list(path string, query url.Values, filters map[string]json.RawMessage, control rest.ListControl) iter.Seq2[*TransferRecord, error] {
	return func(yield func(*TransferRecord, error) bool) {
		var origin *rest.Response
		// A Source version-prefixed continuation loses the leading /v segment.
		// Permit the one fixed catalog route alias, never another collection.
		parsed, _ := url.Parse(p.source.Client.ServiceURL(path))
		spec := rest.CollectionSpec[TransferRecord]{Client: &p.source.Client, Path: path, Kind: "transfers", PluralKey: "transfers", Metadata: recordMetadata, Validate: p.guard, SourceGuard: p.guard, ListCodes: recordCodes(), ReadPage: func(_ context.Context, target string, _ ...int) (*rest.Response, error) {
			return p.request(http.MethodGet, target, nil)
		}, ValidateResponse: func(r *rest.Response) error { origin = r; return p.guard(p.ctx) }, ValidateItem: func(v *TransferRecord) error {
			v.Envelope = bytes.Clone(origin.Body)
			v.Header, v.StatusCode = v.Wire.Header.Clone(), v.Wire.StatusCode
			projectRecord(v, nil)
			p.proof(v)
			return p.guard(p.ctx)
		}, Paging: rest.PagePolicy[TransferRecord]{VersionedPath: parsed.EscapedPath(), HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, MarkerFallback: true, Marker: recordMarker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true}}
		for value, err := range rest.ListWithControl(p.ctx, spec, query, control) {
			if err != nil {
				yield(nil, err)
				return
			}
			matched, err := jsonfilter.MatchFilters(value.Resource.Body, filters)
			if err = errors.Join(err, p.guard(p.ctx)); err != nil {
				yield(nil, origin.Fail(err))
				return
			}
			if matched && !yield(value, nil) {
				return
			}
		}
	}
}

// ListRecords mirrors transfers(details=True), including discovery, lazy pages,
// Source limit/marker queries and local declared Body filters. Unknown kwargs
// and all_projects are discarded as they are in the pinned Transfer Source.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*TransferRecord, error] {
	owned := slices.Clone(options)
	return func(yield func(*TransferRecord, error) bool) {
		const op = "ListRecords"
		p, err := a.captureRecord(ctx)
		if err != nil {
			yield(nil, recordWrap(ctx, op, err))
			return
		}
		policy, err := applyRecordOptions(ctx, p.guard, RecordListOpts{}, copyRecordList, owned)
		if err != nil {
			yield(nil, recordWrap(ctx, op, err))
			return
		}
		query, filters, control, err := prepareRecordList(policy)
		if err != nil {
			yield(nil, recordWrap(ctx, op, err))
			return
		}
		headers := policy.Headers
		if headers == nil {
			headers = make(map[string]string)
		}
		if _, ok := headers["Accept"]; !ok {
			headers["Accept"] = "application/json"
		}
		if err = p.prepare(headers); err != nil {
			yield(nil, recordWrap(ctx, op, err))
			return
		}
		path := p.path
		if policy.Details == nil || *policy.Details {
			path += "/detail"
		}
		for value, err := range p.list(path, query, filters, control) {
			if !yield(value, recordWrap(ctx, op, err)) {
				return
			}
		}
	}
}

// FindRecord follows GET-first clean400/403/404 fallback and exact ID/name
// resolution. Source Resource.find calls Transfer.list directly, whose summary
// base path remains /volume-transfers even when fetch chose the legacy route.
func (a *API) FindRecord(ctx context.Context, nameOrID string, options ...RecordFindOption) (*TransferRecord, error) {
	const op = "FindRecord"
	p, err := a.captureRecord(ctx)
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	if err = recordID(nameOrID); err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	policy, err := applyRecordOptions(ctx, p.guard, RecordFindOpts{}, copyRecordFind, slices.Clone(options))
	if err == nil {
		err = p.prepare(nil)
	}
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	collection := resource.NewCollection(resource.Adapter[TransferRecord]{Kind: "transfers", IdentityFind: true, ValidateID: recordID, IdentityDirectGet: recordID, IdentityResponseID: func(value *TransferRecord) (string, error) {
		if value == nil {
			return "", recordInvalid("find response missing")
		}
		return recordText(value, "id"), nil
	}, Name: func(value *TransferRecord) string { return recordText(value, "name") }, Get: func(_ context.Context, id string) (*TransferRecord, error) { return p.get(id) }, Iterate: func(context.Context, url.Values) iter.Seq2[*TransferRecord, error] {
		if p.source.Client.MoreHeaders == nil {
			p.source.Client.MoreHeaders = make(map[string]string)
		}
		if _, ok := p.source.Client.MoreHeaders["Accept"]; !ok {
			p.source.Client.MoreHeaders["Accept"] = "application/json"
		}
		return p.list("volume-transfers", nil, nil, rest.ListControl{})
	}})
	ignored := policy.IgnoreMissing == nil || *policy.IgnoreMissing
	value, err := collection.FindIdentity(ctx, nameOrID, resource.WithIdentityFindIgnoreMissing(ignored))
	err = errors.Join(err, p.guard(ctx))
	return value, recordWrap(ctx, op, err)
}

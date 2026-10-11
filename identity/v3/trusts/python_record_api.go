package trusts

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
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type preparedRecord struct {
	ctx    context.Context
	source *cloudread.Source
	guard  func(context.Context) error
}

func recordRejections() []int {
	codes := make([]int, 0, 700)
	for code := 100; code <= 999; code++ {
		if code < 200 || code >= 400 {
			codes = append(codes, code)
		}
	}
	return codes
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
		return recordInvalid("trust ID must be a nonblank UTF-8 segment")
	}
	for _, c := range id {
		if unicode.IsControl(c) {
			return recordInvalid("trust ID has controls")
		}
	}
	return nil
}
func recordWrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "trusts", cloudread.ContextError(ctx, err))
}
func (a *API) captureRecord(ctx context.Context) (*preparedRecord, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, recordInvalid("trust API is required")
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "identity")
	if err != nil {
		return nil, err
	}
	guard := func(ctx context.Context) error {
		var replaced error
		if a.client != selected {
			replaced = recordInvalid("selected trust client changed")
		}
		return errors.Join(replaced, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	return &preparedRecord{ctx: ctx, source: source, guard: guard}, guard(ctx)
}
func (p *preparedRecord) headers(values map[string]string) error {
	return errors.Join(p.source.WithPolicy(p.ctx, nil, values), p.guard(p.ctx))
}
func (p *preparedRecord) get(id string) (*TrustRecord, error) {
	if err := recordID(id); err != nil {
		return nil, err
	}
	response, err := rest.DoJSONGuardedRejections(p.ctx, &p.source.Client, p.guard, http.MethodGet, p.source.Client.ServiceURL("OS-TRUST", "trusts", url.PathEscape(id)), nil, nil, rest.RejectionPolicy{Codes: recordRejections(), PreserveCleanRetry: true}, recordCodes()...)
	if err != nil {
		if response == nil {
			return nil, err
		}
		return &TrustRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}, err
	}
	raw, _ := json.Marshal(id)
	value, err := recordFromResponse(response, map[string]json.RawMessage{"id": raw})
	if err = errors.Join(err, p.guard(p.ctx)); err != nil {
		err = response.Fail(err)
	}
	return value, err
}

// GetRecord fetches a literal /trusts member without native expiry/roles/count coercion.
func (a *API) GetRecord(ctx context.Context, id string, options ...RecordReadOption) (*TrustRecord, error) {
	const op = "GetRecord"
	p, err := a.captureRecord(ctx)
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	policy, err := applyRecordOptions(ctx, p.guard, RecordReadOpts{}, copyRecordRead, slices.Clone(options))
	if err == nil {
		err = p.headers(policy.Headers)
	}
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	v, err := p.get(id)
	return v, recordWrap(ctx, op, err)
}

// CreateRecord sends declared Python Trust Body attrs only. Missing values
// are omitted and explicit null/false/empty values remain in the request.
func (a *API) CreateRecord(ctx context.Context, options ...RecordCreateOption) (*TrustRecord, error) {
	const op = "CreateRecord"
	p, err := a.captureRecord(ctx)
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	policy, err := applyRecordOptions(ctx, p.guard, RecordCreateOpts{}, copyRecordCreate, slices.Clone(options))
	if err == nil {
		err = p.headers(policy.Headers)
	}
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	fields, err := prepareRecordCreate(policy.Fields)
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	response, err := rest.DoJSONGuardedRejections(ctx, &p.source.Client, p.guard, http.MethodPost, p.source.Client.ServiceURL("OS-TRUST", "trusts"), map[string]any{"trust": cloneRecordFields(fields)}, nil, rest.RejectionPolicy{Codes: recordRejections(), PreserveCleanRetry: true}, recordCodes()...)
	if err != nil {
		if response == nil {
			return nil, recordWrap(ctx, op, err)
		}
		return &TrustRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}, recordWrap(ctx, op, err)
	}
	v, err := recordFromResponse(response, fields)
	if err = errors.Join(err, p.guard(ctx)); err != nil {
		err = response.Fail(err)
	}
	return v, recordWrap(ctx, op, err)
}

// DeleteRecord defaults to ignoring a clean HTTP404, preserving all mixed causes.
func (a *API) DeleteRecord(ctx context.Context, id string, options ...RecordDeleteOption) error {
	const op = "DeleteRecord"
	p, err := a.captureRecord(ctx)
	if err != nil {
		return recordWrap(ctx, op, err)
	}
	policy, err := applyRecordOptions(ctx, p.guard, RecordDeleteOpts{}, copyRecordDelete, slices.Clone(options))
	if err == nil {
		err = recordID(id)
	}
	if err == nil {
		err = p.headers(policy.Headers)
	}
	if err != nil {
		return recordWrap(ctx, op, err)
	}
	_, err = rest.DoJSONGuardedRejections(ctx, &p.source.Client, p.guard, http.MethodDelete, p.source.Client.ServiceURL("OS-TRUST", "trusts", url.PathEscape(id)), nil, nil, rest.RejectionPolicy{Codes: recordRejections(), PreserveCleanRetry: true}, recordCodes()...)
	if policy.IgnoreMissing == nil || *policy.IgnoreMissing {
		err = resource.IgnoreMissing(err)
	}
	return recordWrap(ctx, op, err)
}

func (p *preparedRecord) list(query url.Values, filters map[string]json.RawMessage, control rest.ListControl) iter.Seq2[*TrustRecord, error] {
	return func(yield func(*TrustRecord, error) bool) {
		var origin *rest.Response
		spec := rest.CollectionSpec[TrustRecord]{Client: &p.source.Client, Path: "OS-TRUST/trusts", Kind: "trusts", PluralKey: "trusts", Metadata: recordMetadata, Validate: p.guard, SourceGuard: p.guard, ListCodes: recordCodes(), ReadPage: func(ctx context.Context, target string, codes ...int) (*rest.Response, error) {
			return rest.DoJSONGuardedRejections(ctx, &p.source.Client, p.guard, http.MethodGet, target, nil, nil, rest.RejectionPolicy{Codes: recordRejections(), PreserveCleanRetry: true}, codes...)
		}, ValidateResponse: func(r *rest.Response) error { origin = r; return p.guard(p.ctx) }, ValidateItem: func(v *TrustRecord) error {
			v.Envelope = bytes.Clone(origin.Body)
			v.Header, v.StatusCode = v.Wire.Header.Clone(), v.Wire.StatusCode
			return errors.Join(projectRecord(v, nil), p.guard(p.ctx))
		}, Paging: rest.PagePolicy[TrustRecord]{HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, MarkerFallback: true, Marker: recordMarker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true}}
		for v, err := range rest.ListWithControl(p.ctx, spec, query, control) {
			if err != nil {
				yield(nil, err)
				return
			}
			matched, err := jsonfilter.MatchFilters(v.Resource.Body, filters)
			if err = errors.Join(err, p.guard(p.ctx)); err != nil {
				yield(nil, origin.Fail(err))
				return
			}
			if matched && !yield(v, nil) {
				return
			}
		}
	}
}

// ListRecords splits Source trustor_user_id/trustee_user_id server query from other Body filters.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*TrustRecord, error] {
	owned := slices.Clone(options)
	return func(yield func(*TrustRecord, error) bool) {
		const op = "ListRecords"
		p, err := a.captureRecord(ctx)
		if err != nil {
			yield(nil, recordWrap(ctx, op, err))
			return
		}
		policy, err := applyRecordOptions(ctx, p.guard, RecordListOpts{}, copyRecordList, owned)
		if err == nil {
			err = p.headers(policy.Headers)
		}
		if err != nil {
			yield(nil, recordWrap(ctx, op, err))
			return
		}
		if _, ok := p.source.Client.MoreHeaders["Accept"]; !ok {
			p.source.Client.MoreHeaders["Accept"] = "application/json"
		}
		query, filters, control, err := prepareRecordList(policy)
		if err != nil {
			yield(nil, recordWrap(ctx, op, err))
			return
		}
		for v, err := range p.list(query, filters, control) {
			if !yield(v, recordWrap(ctx, op, err)) {
				return
			}
		}
	}
}

// FindRecord follows Python's GET-first, exact-ID-or-name fallback. It exposes
// the only Source find_trust option: IgnoreMissing, defaulting to true.
func (a *API) FindRecord(ctx context.Context, nameOrID string, options ...RecordFindOption) (*TrustRecord, error) {
	const op = "FindRecord"
	p, err := a.captureRecord(ctx)
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	policy, err := applyRecordOptions(ctx, p.guard, RecordFindOpts{}, copyRecordFind, slices.Clone(options))
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	collection := resource.NewCollection(resource.Adapter[TrustRecord]{Kind: "trusts", IdentityFind: true, ValidateID: recordID, IdentityDirectGet: recordID, IdentityResponseID: func(v *TrustRecord) (string, error) {
		if v == nil {
			return "", recordInvalid("find response missing")
		}
		return recordText(v, "id"), nil
	}, Name: func(v *TrustRecord) string { return recordText(v, "name") }, Get: func(_ context.Context, stringID string) (*TrustRecord, error) { return p.get(stringID) }, Iterate: func(context.Context, url.Values) iter.Seq2[*TrustRecord, error] {
		return p.list(nil, nil, rest.ListControl{})
	}})
	ignored := policy.IgnoreMissing == nil || *policy.IgnoreMissing
	value, err := collection.FindIdentity(ctx, nameOrID, resource.WithIdentityFindIgnoreMissing(ignored))
	err = errors.Join(err, p.guard(ctx))
	return value, recordWrap(ctx, op, err)
}

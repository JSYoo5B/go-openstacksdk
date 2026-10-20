package credentials

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

func recordCodes() []int {
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = 200 + i
	}
	return codes
}
func recordID(id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." || !utf8.ValidString(id) {
		return recordInvalid("credential ID must be a nonblank UTF-8 segment")
	}
	for _, c := range id {
		if unicode.IsControl(c) {
			return recordInvalid("credential ID has controls")
		}
	}
	return nil
}
func recordWrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "credentials", cloudread.ContextError(ctx, err))
}
func (a *API) captureRecord(ctx context.Context) (*preparedRecord, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, recordInvalid("credential API is required")
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "identity")
	if err != nil {
		return nil, err
	}
	guard := func(ctx context.Context) error {
		var replaced error
		if a.client != selected {
			replaced = recordInvalid("selected credential client changed")
		}
		return errors.Join(replaced, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	return &preparedRecord{ctx: ctx, source: source, guard: guard}, guard(ctx)
}
func (p *preparedRecord) headers(values map[string]string) error {
	return errors.Join(p.source.WithPolicy(p.ctx, nil, values), p.guard(p.ctx))
}
func (p *preparedRecord) get(id string) (*CredentialRecord, error) {
	if err := recordID(id); err != nil {
		return nil, err
	}
	response, err := rest.DoJSONGuardedRejections(p.ctx, &p.source.Client, p.guard, http.MethodGet, p.source.Client.ServiceURL("credentials", url.PathEscape(id)), nil, nil, rest.RejectionPolicy{Codes: []int{400, 403, 404}, PreserveCleanRetry: true}, recordCodes()...)
	if err != nil {
		if response == nil {
			return nil, err
		}
		return &CredentialRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}, err
	}
	raw, _ := json.Marshal(id)
	value, err := recordFromResponse(response, map[string]json.RawMessage{"id": raw})
	if err = errors.Join(err, p.guard(p.ctx)); err != nil {
		err = response.Fail(err)
	}
	return value, err
}

// GetRecord fetches a literal /credentials member without native field coercion.
func (a *API) GetRecord(ctx context.Context, id string, options ...RecordReadOption) (*CredentialRecord, error) {
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

// CreateRecord sends declared Python Credential Body attrs only. Missing values
// are omitted and explicit null/false/empty values remain in the request.
func (a *API) CreateRecord(ctx context.Context, options ...RecordCreateOption) (*CredentialRecord, error) {
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
	response, err := rest.DoJSONGuarded(ctx, &p.source.Client, p.guard, http.MethodPost, p.source.Client.ServiceURL("credentials"), map[string]any{"credential": cloneRecordFields(fields)}, nil, recordCodes()...)
	if err != nil {
		if response == nil {
			return nil, recordWrap(ctx, op, err)
		}
		return &CredentialRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}, recordWrap(ctx, op, err)
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
	_, err = rest.DoJSONGuardedRejections(ctx, &p.source.Client, p.guard, http.MethodDelete, p.source.Client.ServiceURL("credentials", url.PathEscape(id)), nil, nil, rest.RejectionPolicy{Codes: []int{404}, PreserveCleanRetry: true}, recordCodes()...)
	if policy.IgnoreMissing == nil || *policy.IgnoreMissing {
		err = resource.IgnoreMissing(err)
	}
	return recordWrap(ctx, op, err)
}

func (p *preparedRecord) list(query url.Values, filters map[string]json.RawMessage, control rest.ListControl) iter.Seq2[*CredentialRecord, error] {
	return func(yield func(*CredentialRecord, error) bool) {
		var origin *rest.Response
		spec := rest.CollectionSpec[CredentialRecord]{Client: &p.source.Client, Path: "credentials", Kind: "credentials", PluralKey: "credentials", Metadata: recordMetadata, Validate: p.guard, SourceGuard: p.guard, ListCodes: recordCodes(), ValidateResponse: func(r *rest.Response) error { origin = r; return p.guard(p.ctx) }, ValidateItem: func(v *CredentialRecord) error {
			v.Envelope = bytes.Clone(origin.Body)
			v.Header, v.StatusCode = v.Wire.Header.Clone(), v.Wire.StatusCode
			projectRecord(v, nil)
			return p.guard(p.ctx)
		}, Paging: rest.PagePolicy[CredentialRecord]{HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, MarkerFallback: true, Marker: recordMarker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true}}
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

// ListRecords splits Source type/user_id server query from other Body filters.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*CredentialRecord, error] {
	owned := slices.Clone(options)
	return func(yield func(*CredentialRecord, error) bool) {
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
// the only Source find_credential option: IgnoreMissing, defaulting to true.
func (a *API) FindRecord(ctx context.Context, nameOrID string, options ...RecordFindOption) (*CredentialRecord, error) {
	const op = "FindRecord"
	p, err := a.captureRecord(ctx)
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	policy, err := applyRecordOptions(ctx, p.guard, RecordFindOpts{}, copyRecordFind, slices.Clone(options))
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	collection := resource.NewCollection(resource.Adapter[CredentialRecord]{Kind: "credentials", IdentityFind: true, ValidateID: recordID, IdentityDirectGet: recordID, IdentityResponseID: func(v *CredentialRecord) (string, error) {
		if v == nil {
			return "", recordInvalid("find response missing")
		}
		return recordText(v, "id"), nil
	}, Name: func(v *CredentialRecord) string { return recordText(v, "name") }, Get: func(_ context.Context, stringID string) (*CredentialRecord, error) { return p.get(stringID) }, Iterate: func(context.Context, url.Values) iter.Seq2[*CredentialRecord, error] {
		return p.list(nil, nil, rest.ListControl{})
	}})
	ignored := policy.IgnoreMissing == nil || *policy.IgnoreMissing
	value, err := collection.FindIdentity(ctx, nameOrID, resource.WithIdentityFindIgnoreMissing(ignored))
	err = errors.Join(err, p.guard(ctx))
	return value, recordWrap(ctx, op, err)
}

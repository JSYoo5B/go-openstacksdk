package applicationcredentials

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
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
	userID string
}

func recordCodes() []int {
	v := make([]int, 200)
	for i := range v {
		v[i] = 200 + i
	}
	return v
}
func recordRouteID(id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." || !utf8.ValidString(id) {
		return recordInvalid("record identity must be a nonblank UTF-8 segment")
	}
	for _, c := range id {
		if unicode.IsControl(c) {
			return recordInvalid("record identity has controls")
		}
	}
	return nil
}
func (a *API) captureRecord(ctx context.Context, userID string) (*preparedRecord, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if err := resource.ID(userID).Validate(); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, recordInvalid("application credential API is required")
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "identity")
	if err != nil {
		return nil, err
	}
	guard := func(ctx context.Context) error {
		var replaced error
		if a.client != selected {
			replaced = recordInvalid("selected application credential client changed")
		}
		return errors.Join(replaced, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	return &preparedRecord{ctx: ctx, source: source, guard: guard, userID: userID}, guard(ctx)
}
func (p *preparedRecord) path(credential bool) string {
	leaf := "access_rules"
	if credential {
		leaf = "application_credentials"
	}
	return "users/" + url.PathEscape(p.userID) + "/" + leaf
}
func recordWrap(ctx context.Context, operation string, err error) error {
	if err != nil {
		err = cloudread.ContextError(ctx, err)
	}
	return request.Wrap(operation, "applicationcredentials", err)
}
func copyRead(v RecordReadOpts) RecordReadOpts { v.Headers = maps.Clone(v.Headers); return v }
func copyDelete(v RecordDeleteOpts) RecordDeleteOpts {
	v.Headers = maps.Clone(v.Headers)
	if v.IgnoreMissing != nil {
		x := *v.IgnoreMissing
		v.IgnoreMissing = &x
	}
	return v
}
func (p *preparedRecord) headers(v map[string]string) error {
	return errors.Join(p.source.WithPolicy(p.ctx, nil, v), p.guard(p.ctx))
}
func (p *preparedRecord) get(id string, credential bool, query url.Values) (*ApplicationCredentialRecord, error) {
	if err := recordRouteID(id); err != nil {
		return nil, err
	}
	target := p.source.Client.ServiceURL(p.path(credential), url.PathEscape(id))
	if len(query) != 0 {
		target += "?" + query.Encode()
	}
	response, err := rest.DoJSONGuardedRejections(p.ctx, &p.source.Client, p.guard, http.MethodGet, target, nil, nil, rest.RejectionPolicy{Codes: []int{400, 403, 404}, PreserveCleanRetry: true}, recordCodes()...)
	seedID, _ := json.Marshal(id)
	seed := map[string]json.RawMessage{"id": seedID}
	for key, values := range query {
		if key != "id" && recordKnown(key, credential) && len(values) == 1 {
			seed[key], _ = json.Marshal(values[0])
		}
	}
	if err != nil {
		if response == nil {
			return nil, err
		}
		return &ApplicationCredentialRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}, err
	}
	v, err := recordFromResponse(response, p.userID, credential, seed)
	if err = errors.Join(err, p.guard(p.ctx)); err != nil {
		return v, response.Fail(err)
	}
	return v, nil
}
func (a *API) getRecord(ctx context.Context, userID, id string, credential bool, options []RecordReadOption) (*ApplicationCredentialRecord, error) {
	p, err := a.captureRecord(ctx, userID)
	if err != nil {
		return nil, err
	}
	policy, err := applyRecordOptions(ctx, p.guard, RecordReadOpts{}, copyRead, slices.Clone(options))
	if err == nil {
		err = p.headers(policy.Headers)
	}
	if err != nil {
		return nil, err
	}
	return p.get(id, credential, nil)
}

// GetCredentialRecord fetches a literal member ID and preserves raw dates and fields.
func (a *API) GetCredentialRecord(ctx context.Context, userID, id string, options ...RecordReadOption) (*ApplicationCredentialRecord, error) {
	v, err := a.getRecord(ctx, userID, id, true, options)
	return v, recordWrap(ctx, "GetCredentialRecord", err)
}
func (a *API) GetAccessRuleRecord(ctx context.Context, userID, id string, options ...RecordReadOption) (*AccessRuleRecord, error) {
	v, err := a.getRecord(ctx, userID, id, false, options)
	return v, recordWrap(ctx, "GetAccessRuleRecord", err)
}

// CreateCredentialRecord sends only declared Body attributes, including explicit
// false/null/empty values. Unknown fields are discarded and name is explicit.
func (a *API) CreateCredentialRecord(ctx context.Context, userID, name string, options ...RecordCreateOption) (*ApplicationCredentialRecord, error) {
	const operation = "CreateCredentialRecord"
	p, err := a.captureRecord(ctx, userID)
	if err != nil {
		return nil, recordWrap(ctx, operation, err)
	}
	policy, err := applyRecordOptions(ctx, p.guard, RecordCreateOpts{}, copyRecordCreate, slices.Clone(options))
	if err == nil {
		err = p.headers(policy.Headers)
	}
	if err == nil && !utf8.ValidString(name) {
		err = recordInvalid("credential name must be UTF-8")
	}
	if err != nil {
		return nil, recordWrap(ctx, operation, err)
	}
	seed := make(map[string]json.RawMessage)
	seed["name"], _ = json.Marshal(name)
	for key, raw := range policy.Fields {
		if key == "user_id" {
			return nil, recordWrap(ctx, operation, recordInvalid("user_id is the fixed create argument"))
		}
		if !recordKnown(key, true) {
			continue
		}
		if key == "name" {
			return nil, recordWrap(ctx, operation, recordInvalid("name is the explicit create argument"))
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, recordWrap(ctx, operation, recordInvalid("attribute %q must be UTF-8 JSON", key))
		}
		seed[key] = bytes.Clone(raw)
	}
	response, err := rest.DoJSONGuarded(ctx, &p.source.Client, p.guard, http.MethodPost, p.source.Client.ServiceURL(p.path(true)), map[string]any{"application_credential": copyRecordFields(seed)}, nil, recordCodes()...)
	if err != nil {
		if response == nil {
			return nil, recordWrap(ctx, operation, err)
		}
		return &ApplicationCredentialRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}, recordWrap(ctx, operation, err)
	}
	value, err := recordFromResponse(response, userID, true, seed)
	if err = errors.Join(err, p.guard(ctx)); err != nil {
		err = response.Fail(err)
	}
	return value, recordWrap(ctx, operation, err)
}
func (a *API) deleteRecord(ctx context.Context, userID, id string, credential bool, options []RecordDeleteOption) error {
	p, err := a.captureRecord(ctx, userID)
	if err != nil {
		return err
	}
	policy, err := applyRecordOptions(ctx, p.guard, RecordDeleteOpts{}, copyDelete, slices.Clone(options))
	if err == nil {
		err = recordRouteID(id)
	}
	if err == nil {
		err = p.headers(policy.Headers)
	}
	if err != nil {
		return err
	}
	_, err = rest.DoJSONGuardedRejections(ctx, &p.source.Client, p.guard, http.MethodDelete, p.source.Client.ServiceURL(p.path(credential), url.PathEscape(id)), nil, nil, rest.RejectionPolicy{Codes: []int{404}, PreserveCleanRetry: true}, recordCodes()...)
	if policy.IgnoreMissing == nil || *policy.IgnoreMissing {
		err = resource.IgnoreMissing(err)
	}
	return err
}
func (a *API) DeleteCredentialRecord(ctx context.Context, userID, id string, options ...RecordDeleteOption) error {
	return recordWrap(ctx, "DeleteCredentialRecord", a.deleteRecord(ctx, userID, id, true, options))
}
func (a *API) DeleteAccessRuleRecord(ctx context.Context, userID, id string, options ...RecordDeleteOption) error {
	return recordWrap(ctx, "DeleteAccessRuleRecord", a.deleteRecord(ctx, userID, id, false, options))
}

func (p *preparedRecord) list(credential bool, query url.Values, filters map[string]json.RawMessage, control rest.ListControl) iter.Seq2[*ApplicationCredentialRecord, error] {
	return func(yield func(*ApplicationCredentialRecord, error) bool) {
		plural := "access_rules"
		if credential {
			plural = "application_credentials"
		}
		var origin *rest.Response
		spec := rest.CollectionSpec[ApplicationCredentialRecord]{Client: &p.source.Client, Path: p.path(credential), Kind: "applicationcredentials", PluralKey: plural, Metadata: recordMetadata, Validate: p.guard, SourceGuard: p.guard, ListCodes: recordCodes(), ValidateResponse: func(r *rest.Response) error { origin = r; return p.guard(p.ctx) }, ValidateItem: func(v *ApplicationCredentialRecord) error {
			v.Envelope = bytes.Clone(origin.Body)
			v.Header, v.StatusCode = v.Wire.Header.Clone(), v.Wire.StatusCode
			return errors.Join(projectRecord(v, p.userID, credential, nil), p.guard(p.ctx))
		}, Paging: rest.PagePolicy[ApplicationCredentialRecord]{HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, MarkerFallback: true, Marker: recordMarker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true}}
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
func (a *API) listRecords(ctx context.Context, userID string, credential bool, options []RecordListOption) iter.Seq2[*ApplicationCredentialRecord, error] {
	owned := slices.Clone(options)
	operation := "ListAccessRuleRecords"
	if credential {
		operation = "ListCredentialRecords"
	}
	return func(yield func(*ApplicationCredentialRecord, error) bool) {
		p, err := a.captureRecord(ctx, userID)
		if err != nil {
			yield(nil, recordWrap(ctx, operation, err))
			return
		}
		policy, err := applyRecordOptions(ctx, p.guard, RecordListOpts{}, copyRecordList, owned)
		if err == nil {
			err = p.headers(policy.Headers)
		}
		if err != nil {
			yield(nil, recordWrap(ctx, operation, err))
			return
		}
		if _, ok := p.source.Client.MoreHeaders["Accept"]; !ok {
			p.source.Client.MoreHeaders["Accept"] = "application/json"
		}
		query, filters, control, err := prepareRecordList(policy, credential)
		if err != nil {
			yield(nil, recordWrap(ctx, operation, err))
			return
		}
		for v, err := range p.list(credential, query, filters, control) {
			if !yield(v, recordWrap(ctx, operation, err)) {
				return
			}
		}
	}
}
func (a *API) ListCredentialRecords(ctx context.Context, userID string, options ...RecordListOption) iter.Seq2[*ApplicationCredentialRecord, error] {
	return a.listRecords(ctx, userID, true, options)
}
func (a *API) ListAccessRuleRecords(ctx context.Context, userID string, options ...RecordListOption) iter.Seq2[*AccessRuleRecord, error] {
	return a.listRecords(ctx, userID, false, options)
}

// FindCredentialRecord reuses the common GET-first Resource identity policy.
// IgnoreMissing defaults to true; direct400/403/404 may fall back to a complete
// exact ID-or-name list. Source sends the fixed user_id in the direct GET query.
func (a *API) FindCredentialRecord(ctx context.Context, userID, nameOrID string, options ...resource.IdentityFindOption) (*ApplicationCredentialRecord, error) {
	const operation = "FindCredentialRecord"
	p, err := a.captureRecord(ctx, userID)
	if err != nil {
		return nil, recordWrap(ctx, operation, err)
	}
	get := func(ctx context.Context, id string, q url.Values) (*ApplicationCredentialRecord, error) {
		if q.Has("user_id") || q.Has("id") {
			return nil, recordInvalid("find cannot replace the fixed user_id or member id")
		}
		owned := maps.Clone(q)
		if owned == nil {
			owned = make(url.Values)
		}
		owned.Set("user_id", userID)
		return p.get(id, true, owned)
	}
	collection := resource.NewCollection(resource.Adapter[ApplicationCredentialRecord]{Kind: "applicationcredentials", IdentityFind: true, ValidateID: recordRouteID, IdentityDirectGet: recordRouteID, IdentityResponseID: func(v *ApplicationCredentialRecord) (string, error) {
		if v == nil {
			return "", recordInvalid("find response is required")
		}
		return recordText(v, "id"), nil
	}, Name: func(v *ApplicationCredentialRecord) string { return recordText(v, "name") }, Get: func(ctx context.Context, id string) (*ApplicationCredentialRecord, error) { return get(ctx, id, nil) }, GetIdentityQuery: get, Iterate: func(ctx context.Context, q url.Values) iter.Seq2[*ApplicationCredentialRecord, error] {
		return func(yield func(*ApplicationCredentialRecord, error) bool) {
			policy := RecordListOpts{Filters: make(map[string]json.RawMessage)}
			for key, values := range q {
				if key == "user_id" {
					yield(nil, recordInvalid("find cannot replace the fixed user_id"))
					return
				}
				if len(values) != 1 {
					yield(nil, recordInvalid("find list query requires one value per key"))
					return
				}
				switch key {
				case "limit":
					n, err := strconv.Atoi(values[0])
					if err != nil {
						yield(nil, err)
						return
					}
					policy.Limit = &n
				case "marker":
					policy.Marker = values[0]
				default:
					if recordKnown(key, true) {
						policy.Filters[key], _ = json.Marshal(values[0])
					}
				}
			}
			query, filters, control, err := prepareRecordList(policy, true)
			if err != nil {
				yield(nil, err)
				return
			}
			for v, err := range p.list(true, query, filters, control) {
				if !yield(v, err) {
					return
				}
			}
		}
	}})
	value, err := collection.FindIdentity(ctx, nameOrID, slices.Clone(options)...)
	if guardErr := p.guard(ctx); guardErr != nil {
		err = errors.Join(err, guardErr)
	}
	return value, recordWrap(ctx, operation, err)
}

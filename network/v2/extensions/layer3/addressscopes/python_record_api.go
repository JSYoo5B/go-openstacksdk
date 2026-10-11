package addressscopes

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
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type preparedScopeRecord struct {
	ctx     context.Context
	source  *cloudread.Source
	guard   func(context.Context) error
	headers rest.RequestHeaderPolicy
}

func scopeRecordCodes(start int) []int {
	count := 200
	if start == 100 {
		count = 300
	}
	codes := make([]int, count)
	for i := range codes {
		codes[i] = start + i
	}
	return codes
}
func scopeRecordWrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "addressscopes", cloudread.ContextError(ctx, err))
}
func (a *API) captureScopeRecord(ctx context.Context) (*preparedScopeRecord, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, scopeRecordInvalid("addressscopes API is required")
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "network")
	if err != nil {
		return nil, err
	}
	var observed error
	guard := func(ctx context.Context) error {
		if observed == nil && a.client != selected {
			observed = scopeRecordInvalid("selected addressscopes client changed")
		}
		return errors.Join(cloudread.ContextError(ctx, observed), source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	p := &preparedScopeRecord{ctx: ctx, source: source, guard: guard}
	return p, guard(ctx)
}
func applyScopeRecordOptions[T any, O ~func(*T) error](ctx context.Context, guard func(context.Context) error, value T, copyValue func(T) T, options []O) (T, error) {
	for _, apply := range options {
		if err := guard(ctx); err != nil {
			return value, err
		}
		if apply == nil {
			return value, scopeRecordInvalid("nil record option")
		}
		next := copyValue(value)
		err := apply(&next)
		if err = errors.Join(err, guard(ctx)); err != nil {
			return value, err
		}
		value = copyValue(next)
	}
	return value, guard(ctx)
}
func (p *preparedScopeRecord) policy(version *string, headers map[string]string, list bool) error {
	if version != nil {
		// The captured matching version header belongs to the selected source.
		// An explicit override replaces it only on this operation's copy.
		p.source.Client.MoreHeaders = maps.Clone(p.source.Client.MoreHeaders)
		for key := range p.source.Client.MoreHeaders {
			if strings.EqualFold(key, "OpenStack-API-Version") {
				delete(p.source.Client.MoreHeaders, key)
			}
		}
	}
	if err := errors.Join(p.source.WithPolicy(p.ctx, version, headers), p.guard(p.ctx)); err != nil {
		return err
	}
	values := maps.Clone(p.source.Client.MoreHeaders)
	if values == nil {
		values = make(map[string]string)
	}
	// Publish exactly one operation-owned spelling, or enforce absence.
	for key := range values {
		if strings.EqualFold(key, "OpenStack-API-Version") {
			delete(values, key)
		}
	}
	if list {
		values["Accept"] = "application/json"
		for key, value := range headers {
			if strings.EqualFold(key, "Accept") {
				values["Accept"] = value
			}
		}
	}
	absent := []string(nil)
	if p.source.Client.Microversion != "" {
		values["OpenStack-API-Version"] = "network " + p.source.Client.Microversion
	} else {
		absent = []string{"OpenStack-API-Version"}
	}
	p.headers = rest.RequestHeaderPolicy{Values: values, Absent: absent}
	p.source.Client.MoreHeaders = maps.Clone(values)
	return p.guard(p.ctx)
}
func scopeRecordKey(key string, find bool) error {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) {
		return scopeRecordInvalid("query key requires nonblank UTF-8")
	}
	for _, char := range key {
		if unicode.IsControl(char) {
			return scopeRecordInvalid("query key contains controls")
		}
	}
	switch key {
	case "base_path", "list_base_path", "resource_type", "connection", "session", "microversion", "headers", "_synchronized", "self", "requires_id", "error_message", "skip_cache", "resource_response_key", "allow_unknown_params", "jmespath_filters", "__conflicting_attrs":
		return scopeRecordInvalid("%q is an SDK-owned control", key)
	case "id":
		if find {
			return scopeRecordInvalid("id kwargs collide with find identity")
		}
	}
	return nil
}
func scopeRecordPrepare(query map[string]json.RawMessage, order []string, identity *string) (url.Values, url.Values, map[string]json.RawMessage, map[string]json.RawMessage, rest.ListControl, error) {
	direct, list := make(url.Values), make(url.Values)
	seed, filters := make(map[string]json.RawMessage), make(map[string]json.RawMessage)
	control := rest.ListControl{LimitHint: true}
	for _, key := range order {
		raw := query[key]
		if err := scopeRecordKey(key, identity != nil); err != nil {
			return nil, nil, nil, nil, control, err
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, nil, nil, control, scopeRecordInvalid("query %q requires complete UTF-8 JSON", key)
		}
		if identity != nil && key == "all_projects" {
			continue
		}
		if identity != nil {
			values, err := cloudfilter.RequestQueryValues(raw)
			if err != nil {
				return nil, nil, nil, nil, control, err
			}
			if len(values) > 0 {
				direct[key] = slices.Clone(values)
			}
			seed[key] = bytes.Clone(raw)
		}
		switch key {
		case "max_items":
			var amount int
			if err := json.Unmarshal(raw, &amount); err != nil || amount < 0 {
				return nil, nil, nil, nil, control, scopeRecordInvalid("max_items requires nonnegative Go integer")
			}
			control.MaxItems = amount
		case "paginated":
			truthy, err := scopeRecordTruthy(raw)
			if err != nil {
				return nil, nil, nil, nil, control, err
			}
			control.SinglePage = !truthy
		default:
			declared := false
			for _, pair := range scopeRecordProfile.Query {
				if key == pair.Name {
					declared = true
				}
			}
			if scopeRecordKnown(key) && !declared {
				filters[key] = bytes.Clone(raw)
			}
		}
	}
	for _, pair := range scopeRecordProfile.Query {
		raw, exists := query[pair.Name]
		if !exists {
			raw, exists = query[pair.Wire]
		}
		if !exists {
			continue
		}
		values, err := cloudfilter.RequestQueryValues(raw)
		if err != nil {
			return nil, nil, nil, nil, control, err
		}
		delete(list, pair.Wire)
		if len(values) > 0 {
			list[pair.Wire] = slices.Clone(values)
		}
	}
	if identity != nil {
		if _, exists := query["name"]; !exists {
			list.Set("name", *identity)
		}
	}
	return direct, list, seed, filters, control, nil
}
func (p *preparedScopeRecord) get(id string, query url.Values, seed map[string]json.RawMessage) (*AddressScopeRecord, error) {
	if err := scopeRecordID(id); err != nil {
		return nil, err
	}
	owned := scopeRecordFields(seed)
	owned["id"], _ = json.Marshal(id)
	if err := scopeRecordProject(&AddressScopeRecord{}, owned); err != nil {
		return nil, errors.Join(err, p.guard(p.ctx))
	}
	target := p.source.Client.ServiceURL("address-scopes", url.PathEscape(id))
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	response, err := rest.DoJSONGuardedRejectionsHeaderPolicy(p.ctx, &p.source.Client, p.guard, http.MethodGet, target, nil, p.headers, rest.RejectionPolicy{Codes: scopeRecordCodes(400), PreserveCleanRetry: true}, scopeRecordCodes(100)...)
	if err != nil {
		return nil, err
	}
	value, err := scopeRecordResponse(response, owned)
	if err = errors.Join(err, p.guard(p.ctx)); err != nil {
		err = response.Fail(err)
	}
	return value, err
}

// GetRecord fetches a fixed literal ID, mapping canonical Fields to wire names.
// Seed is copied immutable state, not a mutable Resource cache; clean404 errors.
func (a *API) GetRecord(ctx context.Context, id string, options ...RecordReadOption) (*AddressScopeRecord, error) {
	const op = "GetRecord"
	p, err := a.captureScopeRecord(ctx)
	if err != nil {
		return nil, scopeRecordWrap(ctx, op, err)
	}
	policy, err := applyScopeRecordOptions(ctx, p.guard, RecordReadOpts{}, copyScopeRead, slices.Clone(options))
	if err != nil {
		return nil, scopeRecordWrap(ctx, op, err)
	}
	if err := scopeRecordID(id); err != nil {
		return nil, scopeRecordWrap(ctx, op, err)
	}
	for key, raw := range policy.Seed {
		if err := scopeRecordKey(key, false); err != nil {
			return nil, scopeRecordWrap(ctx, op, err)
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, scopeRecordWrap(ctx, op, scopeRecordInvalid("seed %q requires complete UTF-8 JSON", key))
		}
	}
	query := make(url.Values)
	for _, field := range policy.Fields {
		if !utf8.ValidString(field) {
			return nil, scopeRecordWrap(ctx, op, scopeRecordInvalid("fields require UTF-8 strings"))
		}
		wire := field
		for _, descriptor := range scopeRecordProfile.Body {
			if field == descriptor.Name {
				wire = descriptor.Wire
				break
			}
		}
		query.Add("fields", wire)
	}
	if err = p.policy(policy.Microversion, policy.Headers, false); err != nil {
		return nil, scopeRecordWrap(ctx, op, err)
	}
	value, err := p.get(id, query, policy.Seed)
	return value, scopeRecordWrap(ctx, op, err)
}
func (p *preparedScopeRecord) list(query url.Values, filters map[string]json.RawMessage, order []string, control rest.ListControl) iter.Seq2[*AddressScopeRecord, error] {
	return func(yield func(*AddressScopeRecord, error) bool) {
		var origin *rest.Response
		spec := rest.CollectionSpec[AddressScopeRecord]{Client: &p.source.Client, Path: "address-scopes", Kind: "addressscopes", PluralKey: "address_scopes", Metadata: scopeRecordMetadata, Validate: p.guard, SourceGuard: p.guard, ListCodes: scopeRecordCodes(100),
			ReadPage: func(ctx context.Context, target string, codes ...int) (*rest.Response, error) {
				return rest.DoJSONGuardedRejectionsHeaderPolicy(ctx, &p.source.Client, p.guard, http.MethodGet, target, nil, p.headers, rest.RejectionPolicy{Codes: scopeRecordCodes(400), PreserveCleanRetry: true}, codes...)
			},
			ValidateResponse: func(response *rest.Response) error { origin = response; return p.guard(p.ctx) },
			ValidateItem: func(value *AddressScopeRecord) error {
				for _, key := range []string{"connection", "microversion", "_synchronized"} {
					if _, exists := value.Wire.Body[key]; exists {
						return scopeRecordInvalid("list row %q collides with Resource.existing", key)
					}
				}
				value.Envelope = bytes.Clone(origin.Body)
				value.Header, value.StatusCode = value.Wire.Header.Clone(), value.Wire.StatusCode
				return errors.Join(scopeRecordProject(value, nil), p.guard(p.ctx))
			},
			Paging: rest.PagePolicy[AddressScopeRecord]{HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, MarkerFallback: true, Marker: scopeRecordMarker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true, VersionedPath: "/v2.0/address-scopes"},
		}
		for value, err := range rest.ListWithControl(p.ctx, spec, query, control) {
			if err != nil {
				yield(nil, err)
				return
			}
			matched := true
			for _, key := range order {
				filter, exists := filters[key]
				if !exists {
					continue
				}
				equal, cause := scopeRecordLocalMatch(value.Resource.Body[key], filter)
				if cause = errors.Join(cause, p.guard(p.ctx)); cause != nil {
					yield(nil, origin.Fail(cause))
					return
				}
				if !equal {
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

// ListRecords is lazy and reusable. It splits declared server query and Body
// local filters, discards unknown list kwargs, and owns paging/actual receipts.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*AddressScopeRecord, error] {
	owned := slices.Clone(options)
	return func(yield func(*AddressScopeRecord, error) bool) {
		const op = "ListRecords"
		p, err := a.captureScopeRecord(ctx)
		if err != nil {
			yield(nil, scopeRecordWrap(ctx, op, err))
			return
		}
		policy, err := applyScopeRecordOptions(ctx, p.guard, RecordListOpts{}, copyScopeList, owned)
		if err == nil {
			err = p.policy(policy.Microversion, policy.Headers, true)
		}
		if err != nil {
			yield(nil, scopeRecordWrap(ctx, op, err))
			return
		}
		_, query, _, filters, control, err := scopeRecordPrepare(policy.Query, policy.queryOrder, nil)
		if err != nil {
			yield(nil, scopeRecordWrap(ctx, op, err))
			return
		}
		for value, err := range p.list(query, filters, policy.queryOrder, control) {
			if !yield(value, scopeRecordWrap(ctx, op, err)) {
				return
			}
		}
	}
}

// FindRecord implements GET-first exact ID-or-name search. Only clean400/403/404
// starts a complete fallback list; default logical missing is nil, duplicates
// and late page/physical failures invalidate earlier candidates.
func (a *API) FindRecord(ctx context.Context, identity string, options ...RecordFindOption) (*AddressScopeRecord, error) {
	const op = "FindRecord"
	p, err := a.captureScopeRecord(ctx)
	if err != nil {
		return nil, scopeRecordWrap(ctx, op, err)
	}
	// Resource.find applies its per-call microversion only to fetch. Keep the
	// original selected list version/headers before any user callback runs.
	listSource := *p.source
	listSource.Client.MoreHeaders = maps.Clone(p.source.Client.MoreHeaders)
	policy, err := applyScopeRecordOptions(ctx, p.guard, RecordFindOpts{}, copyScopeFind, slices.Clone(options))
	if err != nil {
		return nil, scopeRecordWrap(ctx, op, err)
	}
	direct, list, seed, filters, control, err := scopeRecordPrepare(policy.Query, policy.queryOrder, &identity)
	if err != nil {
		return nil, scopeRecordWrap(ctx, op, err)
	}
	// Source constructor calls to_dict before the member request. Invalid
	// descriptor kwargs cannot be rescued by a later response overlay.
	if err = scopeRecordProject(&AddressScopeRecord{}, seed); err == nil {
		err = p.policy(policy.Microversion, policy.Headers, false)
	}
	if err != nil {
		return nil, scopeRecordWrap(ctx, op, errors.Join(err, p.guard(ctx)))
	}
	getHeaders := p.headers
	collection := resource.NewCollection(resource.Adapter[AddressScopeRecord]{Kind: "addressscopes", IdentityFind: true, ValidateID: scopeRecordID, IdentityDirectGet: scopeRecordID,
		IdentityResponseID: func(value *AddressScopeRecord) (string, error) { return scopeRecordText(value, "id"), nil }, Name: func(value *AddressScopeRecord) string { return scopeRecordText(value, "name") },
		Get: func(_ context.Context, id string) (*AddressScopeRecord, error) {
			p.headers = getHeaders
			return p.get(id, direct, seed)
		},
		Iterate: func(context.Context, url.Values) iter.Seq2[*AddressScopeRecord, error] {
			return func(yield func(*AddressScopeRecord, error) bool) {
				fallback := &preparedScopeRecord{ctx: ctx, source: &listSource, guard: p.guard}
				if err := fallback.policy(nil, policy.Headers, true); err != nil {
					yield(nil, err)
					return
				}
				for value, err := range fallback.list(list, filters, policy.queryOrder, control) {
					if !yield(value, err) {
						return
					}
				}
			}
		},
	})
	value, err := collection.FindIdentity(ctx, identity, resource.WithIdentityFindIgnoreMissing(policy.IgnoreMissing == nil || *policy.IgnoreMissing))
	return value, scopeRecordWrap(ctx, op, errors.Join(err, p.guard(ctx)))
}

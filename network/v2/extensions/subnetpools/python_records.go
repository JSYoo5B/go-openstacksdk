package subnetpools

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
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type preparedPoolRecord struct {
	source          *cloudread.Source
	guard           func(context.Context) error
	explicitHeaders map[string]string
}

func poolRecordID(id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." || !utf8.ValidString(id) {
		return poolRecordInvalid("nonblank UTF-8 identity required")
	}
	for _, char := range id {
		if unicode.IsControl(char) {
			return poolRecordInvalid("identity contains controls")
		}
	}
	return nil
}
func poolRecordCodes(first, last int) []int {
	codes := make([]int, last-first)
	for i := range codes {
		codes[i] = first + i
	}
	return codes
}
func poolRecordWrap(ctx context.Context, operation string, err error) error {
	return request.Wrap(operation, "subnetpools", cloudread.ContextError(ctx, err))
}
func (a *API) preparePoolRecord(ctx context.Context) (*preparedPoolRecord, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, poolRecordInvalid("configured API required")
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "network")
	if err != nil {
		return nil, err
	}
	var changed error
	guard := func(ctx context.Context) error {
		if changed == nil && a.client != selected {
			changed = poolRecordInvalid("selected Neutron client changed")
		}
		return errors.Join(changed, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	return &preparedPoolRecord{source: source, guard: guard}, guard(ctx)
}
func (p *preparedPoolRecord) withPolicy(ctx context.Context, version *string, headers map[string]string) error {
	if err := p.source.WithPolicy(ctx, version, headers); err != nil {
		return errors.Join(err, p.guard(ctx))
	}
	p.explicitHeaders = maps.Clone(headers)
	return p.guard(ctx)
}
func poolRecordHeaderPolicy(clientHeaders map[string]string, version string, body bool) rest.RequestHeaderPolicy {
	policy := rest.RequestHeaderPolicy{Values: make(map[string]string), Absent: []string{
		"Authorization", "X-Service-Token", "Cookie", "Host", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "TE", "Upgrade",
		"X-OpenStack-Nova-API-Version", "X-OpenStack-Volume-API-Version",
	}}
	if version == "" {
		policy.Absent = append(policy.Absent, "OpenStack-API-Version")
	} else {
		policy.Values["OpenStack-API-Version"] = "network " + version
	}
	if body {
		contentType := "application/json"
		for key, value := range clientHeaders {
			if strings.EqualFold(key, "Content-Type") {
				contentType = value
			}
		}
		policy.Values["Content-Type"] = contentType
	}
	return policy
}
func (p *preparedPoolRecord) read(ctx context.Context, method, target string, body any, list bool) (*rest.Response, error) {
	client := p.source.Client
	client.MoreHeaders = maps.Clone(client.MoreHeaders)
	if list {
		if client.MoreHeaders == nil {
			client.MoreHeaders = make(map[string]string)
		}
		client.MoreHeaders["Accept"] = "application/json"
		for key, value := range p.explicitHeaders {
			if strings.EqualFold(key, "Accept") {
				client.MoreHeaders["Accept"] = value
			}
		}
	}
	policy := poolRecordHeaderPolicy(client.MoreHeaders, client.Microversion, body != nil)
	return rest.DoJSONGuardedRejectionsHeaderPolicy(ctx, &client, p.guard, method, target, body, policy,
		rest.RejectionPolicy{Codes: poolRecordCodes(400, 600), PreserveCleanRetry: true}, poolRecordCodes(200, 400)...)
}
func (p *preparedPoolRecord) list(ctx context.Context, query url.Values, filters map[string]json.RawMessage, control rest.ListControl) iter.Seq2[*SubnetPoolRecord, error] {
	return func(yield func(*SubnetPoolRecord, error) bool) {
		var page *rest.Response
		spec := rest.CollectionSpec[SubnetPoolRecord]{Client: &p.source.Client, Path: "subnetpools", Kind: "subnetpools", PluralKey: "subnetpools", Metadata: poolRecordMetadata, Validate: p.guard, SourceGuard: p.guard, ListCodes: poolRecordCodes(200, 400),
			ReadPage: func(pageCtx context.Context, target string, codes ...int) (*rest.Response, error) {
				return p.read(pageCtx, http.MethodGet, target, nil, true)
			},
			ValidateResponse: func(response *rest.Response) error { page = response; return p.guard(ctx) },
			ValidateItem: func(value *SubnetPoolRecord) error {
				value.Envelope = bytes.Clone(page.Body)
				value.Header, value.StatusCode = value.Wire.Header.Clone(), value.Wire.StatusCode
				return errors.Join(poolRecordProject(value, value.fields), p.guard(ctx))
			},
			Paging: rest.PagePolicy[SubnetPoolRecord]{VersionedPath: "/v2.0/subnetpools", HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, MarkerFallback: true, Marker: poolRecordMarker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true},
		}
		keys := make([]string, 0, len(filters))
		for key := range filters {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for value, err := range rest.ListWithControl(ctx, spec, query, control) {
			if err != nil {
				yield(nil, err)
				return
			}
			matched := true
			for _, key := range keys {
				equal, cause := poolRecordLocalMatch(value.Resource.Body[key], filters[key])
				if cause = errors.Join(cause, p.guard(ctx)); cause != nil {
					yield(nil, page.Fail(cause))
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

// UpdateRecord commits only supplied canonical/wire Body attrs for a literal ID.
// No attrs performs no HTTP. Unknown if_match/if_revision attrs stay in Body;
// this SubnetPool does not inherit NetworkResource conditional headers.
func (a *API) UpdateRecord(ctx context.Context, id string, options ...RecordMutationOption) (*SubnetPoolRecord, error) {
	const operation = "UpdateRecord"
	p, err := a.preparePoolRecord(ctx)
	if err == nil {
		err = poolRecordID(id)
	}
	if err != nil {
		return nil, poolRecordWrap(ctx, operation, err)
	}
	policy, err := poolRecordOptions(ctx, p.guard, RecordMutationOpts{}, copyPoolMutation, slices.Clone(options))
	if err != nil {
		return nil, poolRecordWrap(ctx, operation, err)
	}
	body, seed, err := poolRecordMutation(policy)
	if err != nil {
		return nil, poolRecordWrap(ctx, operation, err)
	}
	seed["id"], _ = json.Marshal(id)
	initial := &SubnetPoolRecord{}
	if err := poolRecordProject(initial, seed); err != nil {
		return nil, poolRecordWrap(ctx, operation, err)
	}
	if err := p.withPolicy(ctx, policy.Microversion, policy.Headers); err != nil {
		return nil, poolRecordWrap(ctx, operation, err)
	}
	if len(body) == 0 {
		return initial, poolRecordWrap(ctx, operation, p.guard(ctx))
	}
	response, err := p.read(ctx, http.MethodPut, p.source.Client.ServiceURL("subnetpools", url.PathEscape(id)), map[string]any{"subnetpool": body}, false)
	if err != nil {
		return poolRecordReceipt(response), poolRecordWrap(ctx, operation, err)
	}
	value, err := poolRecordResponse(response, seed)
	if err = errors.Join(err, p.guard(ctx)); err != nil {
		return value, poolRecordWrap(ctx, operation, response.Fail(err))
	}
	return value, nil
}

// ListRecords lazily yields Source-shaped rows, with complete owned pagination
// and declared query/Body filtering. It does not enforce native required prefix
// lengths or synthesize id/unknown keys into the literal Wire observation.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*SubnetPoolRecord, error] {
	snapshot := slices.Clone(options)
	return func(yield func(*SubnetPoolRecord, error) bool) {
		p, err := a.preparePoolRecord(ctx)
		if err != nil {
			yield(nil, poolRecordWrap(ctx, "ListRecords", err))
			return
		}
		policy, err := poolRecordOptions(ctx, p.guard, RecordListOpts{}, copyPoolList, snapshot)
		if err != nil {
			yield(nil, poolRecordWrap(ctx, "ListRecords", err))
			return
		}
		_, query, filters, _, control, err := poolRecordQuery(policy, false, "")
		if err == nil {
			err = p.withPolicy(ctx, policy.Microversion, policy.Headers)
		}
		if err = errors.Join(err, p.guard(ctx)); err != nil {
			yield(nil, poolRecordWrap(ctx, "ListRecords", err))
			return
		}
		for value, err := range p.list(ctx, query, filters, control) {
			if !yield(value, poolRecordWrap(ctx, "ListRecords", err)) {
				return
			}
		}
	}
}

// AllRecords collects ListRecords and discards partial rows on any failure.
func (a *API) AllRecords(ctx context.Context, options ...RecordListOption) ([]*SubnetPoolRecord, error) {
	var values []*SubnetPoolRecord
	for value, err := range a.ListRecords(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// FindRecord follows Resource.find's literal member-first exact ID/name lookup.
// Only clean native400/403/404 fall back. Default missing follows a completely
// successful search; late duplicates, page and physical errors remain errors.
func (a *API) FindRecord(ctx context.Context, identity string, options ...RecordFindOption) (*SubnetPoolRecord, error) {
	const operation = "FindRecord"
	p, err := a.preparePoolRecord(ctx)
	if err == nil {
		err = poolRecordID(identity)
	}
	if err != nil {
		return nil, poolRecordWrap(ctx, operation, err)
	}
	policy, err := poolRecordOptions(ctx, p.guard, RecordFindOpts{}, copyPoolFind, slices.Clone(options))
	if err != nil {
		return nil, poolRecordWrap(ctx, operation, err)
	}
	member, list, filters, seed, control, err := poolRecordQuery(poolFindList(policy), true, identity)
	// Resource.find consumes microversion for fetch, then calls list without it.
	// Keep the selected/default client snapshot for fallback before GET policy.
	listSource := *p.source
	listSource.Client.MoreHeaders = maps.Clone(p.source.Client.MoreHeaders)
	fallback := &preparedPoolRecord{source: &listSource, guard: p.guard}
	if err == nil {
		err = p.withPolicy(ctx, policy.Microversion, policy.Headers)
	}
	if err = errors.Join(err, p.guard(ctx)); err != nil {
		return nil, poolRecordWrap(ctx, operation, err)
	}
	var failedMember *SubnetPoolRecord
	get := func(ctx context.Context, id string) (*SubnetPoolRecord, error) {
		target, err := url.Parse(p.source.Client.ServiceURL("subnetpools", url.PathEscape(id)))
		if err != nil {
			return nil, err
		}
		target.RawQuery = member.Encode()
		response, err := p.read(ctx, http.MethodGet, target.String(), nil, false)
		if err != nil {
			failedMember = poolRecordReceipt(response)
			return failedMember, err
		}
		value, err := poolRecordResponse(response, seed)
		if err = errors.Join(err, p.guard(ctx)); err != nil {
			failedMember = value
			return value, response.Fail(err)
		}
		return value, nil
	}
	listRows := func(ctx context.Context, _ url.Values) iter.Seq2[*SubnetPoolRecord, error] {
		return func(yield func(*SubnetPoolRecord, error) bool) {
			if err := fallback.withPolicy(ctx, nil, policy.Headers); err != nil {
				yield(nil, err)
				return
			}
			for value, err := range fallback.list(ctx, list, filters, control) {
				if !yield(value, err) {
					return
				}
			}
		}
	}
	collection := resource.NewCollection(resource.Adapter[SubnetPoolRecord]{Kind: "subnetpools", IdentityFind: true, ValidateID: poolRecordID, IdentityDirectGet: poolRecordID,
		IdentityResponseID: func(v *SubnetPoolRecord) (string, error) { return poolRecordText(v, "id"), nil }, Name: func(v *SubnetPoolRecord) string { return poolRecordText(v, "name") }, Get: get, Iterate: listRows})
	value, err := collection.FindIdentity(ctx, identity, resource.WithIdentityFindIgnoreMissing(policy.IgnoreMissing == nil || *policy.IgnoreMissing))
	if value == nil && err != nil && failedMember != nil {
		value = failedMember
	}
	return value, poolRecordWrap(ctx, operation, errors.Join(err, p.guard(ctx)))
}

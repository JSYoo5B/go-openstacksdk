package portforwarding

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

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type preparedForwarding struct {
	ctx          context.Context
	client       gophercloud.ServiceClient
	parent, path string
	guard        func(context.Context) error
}

func forwardingCodes() []int {
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = 200 + i
	}
	return codes
}
func forwardingID(id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." || !utf8.ValidString(id) {
		return forwardingInvalid("ID must be a nonblank UTF-8 segment")
	}
	for _, char := range id {
		if unicode.IsControl(char) {
			return forwardingInvalid("ID contains controls")
		}
	}
	return nil
}
func forwardingWrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "portforwarding", cloudread.ContextError(ctx, err))
}
func forwardingHeaders(values map[string]string) (map[string]string, error) {
	headers := make(map[string]string, len(values))
	for key, value := range values {
		if key == "" || !utf8.ValidString(value) {
			return nil, forwardingInvalid("invalid request Header")
		}
		for _, char := range []byte(key) {
			if !(char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(char))) {
				return nil, forwardingInvalid("invalid Header name")
			}
		}
		for _, char := range value {
			if unicode.IsControl(char) && char != '\t' {
				return nil, forwardingInvalid("invalid Header value")
			}
		}
		switch strings.ToLower(key) {
		case "x-auth-token", "x-service-token", "authorization", "host", "cookie", "content-length", "transfer-encoding", "connection", "trailer", "te", "upgrade", "openstack-api-version", "x-openstack-nova-api-version", "x-openstack-volume-api-version":
			return nil, forwardingInvalid("Header %q is SDK owned", key)
		}
		name := http.CanonicalHeaderKey(key)
		if old, ok := headers[name]; ok && old != value {
			return nil, forwardingInvalid("conflicting Header aliases")
		}
		headers[name] = value
	}
	return headers, nil
}
func (a *API) captureForwarding(ctx context.Context, parent string) (*preparedForwarding, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil || a.client == nil || a.client.ProviderClient == nil {
		return nil, forwardingInvalid("configured Neutron client is required")
	}
	if err := forwardingID(parent); err != nil {
		return nil, err
	}
	source := a.client
	if source.Type != "" && source.Type != "network" {
		return nil, resource.ErrUnsupported
	}
	for _, endpoint := range []string{source.Endpoint, source.ResourceBaseURL()} {
		parsed, err := url.Parse(endpoint)
		if err != nil || !utf8.ValidString(endpoint) || parsed == nil || parsed.User != nil || parsed.Host == "" || parsed.Opaque != "" || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || parsed.Scheme != "http" && parsed.Scheme != "https" {
			return nil, forwardingInvalid("service endpoint must be an absolute HTTP(S) URL without query, fragment or credentials")
		}
	}
	if !strings.HasSuffix(source.ResourceBaseURL(), "/") {
		return nil, forwardingInvalid("resource base must end in slash")
	}
	if err := rest.ValidateTarget(source, source.ResourceBaseURL()); err != nil {
		return nil, err
	}
	headers, err := forwardingHeaders(source.MoreHeaders)
	if err != nil {
		return nil, err
	}
	captured := *source
	captured.MoreHeaders = headers
	provider, endpoint, base, kind, version := source.ProviderClient, source.Endpoint, source.ResourceBase, source.Type, source.Microversion
	var observed error
	guard := func(ctx context.Context) error {
		if observed == nil && (a.client != source || source.ProviderClient != provider || source.Endpoint != endpoint || source.ResourceBase != base || source.Type != kind || source.Microversion != version) {
			observed = forwardingInvalid("selected Neutron source or route changed")
		}
		return errors.Join(cloudread.ContextError(ctx, observed), rest.CheckOperationGuard(ctx))
	}
	p := &preparedForwarding{ctx: ctx, client: captured, parent: parent, path: "floatingips/" + url.PathEscape(parent) + "/port_forwardings", guard: guard}
	return p, guard(ctx)
}
func (p *preparedForwarding) get(id string, query url.Values) (*PortForwardingRecord, error) {
	if err := forwardingID(id); err != nil {
		return nil, err
	}
	target := p.client.ServiceURL(p.path, url.PathEscape(id))
	parsed, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	parsed.RawQuery = query.Encode()
	response, err := rest.DoJSONGuardedRejections(p.ctx, &p.client, p.guard, http.MethodGet, parsed.String(), nil, nil, rest.RejectionPolicy{Codes: []int{400, 403, 404}, PreserveCleanRetry: true}, forwardingCodes()...)
	if err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(id)
	value, err := forwardingFromResponse(response, p.parent, map[string]json.RawMessage{"id": raw})
	if err = errors.Join(err, p.guard(p.ctx)); err != nil {
		err = response.Fail(err)
	}
	return value, err
}

// CreateRecord implements both Python create naming families with an explicit
// literal floating-IP parent. Supplied attrs are posted once in port_forwarding;
// omitted fields are not synthesized and Neutron owns required-field checks.
func (a *API) CreateRecord(ctx context.Context, parent string, options ...RecordMutationOption) (*PortForwardingRecord, error) {
	const op = "CreateRecord"
	p, err := a.captureForwarding(ctx, parent)
	if err != nil {
		return nil, forwardingWrap(ctx, op, err)
	}
	policy, err := applyForwardingOptions(ctx, p.guard, RecordMutationOpts{}, copyForwardingMutation, slices.Clone(options))
	if err != nil {
		return nil, forwardingWrap(ctx, op, err)
	}
	body, err := forwardingBody(policy, false)
	if err != nil {
		return nil, forwardingWrap(ctx, op, err)
	}
	response, err := rest.DoJSONGuarded(ctx, &p.client, p.guard, http.MethodPost, p.client.ServiceURL(p.path), map[string]any{"port_forwarding": body}, nil, forwardingCodes()...)
	if err != nil {
		return nil, forwardingWrap(ctx, op, err)
	}
	value, err := forwardingFromResponse(response, parent, body)
	if err = errors.Join(err, p.guard(ctx)); err != nil {
		err = response.Fail(err)
	}
	return value, forwardingWrap(ctx, op, err)
}

// UpdateRecord commits supplied Body attrs to a fixed member. No attrs sends
// no HTTP and returns an ID-only descriptor view with the same parent.
func (a *API) UpdateRecord(ctx context.Context, parent, id string, options ...RecordMutationOption) (*PortForwardingRecord, error) {
	const op = "UpdateRecord"
	p, err := a.captureForwarding(ctx, parent)
	if err == nil {
		err = forwardingID(id)
	}
	if err != nil {
		return nil, forwardingWrap(ctx, op, err)
	}
	policy, err := applyForwardingOptions(ctx, p.guard, RecordMutationOpts{}, copyForwardingMutation, slices.Clone(options))
	if err != nil {
		return nil, forwardingWrap(ctx, op, err)
	}
	body, err := forwardingBody(policy, true)
	if err != nil {
		return nil, forwardingWrap(ctx, op, err)
	}
	raw, _ := json.Marshal(id)
	seed := forwardingClone(body)
	seed["id"] = raw
	if len(body) == 0 {
		value := &PortForwardingRecord{ParentID: parent}
		err := projectForwarding(value, seed)
		return value, forwardingWrap(ctx, op, err)
	}
	response, err := rest.DoJSONGuarded(ctx, &p.client, p.guard, http.MethodPut, p.client.ServiceURL(p.path, url.PathEscape(id)), map[string]any{"port_forwarding": body}, nil, forwardingCodes()...)
	if err != nil {
		return nil, forwardingWrap(ctx, op, err)
	}
	value, err := forwardingFromResponse(response, parent, seed)
	if err = errors.Join(err, p.guard(ctx)); err != nil {
		err = response.Fail(err)
	}
	return value, forwardingWrap(ctx, op, err)
}

// GetRecord GETs a literal scoped member. Fields are repeated query parameters;
// empty fields omit that query and unknown names are preserved.
func (a *API) GetRecord(ctx context.Context, parent, id string, options ...RecordReadOption) (*PortForwardingRecord, error) {
	const op = "GetRecord"
	p, err := a.captureForwarding(ctx, parent)
	if err != nil {
		return nil, forwardingWrap(ctx, op, err)
	}
	policy, err := applyForwardingOptions(ctx, p.guard, RecordReadOpts{}, copyForwardingRead, slices.Clone(options))
	if err != nil {
		return nil, forwardingWrap(ctx, op, err)
	}
	query := make(url.Values)
	for _, field := range policy.Fields {
		if !utf8.ValidString(field) {
			return nil, forwardingWrap(ctx, op, forwardingInvalid("fields require UTF-8 strings"))
		}
		query.Add("fields", field)
	}
	value, err := p.get(id, query)
	return value, forwardingWrap(ctx, op, err)
}

// DeleteRecord ignores only a clean 404 by default. Rejected-body read/close,
// retry, route-change and cancellation causes remain failures with HTTP proof.
func (a *API) DeleteRecord(ctx context.Context, parent, id string, options ...RecordDeleteOption) error {
	const op = "DeleteRecord"
	p, err := a.captureForwarding(ctx, parent)
	if err == nil {
		err = forwardingID(id)
	}
	if err != nil {
		return forwardingWrap(ctx, op, err)
	}
	policy, err := applyForwardingOptions(ctx, p.guard, RecordDeleteOpts{}, copyForwardingDelete, slices.Clone(options))
	if err != nil {
		return forwardingWrap(ctx, op, err)
	}
	_, err = rest.DoJSONGuardedRejections(ctx, &p.client, p.guard, http.MethodDelete, p.client.ServiceURL(p.path, url.PathEscape(id)), nil, nil, rest.RejectionPolicy{Codes: []int{404}, PreserveCleanRetry: true}, forwardingCodes()...)
	if policy.IgnoreMissing == nil || *policy.IgnoreMissing {
		err = resource.IgnoreMissing(err)
	}
	return forwardingWrap(ctx, op, err)
}
func (p *preparedForwarding) list(query url.Values, filters map[string]json.RawMessage, control rest.ListControl) iter.Seq2[*PortForwardingRecord, error] {
	return func(yield func(*PortForwardingRecord, error) bool) {
		var origin *rest.Response
		spec := rest.CollectionSpec[PortForwardingRecord]{Client: &p.client, Path: p.path, Kind: "portforwarding", PluralKey: "port_forwardings", Metadata: forwardingMetadata, Validate: p.guard, SourceGuard: p.guard, ListCodes: forwardingCodes(),
			ValidateResponse: func(r *rest.Response) error { origin = r; return p.guard(p.ctx) },
			ValidateItem: func(v *PortForwardingRecord) error {
				v.ParentID = p.parent
				v.Envelope = bytes.Clone(origin.Body)
				v.Header, v.StatusCode = v.Wire.Header.Clone(), v.Wire.StatusCode
				return errors.Join(projectForwarding(v, nil), p.guard(p.ctx))
			},
			Paging: rest.PagePolicy[PortForwardingRecord]{HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, MarkerFallback: true, Marker: forwardingMarker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true, VersionedPath: "/v2.0/" + p.path}}
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
func (p *preparedForwarding) listHeaders(headers map[string]string) error {
	overrides, err := forwardingHeaders(headers)
	if err != nil {
		return err
	}
	if p.client.MoreHeaders == nil {
		p.client.MoreHeaders = make(map[string]string)
	}
	p.client.MoreHeaders["Accept"] = "application/json"
	maps.Copy(p.client.MoreHeaders, overrides)
	return p.guard(p.ctx)
}

// ListRecords lazily splits Source query kwargs into declared server filters and
// other declared Body filters. Unknown attrs are ignored; max-items counts raw
// rows before local filtering. Each iteration captures an independent client.
func (a *API) ListRecords(ctx context.Context, parent string, options ...RecordListOption) iter.Seq2[*PortForwardingRecord, error] {
	owned := slices.Clone(options)
	return func(yield func(*PortForwardingRecord, error) bool) {
		const op = "ListRecords"
		p, err := a.captureForwarding(ctx, parent)
		if err != nil {
			yield(nil, forwardingWrap(ctx, op, err))
			return
		}
		policy, err := applyForwardingOptions(ctx, p.guard, RecordListOpts{}, copyForwardingList, owned)
		if err == nil {
			err = p.listHeaders(policy.Headers)
		}
		if err != nil {
			yield(nil, forwardingWrap(ctx, op, err))
			return
		}
		query, filters, control, err := forwardingList(policy)
		if err != nil {
			yield(nil, forwardingWrap(ctx, op, err))
			return
		}
		for value, err := range p.list(query, filters, control) {
			if !yield(value, forwardingWrap(ctx, op, err)) {
				return
			}
		}
	}
}

// FindRecord performs Source GET-first lookup, preserving the floatingip_id
// fetch query, then fully observes the scoped ID-or-name list after only clean
// 400/403/404. Its default missing result is nil; duplicate matches are errors.
func (a *API) FindRecord(ctx context.Context, parent, nameOrID string, options ...RecordFindOption) (*PortForwardingRecord, error) {
	const op = "FindRecord"
	p, err := a.captureForwarding(ctx, parent)
	if err != nil {
		return nil, forwardingWrap(ctx, op, err)
	}
	policy, err := applyForwardingOptions(ctx, p.guard, RecordFindOpts{}, copyForwardingFind, slices.Clone(options))
	if err != nil {
		return nil, forwardingWrap(ctx, op, err)
	}
	direct, list, err := forwardingFind(policy, parent)
	if err != nil {
		return nil, forwardingWrap(ctx, op, err)
	}
	// Validate fallback configuration now without invoking callbacks after GET.
	query, filters, control, err := forwardingList(list)
	if err != nil {
		return nil, forwardingWrap(ctx, op, err)
	}
	headers, err := forwardingHeaders(list.Headers)
	if err != nil {
		return nil, forwardingWrap(ctx, op, err)
	}
	collection := resource.NewCollection(resource.Adapter[PortForwardingRecord]{Kind: "portforwarding", IdentityFind: true, ValidateID: forwardingID, IdentityDirectGet: forwardingID,
		IdentityResponseID: func(v *PortForwardingRecord) (string, error) {
			if v == nil {
				return "", forwardingInvalid("find response is missing")
			}
			return forwardingText(v, "id"), nil
		}, Name: func(v *PortForwardingRecord) string { return forwardingText(v, "name") },
		Get: func(_ context.Context, stringID string) (*PortForwardingRecord, error) {
			return p.get(stringID, direct)
		},
		Iterate: func(context.Context, url.Values) iter.Seq2[*PortForwardingRecord, error] {
			return func(yield func(*PortForwardingRecord, error) bool) {
				if err := p.listHeaders(headers); err != nil {
					yield(nil, err)
					return
				}
				for value, err := range p.list(query, filters, control) {
					if !yield(value, err) {
						return
					}
				}
			}
		},
	})
	ignored := policy.IgnoreMissing == nil || *policy.IgnoreMissing
	value, err := collection.FindIdentity(ctx, nameOrID, resource.WithIdentityFindIgnoreMissing(ignored))
	err = errors.Join(err, p.guard(ctx))
	return value, forwardingWrap(ctx, op, err)
}

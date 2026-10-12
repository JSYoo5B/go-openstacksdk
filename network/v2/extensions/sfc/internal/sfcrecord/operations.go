package sfcrecord

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Target is one collection route bound to the API's selected client.
type Target struct {
	Spec   *Spec
	Client func() *gophercloud.ServiceClient
}

// Prepared is one operation's captured network client and source guard.
type Prepared struct {
	target Target
	ctx    context.Context
	source *cloudread.Source
	guard  func(context.Context) error
}

// Codes is Python raise_from_response: every status from 200 through 399.
func Codes() []int {
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = 200 + i
	}
	return codes
}

func rejectedCodes() []int {
	codes := make([]int, 0, 700)
	for code := 100; code <= 999; code++ {
		if code < 200 || code >= 400 {
			codes = append(codes, code)
		}
	}
	return codes
}

// Wrap adds operation context and any context cancellation cause.
func (t Target) Wrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, t.Spec.Kind, cloudread.ContextError(ctx, err))
}

// Prepare captures the selected network client once. Later replacement of
// the API's client, provider, endpoint or headers fails the operation.
func (t Target) Prepare(ctx context.Context) (*Prepared, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if t.Client == nil {
		return nil, Invalid("%s API is required", t.Spec.Kind)
	}
	selected := t.Client()
	source, err := cloudread.Capture(ctx, selected, "network")
	if err != nil {
		return nil, err
	}
	guard := func(ctx context.Context) error {
		var replaced error
		if t.Client() != selected {
			replaced = Invalid("selected %s client changed", t.Spec.Kind)
		}
		return errors.Join(replaced, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	return &Prepared{target: t, ctx: ctx, source: source, guard: guard}, guard(ctx)
}

// Headers applies caller headers after the SDK-owned header policy.
func (p *Prepared) Headers(values map[string]string) error {
	return errors.Join(p.source.WithPolicy(p.ctx, nil, values), p.guard(p.ctx))
}

func (p *Prepared) collection() string {
	return p.source.Client.ServiceURL(p.target.Spec.Path)
}

func (p *Prepared) member(id string) string {
	return p.source.Client.ServiceURL(p.target.Spec.Path, url.PathEscape(id))
}

func (p *Prepared) seed(fields map[string]json.RawMessage, id string) map[string]json.RawMessage {
	seed := cloneFields(fields)
	if seed == nil {
		seed = make(map[string]json.RawMessage)
	}
	if id != "" {
		raw, _ := json.Marshal(id)
		seed["id"] = raw
	}
	return seed
}

func (p *Prepared) record(response *rest.Response, seed map[string]json.RawMessage) (*Record, error) {
	value, err := p.target.Spec.fromResponse(response, seed)
	if err = errors.Join(err, p.guard(p.ctx)); err != nil {
		err = response.Fail(err)
	}
	return value, err
}

func evidence(response *rest.Response) *Record {
	if response == nil {
		return nil
	}
	return &Record{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
}

// Get sends Resource.fetch: GET {collection}/{id} with the optional query.
// 400, 403 and 404 bodies are observed so find may fall back cleanly.
func (p *Prepared) Get(id string, query url.Values, seed map[string]json.RawMessage) (*Record, error) {
	if err := ValidateID("ID", id); err != nil {
		return nil, err
	}
	target := p.member(id)
	if len(query) != 0 {
		target += "?" + query.Encode()
	}
	response, err := rest.DoJSONGuardedRejections(p.ctx, &p.source.Client, p.guard, http.MethodGet, target, nil, nil, rest.RejectionPolicy{Codes: []int{400, 403, 404}, PreserveCleanRetry: true}, Codes()...)
	if err != nil {
		return evidence(response), err
	}
	return p.record(response, p.seed(seed, id))
}

// Create sends Resource.create with create_method POST: every given declared
// attribute (id included) enters the resource_key envelope.
func (p *Prepared) Create(values map[string]json.RawMessage) (*Record, error) {
	fields, err := p.target.Spec.fields(values, false)
	if err != nil {
		return nil, err
	}
	body := map[string]any{p.target.Spec.ResourceKey: cloneFields(fields)}
	response, err := rest.DoJSONGuardedRejections(p.ctx, &p.source.Client, p.guard, http.MethodPost, p.collection(), body, nil, rest.RejectionPolicy{Codes: rejectedCodes(), PreserveCleanRetry: true}, Codes()...)
	if err != nil {
		return evidence(response), err
	}
	return p.record(response, p.seed(fields, ""))
}

// Update sends Resource.commit with commit_method PUT and only the dirty
// declared attributes. The ID is never dirty; nothing to commit means no HTTP.
func (p *Prepared) Update(id string, values map[string]json.RawMessage) (*Record, error) {
	if err := ValidateID("ID", id); err != nil {
		return nil, err
	}
	fields, err := p.target.Spec.fields(values, true)
	if err != nil {
		return nil, err
	}
	seed := p.seed(fields, id)
	if len(fields) == 0 {
		value := &Record{}
		return value, errors.Join(p.target.Spec.project(value, seed), p.guard(p.ctx))
	}
	body := map[string]any{p.target.Spec.ResourceKey: cloneFields(fields)}
	response, err := rest.DoJSONGuardedRejections(p.ctx, &p.source.Client, p.guard, http.MethodPut, p.member(id), body, nil, rest.RejectionPolicy{Codes: rejectedCodes(), PreserveCleanRetry: true}, Codes()...)
	if err != nil {
		return evidence(response), err
	}
	return p.record(response, seed)
}

// Delete sends Resource.delete. A clean 404 is nil when ignoreMissing is true.
func (p *Prepared) Delete(id string, ignoreMissing bool) error {
	if err := ValidateID("ID", id); err != nil {
		return err
	}
	_, err := rest.DoJSONGuardedRejections(p.ctx, &p.source.Client, p.guard, http.MethodDelete, p.member(id), nil, nil, rest.RejectionPolicy{Codes: []int{404}, PreserveCleanRetry: true}, Codes()...)
	if ignoreMissing {
		err = resource.IgnoreMissing(err)
	}
	return err
}

// List sends Resource.list: server query, Accept: application/json, local
// Body filters after the raw row cap, Python pagination continuation rules
// (the {resources_key}_links next link, an HTTP Link header, and limit-driven
// marker continuation).
func (p *Prepared) List(query url.Values, filters map[string]json.RawMessage, control rest.ListControl) iter.Seq2[*Record, error] {
	return func(yield func(*Record, error) bool) {
		var origin *rest.Response
		spec := p.target.Spec
		collection := rest.CollectionSpec[row]{Client: &p.source.Client, Path: spec.Path, Kind: spec.Kind, PluralKey: spec.ResourcesKey, Metadata: func(r *row) *resource.Metadata { return &r.meta }, Validate: p.guard, SourceGuard: p.guard, ListCodes: Codes(), ValidateResponse: func(r *rest.Response) error { origin = r; return p.guard(p.ctx) },
			Paging: rest.PagePolicy[row]{VersionedPath: "/v2.0/" + spec.Path, HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, NextTruthy: cloudfilter.PythonJSONTruthy, MarkerFallback: true, Marker: marker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true}}
		for item, err := range rest.ListWithControl(p.ctx, collection, query, control) {
			if err != nil {
				yield(nil, err)
				return
			}
			value, err := p.row(item, origin)
			if err == nil {
				var matched bool
				matched, err = jsonfilter.MatchFilters(value.View, filters)
				if err = errors.Join(err, p.guard(p.ctx)); err == nil && !matched {
					continue
				}
			}
			if err != nil {
				yield(nil, origin.Fail(err))
				return
			}
			if !yield(value, nil) {
				return
			}
		}
	}
}

// acceptJSON adds the Resource.list Accept header unless the caller set one.
func (p *Prepared) acceptJSON() {
	for key := range p.source.Client.MoreHeaders {
		if http.CanonicalHeaderKey(key) == "Accept" {
			return
		}
	}
	if p.source.Client.MoreHeaders == nil {
		p.source.Client.MoreHeaders = make(map[string]string, 1)
	}
	p.source.Client.MoreHeaders["Accept"] = "application/json"
}

// ListOptions prepares Python **query and runs List.
func (p *Prepared) ListOptions(options ListOpts) iter.Seq2[*Record, error] {
	return func(yield func(*Record, error) bool) {
		query, filters, control, err := p.target.Spec.prepareList(options)
		if err == nil {
			err = p.Headers(options.Headers)
		}
		if err != nil {
			yield(nil, err)
			return
		}
		p.acceptJSON()
		for value, err := range p.List(query, filters, control) {
			if !yield(value, err) || err != nil {
				return
			}
		}
	}
}

func (p *Prepared) row(item *row, page *rest.Response) (*Record, error) {
	if item == nil || item.meta.Body == nil || page == nil {
		return nil, Invalid("%s list item is missing", p.target.Spec.Kind)
	}
	wire := cloneFields(item.meta.Body)
	delete(wire, "self")
	value := &Record{Wire: wire, Envelope: bytes.Clone(page.Body), Header: page.Header.Clone(), StatusCode: page.StatusCode}
	return value, errors.Join(p.target.Spec.project(value, nil), p.guard(p.ctx))
}

// row is one raw list item. DecodeObject keeps every original field in
// meta.Body; nothing is decoded into typed fields.
type row struct{ meta resource.Metadata }

func (*row) UnmarshalJSON([]byte) error { return nil }

func marker(r *row) (string, error) {
	if r == nil || r.meta.Body == nil {
		return "", Invalid("record marker missing")
	}
	var id string
	if err := json.Unmarshal(r.meta.Body["id"], &id); err != nil || id == "" {
		return "", Invalid("record marker must be a nonempty string")
	}
	return id, nil
}

// Find follows Resource.find: GET {collection}/{name_or_id} with every find
// query name, then on a clean 400, 403 or 404 the complete list with the
// query (plus name=name_or_id unless the query names one) matched by exact
// ID or name.
func (p *Prepared) Find(ctx context.Context, nameOrID string, options FindOpts) (*Record, error) {
	member, seed, list, err := p.target.Spec.prepareFind(options.Query, nameOrID)
	if err != nil {
		return nil, err
	}
	query, filters, control, err := p.target.Spec.prepareList(list)
	if err != nil {
		return nil, err
	}
	p.acceptJSON()
	collection := resource.NewCollection(resource.Adapter[Record]{Kind: p.target.Spec.Kind, IdentityFind: true,
		ValidateID:        func(id string) error { return ValidateID("ID", id) },
		IdentityDirectGet: func(id string) error { return ValidateID("ID", id) },
		IdentityResponseID: func(v *Record) (string, error) {
			if v == nil {
				return "", Invalid("find response missing")
			}
			return v.Text("id"), nil
		},
		Name: func(v *Record) string { return v.Text("name") },
		Get:  func(_ context.Context, id string) (*Record, error) { return p.Get(id, member, seed) },
		Iterate: func(context.Context, url.Values) iter.Seq2[*Record, error] {
			return p.List(query, filters, control)
		}})
	value, err := collection.FindIdentity(ctx, nameOrID, resource.WithIdentityFindIgnoreMissing(options.IgnoreMissing == nil || *options.IgnoreMissing))
	return value, errors.Join(err, p.guard(ctx))
}

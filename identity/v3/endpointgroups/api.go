package endpointgroups

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
	"github.com/gophercloud/gophercloud/v2"
)

const kind = "endpointgroups"

// API shares the selected authenticated Keystone v3 client without taking
// ownership. Every route is under OS-EP-FILTER on that client.
type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }
func (a *API) RawClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}

type prepared struct {
	ctx    context.Context
	source *cloudread.Source
	guard  func(context.Context) error
}

// acceptedCodes is Resource raise_from_response: every status below 400.
func acceptedCodes() []int {
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

func validID(id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." || !utf8.ValidString(id) {
		return invalid("ID must be a nonblank UTF-8 segment")
	}
	for _, c := range id {
		if unicode.IsControl(c) {
			return invalid("ID has controls")
		}
	}
	return nil
}

func wrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, kind, cloudread.ContextError(ctx, err))
}

func (a *API) capture(ctx context.Context) (*prepared, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, invalid("endpoint group API is required")
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "identity")
	if err != nil {
		return nil, err
	}
	guard := func(ctx context.Context) error {
		var replaced error
		if a.client != selected {
			replaced = invalid("selected endpoint group client changed")
		}
		return errors.Join(replaced, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	return &prepared{ctx: ctx, source: source, guard: guard}, guard(ctx)
}

func (p *prepared) headers(values map[string]string) error {
	return errors.Join(p.source.WithPolicy(p.ctx, nil, values), p.guard(p.ctx))
}

// member escapes each ID as exactly one path segment under OS-EP-FILTER.
func (p *prepared) member(collection string, ids ...string) string {
	parts := []string{"OS-EP-FILTER", collection}
	for i, id := range ids {
		parts = append(parts, url.PathEscape(id))
		if i == 0 && len(ids) > 1 {
			parts = append(parts, "projects")
		}
	}
	return p.source.Client.ServiceURL(parts...)
}

// finish turns an accepted response into a Record and keeps the receipt of a
// failed one, the way the other SDK-owned identity records do.
func (p *prepared) finish(response *rest.Response, err error, seed map[string]json.RawMessage) (*Record, error) {
	if err != nil {
		if response == nil {
			return nil, err
		}
		return receipt(response), err
	}
	v, err := endpointGroupClass.fromResponse(response, seed)
	if err = errors.Join(err, p.guard(p.ctx)); err != nil {
		err = response.Fail(err)
	}
	return v, err
}

func (p *prepared) get(id string) (*Record, error) {
	if err := validID(id); err != nil {
		return nil, err
	}
	response, err := rest.DoJSONGuardedRejections(p.ctx, &p.source.Client, p.guard, http.MethodGet, p.member("endpoint_groups", id), nil, nil, rest.RejectionPolicy{Codes: []int{400, 403, 404}, PreserveCleanRetry: true}, acceptedCodes()...)
	if err != nil && response == nil {
		return nil, err
	}
	raw, _ := json.Marshal(id)
	return p.finish(response, err, map[string]json.RawMessage{"id": raw})
}

// Get implements get_endpoint_group: GET OS-EP-FILTER/endpoint_groups/{id}.
func (a *API) Get(ctx context.Context, id string, options ...ReadOption) (*Record, error) {
	const op = "Get"
	p, err := a.capture(ctx)
	if err != nil {
		return nil, wrap(ctx, op, err)
	}
	policy, err := applyOptions(ctx, p.guard, ReadOpts{}, copyRead, slices.Clone(options))
	if err == nil {
		err = p.headers(policy.Headers)
	}
	if err != nil {
		return nil, wrap(ctx, op, err)
	}
	v, err := p.get(id)
	return v, wrap(ctx, op, err)
}

func declaredFields(values map[string]json.RawMessage, update bool) (map[string]json.RawMessage, error) {
	fields := make(map[string]json.RawMessage)
	for key, raw := range values {
		if !endpointGroupClass.known(key) || update && key == "id" {
			continue
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, invalid("attribute %q must be UTF-8 JSON", key)
		}
		fields[key] = bytes.Clone(raw)
	}
	return fields, nil
}

// Create implements create_endpoint_group: POST OS-EP-FILTER/endpoint_groups
// with the declared attributes under endpoint_group.
func (a *API) Create(ctx context.Context, options ...CreateOption) (*Record, error) {
	return a.create(ctx, "Create", slices.Clone(options))
}

func (a *API) create(ctx context.Context, op string, options []CreateOption) (*Record, error) {
	p, err := a.capture(ctx)
	if err != nil {
		return nil, wrap(ctx, op, err)
	}
	policy, err := applyOptions(ctx, p.guard, CreateOpts{}, copyCreate, options)
	var fields map[string]json.RawMessage
	if err == nil {
		fields, err = declaredFields(policy.Fields, false)
	}
	if err == nil {
		err = p.headers(policy.Headers)
	}
	if err != nil {
		return nil, wrap(ctx, op, err)
	}
	response, err := rest.DoJSONGuarded(ctx, &p.source.Client, p.guard, http.MethodPost, p.member("endpoint_groups"), map[string]any{"endpoint_group": cloneFields(fields)}, nil, acceptedCodes()...)
	v, err := p.finish(response, err, fields)
	return v, wrap(ctx, op, err)
}

// Update implements update_endpoint_group: PATCH on the literal member with
// only the declared attributes. No declared attribute means no request, and
// the returned view carries the ID and nothing else, as Resource.commit does.
func (a *API) Update(ctx context.Context, id string, options ...UpdateOption) (*Record, error) {
	const op = "Update"
	p, err := a.capture(ctx)
	if err != nil {
		return nil, wrap(ctx, op, err)
	}
	policy, err := applyOptions(ctx, p.guard, UpdateOpts{}, copyUpdate, slices.Clone(options))
	if err == nil {
		err = validID(id)
	}
	var fields map[string]json.RawMessage
	if err == nil {
		fields, err = declaredFields(policy.Fields, true)
	}
	if err == nil {
		err = p.headers(policy.Headers)
	}
	if err != nil {
		return nil, wrap(ctx, op, err)
	}
	rawID, _ := json.Marshal(id)
	seed := cloneFields(fields)
	seed["id"] = rawID
	if len(fields) == 0 {
		v := &Record{}
		err = errors.Join(endpointGroupClass.project(v, seed, ""), p.guard(ctx))
		return v, wrap(ctx, op, err)
	}
	response, err := rest.DoJSONGuardedRejections(ctx, &p.source.Client, p.guard, http.MethodPatch, p.member("endpoint_groups", id), map[string]any{"endpoint_group": cloneFields(fields)}, nil, rest.RejectionPolicy{Codes: rejectedCodes(), PreserveCleanRetry: true}, acceptedCodes()...)
	v, err := p.finish(response, err, seed)
	return v, wrap(ctx, op, err)
}

// Delete implements delete_endpoint_group. A clean 404 is ignored unless
// WithDeleteIgnoreMissing(false) is given.
func (a *API) Delete(ctx context.Context, id string, options ...DeleteOption) error {
	const op = "Delete"
	p, err := a.capture(ctx)
	if err != nil {
		return wrap(ctx, op, err)
	}
	policy, err := applyOptions(ctx, p.guard, DeleteOpts{}, copyDelete, slices.Clone(options))
	if err == nil {
		err = validID(id)
	}
	if err == nil {
		err = p.headers(policy.Headers)
	}
	if err != nil {
		return wrap(ctx, op, err)
	}
	_, err = rest.DoJSONGuardedRejections(ctx, &p.source.Client, p.guard, http.MethodDelete, p.member("endpoint_groups", id), nil, nil, rest.RejectionPolicy{Codes: []int{404}, PreserveCleanRetry: true}, acceptedCodes()...)
	if policy.IgnoreMissing == nil || *policy.IgnoreMissing {
		err = resource.IgnoreMissing(err)
	}
	return wrap(ctx, op, err)
}

// list streams one OS-EP-FILTER collection for a Resource class. Rows are
// projected with the scope URI attribute and filtered by local Body filters
// after the raw-row cap, as Resource.list does.
func (p *prepared) list(class descriptor, path []string, scope string, query url.Values, filters map[string]json.RawMessage, control rest.ListControl) iter.Seq2[*Record, error] {
	return func(yield func(*Record, error) bool) {
		var origin *rest.Response
		spec := rest.CollectionSpec[Record]{Client: &p.source.Client, Path: strings.Join(path, "/"), Kind: kind, PluralKey: class.plural, Metadata: recordMetadata, Validate: p.guard, SourceGuard: p.guard, ListCodes: acceptedCodes(), ValidateResponse: func(r *rest.Response) error { origin = r; return p.guard(p.ctx) }, ValidateItem: func(v *Record) error {
			v.Envelope = bytes.Clone(origin.Body)
			v.Header, v.StatusCode = v.Wire.Header.Clone(), v.Wire.StatusCode
			return errors.Join(class.project(v, nil, scope), p.guard(p.ctx))
		}, Paging: rest.PagePolicy[Record]{HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, MarkerFallback: true, Marker: marker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true}}
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

func (a *API) listOperation(ctx context.Context, op string, class descriptor, scope string, path func(string) []string, options []ListOption) iter.Seq2[*Record, error] {
	owned := slices.Clone(options)
	return func(yield func(*Record, error) bool) {
		p, err := a.capture(ctx)
		if err != nil {
			yield(nil, wrap(ctx, op, err))
			return
		}
		policy, err := applyOptions(ctx, p.guard, ListOpts{}, copyList, owned)
		if err == nil && class.uri != "" {
			err = validID(scope)
		}
		if err == nil {
			err = p.headers(policy.Headers)
		}
		if err != nil {
			yield(nil, wrap(ctx, op, err))
			return
		}
		if _, ok := p.source.Client.MoreHeaders["Accept"]; !ok {
			p.source.Client.MoreHeaders["Accept"] = "application/json"
		}
		query, filters, control, err := class.prepareList(policy)
		if err != nil {
			yield(nil, wrap(ctx, op, err))
			return
		}
		for v, err := range p.list(class, path(url.PathEscape(scope)), scope, query, filters, control) {
			if !yield(v, wrap(ctx, op, err)) {
				return
			}
		}
	}
}

// List implements endpoint_groups(**query): GET OS-EP-FILTER/endpoint_groups.
// name is the only wire query; id, description, filters and links filter
// rows locally; limit and marker are dropped because EndpointGroup excludes
// the pagination defaults, while WithListMaxItems still sends limit.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*Record, error] {
	return a.listOperation(ctx, "List", endpointGroupClass, "", func(string) []string {
		return []string{"OS-EP-FILTER", "endpoint_groups"}
	}, options)
}

// Find implements find_endpoint_group: GET by ID first and, after a clean
// 400, 403 or 404, the complete name-filtered list matched by exact ID or name.
func (a *API) Find(ctx context.Context, nameOrID string, options ...FindOption) (*Record, error) {
	const op = "Find"
	p, err := a.capture(ctx)
	if err != nil {
		return nil, wrap(ctx, op, err)
	}
	policy, err := applyOptions(ctx, p.guard, FindOpts{}, copyFind, slices.Clone(options))
	if err != nil {
		return nil, wrap(ctx, op, err)
	}
	collection := resource.NewCollection(resource.Adapter[Record]{Kind: kind, IdentityFind: true, ValidateID: validID, IdentityDirectGet: validID,
		IdentityResponseID: func(v *Record) (string, error) {
			if v == nil {
				return "", invalid("find response missing")
			}
			return v.Text("id"), nil
		},
		Name:      func(v *Record) string { return v.Text("name") },
		NameQuery: func(identity string) string { return identity },
		Get:       func(_ context.Context, id string) (*Record, error) { return p.get(id) },
		Iterate: func(_ context.Context, query url.Values) iter.Seq2[*Record, error] {
			return p.list(endpointGroupClass, []string{"OS-EP-FILTER", "endpoint_groups"}, "", query, nil, rest.ListControl{})
		}})
	ignored := policy.IgnoreMissing == nil || *policy.IgnoreMissing
	value, err := collection.FindIdentity(ctx, nameOrID, resource.WithIdentityFindIgnoreMissing(ignored))
	return value, wrap(ctx, op, errors.Join(err, p.guard(ctx)))
}

// ListEndpoints implements endpoint_group_endpoints(endpoint_group, **query):
// GET OS-EP-FILTER/endpoint_groups/{id}/endpoints as EndpointGroupEndpoint.
func (a *API) ListEndpoints(ctx context.Context, endpointGroupID string, options ...ListOption) iter.Seq2[*Record, error] {
	return a.listOperation(ctx, "ListEndpoints", endpointGroupEndpointClass, endpointGroupID, func(id string) []string {
		return []string{"OS-EP-FILTER", "endpoint_groups", id, "endpoints"}
	}, options)
}

// ListProjects implements endpoint_group_projects(endpoint_group, **query):
// GET OS-EP-FILTER/endpoint_groups/{id}/projects as EndpointGroupProject.
func (a *API) ListProjects(ctx context.Context, endpointGroupID string, options ...ListOption) iter.Seq2[*Record, error] {
	return a.listOperation(ctx, "ListProjects", endpointGroupProjectClass, endpointGroupID, func(id string) []string {
		return []string{"OS-EP-FILTER", "endpoint_groups", id, "projects"}
	}, options)
}

// ListProjectEndpointGroups implements project_endpoint_groups(project,
// **query): GET OS-EP-FILTER/projects/{id}/endpoint_groups.
func (a *API) ListProjectEndpointGroups(ctx context.Context, projectID string, options ...ListOption) iter.Seq2[*Record, error] {
	return a.listOperation(ctx, "ListProjectEndpointGroups", projectEndpointGroupClass, projectID, func(id string) []string {
		return []string{"OS-EP-FILTER", "projects", id, "endpoint_groups"}
	}, options)
}

// ListEndpointProjects implements endpoint_projects(endpoint, **query):
// GET OS-EP-FILTER/endpoints/{id}/projects as EndpointProject.
func (a *API) ListEndpointProjects(ctx context.Context, endpointID string, options ...ListOption) iter.Seq2[*Record, error] {
	return a.listOperation(ctx, "ListEndpointProjects", endpointProjectClass, endpointID, func(id string) []string {
		return []string{"OS-EP-FILTER", "endpoints", id, "projects"}
	}, options)
}

func (a *API) projectAction(ctx context.Context, op, method, endpointGroupID, projectID string, options []ReadOption) error {
	p, err := a.capture(ctx)
	if err != nil {
		return wrap(ctx, op, err)
	}
	policy, err := applyOptions(ctx, p.guard, ReadOpts{}, copyRead, slices.Clone(options))
	if err == nil {
		err = errors.Join(validID(endpointGroupID), validID(projectID))
	}
	if err == nil {
		err = p.headers(policy.Headers)
	}
	if err != nil {
		return wrap(ctx, op, err)
	}
	_, err = rest.DoJSONGuarded(ctx, &p.source.Client, p.guard, method, p.member("endpoint_groups", endpointGroupID, projectID), nil, nil, acceptedCodes()...)
	return wrap(ctx, op, err)
}

// AssociateProject implements associate_project_with_endpoint_group:
// PUT OS-EP-FILTER/endpoint_groups/{id}/projects/{project_id} without a body.
func (a *API) AssociateProject(ctx context.Context, endpointGroupID, projectID string, options ...ReadOption) error {
	return a.projectAction(ctx, "AssociateProject", http.MethodPut, endpointGroupID, projectID, options)
}

// DisassociateProject implements disassociate_project_from_endpoint_group:
// DELETE on the same route. A 404 is returned; Python has no ignore_missing.
func (a *API) DisassociateProject(ctx context.Context, endpointGroupID, projectID string, options ...ReadOption) error {
	return a.projectAction(ctx, "DisassociateProject", http.MethodDelete, endpointGroupID, projectID, options)
}

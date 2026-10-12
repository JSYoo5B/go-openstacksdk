// Package grouptypes implements the pinned openstacksdk block_storage v3
// group type proxy calls, including the group_specs sub-API, over Cinder
// group_types, a route Gophercloud v2.15.0 does not bind.
package grouptypes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/internal/cinderowned"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/gophercloud/gophercloud/v2"
)

const collectionPath = "group_types"

var spec = &cinderowned.Spec{
	Kind: "group_types", ResourceKey: "group_type", ResourcesKey: "group_types",
	Fields: []string{"id", "name", "description", "group_specs", "is_public"},
	Attrs: map[string]string{"id": "id", "name": "name", "description": "description",
		"group_specs": "group_specs", "is_public": "is_public"},
	Defaults: map[string]json.RawMessage{"group_specs": json.RawMessage("{}")},
	Query: map[string]string{"is_public": "is_public", "offset": "offset", "sort_dir": "sort_dir",
		"sort_key": "sort_key", "sort": "sort", "all_projects": "all_tenants"},
	Filters: map[string]string{"id": "id", "name": "name", "description": "description", "group_specs": "group_specs"},
}

// version is GroupType._get_microversion: the configured microversion, or
// min(3.11, server maximum) from discovery.
var version = cinderowned.Version{Ceiling: "3.11"}

// API shares the selected authenticated Cinder client without owning it.
type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }
func (a *API) RawClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}

func (a *API) target() cinderowned.Target {
	var client func() *gophercloud.ServiceClient
	if a != nil {
		client = func() *gophercloud.ServiceClient { return a.client }
	}
	return cinderowned.Target{Spec: spec, Client: client, Path: collectionPath}
}

func readHeaders(v GetOpts) map[string]string     { return v.Headers }
func writeHeaders(v CreateOpts) map[string]string { return v.Headers }
func deleteHeaders(v DeleteOpts) map[string]string {
	return v.Headers
}
func noHeaders(FindOpts) map[string]string { return nil }

// Get implements get_group_type: GET group_types/{id}.
func (a *API) Get(ctx context.Context, id string, options ...GetOption) (*GroupType, error) {
	const op = "Get"
	t := a.target()
	if err := cinderowned.ValidateID("group type ID", id); err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	p, _, err := cinderowned.Begin(ctx, t, GetOpts{}, cinderowned.CopyRead, readHeaders, version, options)
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	return convert(t.Wrapper(ctx, op)(p.Get(id, nil)))
}

// Create implements create_group_type: POST group_types with
// {"group_type": attrs} of the declared attributes given.
func (a *API) Create(ctx context.Context, options ...CreateOption) (*GroupType, error) {
	const op = "Create"
	t := a.target()
	p, v, err := cinderowned.Begin(ctx, t, CreateOpts{}, cinderowned.CopyWrite, writeHeaders, version, options)
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	return convert(t.Wrapper(ctx, op)(p.Create(v.Fields)))
}

// Update implements update_group_type: PUT group_types/{id} with only the
// given declared attributes. No declared attribute means no request.
func (a *API) Update(ctx context.Context, id string, options ...UpdateOption) (*GroupType, error) {
	const op = "Update"
	t := a.target()
	if err := cinderowned.ValidateID("group type ID", id); err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	// Resource.commit returns before selecting a microversion when nothing
	// is dirty, so discovery runs only for a real update.
	p, v, err := cinderowned.Begin(ctx, t, UpdateOpts{}, cinderowned.CopyWrite, writeHeaders, cinderowned.Version{Default: true}, options)
	var fields map[string]json.RawMessage
	if err == nil {
		fields, err = spec.BodyFields(v.Fields, true)
	}
	if err == nil && len(fields) != 0 {
		err = p.Select(version)
	}
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	return convert(t.Wrapper(ctx, op)(p.Update(id, v.Fields)))
}

// Delete implements delete_group_type: DELETE group_types/{id}. A clean 404
// returns nil unless WithDeleteIgnoreMissing(false) is given.
func (a *API) Delete(ctx context.Context, id string, options ...DeleteOption) error {
	const op = "Delete"
	t := a.target()
	if err := cinderowned.ValidateID("group type ID", id); err != nil {
		return t.Wrap(ctx, op, err)
	}
	p, v, err := cinderowned.Begin(ctx, t, DeleteOpts{}, cinderowned.CopyDelete, deleteHeaders, version, options)
	if err != nil {
		return t.Wrap(ctx, op, err)
	}
	return t.Wrap(ctx, op, p.Delete(p.URL(id), v.IgnoreMissing == nil || *v.IgnoreMissing))
}

// List implements group_types(**query) as a lazy iterator over GET
// group_types. Every iteration sends the requests again.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*GroupType, error] {
	return func(yield func(*GroupType, error) bool) {
		const op = "List"
		t := a.target()
		p, v, err := cinderowned.Begin(ctx, t, ListOpts{}, cinderowned.CopyList, func(v ListOpts) map[string]string { return v.Headers }, version, options)
		if err != nil {
			yield(nil, t.Wrap(ctx, op, err))
			return
		}
		for record, err := range p.ListOptions(collectionPath, v) {
			if !yield(fromRecord(record), t.Wrap(ctx, op, err)) || err != nil {
				return
			}
		}
	}
}

// All collects List.
func (a *API) All(ctx context.Context, options ...ListOption) ([]*GroupType, error) {
	var values []*GroupType
	for value, err := range a.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// Find implements find_group_type: GET group_types/{name_or_id}?is_public=none
// and, after a clean 400, 403 or 404, the complete group_types?is_public=none
// list matched by exact ID or name. A missing group type returns nil, nil
// unless WithFindIgnoreMissing(false) is given.
func (a *API) Find(ctx context.Context, nameOrID string, options ...FindOption) (*GroupType, error) {
	const op = "Find"
	t := a.target()
	p, v, err := cinderowned.Begin(ctx, t, FindOpts{}, cinderowned.CopyFind, noHeaders, version, options)
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	query := url.Values{"is_public": {"none"}}
	return convert(t.Wrapper(ctx, op)(p.Find(ctx, nameOrID, query, collectionPath, query, v.IgnoreMissing == nil || *v.IgnoreMissing)))
}

// FetchGroupSpecs implements fetch_group_type_group_specs: GET
// group_types/{id}/group_specs. The result is the group type seeded with the
// ID and the response group_specs ({} when the key is absent).
func (a *API) FetchGroupSpecs(ctx context.Context, id string, options ...GetOption) (*GroupType, error) {
	return a.specs(ctx, "FetchGroupSpecs", id, http.MethodGet, nil, options)
}

// CreateGroupSpecs implements create_group_type_group_specs: POST
// group_types/{id}/group_specs with {"group_specs": specs}.
func (a *API) CreateGroupSpecs(ctx context.Context, id string, specs map[string]any, options ...GetOption) (*GroupType, error) {
	raw, err := json.Marshal(specs)
	if err != nil {
		return nil, a.target().Wrap(ctx, "CreateGroupSpecs", err)
	}
	return a.specs(ctx, "CreateGroupSpecs", id, http.MethodPost, map[string]json.RawMessage{"group_specs": raw}, options)
}

func (a *API) specs(ctx context.Context, op, id, method string, body map[string]json.RawMessage, options []GetOption) (*GroupType, error) {
	t := a.target()
	if err := cinderowned.ValidateID("group type ID", id); err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	p, _, err := cinderowned.Begin(ctx, t, GetOpts{}, cinderowned.CopyRead, readHeaders, version, options)
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	var payload any
	if body != nil {
		payload = body
	}
	response, err := p.Do(method, p.URL(id, "group_specs"), payload)
	if err != nil {
		return convert(p.Evidence(response, err), t.Wrap(ctx, op, err))
	}
	members, err := object(response)
	if err != nil {
		return convert(p.Evidence(response, nil), t.Wrap(ctx, op, response.Fail(err)))
	}
	specs, present := members["group_specs"]
	if !present {
		specs = json.RawMessage("{}")
	}
	value := p.Evidence(response, nil)
	spec.Project(value, p.Seed(map[string]json.RawMessage{"group_specs": specs}, id))
	return convert(value, t.Wrap(ctx, op, p.Guard(ctx)))
}

// GetGroupSpecsProperty implements get_group_type_group_specs_property: GET
// group_types/{id}/group_specs/{prop}. Value is the response member named
// prop, JSON null when the member is absent.
func (a *API) GetGroupSpecsProperty(ctx context.Context, id, prop string, options ...GetOption) (*SpecProperty, error) {
	return a.property(ctx, "GetGroupSpecsProperty", id, prop, http.MethodGet, nil, false, options)
}

// UpdateGroupSpecsProperty implements update_group_type_group_specs_property:
// PUT group_types/{id}/group_specs/{prop} with {prop: value}. The response
// must contain the prop member, as Python indexes it.
func (a *API) UpdateGroupSpecsProperty(ctx context.Context, id, prop string, value any, options ...GetOption) (*SpecProperty, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, a.target().Wrap(ctx, "UpdateGroupSpecsProperty", err)
	}
	return a.property(ctx, "UpdateGroupSpecsProperty", id, prop, http.MethodPut, map[string]json.RawMessage{prop: raw}, true, options)
}

func (a *API) property(ctx context.Context, op, id, prop, method string, body map[string]json.RawMessage, required bool, options []GetOption) (*SpecProperty, error) {
	t := a.target()
	err := errors.Join(cinderowned.ValidateID("group type ID", id), cinderowned.ValidateID("group spec name", prop))
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	p, _, err := cinderowned.Begin(ctx, t, GetOpts{}, cinderowned.CopyRead, readHeaders, version, options)
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	var payload any
	if body != nil {
		payload = body
	}
	response, err := p.Do(method, p.URL(id, "group_specs", prop), payload)
	result := &SpecProperty{GroupTypeID: id, Key: prop}
	evidence := p.Evidence(response, err)
	if evidence != nil {
		result.Envelope, result.Header, result.StatusCode = evidence.Envelope, evidence.Header, evidence.StatusCode
	}
	if err != nil {
		return result, t.Wrap(ctx, op, err)
	}
	members, err := object(response)
	if err == nil {
		raw, present := members[prop]
		if !present && required {
			err = cinderowned.Invalid("group spec response has no %q member", prop)
		}
		if !present {
			raw = json.RawMessage("null")
		}
		result.Value = bytes.Clone(raw)
	}
	if err = errors.Join(err, p.Guard(ctx)); err != nil {
		return result, t.Wrap(ctx, op, response.Fail(err))
	}
	return result, nil
}

// DeleteGroupSpecsProperty implements delete_group_type_group_specs_property:
// DELETE group_types/{id}/group_specs/{prop}. A clean 404 returns nil unless
// WithDeleteIgnoreMissing(false) is given.
func (a *API) DeleteGroupSpecsProperty(ctx context.Context, id, prop string, options ...DeleteOption) error {
	const op = "DeleteGroupSpecsProperty"
	t := a.target()
	if err := errors.Join(cinderowned.ValidateID("group type ID", id), cinderowned.ValidateID("group spec name", prop)); err != nil {
		return t.Wrap(ctx, op, err)
	}
	p, v, err := cinderowned.Begin(ctx, t, DeleteOpts{}, cinderowned.CopyDelete, deleteHeaders, version, options)
	if err != nil {
		return t.Wrap(ctx, op, err)
	}
	return t.Wrap(ctx, op, p.Delete(p.URL(id, "group_specs", prop), v.IgnoreMissing == nil || *v.IgnoreMissing))
}

// object follows response.json() followed by dict access: the accepted body
// must be one JSON object.
func object(response *rest.Response) (map[string]json.RawMessage, error) {
	raw := bytes.TrimSpace(response.Body)
	if len(raw) == 0 || raw[0] != '{' || !json.Valid(raw) {
		return nil, cinderowned.Invalid("group spec response must be a JSON object")
	}
	var members map[string]json.RawMessage
	if err := json.Unmarshal(raw, &members); err != nil {
		return nil, err
	}
	return members, nil
}

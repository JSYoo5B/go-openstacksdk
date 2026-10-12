// Package defaulttypes implements the pinned openstacksdk block_storage v3
// default volume type proxy calls over Cinder default-types, a route
// Gophercloud v2.15.0 does not bind.
package defaulttypes

import (
	"context"
	"encoding/json"
	"iter"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/internal/cinderowned"
	"github.com/gophercloud/gophercloud/v2"
)

const collectionPath = "default-types"

var spec = &cinderowned.Spec{
	Kind: "default_types", ResourceKey: "default_type", ResourcesKey: "default_types",
	Fields: []string{"id", "name", "project_id", "volume_type_id"},
}

// version follows the proxy: utils.supports_microversion(3.67) must be true,
// then DefaultType._get_microversion selects the configured microversion or
// min(3.67, server maximum).
var version = cinderowned.Version{Ceiling: "3.67", Required: "3.67"}

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

func readHeaders(v Opts) map[string]string         { return v.Headers }
func deleteHeaders(v DeleteOpts) map[string]string { return v.Headers }

// List implements default_types(): GET default-types as a lazy iterator.
// Every iteration checks the microversion and sends the requests again.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*DefaultType, error] {
	return func(yield func(*DefaultType, error) bool) {
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
func (a *API) All(ctx context.Context, options ...ListOption) ([]*DefaultType, error) {
	var values []*DefaultType
	for value, err := range a.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// Get implements show_default_type: GET default-types/{project_id}.
func (a *API) Get(ctx context.Context, projectID string, options ...Option) (*DefaultType, error) {
	const op = "Get"
	t := a.target()
	if err := cinderowned.ValidateID("project ID", projectID); err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	p, _, err := cinderowned.Begin(ctx, t, Opts{}, cinderowned.CopyRead, readHeaders, version, options)
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	return convert(t.Wrapper(ctx, op)(p.Get(projectID, nil)))
}

// Set implements set_default_type: PUT default-types/{project_id} with
// {"default_type": {"volume_type": volumeTypeID}}. Like Python, an empty
// volumeTypeID is sent under its attribute name volume_type_id instead.
func (a *API) Set(ctx context.Context, projectID, volumeTypeID string, options ...Option) (*DefaultType, error) {
	const op = "Set"
	t := a.target()
	if err := cinderowned.ValidateID("project ID", projectID); err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	p, _, err := cinderowned.Begin(ctx, t, Opts{}, cinderowned.CopyRead, readHeaders, version, options)
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	raw, _ := json.Marshal(volumeTypeID)
	key := "volume_type"
	if volumeTypeID == "" {
		key = "volume_type_id"
	}
	body := map[string]any{"default_type": map[string]json.RawMessage{key: raw}}
	response, err := p.Do(http.MethodPut, p.URL(projectID), body)
	if err != nil {
		return convert(p.Evidence(response, err), t.Wrap(ctx, op, err))
	}
	return convert(t.Wrapper(ctx, op)(p.Record(response, p.Seed(map[string]json.RawMessage{"volume_type_id": raw}, projectID))))
}

// Unset implements unset_default_type: DELETE default-types/{project_id}.
// Python always ignores a 404; WithDeleteIgnoreMissing(false) keeps it.
func (a *API) Unset(ctx context.Context, projectID string, options ...DeleteOption) error {
	const op = "Unset"
	t := a.target()
	if err := cinderowned.ValidateID("project ID", projectID); err != nil {
		return t.Wrap(ctx, op, err)
	}
	p, v, err := cinderowned.Begin(ctx, t, DeleteOpts{}, cinderowned.CopyDelete, deleteHeaders, version, options)
	if err != nil {
		return t.Wrap(ctx, op, err)
	}
	return t.Wrap(ctx, op, p.Delete(p.URL(projectID), v.IgnoreMissing == nil || *v.IgnoreMissing))
}

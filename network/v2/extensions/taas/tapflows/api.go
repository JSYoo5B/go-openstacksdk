// Package tapflows implements the pinned openstacksdk network v2 tap flow
// proxy calls over the tap-as-a-service /taas/tap_flows route, which
// Gophercloud v2.15.0 does not bind.
package tapflows

import (
	"context"
	"iter"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/internal/ownedrecord"
	"github.com/gophercloud/gophercloud/v2"
)

// TapFlow inherits Resource directly: no revision_number cast, If-Match
// header or tag defaults. Unknown Body attributes stay in the body.
var spec = &ownedrecord.Spec{
	Kind: "tapflows",
	Profile: neutronfind.Profile{
		Kind: "tapflows", Path: "taas/tap_flows", Singular: "tap_flow", Plural: "tap_flows",
		IDKey: "id", NameKey: "name", PreserveUnknownBody: true, NameQueryHint: true,
		SourceFindPolicy: true, StopFaultRetryCallbacks: true,
		Body: []neutronfind.Descriptor{
			{Name: "id", Wire: "id", Missing: "null"},
			{Name: "name", Wire: "name", Missing: "null"},
			{Name: "description", Wire: "description", Missing: "null"},
			{Name: "project_id", Wire: "project_id", Alias: "tenant_id", Missing: "null"},
			{Name: "tenant_id", Wire: "tenant_id", Missing: "null"},
			{Name: "tap_service_id", Wire: "tap_service_id", Missing: "null"},
			{Name: "direction", Wire: "direction", Missing: "null"},
			{Name: "status", Wire: "status", Missing: "null"},
			{Name: "source_port", Wire: "source_port", Missing: "null"},
		},
		Query: []neutronfind.QueryPair{
			{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"},
			{Name: "sort_key", Wire: "sort_key"}, {Name: "sort_dir", Wire: "sort_dir"},
			{Name: "fields", Wire: "fields"}, {Name: "name", Wire: "name"},
			{Name: "project_id", Wire: "project_id"},
		},
	},
	UpdateReserved: []string{"tap_flow"},
}

// API shares the selected authenticated network client without owning it.
type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }
func (a *API) RawClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}

func (a *API) binding() ownedrecord.Binding {
	var client func() *gophercloud.ServiceClient
	if a != nil {
		client = func() *gophercloud.ServiceClient { return a.client }
	}
	return ownedrecord.Binding{Spec: spec, Client: client}
}

// Create implements create_tap_flow(**attrs): POST taas/tap_flows with
// {"tap_flow": attrs} carrying only the given attributes.
func (a *API) Create(ctx context.Context, options ...CreateOption) (*TapFlow, error) {
	return convert(ownedrecord.Create(ctx, a.binding(), "", options))
}

// Get implements get_tap_flow(tap_flow, fields=None): GET taas/tap_flows/{id}.
func (a *API) Get(ctx context.Context, id string, options ...GetOption) (*TapFlow, error) {
	return convert(ownedrecord.Get(ctx, a.binding(), "", id, options))
}

// Update implements update_tap_flow(tap_flow, **attrs): PUT with only the
// given attributes. No attribute means no request.
func (a *API) Update(ctx context.Context, id string, options ...UpdateOption) (*TapFlow, error) {
	return convert(ownedrecord.Update(ctx, a.binding(), "", id, options))
}

// Delete implements delete_tap_flow. A clean 404 returns nil unless
// WithDeleteIgnoreMissing(false) is given.
func (a *API) Delete(ctx context.Context, id string, options ...DeleteOption) error {
	return ownedrecord.Delete(ctx, a.binding(), "", id, options)
}

// List implements tap_flows(**query) as a lazy iterator.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*TapFlow, error] {
	source := ownedrecord.List(ctx, a.binding(), "", options)
	return func(yield func(*TapFlow, error) bool) {
		for record, err := range source {
			if !yield(fromRecord(record), err) || err != nil {
				return
			}
		}
	}
}

// All collects List and discards partial rows on failure.
func (a *API) All(ctx context.Context, options ...ListOption) ([]*TapFlow, error) {
	var values []*TapFlow
	for value, err := range a.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// Find implements find_tap_flow: GET by ID first and, after a clean 400, 403
// or 404, the complete list with ?name= matched by exact ID or name. A
// missing flow returns nil, nil unless WithFindIgnoreMissing(false).
func (a *API) Find(ctx context.Context, nameOrID string, options ...FindOption) (*TapFlow, error) {
	return convert(ownedrecord.Find(ctx, a.binding(), "", nameOrID, options))
}

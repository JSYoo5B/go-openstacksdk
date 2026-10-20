// Package tapservices implements the pinned openstacksdk network v2 tap service
// proxy calls over the tap-as-a-service /taas/tap_services route, which
// Gophercloud v2.15.0 does not bind.
package tapservices

import (
	"context"
	"iter"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/internal/ownedrecord"
	"github.com/gophercloud/gophercloud/v2"
)

// TapService inherits Resource directly: no revision_number cast, If-Match
// header or tag defaults. Unknown Body attributes stay in the body.
var spec = &ownedrecord.Spec{
	Kind: "tapservices",
	Profile: neutronfind.Profile{
		Kind: "tapservices", Path: "taas/tap_services", Singular: "tap_service", Plural: "tap_services",
		IDKey: "id", NameKey: "name", PreserveUnknownBody: true, NameQueryHint: true,
		SourceFindPolicy: true, StopFaultRetryCallbacks: true,
		Body: []neutronfind.Descriptor{
			{Name: "id", Wire: "id", Missing: "null"},
			{Name: "name", Wire: "name", Missing: "null"},
			{Name: "description", Wire: "description", Missing: "null"},
			{Name: "project_id", Wire: "project_id", Alias: "tenant_id", Missing: "null"},
			{Name: "tenant_id", Wire: "tenant_id", Missing: "null"},
			{Name: "port_id", Wire: "port_id", Missing: "null"},
			{Name: "status", Wire: "status", Missing: "null"},
		},
		Query: []neutronfind.QueryPair{
			{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"},
			{Name: "sort_key", Wire: "sort_key"}, {Name: "sort_dir", Wire: "sort_dir"},
			{Name: "fields", Wire: "fields"}, {Name: "name", Wire: "name"},
			{Name: "project_id", Wire: "project_id"},
		},
	},
	UpdateReserved: []string{"tap_service"},
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

// Create implements create_tap_service(**attrs): POST taas/tap_services with
// {"tap_service": attrs} carrying only the given attributes.
func (a *API) Create(ctx context.Context, options ...CreateOption) (*TapService, error) {
	return convert(ownedrecord.Create(ctx, a.binding(), "", options))
}

// Get implements get_tap_service(tap_service, fields=None): GET taas/tap_services/{id}.
func (a *API) Get(ctx context.Context, id string, options ...GetOption) (*TapService, error) {
	return convert(ownedrecord.Get(ctx, a.binding(), "", id, options))
}

// Update implements update_tap_service(tap_service, **attrs): PUT with only the
// given attributes. No attribute means no request.
func (a *API) Update(ctx context.Context, id string, options ...UpdateOption) (*TapService, error) {
	return convert(ownedrecord.Update(ctx, a.binding(), "", id, options))
}

// Delete implements delete_tap_service. A clean 404 returns nil unless
// WithDeleteIgnoreMissing(false) is given.
func (a *API) Delete(ctx context.Context, id string, options ...DeleteOption) error {
	return ownedrecord.Delete(ctx, a.binding(), "", id, options)
}

// List implements tap_services(**query) as a lazy iterator.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*TapService, error] {
	source := ownedrecord.List(ctx, a.binding(), "", options)
	return func(yield func(*TapService, error) bool) {
		for record, err := range source {
			if !yield(fromRecord(record), err) || err != nil {
				return
			}
		}
	}
}

// All collects List and discards partial rows on failure.
func (a *API) All(ctx context.Context, options ...ListOption) ([]*TapService, error) {
	var values []*TapService
	for value, err := range a.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// Find implements find_tap_service: GET by ID first and, after a clean 400, 403
// or 404, the complete list with ?name= matched by exact ID or name. A
// missing service returns nil, nil unless WithFindIgnoreMissing(false).
func (a *API) Find(ctx context.Context, nameOrID string, options ...FindOption) (*TapService, error) {
	return convert(ownedrecord.Find(ctx, a.binding(), "", nameOrID, options))
}

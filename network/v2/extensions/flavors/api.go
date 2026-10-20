// Package flavors implements the pinned openstacksdk network v2 service
// flavor proxy calls over Neutron /v2.0/flavors, a route Gophercloud v2.15.0
// does not bind, including the flavor and service profile association.
package flavors

import (
	"context"
	"iter"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/internal/ownedrecords"
	"github.com/gophercloud/gophercloud/v2"
)

// spec is the pinned Flavor Resource: it inherits Resource directly, so no
// NetworkResource revision_number or If-Match header applies. name is a
// declared query parameter, so find adds name=<name_or_id> to its list.
var spec = &ownedrecords.Spec{Resource: "flavors", Bound: "flavor", Profile: neutronfind.Profile{
	Kind: "flavors", Path: "flavors", Singular: "flavor", Plural: "flavors",
	IDKey: "id", NameKey: "name", PreserveUnknownBody: true, NameQueryHint: true,
	VersionedPath: "/v2.0/flavors", SourceFindPolicy: true, StopFaultRetryCallbacks: true,
	Body: []neutronfind.Descriptor{
		{Name: "id", Wire: "id", Missing: "null"},
		{Name: "name", Wire: "name", Missing: "null"},
		{Name: "description", Wire: "description", Missing: "null"},
		{Name: "is_enabled", Wire: "enabled", Kind: neutronfind.Boolean, Missing: "null"},
		{Name: "service_type", Wire: "service_type", Missing: "null"},
		{Name: "service_profile_ids", Wire: "service_profiles", Kind: neutronfind.List, Missing: "null"},
	},
	Query: []neutronfind.QueryPair{
		{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"},
		{Name: "description", Wire: "description"}, {Name: "fields", Wire: "fields"},
		{Name: "name", Wire: "name"}, {Name: "service_type", Wire: "service_type"},
		{Name: "sort_key", Wire: "sort_key"}, {Name: "sort_dir", Wire: "sort_dir"},
		{Name: "is_enabled", Wire: "enabled"},
	},
}}

// API shares the selected authenticated network client without owning it.
type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }
func (a *API) RawClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}

func (a *API) target() ownedrecords.Target {
	var client func() *gophercloud.ServiceClient
	if a != nil {
		client = func() *gophercloud.ServiceClient { return a.client }
	}
	return ownedrecords.Target{Spec: spec, Client: client}
}

// Create implements create_flavor(**attrs): POST flavors with
// {"flavor": attrs}. Python names map to wire keys; unknown keys are kept.
func (a *API) Create(ctx context.Context, options ...CreateOption) (*Flavor, error) {
	return convert(ownedrecords.Create(ctx, a.target(), options))
}

// Get implements get_flavor(flavor, fields=...): GET flavors/{id}.
func (a *API) Get(ctx context.Context, id string, options ...GetOption) (*Flavor, error) {
	return convert(ownedrecords.Get(ctx, a.target(), id, options))
}

// Update implements update_flavor(flavor, **attrs): PUT flavors/{id} with
// only the given attributes. No attribute means no request.
func (a *API) Update(ctx context.Context, id string, options ...UpdateOption) (*Flavor, error) {
	return convert(ownedrecords.Update(ctx, a.target(), id, options))
}

// Delete implements delete_flavor(flavor, ignore_missing=True). A clean 404
// returns a receipt with IgnoredMissing unless WithDeleteIgnoreMissing(false).
func (a *API) Delete(ctx context.Context, id string, options ...DeleteOption) (*DeleteResult, error) {
	return ownedrecords.Delete(ctx, a.target(), id, options)
}

// List implements flavors(**query) as a lazy iterator.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*Flavor, error] {
	source := ownedrecords.List(ctx, a.target(), options)
	return func(yield func(*Flavor, error) bool) {
		for record, err := range source {
			if !yield(fromRecord(record), err) || err != nil {
				return
			}
		}
	}
}

// All collects List and discards partial rows on failure.
func (a *API) All(ctx context.Context, options ...ListOption) ([]*Flavor, error) {
	var values []*Flavor
	for value, err := range a.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// Find implements find_flavor(name_or_id, ignore_missing=True, **query).
func (a *API) Find(ctx context.Context, nameOrID string, options ...FindOption) (*Flavor, error) {
	return convert(ownedrecords.Find(ctx, a.target(), nameOrID, options))
}

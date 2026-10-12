// Package serviceprofiles implements the pinned openstacksdk network v2
// service flavor profile proxy calls over Neutron /v2.0/service_profiles, a
// route Gophercloud v2.15.0 does not bind.
package serviceprofiles

import (
	"context"
	"iter"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/internal/ownedrecords"
	"github.com/gophercloud/gophercloud/v2"
)

// spec is the pinned ServiceProfile Resource: it inherits Resource directly
// (id and name), and project_id falls back to tenant_id. name is not a
// declared query parameter, so find lists without a name hint.
var spec = &ownedrecords.Spec{Resource: "serviceprofiles", Bound: "service_profile", Profile: neutronfind.Profile{
	Kind: "serviceprofiles", Path: "service_profiles", Singular: "service_profile", Plural: "service_profiles",
	IDKey: "id", NameKey: "name", PreserveUnknownBody: true, NameQueryHint: false,
	VersionedPath: "/v2.0/service_profiles", SourceFindPolicy: true, StopFaultRetryCallbacks: true,
	Body: []neutronfind.Descriptor{
		{Name: "id", Wire: "id", Missing: "null"},
		{Name: "name", Wire: "name", Missing: "null"},
		{Name: "description", Wire: "description", Missing: "null"},
		{Name: "driver", Wire: "driver", Missing: "null"},
		{Name: "is_enabled", Wire: "enabled", Kind: neutronfind.Boolean, Missing: "null"},
		{Name: "meta_info", Wire: "metainfo", Missing: "null"},
		{Name: "project_id", Wire: "project_id", Alias: "tenant_id", Missing: "null"},
		{Name: "tenant_id", Wire: "tenant_id", Missing: "null"},
	},
	Query: []neutronfind.QueryPair{
		{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"},
		{Name: "description", Wire: "description"}, {Name: "fields", Wire: "fields"},
		{Name: "driver", Wire: "driver"}, {Name: "project_id", Wire: "project_id"},
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

// Create implements create_service_profile(**attrs): POST service_profiles
// with {"service_profile": attrs}. Unknown keys are kept.
func (a *API) Create(ctx context.Context, options ...CreateOption) (*ServiceProfile, error) {
	return convert(ownedrecords.Create(ctx, a.target(), options))
}

// Get implements get_service_profile(service_profile, fields=...).
func (a *API) Get(ctx context.Context, id string, options ...GetOption) (*ServiceProfile, error) {
	return convert(ownedrecords.Get(ctx, a.target(), id, options))
}

// Update implements update_service_profile(service_profile, **attrs): PUT
// service_profiles/{id} with only the given attributes.
func (a *API) Update(ctx context.Context, id string, options ...UpdateOption) (*ServiceProfile, error) {
	return convert(ownedrecords.Update(ctx, a.target(), id, options))
}

// Delete implements delete_service_profile(service_profile,
// ignore_missing=True).
func (a *API) Delete(ctx context.Context, id string, options ...DeleteOption) (*DeleteResult, error) {
	return ownedrecords.Delete(ctx, a.target(), id, options)
}

// List implements service_profiles(**query) as a lazy iterator.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*ServiceProfile, error] {
	source := ownedrecords.List(ctx, a.target(), options)
	return func(yield func(*ServiceProfile, error) bool) {
		for record, err := range source {
			if !yield(fromRecord(record), err) || err != nil {
				return
			}
		}
	}
}

// All collects List and discards partial rows on failure.
func (a *API) All(ctx context.Context, options ...ListOption) ([]*ServiceProfile, error) {
	var values []*ServiceProfile
	for value, err := range a.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// Find implements find_service_profile(name_or_id, ignore_missing=True,
// **query).
func (a *API) Find(ctx context.Context, nameOrID string, options ...FindOption) (*ServiceProfile, error) {
	return convert(ownedrecords.Find(ctx, a.target(), nameOrID, options))
}

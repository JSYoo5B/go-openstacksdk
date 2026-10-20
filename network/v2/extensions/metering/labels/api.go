// Package labels implements the pinned openstacksdk network v2 metering
// label proxy calls over Neutron /v2.0/metering/metering-labels, a route
// Gophercloud v2.15.0 does not bind.
package labels

import (
	"context"
	"iter"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/internal/ownedrecords"
	"github.com/gophercloud/gophercloud/v2"
)

// spec is the pinned MeteringLabel Resource: it inherits Resource directly
// (id and name), and project_id falls back to tenant_id. name is a declared
// query parameter, so find adds name=<name_or_id> to its list.
var spec = &ownedrecords.Spec{Resource: "meteringlabels", Bound: "metering_label", Profile: neutronfind.Profile{
	Kind: "meteringlabels", Path: "metering/metering-labels", Singular: "metering_label", Plural: "metering_labels",
	IDKey: "id", NameKey: "name", PreserveUnknownBody: true, NameQueryHint: true,
	VersionedPath: "/v2.0/metering/metering-labels", SourceFindPolicy: true, StopFaultRetryCallbacks: true,
	Body: []neutronfind.Descriptor{
		{Name: "id", Wire: "id", Missing: "null"},
		{Name: "name", Wire: "name", Missing: "null"},
		{Name: "description", Wire: "description", Missing: "null"},
		{Name: "project_id", Wire: "project_id", Alias: "tenant_id", Missing: "null"},
		{Name: "tenant_id", Wire: "tenant_id", Missing: "null"},
		{Name: "is_shared", Wire: "shared", Kind: neutronfind.Boolean, Missing: "null"},
	},
	Query: []neutronfind.QueryPair{
		{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"},
		{Name: "description", Wire: "description"}, {Name: "fields", Wire: "fields"},
		{Name: "name", Wire: "name"}, {Name: "project_id", Wire: "project_id"},
		{Name: "sort_key", Wire: "sort_key"}, {Name: "sort_dir", Wire: "sort_dir"},
		{Name: "is_shared", Wire: "shared"},
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

// Create implements create_metering_label(**attrs): POST
// metering/metering-labels with {"metering_label": attrs}.
func (a *API) Create(ctx context.Context, options ...CreateOption) (*MeteringLabel, error) {
	return convert(ownedrecords.Create(ctx, a.target(), options))
}

// Get implements get_metering_label(metering_label, fields=...).
func (a *API) Get(ctx context.Context, id string, options ...GetOption) (*MeteringLabel, error) {
	return convert(ownedrecords.Get(ctx, a.target(), id, options))
}

// Update implements update_metering_label(metering_label, **attrs): PUT
// metering/metering-labels/{id} with only the given attributes.
func (a *API) Update(ctx context.Context, id string, options ...UpdateOption) (*MeteringLabel, error) {
	return convert(ownedrecords.Update(ctx, a.target(), id, options))
}

// Delete implements delete_metering_label(metering_label,
// ignore_missing=True).
func (a *API) Delete(ctx context.Context, id string, options ...DeleteOption) (*DeleteResult, error) {
	return ownedrecords.Delete(ctx, a.target(), id, options)
}

// List implements metering_labels(**query) as a lazy iterator.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*MeteringLabel, error] {
	source := ownedrecords.List(ctx, a.target(), options)
	return func(yield func(*MeteringLabel, error) bool) {
		for record, err := range source {
			if !yield(fromRecord(record), err) || err != nil {
				return
			}
		}
	}
}

// All collects List and discards partial rows on failure.
func (a *API) All(ctx context.Context, options ...ListOption) ([]*MeteringLabel, error) {
	var values []*MeteringLabel
	for value, err := range a.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// Find implements find_metering_label(name_or_id, ignore_missing=True,
// **query).
func (a *API) Find(ctx context.Context, nameOrID string, options ...FindOption) (*MeteringLabel, error) {
	return convert(ownedrecords.Find(ctx, a.target(), nameOrID, options))
}

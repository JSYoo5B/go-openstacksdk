// Package localips implements the pinned openstacksdk network v2 local IP
// proxy calls over the Neutron /local_ips route, which Gophercloud v2.15.0
// does not bind.
package localips

import (
	"context"
	"iter"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/internal/ownedrecord"
	"github.com/gophercloud/gophercloud/v2"
)

// LocalIP inherits Resource directly, not NetworkResource: revision_number
// is an untyped Body value and if_revision never becomes an If-Match header.
// project_id has no tenant_id alias. Unknown Body attributes stay in the body.
var spec = &ownedrecord.Spec{
	Kind: "localips",
	Profile: neutronfind.Profile{
		Kind: "localips", Path: "local_ips", Singular: "local_ip", Plural: "local_ips",
		IDKey: "id", NameKey: "name", PreserveUnknownBody: true, NameQueryHint: true,
		SourceFindPolicy: true, StopFaultRetryCallbacks: true,
		Body: []neutronfind.Descriptor{
			{Name: "id", Wire: "id", Missing: "null"},
			{Name: "name", Wire: "name", Missing: "null"},
			{Name: "created_at", Wire: "created_at", Missing: "null"},
			{Name: "description", Wire: "description", Missing: "null"},
			{Name: "ip_mode", Wire: "ip_mode", Missing: "null"},
			{Name: "local_ip_address", Wire: "local_ip_address", Missing: "null"},
			{Name: "local_port_id", Wire: "local_port_id", Missing: "null"},
			{Name: "network_id", Wire: "network_id", Missing: "null"},
			{Name: "project_id", Wire: "project_id", Missing: "null"},
			{Name: "revision_number", Wire: "revision_number", Missing: "null"},
			{Name: "updated_at", Wire: "updated_at", Missing: "null"},
		},
		Query: []neutronfind.QueryPair{
			{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"},
			{Name: "sort_key", Wire: "sort_key"}, {Name: "sort_dir", Wire: "sort_dir"},
			{Name: "name", Wire: "name"}, {Name: "description", Wire: "description"},
			{Name: "fields", Wire: "fields"}, {Name: "project_id", Wire: "project_id"},
			{Name: "network_id", Wire: "network_id"}, {Name: "local_port_id", Wire: "local_port_id"},
			{Name: "local_ip_address", Wire: "local_ip_address"}, {Name: "ip_mode", Wire: "ip_mode"},
		},
	},
	UpdateReserved: []string{"local_ip", "if_revision"},
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

// Create implements create_local_ip(**attrs): POST local_ips with
// {"local_ip": attrs} carrying only the given attributes.
func (a *API) Create(ctx context.Context, options ...CreateOption) (*LocalIP, error) {
	return convert(ownedrecord.Create(ctx, a.binding(), "", options))
}

// Get implements get_local_ip(local_ip, fields=None): GET local_ips/{id}.
func (a *API) Get(ctx context.Context, id string, options ...GetOption) (*LocalIP, error) {
	return convert(ownedrecord.Get(ctx, a.binding(), "", id, options))
}

// Update implements update_local_ip(local_ip, if_revision=None, **attrs): PUT
// with only the given attributes. No attribute means no request. Python's
// if_revision has no effect for LocalIP, so there is no option for it.
func (a *API) Update(ctx context.Context, id string, options ...UpdateOption) (*LocalIP, error) {
	return convert(ownedrecord.Update(ctx, a.binding(), "", id, options))
}

// Delete implements delete_local_ip. A clean 404 returns nil unless
// WithDeleteIgnoreMissing(false) is given.
func (a *API) Delete(ctx context.Context, id string, options ...DeleteOption) error {
	return ownedrecord.Delete(ctx, a.binding(), "", id, options)
}

// List implements local_ips(**query) as a lazy iterator.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*LocalIP, error] {
	source := ownedrecord.List(ctx, a.binding(), "", options)
	return func(yield func(*LocalIP, error) bool) {
		for record, err := range source {
			if !yield(fromRecord(record), err) || err != nil {
				return
			}
		}
	}
}

// All collects List and discards partial rows on failure.
func (a *API) All(ctx context.Context, options ...ListOption) ([]*LocalIP, error) {
	var values []*LocalIP
	for value, err := range a.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// Find implements find_local_ip: GET by ID first and, after a clean 400, 403
// or 404, the complete list with ?name= matched by exact ID or name. A
// missing local IP returns nil, nil unless WithFindIgnoreMissing(false).
func (a *API) Find(ctx context.Context, nameOrID string, options ...FindOption) (*LocalIP, error) {
	return convert(ownedrecord.Find(ctx, a.binding(), "", nameOrID, options))
}

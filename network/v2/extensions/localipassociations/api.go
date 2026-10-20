// Package localipassociations implements the pinned openstacksdk network v2
// local IP association proxy calls over the Neutron
// /local_ips/{local_ip_id}/port_associations route, which Gophercloud
// v2.15.0 does not bind.
package localipassociations

import (
	"context"
	"iter"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/internal/ownedrecord"
	"github.com/gophercloud/gophercloud/v2"
)

// LocalIPAssociation inherits Resource directly. local_ip_id is a URI
// attribute; id and name are the Resource base Body attributes, which Neutron
// port associations normally omit. The query mapping has no name, so find
// sends no name hint.
var spec = &ownedrecord.Spec{
	Kind: "localipassociations",
	Profile: neutronfind.Profile{
		Kind: "localipassociations", Singular: "port_association", Plural: "port_associations",
		IDKey: "id", NameKey: "name", PreserveUnknownBody: true,
		SourceFindPolicy: true, StopFaultRetryCallbacks: true,
		Body: []neutronfind.Descriptor{
			{Name: "id", Wire: "id", Missing: "null"},
			{Name: "name", Wire: "name", Missing: "null"},
			{Name: "fixed_port_id", Wire: "fixed_port_id", Missing: "null"},
			{Name: "fixed_ip", Wire: "fixed_ip", Missing: "null"},
			{Name: "host", Wire: "host", Missing: "null"},
			{Name: "local_ip_address", Wire: "local_ip_address", Missing: "null"},
		},
		Query: []neutronfind.QueryPair{
			{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"},
			{Name: "fields", Wire: "fields"}, {Name: "fixed_port_id", Wire: "fixed_port_id"},
			{Name: "fixed_ip", Wire: "fixed_ip"}, {Name: "host", Wire: "host"},
			{Name: "sort_key", Wire: "sort_key"}, {Name: "sort_dir", Wire: "sort_dir"},
		},
	},
	Parent: "local_ip_id", ParentArg: "local_ip", ParentLabel: "local IP ID",
	Route: func(parent string) string { return "local_ips/" + parent + "/port_associations" },
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

// Create implements create_local_ip_association(local_ip, **attrs): POST
// local_ips/{local_ip_id}/port_associations with {"port_association": attrs}.
func (a *API) Create(ctx context.Context, localIPID string, options ...CreateOption) (*LocalIPAssociation, error) {
	return convert(ownedrecord.Create(ctx, a.binding(), localIPID, options))
}

// Get implements get_local_ip_association(local_ip_association, local_ip,
// fields=None): GET local_ips/{local_ip_id}/port_associations/{id}.
func (a *API) Get(ctx context.Context, localIPID, id string, options ...GetOption) (*LocalIPAssociation, error) {
	return convert(ownedrecord.Get(ctx, a.binding(), localIPID, id, options))
}

// Delete implements delete_local_ip_association(local_ip, fixed_port_id):
// DELETE local_ips/{local_ip_id}/port_associations/{fixed_port_id}. A clean
// 404 returns nil unless WithDeleteIgnoreMissing(false) is given.
func (a *API) Delete(ctx context.Context, localIPID, fixedPortID string, options ...DeleteOption) error {
	return ownedrecord.Delete(ctx, a.binding(), localIPID, fixedPortID, options)
}

// List implements local_ip_associations(local_ip, **query) as a lazy
// iterator.
func (a *API) List(ctx context.Context, localIPID string, options ...ListOption) iter.Seq2[*LocalIPAssociation, error] {
	source := ownedrecord.List(ctx, a.binding(), localIPID, options)
	return func(yield func(*LocalIPAssociation, error) bool) {
		for record, err := range source {
			if !yield(fromRecord(record), err) || err != nil {
				return
			}
		}
	}
}

// All collects List and discards partial rows on failure.
func (a *API) All(ctx context.Context, localIPID string, options ...ListOption) ([]*LocalIPAssociation, error) {
	var values []*LocalIPAssociation
	for value, err := range a.List(ctx, localIPID, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// Find implements find_local_ip_association(name_or_id, local_ip): GET by
// ID with ?local_ip_id= first and, after a clean 400, 403 or 404, the complete
// association list matched by exact id or name Body value. A missing
// association returns nil, nil unless WithFindIgnoreMissing(false).
func (a *API) Find(ctx context.Context, localIPID, nameOrID string, options ...FindOption) (*LocalIPAssociation, error) {
	return convert(ownedrecord.Find(ctx, a.binding(), localIPID, nameOrID, options))
}

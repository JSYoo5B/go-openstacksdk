// Package federationprotocols implements the pinned openstacksdk identity v3
// federation protocol proxy calls over Keystone
// OS-FEDERATION/identity_providers/{idp_id}/protocols, a route Gophercloud
// v2.15.0 does not bind. Every call names its identity provider explicitly.
package federationprotocols

import (
	"context"
	"iter"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/internal/osfederation"
	"github.com/gophercloud/gophercloud/v2"
)

var spec = &osfederation.Spec{
	Kind: "federation_protocols", ResourceKey: "protocol", ResourcesKey: "protocols",
	Fields:  []string{"id", "mapping_id"},
	Attrs:   map[string]string{"id": "id", "name": "id", "mapping_id": "mapping_id"},
	Query:   map[string]string{"id": "id"},
	Filters: map[string]string{"name": "id", "mapping_id": "mapping_id"},
}

// API shares the selected authenticated identity client without owning it.
type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }
func (a *API) RawClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}

func (a *API) target(idpID string) osfederation.Target {
	var client func() *gophercloud.ServiceClient
	if a != nil {
		client = func() *gophercloud.ServiceClient { return a.client }
	}
	return osfederation.Target{Spec: spec, Client: client, Err: osfederation.ValidateID("identity provider ID", idpID),
		Path: "OS-FEDERATION/identity_providers/" + url.PathEscape(idpID) + "/protocols"}
}

// Create implements create_federation_protocol: PUT
// OS-FEDERATION/identity_providers/{idp_id}/protocols/{id} with
// {"protocol": attrs} where id (or its alias name) is removed from the body.
func (a *API) Create(ctx context.Context, idpID, id string, options ...CreateOption) (*FederationProtocol, error) {
	return convert(idpID)(osfederation.Create(ctx, a.target(idpID), id, options))
}

// Get implements get_federation_protocol.
func (a *API) Get(ctx context.Context, idpID, id string, options ...GetOption) (*FederationProtocol, error) {
	return convert(idpID)(osfederation.Get(ctx, a.target(idpID), id, options))
}

// Update implements update_federation_protocol: PATCH with only the given
// declared attributes. No declared attribute means no request.
func (a *API) Update(ctx context.Context, idpID, id string, options ...UpdateOption) (*FederationProtocol, error) {
	return convert(idpID)(osfederation.Update(ctx, a.target(idpID), id, options))
}

// Delete implements delete_federation_protocol. A clean 404 returns nil
// unless WithDeleteIgnoreMissing(false) is given.
func (a *API) Delete(ctx context.Context, idpID, id string, options ...DeleteOption) error {
	return osfederation.Delete(ctx, a.target(idpID), id, options)
}

// List implements federation_protocols(idp, **query) as a lazy iterator.
func (a *API) List(ctx context.Context, idpID string, options ...ListOption) iter.Seq2[*FederationProtocol, error] {
	source := osfederation.List(ctx, a.target(idpID), options)
	return func(yield func(*FederationProtocol, error) bool) {
		for record, err := range source {
			if !yield(fromRecord(record, idpID), err) || err != nil {
				return
			}
		}
	}
}

// All collects List.
func (a *API) All(ctx context.Context, idpID string, options ...ListOption) ([]*FederationProtocol, error) {
	var values []*FederationProtocol
	for value, err := range a.List(ctx, idpID, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// Find implements find_federation_protocol. Like Resource.find, the direct
// GET carries the URI attribute as ?idp_id={idp_id}; after a clean 400, 403
// or 404 the complete protocol list is matched by exact ID (name is the ID).
// A missing protocol returns nil, nil unless WithFindIgnoreMissing(false).
func (a *API) Find(ctx context.Context, idpID, nameOrID string, options ...FindOption) (*FederationProtocol, error) {
	return convert(idpID)(osfederation.Find(ctx, a.target(idpID), nameOrID, url.Values{"idp_id": {idpID}}, options))
}

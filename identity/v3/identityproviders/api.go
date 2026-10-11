// Package identityproviders implements the pinned openstacksdk identity v3
// identity provider proxy calls over Keystone OS-FEDERATION/identity_providers,
// a route Gophercloud v2.15.0 does not bind.
package identityproviders

import (
	"context"
	"iter"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/internal/osfederation"
	"github.com/gophercloud/gophercloud/v2"
)

const collectionPath = "OS-FEDERATION/identity_providers"

var spec = &osfederation.Spec{
	Kind: "identity_providers", ResourceKey: "identity_provider", ResourcesKey: "identity_providers",
	Fields: []string{"id", "authorization_ttl", "description", "domain_id", "enabled", "remote_ids"},
	Attrs: map[string]string{"id": "id", "name": "id", "authorization_ttl": "authorization_ttl", "description": "description",
		"domain_id": "domain_id", "enabled": "enabled", "is_enabled": "enabled", "remote_ids": "remote_ids"},
	Query: map[string]string{"id": "id", "is_enabled": "enabled", "enabled": "enabled"},
	Filters: map[string]string{"name": "id", "authorization_ttl": "authorization_ttl", "description": "description",
		"domain_id": "domain_id", "remote_ids": "remote_ids"},
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

func (a *API) target() osfederation.Target {
	var client func() *gophercloud.ServiceClient
	if a != nil {
		client = func() *gophercloud.ServiceClient { return a.client }
	}
	return osfederation.Target{Spec: spec, Client: client, Path: collectionPath}
}

// Create implements create_identity_provider: PUT
// OS-FEDERATION/identity_providers/{id} with {"identity_provider": attrs}
// where id (or its alias name) is removed from the body.
func (a *API) Create(ctx context.Context, id string, options ...CreateOption) (*IdentityProvider, error) {
	return convert(osfederation.Create(ctx, a.target(), id, options))
}

// Get implements get_identity_provider: GET OS-FEDERATION/identity_providers/{id}.
func (a *API) Get(ctx context.Context, id string, options ...GetOption) (*IdentityProvider, error) {
	return convert(osfederation.Get(ctx, a.target(), id, options))
}

// Update implements update_identity_provider: PATCH with only the given
// declared attributes. No declared attribute means no request.
func (a *API) Update(ctx context.Context, id string, options ...UpdateOption) (*IdentityProvider, error) {
	return convert(osfederation.Update(ctx, a.target(), id, options))
}

// Delete implements delete_identity_provider. A clean 404 returns nil unless
// WithDeleteIgnoreMissing(false) is given.
func (a *API) Delete(ctx context.Context, id string, options ...DeleteOption) error {
	return osfederation.Delete(ctx, a.target(), id, options)
}

// List implements identity_providers(**query) as a lazy iterator.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*IdentityProvider, error] {
	source := osfederation.List(ctx, a.target(), options)
	return func(yield func(*IdentityProvider, error) bool) {
		for record, err := range source {
			if !yield(fromRecord(record), err) || err != nil {
				return
			}
		}
	}
}

// All collects List.
func (a *API) All(ctx context.Context, options ...ListOption) ([]*IdentityProvider, error) {
	var values []*IdentityProvider
	for value, err := range a.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// Find implements find_identity_provider: GET by ID first and, after a clean
// 400, 403 or 404, the complete list matched by exact ID (name is the ID).
// A missing provider returns nil, nil unless WithFindIgnoreMissing(false).
func (a *API) Find(ctx context.Context, nameOrID string, options ...FindOption) (*IdentityProvider, error) {
	return convert(osfederation.Find(ctx, a.target(), nameOrID, nil, options))
}

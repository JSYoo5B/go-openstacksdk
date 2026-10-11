// Package serviceproviders implements the pinned openstacksdk identity v3
// service provider proxy calls over Keystone OS-FEDERATION/service_providers,
// a route Gophercloud v2.15.0 does not bind.
package serviceproviders

import (
	"context"
	"iter"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/internal/osfederation"
	"github.com/gophercloud/gophercloud/v2"
)

const collectionPath = "OS-FEDERATION/service_providers"

var spec = &osfederation.Spec{
	Kind: "service_providers", ResourceKey: "service_provider", ResourcesKey: "service_providers",
	Fields: []string{"id", "auth_url", "description", "enabled", "relay_state_prefix", "sp_url"},
	Attrs: map[string]string{"id": "id", "name": "id", "auth_url": "auth_url", "description": "description",
		"enabled": "enabled", "is_enabled": "enabled", "relay_state_prefix": "relay_state_prefix", "sp_url": "sp_url"},
	Query: map[string]string{"id": "id", "is_enabled": "enabled", "enabled": "enabled"},
	Filters: map[string]string{"name": "id", "auth_url": "auth_url", "description": "description",
		"relay_state_prefix": "relay_state_prefix", "sp_url": "sp_url"},
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

// Create implements create_service_provider: PUT
// OS-FEDERATION/service_providers/{id} with {"service_provider": attrs}
// where id (or its alias name) is removed from the body.
func (a *API) Create(ctx context.Context, id string, options ...CreateOption) (*ServiceProvider, error) {
	return convert(osfederation.Create(ctx, a.target(), id, options))
}

// Get implements get_service_provider: GET OS-FEDERATION/service_providers/{id}.
func (a *API) Get(ctx context.Context, id string, options ...GetOption) (*ServiceProvider, error) {
	return convert(osfederation.Get(ctx, a.target(), id, options))
}

// Update implements update_service_provider: PATCH with only the given
// declared attributes. No declared attribute means no request.
func (a *API) Update(ctx context.Context, id string, options ...UpdateOption) (*ServiceProvider, error) {
	return convert(osfederation.Update(ctx, a.target(), id, options))
}

// Delete implements delete_service_provider. A clean 404 returns nil unless
// WithDeleteIgnoreMissing(false) is given.
func (a *API) Delete(ctx context.Context, id string, options ...DeleteOption) error {
	return osfederation.Delete(ctx, a.target(), id, options)
}

// List implements service_providers(**query) as a lazy iterator.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*ServiceProvider, error] {
	source := osfederation.List(ctx, a.target(), options)
	return func(yield func(*ServiceProvider, error) bool) {
		for record, err := range source {
			if !yield(fromRecord(record), err) || err != nil {
				return
			}
		}
	}
}

// All collects List.
func (a *API) All(ctx context.Context, options ...ListOption) ([]*ServiceProvider, error) {
	var values []*ServiceProvider
	for value, err := range a.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// Find implements find_service_provider: GET by ID first and, after a clean
// 400, 403 or 404, the complete list matched by exact ID (name is the ID).
// A missing provider returns nil, nil unless WithFindIgnoreMissing(false).
func (a *API) Find(ctx context.Context, nameOrID string, options ...FindOption) (*ServiceProvider, error) {
	return convert(osfederation.Find(ctx, a.target(), nameOrID, nil, options))
}

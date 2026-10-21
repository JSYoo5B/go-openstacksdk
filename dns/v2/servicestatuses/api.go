// Package servicestatuses binds Designate service heartbeats
// (/v2/service_statuses). The route is typically administrative.
package servicestatuses

import (
	"context"
	"encoding/json"
	"iter"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/dns/v2/internal/designate"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// ServiceStatus is one Designate service instance. Capabilities and Stats keep
// their exact JSON values.
type ServiceStatus struct {
	designate.Evidence
	ID            string          `json:"id"`
	Capabilities  json.RawMessage `json:"capabilities"`
	CreatedAt     string          `json:"created_at"`
	HeartbeatedAt string          `json:"heartbeated_at"`
	Hostname      string          `json:"hostname"`
	Links         map[string]any  `json:"links"`
	ServiceName   string          `json:"service_name"`
	Stats         json.RawMessage `json:"stats"`
	Status        string          `json:"status"`
	UpdatedAt     string          `json:"updated_at"`
}

// API owns one authenticated Designate client.
type API struct {
	Resources *resource.Collection[ServiceStatus]
	client    *gophercloud.ServiceClient
	route     designate.Route[ServiceStatus, *ServiceStatus]
}

func New(client *gophercloud.ServiceClient) *API {
	a := &API{client: client, route: designate.Route[ServiceStatus, *ServiceStatus]{Client: client, Segments: []string{"service_statuses"}, PluralKey: "service_statuses"}}
	a.Resources = resource.NewCollection(resource.Adapter[ServiceStatus]{
		Kind: "service statuses",
		Get:  func(ctx context.Context, id string) (*ServiceStatus, error) { return a.Get(ctx, id) },
		ID:   func(v *ServiceStatus) string { return v.ID },
		IterateControlled: func(ctx context.Context, q url.Values, control resource.ListControl) iter.Seq2[*ServiceStatus, error] {
			return a.route.List(ctx, q, nil, control)
		},
	})
	return a
}

func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

// Get sends GET /service_statuses/{id}, accepting only 200.
func (a *API) Get(ctx context.Context, id string) (*ServiceStatus, error) {
	value, err := a.route.Get(ctx, id, nil)
	return value, request.Wrap("Get", "service statuses", designate.NotFound("service status", id, err))
}

// List streams service statuses. Python service_statuses() accepts no filters,
// so this takes none; paging follows links.next like every Designate list.
func (a *API) List(ctx context.Context) iter.Seq2[*ServiceStatus, error] {
	return a.route.List(ctx, nil, nil, resource.ListControl{})
}

// All collects the list with common list options.
func (a *API) All(ctx context.Context, options ...resource.ListOption) ([]*ServiceStatus, error) {
	return a.Resources.All(ctx, options...)
}

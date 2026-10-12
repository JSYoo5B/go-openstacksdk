// Package portpairs implements the pinned openstacksdk network v2
// sfc_port_pair proxy calls over the Neutron networking-sfc
// /v2.0/sfc/port_pairs route, which Gophercloud v2.15.0 does not bind.
package portpairs

import (
	"context"
	"iter"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/internal/sfcrecord"
	"github.com/gophercloud/gophercloud/v2"
)

// spec is the Python SfcPortPair Resource class: Body attributes, descriptor
// types and QueryParameters names. Body attributes outside the query
// mapping (id, service_function_parameters) filter locally.
var spec = sfcrecord.NewSpec("port_pairs", "sfc/port_pairs", "port_pair", "port_pairs",
	map[string]string{"description": "description", "name": "name", "project_id": "project_id", "tenant_id": "tenant_id", "ingress": "ingress", "egress": "egress", "service_function_parameters": "service_function_parameters"},
	map[string]string{"service_function_parameters": sfcrecord.TypeDict},
	[]string{"description", "name", "egress", "ingress", "project_id", "tenant_id"})

// API shares the selected authenticated network client without owning it.
type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }
func (a *API) RawClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}

func (a *API) target() sfcrecord.Target {
	var client func() *gophercloud.ServiceClient
	if a != nil {
		client = func() *gophercloud.ServiceClient { return a.client }
	}
	return sfcrecord.Target{Spec: spec, Client: client}
}

// Create implements create_sfc_port_pair: POST sfc/port_pairs with
// {"port_pair": attrs} holding every given declared attribute.
func (a *API) Create(ctx context.Context, options ...CreateOption) (*PortPair, error) {
	return convert(sfcrecord.Create(ctx, a.target(), options))
}

// Get implements get_sfc_port_pair: GET sfc/port_pairs/{id}.
func (a *API) Get(ctx context.Context, id string, options ...GetOption) (*PortPair, error) {
	return convert(sfcrecord.Get(ctx, a.target(), id, options))
}

// Update implements update_sfc_port_pair: PUT sfc/port_pairs/{id} with only the
// given declared attributes. No declared attribute means no request.
func (a *API) Update(ctx context.Context, id string, options ...UpdateOption) (*PortPair, error) {
	return convert(sfcrecord.Update(ctx, a.target(), id, options))
}

// Delete implements delete_sfc_port_pair. A clean 404 returns nil unless
// WithDeleteIgnoreMissing(false) is given.
func (a *API) Delete(ctx context.Context, id string, options ...DeleteOption) error {
	return sfcrecord.Delete(ctx, a.target(), id, options)
}

// List implements sfc_port_pairs(**query) as a lazy iterator that follows
// the port_pairs_links next link and limit-driven marker pages.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*PortPair, error] {
	source := sfcrecord.List(ctx, a.target(), options)
	return func(yield func(*PortPair, error) bool) {
		for record, err := range source {
			if !yield(fromRecord(record), err) || err != nil {
				return
			}
		}
	}
}

// All collects List.
func (a *API) All(ctx context.Context, options ...ListOption) ([]*PortPair, error) {
	var values []*PortPair
	for value, err := range a.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// Find implements find_sfc_port_pair: GET by ID first and, after a clean 400,
// 403 or 404, the complete list queried by name=name_or_id and matched by
// exact ID or name. A missing port pair returns nil, nil unless
// WithFindIgnoreMissing(false) is given.
func (a *API) Find(ctx context.Context, nameOrID string, options ...FindOption) (*PortPair, error) {
	return convert(sfcrecord.Find(ctx, a.target(), nameOrID, options))
}

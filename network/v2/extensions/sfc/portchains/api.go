// Package portchains implements the pinned openstacksdk network v2
// sfc_port_chain proxy calls over the Neutron networking-sfc
// /v2.0/sfc/port_chains route, which Gophercloud v2.15.0 does not bind.
package portchains

import (
	"context"
	"iter"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/internal/sfcrecord"
	"github.com/gophercloud/gophercloud/v2"
)

// spec is the Python SfcPortChain Resource class: Body attributes, descriptor
// types and QueryParameters names. Body attributes outside the query
// mapping (chain_parameters, flow_classifiers, id, port_pair_groups) filter locally.
var spec = sfcrecord.NewSpec("port_chains", "sfc/port_chains", "port_chain", "port_chains",
	map[string]string{"description": "description", "name": "name", "project_id": "project_id", "tenant_id": "tenant_id", "port_pair_groups": "port_pair_groups", "flow_classifiers": "flow_classifiers", "chain_parameters": "chain_parameters"},
	map[string]string{"port_pair_groups": sfcrecord.TypeList, "flow_classifiers": sfcrecord.TypeList, "chain_parameters": sfcrecord.TypeDict},
	[]string{"description", "name", "project_id", "tenant_id"})

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

// Create implements create_sfc_port_chain: POST sfc/port_chains with
// {"port_chain": attrs} holding every given declared attribute.
func (a *API) Create(ctx context.Context, options ...CreateOption) (*PortChain, error) {
	return convert(sfcrecord.Create(ctx, a.target(), options))
}

// Get implements get_sfc_port_chain: GET sfc/port_chains/{id}.
func (a *API) Get(ctx context.Context, id string, options ...GetOption) (*PortChain, error) {
	return convert(sfcrecord.Get(ctx, a.target(), id, options))
}

// Update implements update_sfc_port_chain: PUT sfc/port_chains/{id} with only the
// given declared attributes. No declared attribute means no request.
func (a *API) Update(ctx context.Context, id string, options ...UpdateOption) (*PortChain, error) {
	return convert(sfcrecord.Update(ctx, a.target(), id, options))
}

// Delete implements delete_sfc_port_chain. A clean 404 returns nil unless
// WithDeleteIgnoreMissing(false) is given.
func (a *API) Delete(ctx context.Context, id string, options ...DeleteOption) error {
	return sfcrecord.Delete(ctx, a.target(), id, options)
}

// List implements sfc_port_chains(**query) as a lazy iterator that follows
// the port_chains_links next link and limit-driven marker pages.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*PortChain, error] {
	source := sfcrecord.List(ctx, a.target(), options)
	return func(yield func(*PortChain, error) bool) {
		for record, err := range source {
			if !yield(fromRecord(record), err) || err != nil {
				return
			}
		}
	}
}

// All collects List.
func (a *API) All(ctx context.Context, options ...ListOption) ([]*PortChain, error) {
	var values []*PortChain
	for value, err := range a.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// Find implements find_sfc_port_chain: GET by ID first and, after a clean 400,
// 403 or 404, the complete list queried by name=name_or_id and matched by
// exact ID or name. A missing port chain returns nil, nil unless
// WithFindIgnoreMissing(false) is given.
func (a *API) Find(ctx context.Context, nameOrID string, options ...FindOption) (*PortChain, error) {
	return convert(sfcrecord.Find(ctx, a.target(), nameOrID, options))
}

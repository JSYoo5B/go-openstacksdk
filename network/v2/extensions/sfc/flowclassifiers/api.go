// Package flowclassifiers implements the pinned openstacksdk network v2
// sfc_flow_classifier proxy calls over the Neutron networking-sfc
// /v2.0/sfc/flow_classifiers route, which Gophercloud v2.15.0 does not bind.
package flowclassifiers

import (
	"context"
	"iter"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/internal/sfcrecord"
	"github.com/gophercloud/gophercloud/v2"
)

// spec is the Python SfcFlowClassifier Resource class: Body attributes, descriptor
// types and QueryParameters names. Body attributes outside the query
// mapping (destination_ip_prefix, id, l7_parameters, source_ip_prefix) filter locally.
var spec = sfcrecord.NewSpec("flow_classifiers", "sfc/flow_classifiers", "flow_classifier", "flow_classifiers",
	map[string]string{"description": "description", "name": "name", "project_id": "project_id", "tenant_id": "tenant_id", "ethertype": "ethertype", "protocol": "protocol", "source_port_range_min": "source_port_range_min", "source_port_range_max": "source_port_range_max", "destination_port_range_min": "destination_port_range_min", "destination_port_range_max": "destination_port_range_max", "source_ip_prefix": "source_ip_prefix", "destination_ip_prefix": "destination_ip_prefix", "logical_source_port": "logical_source_port", "logical_destination_port": "logical_destination_port", "l7_parameters": "l7_parameters"},
	map[string]string{"source_port_range_min": sfcrecord.TypeInt, "source_port_range_max": sfcrecord.TypeInt, "destination_port_range_min": sfcrecord.TypeInt, "destination_port_range_max": sfcrecord.TypeInt, "l7_parameters": sfcrecord.TypeDict},
	[]string{"description", "name", "project_id", "tenant_id", "ethertype", "protocol", "source_port_range_min", "source_port_range_max", "destination_port_range_min", "destination_port_range_max", "logical_source_port", "logical_destination_port"})

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

// Create implements create_sfc_flow_classifier: POST sfc/flow_classifiers with
// {"flow_classifier": attrs} holding every given declared attribute.
func (a *API) Create(ctx context.Context, options ...CreateOption) (*FlowClassifier, error) {
	return convert(sfcrecord.Create(ctx, a.target(), options))
}

// Get implements get_sfc_flow_classifier: GET sfc/flow_classifiers/{id}.
func (a *API) Get(ctx context.Context, id string, options ...GetOption) (*FlowClassifier, error) {
	return convert(sfcrecord.Get(ctx, a.target(), id, options))
}

// Update implements update_sfc_flow_classifier: PUT sfc/flow_classifiers/{id} with only the
// given declared attributes. No declared attribute means no request.
func (a *API) Update(ctx context.Context, id string, options ...UpdateOption) (*FlowClassifier, error) {
	return convert(sfcrecord.Update(ctx, a.target(), id, options))
}

// Delete implements delete_sfc_flow_classifier. A clean 404 returns nil unless
// WithDeleteIgnoreMissing(false) is given.
func (a *API) Delete(ctx context.Context, id string, options ...DeleteOption) error {
	return sfcrecord.Delete(ctx, a.target(), id, options)
}

// List implements sfc_flow_classifiers(**query) as a lazy iterator that follows
// the flow_classifiers_links next link and limit-driven marker pages.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*FlowClassifier, error] {
	source := sfcrecord.List(ctx, a.target(), options)
	return func(yield func(*FlowClassifier, error) bool) {
		for record, err := range source {
			if !yield(fromRecord(record), err) || err != nil {
				return
			}
		}
	}
}

// All collects List.
func (a *API) All(ctx context.Context, options ...ListOption) ([]*FlowClassifier, error) {
	var values []*FlowClassifier
	for value, err := range a.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// Find implements find_sfc_flow_classifier: GET by ID first and, after a clean 400,
// 403 or 404, the complete list queried by name=name_or_id and matched by
// exact ID or name. A missing flow classifier returns nil, nil unless
// WithFindIgnoreMissing(false) is given.
func (a *API) Find(ctx context.Context, nameOrID string, options ...FindOption) (*FlowClassifier, error) {
	return convert(sfcrecord.Find(ctx, a.target(), nameOrID, options))
}

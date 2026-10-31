// Package inspectionrules manages bare metal (Ironic) inspection rules:
// GET/POST /inspection_rules and GET/PATCH/DELETE /inspection_rules/{id}.
//
// The pinned Gophercloud release has no inspection rule binding, so this
// package is SDK-owned. The client microversion is used as configured and is
// never raised by this package: inspection rules need 1.96. This is the Ironic
// service API, not the ironic-inspector /rules API.
package inspectionrules

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/ironicrest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const kind = "inspection_rules"

type API struct {
	client *gophercloud.ServiceClient
	rest   ironicrest.Resource[InspectionRule]
}

func New(client *gophercloud.ServiceClient) *API {
	return &API{client: client, rest: ironicrest.Resource[InspectionRule]{
		Client: client, Kind: "inspection rule", Collection: "inspection_rules", ListKey: "inspection_rules", Decode: decode,
		Identify: func(r *InspectionRule) (string, string) { return r.UUID, "" },
	}}
}

func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

// Create sends POST /inspection_rules. Only 201 succeeds; a failed or uncertain call is
// never replayed.
func (a *API) Create(ctx context.Context, opts CreateOpts, options ...CreateOption) (*InspectionRule, error) {
	config, err := calls.ApplyConfig(opts, createCalls, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, true, false, true)
	}
	var body json.RawMessage
	if err == nil {
		body, err = ironicrest.BuildBody(config.Options, config.Fields)
	}
	if err != nil {
		return nil, request.Wrap("Create", kind, err)
	}
	value, err := a.rest.Create(ctx, body, config.Headers)
	return value, request.Wrap("Create", kind, err)
}

// Get sends GET /inspection_rules/{id}, with an optional fields query. Only 200 succeeds.
func (a *API) Get(ctx context.Context, id string, options ...GetOption) (*InspectionRule, error) {
	config, err := calls.ApplyConfig(GetOpts{}, getCalls, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	if err != nil {
		return nil, request.Wrap("Get", kind, err)
	}
	value, err := a.rest.Get(ctx, id, config.Options.Fields, config.Headers)
	return value, request.Wrap("Get", kind, err)
}

// List walks GET /inspection_rules lazily; see ListOpts for paging.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*InspectionRule, error] {
	cfg, err := listConfig(options)
	if err != nil {
		return func(yield func(*InspectionRule, error) bool) { yield(nil, request.Wrap("List", kind, err)) }
	}
	return func(yield func(*InspectionRule, error) bool) {
		for value, err := range a.rest.List(ctx, cfg) {
			if !yield(value, request.Wrap("List", kind, err)) {
				return
			}
		}
	}
}

// Update sends PATCH /inspection_rules/{id} with one add operation per set member. An
// update that sets nothing is rejected before HTTP.
func (a *API) Update(ctx context.Context, id string, opts UpdateOpts, options ...UpdateOption) (*InspectionRule, error) {
	config, err := calls.ApplyConfig(opts, updateCalls, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, true, false, true)
	}
	var ops []PatchOperation
	if err == nil {
		var body json.RawMessage
		if body, err = ironicrest.BuildBody(config.Options, config.Fields); err == nil {
			var members map[string]json.RawMessage
			if err = json.Unmarshal(body, &members); err == nil {
				if len(members) == 0 {
					err = fmt.Errorf("%w: update sets no member", resource.ErrInvalidOption)
				}
				ops = ironicrest.AddOps(members)
			}
		}
	}
	if err != nil {
		return nil, request.Wrap("Update", kind, err)
	}
	value, err := a.rest.Patch(ctx, id, ops, config.Headers)
	return value, request.Wrap("Update", kind, err)
}

// PatchOption is one option of Patch: request.Header or request.Custom over
// *request.Config[struct{}]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type PatchOption = request.WriteCall

var patchCalls = calls.Spec[request.Config[struct{}]]{Header: calls.ConfigHeader[struct{}]}

// Patch sends PATCH /inspection_rules/{id} with the operations exactly as given. Only
// 200 succeeds.
func (a *API) Patch(ctx context.Context, id string, ops []PatchOperation, options ...PatchOption) (*InspectionRule, error) {
	config, err := calls.ApplyConfig(struct{}{}, patchCalls, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	if err != nil {
		return nil, request.Wrap("Patch", kind, err)
	}
	value, err := a.rest.Patch(ctx, id, ops, config.Headers)
	return value, request.Wrap("Patch", kind, err)
}

// Delete sends DELETE /inspection_rules/{id}. Only 204 succeeds. By default a missing
// runbook is not an error and deleted is false; request.IgnoreMissing(false)
// returns a resource.NotFoundError instead.
func (a *API) Delete(ctx context.Context, id string, options ...DeleteOption) (deleted bool, err error) {
	config, err := calls.ApplyConfig(DeleteOpts{IgnoreMissing: true}, deleteCalls, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	if err != nil {
		return false, request.Wrap("Delete", kind, err)
	}
	deleted, err = a.rest.Delete(ctx, id, config.Options.IgnoreMissing, config.Headers)
	return deleted, request.Wrap("Delete", kind, err)
}

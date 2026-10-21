// Package runbooks manages bare metal (Ironic) runbooks: GET/POST /runbooks,
// GET/PATCH/DELETE /runbooks/{id} and the trait routes below a runbook.
//
// The pinned Gophercloud release has no runbook binding, so this package is
// SDK-owned. The client microversion is used as configured and is never raised
// by this package: runbooks need 1.92 and descriptions and traits 1.112.
package runbooks

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/ironicrest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const kind = "runbooks"

type API struct {
	client *gophercloud.ServiceClient
	rest   ironicrest.Resource[Runbook]
}

func New(client *gophercloud.ServiceClient) *API {
	return &API{client: client, rest: ironicrest.Resource[Runbook]{
		Client: client, Kind: "runbook", Collection: "runbooks", ListKey: "runbooks", Decode: decode,
		Identify: func(r *Runbook) (string, string) { return r.UUID, r.Name },
	}}
}

func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

// Create sends POST /runbooks. Only 201 succeeds; a failed or uncertain call is
// never replayed.
func (a *API) Create(ctx context.Context, opts CreateOpts, options ...CreateOption) (*Runbook, error) {
	config, err := request.Apply(opts, options...)
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

// Get sends GET /runbooks/{id}, with an optional fields query. Only 200 succeeds.
func (a *API) Get(ctx context.Context, id string, options ...GetOption) (*Runbook, error) {
	config, err := request.Apply(GetOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	if err != nil {
		return nil, request.Wrap("Get", kind, err)
	}
	value, err := a.rest.Get(ctx, id, config.Options.Fields, config.Headers)
	return value, request.Wrap("Get", kind, err)
}

// List walks GET /runbooks lazily; see ListOpts for paging.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*Runbook, error] {
	cfg, err := listConfig(options)
	if err != nil {
		return func(yield func(*Runbook, error) bool) { yield(nil, request.Wrap("List", kind, err)) }
	}
	return func(yield func(*Runbook, error) bool) {
		for value, err := range a.rest.List(ctx, cfg) {
			if !yield(value, request.Wrap("List", kind, err)) {
				return
			}
		}
	}
}

// Update sends PATCH /runbooks/{id} with one add operation per set member. An
// update that sets nothing is rejected before HTTP.
func (a *API) Update(ctx context.Context, id string, opts UpdateOpts, options ...UpdateOption) (*Runbook, error) {
	config, err := request.Apply(opts, options...)
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

// PatchOption configures Patch.
type PatchOption = request.Option[struct{}]

func WithPatchHeader(key, value string) PatchOption { return request.WithHeader[struct{}](key, value) }

// Patch sends PATCH /runbooks/{id} with the operations exactly as given. Only
// 200 succeeds.
func (a *API) Patch(ctx context.Context, id string, ops []PatchOperation, options ...PatchOption) (*Runbook, error) {
	config, err := request.Apply(struct{}{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	if err != nil {
		return nil, request.Wrap("Patch", kind, err)
	}
	value, err := a.rest.Patch(ctx, id, ops, config.Headers)
	return value, request.Wrap("Patch", kind, err)
}

// Delete sends DELETE /runbooks/{id}. Only 204 succeeds. By default a missing
// runbook is not an error and deleted is false; WithDeleteIgnoreMissing(false)
// returns a resource.NotFoundError instead.
func (a *API) Delete(ctx context.Context, id string, options ...DeleteOption) (deleted bool, err error) {
	config, err := request.Apply(DeleteOpts{IgnoreMissing: true}, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	if err != nil {
		return false, request.Wrap("Delete", kind, err)
	}
	deleted, err = a.rest.Delete(ctx, id, config.Options.IgnoreMissing, config.Headers)
	return deleted, request.Wrap("Delete", kind, err)
}

func (a *API) traitPrecheck(ctx context.Context, id string) error {
	if err := ironicrest.Preflight(ctx, a.client); err != nil {
		return err
	}
	return ironicrest.ValidateID("runbook", id)
}

// AddTrait sends a bodyless PUT /runbooks/{id}/traits/{trait}. Only 204
// succeeds. The trait is path-escaped.
func (a *API) AddTrait(ctx context.Context, id, trait string) error {
	err := a.traitPrecheck(ctx, id)
	if err == nil && trait == "" {
		err = fmt.Errorf("%w: trait must not be empty", resource.ErrInvalidOption)
	}
	if err != nil {
		return request.Wrap("AddTrait", kind, err)
	}
	_, err = ironicrest.Send(ctx, a.client, http.MethodPut, a.rest.SubURL(id, "traits", trait), nil, nil, http.StatusNoContent)
	return request.Wrap("AddTrait", kind, err)
}

// RemoveTrait sends DELETE /runbooks/{id}/traits/{trait}. Only 204 succeeds;
// the service treats removing an absent trait as success.
func (a *API) RemoveTrait(ctx context.Context, id, trait string) error {
	err := a.traitPrecheck(ctx, id)
	if err == nil && trait == "" {
		err = fmt.Errorf("%w: trait must not be empty", resource.ErrInvalidOption)
	}
	if err != nil {
		return request.Wrap("RemoveTrait", kind, err)
	}
	_, err = ironicrest.Send(ctx, a.client, http.MethodDelete, a.rest.SubURL(id, "traits", trait), nil, nil, http.StatusNoContent)
	return request.Wrap("RemoveTrait", kind, err)
}

// SetTraits sends PUT /runbooks/{id}/traits with {"traits": [...]}. A nil or
// empty slice sends [] and so removes every trait. Only 204 succeeds.
func (a *API) SetTraits(ctx context.Context, id string, traits []string) error {
	err := a.traitPrecheck(ctx, id)
	for _, t := range traits {
		if t == "" && err == nil {
			err = fmt.Errorf("%w: trait must not be empty", resource.ErrInvalidOption)
		}
	}
	if err != nil {
		return request.Wrap("SetTraits", kind, err)
	}
	if traits == nil {
		traits = []string{}
	}
	_, err = ironicrest.Send(ctx, a.client, http.MethodPut, a.rest.SubURL(id, "traits"), map[string][]string{"traits": traits}, nil, http.StatusNoContent)
	return request.Wrap("SetTraits", kind, err)
}

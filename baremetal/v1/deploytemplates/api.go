// Package deploytemplates manages bare metal (Ironic) deploy templates:
// GET/POST /deploy_templates and GET/PATCH/DELETE /deploy_templates/{id}.
//
// The pinned Gophercloud release has no deploy template binding, so this
// package is SDK-owned. The client microversion is used as configured and is
// never raised by this package: deploy templates need 1.55.
package deploytemplates

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"

	"github.com/JSYoo5B/go-openstacksdk/internal/ironicrest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const kind = "deploy_templates"

type API struct {
	client *gophercloud.ServiceClient
	rest   ironicrest.Resource[DeployTemplate]
}

func New(client *gophercloud.ServiceClient) *API {
	return &API{client: client, rest: ironicrest.Resource[DeployTemplate]{
		Client: client, Kind: "deploy template", Collection: "deploy_templates", ListKey: "deploy_templates", Decode: decode,
		Identify: func(r *DeployTemplate) (string, string) { return r.UUID, r.Name },
	}}
}

func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

// Create sends POST /deploy_templates. Only 201 succeeds; a failed or uncertain call is
// never replayed.
func (a *API) Create(ctx context.Context, opts CreateOpts, options ...CreateOption) (*DeployTemplate, error) {
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

// Get sends GET /deploy_templates/{id}, with an optional fields query. Only 200 succeeds.
func (a *API) Get(ctx context.Context, id string, options ...GetOption) (*DeployTemplate, error) {
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

// List walks GET /deploy_templates lazily; see ListOpts for paging.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*DeployTemplate, error] {
	cfg, err := listConfig(options)
	if err != nil {
		return func(yield func(*DeployTemplate, error) bool) { yield(nil, request.Wrap("List", kind, err)) }
	}
	return func(yield func(*DeployTemplate, error) bool) {
		for value, err := range a.rest.List(ctx, cfg) {
			if !yield(value, request.Wrap("List", kind, err)) {
				return
			}
		}
	}
}

// Update sends PATCH /deploy_templates/{id} with one add operation per set member. An
// update that sets nothing is rejected before HTTP.
func (a *API) Update(ctx context.Context, id string, opts UpdateOpts, options ...UpdateOption) (*DeployTemplate, error) {
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

// Patch sends PATCH /deploy_templates/{id} with the operations exactly as given. Only
// 200 succeeds.
func (a *API) Patch(ctx context.Context, id string, ops []PatchOperation, options ...PatchOption) (*DeployTemplate, error) {
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

// Delete sends DELETE /deploy_templates/{id}. Only 204 succeeds. By default a missing
// deploy template is not an error and deleted is false; WithDeleteIgnoreMissing(false)
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

// FindOpts: IgnoreMissing defaults to true and Detail to true, as in Python.
type FindOpts struct{ IgnoreMissing, Detail bool }

type FindOption = request.Option[FindOpts]

func WithFindIgnoreMissing(ignore bool) FindOption {
	return func(c *request.Config[FindOpts]) error { c.Options.IgnoreMissing = ignore; return nil }
}

// WithFindDetail selects detail=true on the fallback list.
func WithFindDetail(detail bool) FindOption {
	return func(c *request.Config[FindOpts]) error { c.Options.Detail = detail; return nil }
}
func WithFindHeader(key, value string) FindOption { return request.WithHeader[FindOpts](key, value) }

// Find resolves a name or UUID. It first reads GET /deploy_templates/{ref}; a
// 400, 403 or 404 there falls back to one complete list matched by exact uuid or
// name. Two matches return *resource.AmbiguousError. A missing template returns
// nil, nil by default and *resource.NotFoundError with WithFindIgnoreMissing(false).
// Nothing is retried.
func (a *API) Find(ctx context.Context, ref string, options ...FindOption) (*DeployTemplate, error) {
	config, err := request.Apply(FindOpts{IgnoreMissing: true, Detail: true}, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	if err != nil {
		return nil, request.Wrap("Find", kind, err)
	}
	value, err := a.rest.Find(ctx, ref, config.Options.IgnoreMissing, config.Options.Detail, config.Headers)
	return value, request.Wrap("Find", kind, err)
}

package chassis

import (
	"context"
	"iter"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/ironic"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// API is the SDK-owned Ironic chassis binding. The client's Microversion is sent as
// X-OpenStack-Ironic-API-Version for a baremetal-typed client and is never raised automatically.
type API struct {
	client *gophercloud.ServiceClient
	core   *ironic.Core[Chassis]
	// Resources applies the shared lookup, missing-resource and listing policies.
	Resources *resource.Collection[Chassis]
}

// New performs no HTTP.
func New(client *gophercloud.ServiceClient) *API {
	core := ironic.New(ironic.Spec[Chassis]{Client: client, Kind: "chassis", Path: "chassis", PluralKey: "chassis", DetailRoute: true,
		UUID: func(v *Chassis) string { return v.UUID }, Name: func(*Chassis) string { return "" },
		Metadata:       func(v *Chassis) *resource.Metadata { return &v.Metadata },
		PatchProtected: []string{"uuid", "created_at", "updated_at", "links", "nodes"}})
	return &API{client: client, core: core, Resources: core.Resources}
}

func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

// Create sends POST /chassis and accepts only 201. Acknowledgement is not proof of any later state.
func (a *API) Create(ctx context.Context, opts CreateOpts, options ...CreateOption) (*Chassis, error) {
	body, err := prepareCreate(opts, options...)
	if err != nil {
		return nil, wrap("Create", err)
	}
	return a.core.Create(ctx, body, nil)
}

// Get sends GET /chassis/{id} and accepts only 200; a 404 stays an error.
func (a *API) Get(ctx context.Context, id string, options ...GetOption) (*Chassis, error) {
	fields, err := prepareGet(options...)
	if err != nil {
		return nil, wrap("Get", err)
	}
	return a.core.Get(ctx, id, fields)
}

// Update sends one JSON Patch to PATCH /chassis/{id} and accepts only 200.
// An empty patch fails before HTTP; the call is never replayed automatically.
func (a *API) Update(ctx context.Context, id string, opts UpdateOpts, options ...UpdateOption) (*Chassis, error) {
	operations, err := prepareUpdate(opts, options...)
	if err != nil {
		return nil, wrap("Update", err)
	}
	return a.core.Update(ctx, id, operations)
}

// Delete sends DELETE /chassis/{id} and accepts only 204; a 404 stays an error.
func (a *API) Delete(ctx context.Context, id string) error { return a.core.Delete(ctx, id) }

// List streams the summary collection, following advertised next links.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*Chassis, error] {
	return a.list(ctx, "List", false, options...)
}

// ListDetail streams the detailed collection (GET /chassis/detail).
func (a *API) ListDetail(ctx context.Context, options ...ListOption) iter.Seq2[*Chassis, error] {
	return a.list(ctx, "ListDetail", true, options...)
}

func (a *API) list(ctx context.Context, op string, detail bool, options ...ListOption) iter.Seq2[*Chassis, error] {
	query, err := prepareList(options...)
	if err != nil {
		return func(yield func(*Chassis, error) bool) { yield(nil, wrap(op, err)) }
	}
	return a.core.List(ctx, op, detail, url.Values(query))
}

// FindIdentity mirrors Python find_*: GET by identity first, then a listing scan on 400, 403 or 404
// (compatible fallback), exact uuid match, duplicates reported as ambiguous. A missing
// resource returns nil, nil unless resource.WithIdentityFindIgnoreMissing(false) is passed.
func (a *API) FindIdentity(ctx context.Context, identity string, options ...resource.IdentityFindOption) (*Chassis, error) {
	return a.Resources.FindIdentity(ctx, identity, options...)
}

// Find resolves an explicit ID or name reference through the shared collection policy.
func (a *API) Find(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) (*Chassis, error) {
	return a.Resources.Find(ctx, ref, options...)
}

// All lists detailed rows with the shared list options.
func (a *API) All(ctx context.Context, options ...resource.ListOption) ([]*Chassis, error) {
	return a.Resources.All(ctx, options...)
}

// Remove deletes by reference. An ID reference sends DELETE directly and ignores a 404 by default,
// like Python delete_*(ignore_missing=True); resource.WithMissingError() reports it.
func (a *API) Remove(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) error {
	return a.Resources.Delete(ctx, ref, options...)
}

func wrap(op string, err error) error { return request.Wrap(op, "chassis", err) }

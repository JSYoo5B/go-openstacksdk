// Package sharegroups owns Manila's share-group API, absent from Gophercloud
// v2.15.0. The routes are experimental before microversion 2.55 (they then need
// an experimental header); this package requires 2.55 or later and never sends
// that header.
package sharegroups

import (
	"context"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/manilaown"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// MinMicroversion is the first non-experimental microversion of the routes.
const MinMicroversion = 55

// Evidence is the actual wire body, headers and status of a response.
type Evidence = manilaown.Evidence

// ShareGroup is the typed view of a share group. The summary list route only
// carries ID, name and links; Evidence.Body tells which members were present.
type ShareGroup struct {
	ID                         string   `json:"id"`
	Name                       string   `json:"name"`
	Description                string   `json:"description"`
	Status                     string   `json:"status"`
	AvailabilityZone           string   `json:"availability_zone"`
	ConsistentSnapshotSupport  string   `json:"consistent_snapshot_support"`
	CreatedAt                  string   `json:"created_at"`
	ProjectID                  string   `json:"project_id"`
	SourceShareGroupSnapshotID string   `json:"source_share_group_snapshot_id"`
	ShareGroupTypeID           string   `json:"share_group_type_id"`
	ShareNetworkID             string   `json:"share_network_id"`
	ShareTypes                 []string `json:"share_types"`
	Evidence                   Evidence `json:"-"`
}

// API owns typed share-group operations and the shared Collection policies.
type API struct {
	client    *gophercloud.ServiceClient
	Resources *resource.Collection[ShareGroup]
}

func New(client *gophercloud.ServiceClient) *API {
	a := &API{client: client}
	a.Resources = resource.NewCollection(resource.Adapter[ShareGroup]{
		Kind:         "sharegroups",
		IdentityFind: true,
		Get:          a.Get,
		ID:           func(v *ShareGroup) string { return v.ID },
		Name:         func(v *ShareGroup) string { return v.Name },
		Status:       func(v *ShareGroup) string { return v.Status },
		Failed:       func(s string) bool { return strings.HasPrefix(strings.ToLower(s), "error") },
		Delete:       a.Delete,
		IterateControlled: func(ctx context.Context, q url.Values, control resource.ListControl) iter.Seq2[*ShareGroup, error] {
			return a.iterate(ctx, "detail", q, control)
		},
		IterateIdentity: func(ctx context.Context, q url.Values, details bool) iter.Seq2[*ShareGroup, error] {
			path := ""
			if details {
				path = "detail"
			}
			return a.iterate(ctx, path, q, resource.ListControl{})
		},
	})
	return a
}

func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

func (a *API) iterate(ctx context.Context, tail string, q url.Values, control resource.ListControl) iter.Seq2[*ShareGroup, error] {
	segments := []string{"share-groups"}
	if tail != "" {
		segments = append(segments, tail)
	}
	call := manilaown.Call{Client: a.client, Floor: MinMicroversion, Method: http.MethodGet, URL: manilaown.WithQuery(a.client.ServiceURL(segments...), q), OK: []int{http.StatusOK}}
	return manilaown.Iterate[ShareGroup](ctx, call, "share_groups", control)
}

func (a *API) check(ctx context.Context) error {
	if a == nil || a.client == nil {
		return fmt.Errorf("%w: share group API is required", resource.ErrInvalidOption)
	}
	return ctx.Err()
}

func wrap(op string, err error) error { return request.Wrap(op, "sharegroups", err) }

// ListOpts are the server filters of the list routes.
type ListOpts struct {
	Name                 string `q:"name"`
	NameLike             string `q:"name~"`
	Description          string `q:"description"`
	DescriptionLike      string `q:"description~"`
	Status               string `q:"status"`
	ProjectID            string `q:"project_id"`
	ShareServerID        string `q:"share_server_id"`
	SnapshotID           string `q:"snapshot_id"`
	Host                 string `q:"host"`
	ShareNetworkID       string `q:"share_network_id"`
	ShareGroupTypeID     string `q:"share_group_type_id"`
	ShareGroupSnapshotID string `q:"share_group_snapshot_id"`
	Limit                *int   `q:"limit"`
	Offset               *int   `q:"offset"`
	SortKey              string `q:"sort_key"`
	SortDir              string `q:"sort_dir"`
}

type ListOption = request.Option[ListOpts]

func WithListOptions(value ListOpts) ListOption {
	value.Limit, value.Offset = clonePtr(value.Limit), clonePtr(value.Offset)
	return request.WithOptions(value)
}

func clonePtr[T any](p *T) *T {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func WithListQuery(key, value string) ListOption { return request.WithQuery[ListOpts](key, value) }

func WithListMaxItems(value int) ListOption {
	return manilaown.WithListControl[ListOpts](func(c *resource.ListControl) { c.MaxItems = value })
}

// List yields the summary rows (id, name, links) of GET /share-groups, as the
// Python proxy does. Use ListDetail for full rows.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*ShareGroup, error] {
	return a.list(ctx, "", options)
}

// ListDetail yields full rows of GET /share-groups/detail (a Go addition).
func (a *API) ListDetail(ctx context.Context, options ...ListOption) iter.Seq2[*ShareGroup, error] {
	return a.list(ctx, "detail", options)
}

func (a *API) list(ctx context.Context, tail string, options []ListOption) iter.Seq2[*ShareGroup, error] {
	if err := a.check(ctx); err != nil {
		return func(yield func(*ShareGroup, error) bool) { yield(nil, wrap("List", err)) }
	}
	route := func(request.Config[ListOpts]) string {
		if tail == "" {
			return a.client.ServiceURL("share-groups")
		}
		return a.client.ServiceURL("share-groups", tail)
	}
	return manilaown.ListFor[ShareGroup](ctx, a.client, MinMicroversion, "share_groups", route, resource.ListControl{}, options)
}

// Get fetches one share group; only HTTP 200 succeeds.
func (a *API) Get(ctx context.Context, id string) (*ShareGroup, error) {
	if err := a.check(ctx); err != nil {
		return nil, wrap("Get", err)
	}
	if err := resource.ID(id).Validate(); err != nil {
		return nil, wrap("Get", err)
	}
	resp, err := manilaown.Call{Client: a.client, Floor: MinMicroversion, Method: http.MethodGet, URL: a.client.ServiceURL("share-groups", id), OK: []int{http.StatusOK}}.Do(ctx)
	if err != nil {
		return nil, wrap("Get", err)
	}
	value, err := manilaown.One[ShareGroup](resp, "share_group")
	return value, wrap("Get", err)
}

// CreateOpts is the body of POST /share-groups. ShareTypes and
// SourceShareGroupSnapshotID are mutually exclusive on the service, which
// validates it. A non-nil empty ShareTypes slice is sent as [].
type CreateOpts struct {
	Name                       request.Optional[string] `json:"name,omitzero"`
	Description                request.Optional[string] `json:"description,omitzero"`
	ShareTypes                 []string                 `json:"share_types,omitzero"`
	ShareGroupTypeID           request.Optional[string] `json:"share_group_type_id,omitzero"`
	SourceShareGroupSnapshotID request.Optional[string] `json:"source_share_group_snapshot_id,omitzero"`
	ShareNetworkID             request.Optional[string] `json:"share_network_id,omitzero"`
	AvailabilityZone           request.Optional[string] `json:"availability_zone,omitzero"`
}

type CreateOption = request.Option[CreateOpts]

func WithCreateOptions(value CreateOpts) CreateOption {
	if value.ShareTypes != nil {
		value.ShareTypes = append([]string{}, value.ShareTypes...)
	}
	return request.WithOptions(value)
}

func WithCreateField(key string, value any) CreateOption {
	return request.WithField[CreateOpts](key, value)
}

type protectedCreate struct {
	CreateOpts
	ID                        string `json:"id"`
	Status                    string `json:"status"`
	ProjectID                 string `json:"project_id"`
	CreatedAt                 string `json:"created_at"`
	ConsistentSnapshotSupport string `json:"consistent_snapshot_support"`
	Envelope                  any    `json:"share_group"`
}

// Create sends one POST and accepts 202 (or 200). Acknowledgement does not
// mean the group is available: use WaitFor. Failures are never replayed.
func (a *API) Create(ctx context.Context, opts CreateOpts, options ...CreateOption) (*ShareGroup, error) {
	if err := a.check(ctx); err != nil {
		return nil, wrap("Create", err)
	}
	cfg, err := request.Apply(opts, options...)
	if err == nil {
		err = request.ValidateCapabilities(cfg, true, false, false)
	}
	var body map[string]any
	if err == nil {
		body, err = manilaown.Envelope("share_group", cfg.Options, cfg.Fields, protectedCreate{})
	}
	if err != nil {
		return nil, wrap("Create", err)
	}
	resp, err := manilaown.Call{Client: a.client, Floor: MinMicroversion, Method: http.MethodPost, URL: a.client.ServiceURL("share-groups"), Body: body, OK: []int{http.StatusOK, http.StatusAccepted}}.Do(ctx)
	if err != nil {
		return nil, wrap("Create", err)
	}
	value, err := manilaown.One[ShareGroup](resp, "share_group")
	return value, wrap("Create", err)
}

// UpdateOpts changes name and description; null clears a member.
type UpdateOpts struct {
	Name        request.Optional[string] `json:"name,omitzero"`
	Description request.Optional[string] `json:"description,omitzero"`
}

type UpdateOption = request.Option[UpdateOpts]

func WithUpdateOptions(value UpdateOpts) UpdateOption { return request.WithOptions(value) }

func WithUpdateField(key string, value any) UpdateOption {
	return request.WithField[UpdateOpts](key, value)
}

type protectedUpdate struct {
	UpdateOpts
	ID       string `json:"id"`
	Status   string `json:"status"`
	Envelope any    `json:"share_group"`
}

// Update sends PUT with only the supplied members; an empty update is rejected
// before HTTP (Python sends nothing when no attribute changed).
func (a *API) Update(ctx context.Context, id string, opts UpdateOpts, options ...UpdateOption) (*ShareGroup, error) {
	if err := a.check(ctx); err != nil {
		return nil, wrap("Update", err)
	}
	if err := resource.ID(id).Validate(); err != nil {
		return nil, wrap("Update", err)
	}
	cfg, err := request.Apply(opts, options...)
	if err == nil {
		err = request.ValidateCapabilities(cfg, true, false, false)
	}
	if err == nil && cfg.Options.Name.IsZero() && cfg.Options.Description.IsZero() && len(cfg.Fields) == 0 {
		err = fmt.Errorf("%w: update supplies no field", resource.ErrInvalidOption)
	}
	var body map[string]any
	if err == nil {
		body, err = manilaown.Envelope("share_group", cfg.Options, cfg.Fields, protectedUpdate{})
	}
	if err != nil {
		return nil, wrap("Update", err)
	}
	resp, err := manilaown.Call{Client: a.client, Floor: MinMicroversion, Method: http.MethodPut, URL: a.client.ServiceURL("share-groups", id), Body: body, OK: []int{http.StatusOK}}.Do(ctx)
	if err != nil {
		return nil, wrap("Update", err)
	}
	value, err := manilaown.One[ShareGroup](resp, "share_group")
	return value, wrap("Update", err)
}

// Delete sends one DELETE and accepts 200, 202 or 204; a 404 stays native.
// Acknowledgement is not completion: use WaitForDeletion.
func (a *API) Delete(ctx context.Context, id string) error {
	if err := a.check(ctx); err != nil {
		return wrap("Delete", err)
	}
	if err := resource.ID(id).Validate(); err != nil {
		return wrap("Delete", err)
	}
	_, err := manilaown.Call{Client: a.client, Floor: MinMicroversion, Method: http.MethodDelete, URL: a.client.ServiceURL("share-groups", id), OK: []int{200, 202, 204}}.Do(ctx)
	return wrap("Delete", err)
}

func (a *API) Find(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) (*ShareGroup, error) {
	return a.Resources.Find(ctx, ref, options...)
}

// FindIdentity mirrors Resource.find: GET by ID, falling back to a list search
// for an exact ID or name after 400, 403 or 404. Use
// resource.WithIdentityFindDetails(false) to search /share-groups instead of
// /share-groups/detail.
func (a *API) FindIdentity(ctx context.Context, identity string, options ...resource.IdentityFindOption) (*ShareGroup, error) {
	return a.Resources.FindIdentity(ctx, identity, options...)
}

func (a *API) All(ctx context.Context, options ...resource.ListOption) ([]*ShareGroup, error) {
	return a.Resources.All(ctx, options...)
}

// Remove deletes by reference and ignores a missing group by default.
func (a *API) Remove(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) error {
	return a.Resources.Delete(ctx, ref, options...)
}

func (a *API) WaitFor(ctx context.Context, ref resource.Ref, status string, options ...resource.WaitOption) (*ShareGroup, error) {
	return a.Resources.Wait(ctx, ref, status, options...)
}

func (a *API) WaitForDeletion(ctx context.Context, ref resource.Ref, options ...resource.WaitOption) error {
	return a.Resources.WaitDeleted(ctx, ref, options...)
}

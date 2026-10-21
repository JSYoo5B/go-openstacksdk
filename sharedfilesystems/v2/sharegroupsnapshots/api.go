// Package sharegroupsnapshots owns Manila's share-group-snapshot API, absent
// from Gophercloud v2.15.0. Like share groups it requires microversion 2.55 or
// later (earlier versions are experimental and need a header this package never
// sends).
package sharegroupsnapshots

import (
	"context"
	"encoding/json"
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

const MinMicroversion = 55

type Evidence = manilaown.Evidence

// ShareGroupSnapshot is the typed view of one snapshot. Members stays raw JSON
// because the detail rows carry service-defined member objects.
type ShareGroupSnapshot struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Status        string          `json:"status"`
	ShareGroupID  string          `json:"share_group_id"`
	ProjectID     string          `json:"project_id"`
	CreatedAt     string          `json:"created_at"`
	Size          *int64          `json:"size"`
	ShareProtocol string          `json:"share_proto"`
	Members       json.RawMessage `json:"members"`
	Evidence      Evidence        `json:"-"`
}

// Member is one share snapshot of a group snapshot.
type Member struct {
	ID            string `json:"id"`
	ShareID       string `json:"share_id"`
	Size          *int64 `json:"size"`
	CreatedAt     string `json:"created_at"`
	Status        string `json:"status"`
	ShareProtocol string `json:"share_proto"`
}

// Members is GET .../members: Items from share_group_snapshot_members and the
// whole response in Evidence.Body.
type Members struct {
	Items    []Member `json:"share_group_snapshot_members"`
	Evidence Evidence `json:"-"`
}

type API struct {
	client    *gophercloud.ServiceClient
	Resources *resource.Collection[ShareGroupSnapshot]
}

func New(client *gophercloud.ServiceClient) *API {
	a := &API{client: client}
	a.Resources = resource.NewCollection(resource.Adapter[ShareGroupSnapshot]{
		Kind:         "sharegroupsnapshots",
		IdentityFind: true,
		Get:          a.Get,
		ID:           func(v *ShareGroupSnapshot) string { return v.ID },
		Name:         func(v *ShareGroupSnapshot) string { return v.Name },
		Status:       func(v *ShareGroupSnapshot) string { return v.Status },
		Failed:       func(s string) bool { return strings.HasPrefix(strings.ToLower(s), "error") },
		Delete:       a.Delete,
		IterateControlled: func(ctx context.Context, q url.Values, control resource.ListControl) iter.Seq2[*ShareGroupSnapshot, error] {
			return a.iterate(ctx, true, q, control)
		},
		IterateIdentity: func(ctx context.Context, q url.Values, details bool) iter.Seq2[*ShareGroupSnapshot, error] {
			return a.iterate(ctx, details, q, resource.ListControl{})
		},
	})
	return a
}

func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

func (a *API) iterate(ctx context.Context, details bool, q url.Values, control resource.ListControl) iter.Seq2[*ShareGroupSnapshot, error] {
	call := manilaown.Call{Client: a.client, Floor: MinMicroversion, Method: http.MethodGet, URL: manilaown.WithQuery(a.route(details), q), OK: []int{http.StatusOK}}
	return manilaown.Iterate[ShareGroupSnapshot](ctx, call, "share_group_snapshots", control)
}

func (a *API) route(details bool) string {
	if details {
		return a.client.ServiceURL("share-group-snapshots", "detail")
	}
	return a.client.ServiceURL("share-group-snapshots")
}

func (a *API) check(ctx context.Context) error {
	if a == nil || a.client == nil {
		return fmt.Errorf("%w: share group snapshot API is required", resource.ErrInvalidOption)
	}
	return ctx.Err()
}

func wrap(op string, err error) error { return request.Wrap(op, "sharegroupsnapshots", err) }

// ListOpts are the server filters of the list routes.
type ListOpts struct {
	ProjectID    string `q:"project_id"`
	AllTenants   *bool  `q:"all_tenants"`
	Name         string `q:"name"`
	Description  string `q:"description"`
	Status       string `q:"status"`
	ShareGroupID string `q:"share_group_id"`
	Limit        *int   `q:"limit"`
	Offset       *int   `q:"offset"`
	SortKey      string `q:"sort_key"`
	SortDir      string `q:"sort_dir"`
}

type ListOption = request.Option[ListOpts]

func WithListOptions(value ListOpts) ListOption {
	if value.AllTenants != nil {
		v := *value.AllTenants
		value.AllTenants = &v
	}
	if value.Limit != nil {
		v := *value.Limit
		value.Limit = &v
	}
	if value.Offset != nil {
		v := *value.Offset
		value.Offset = &v
	}
	return request.WithOptions(value)
}

func WithListQuery(key, value string) ListOption { return request.WithQuery[ListOpts](key, value) }

func WithListMaxItems(value int) ListOption {
	return manilaown.WithListControl[ListOpts](func(c *resource.ListControl) { c.MaxItems = value })
}

const detailsArgument = "sharegroupsnapshots.details"

// WithListDetails selects /share-group-snapshots/detail (true, the default,
// like Python's details=True) or the summary route /share-group-snapshots.
func WithListDetails(value bool) ListOption {
	return request.WithArgument[ListOpts](detailsArgument, value)
}

func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*ShareGroupSnapshot, error] {
	if err := a.check(ctx); err != nil {
		return func(yield func(*ShareGroupSnapshot, error) bool) { yield(nil, wrap("List", err)) }
	}
	route := func(cfg request.Config[ListOpts]) string {
		details, ok, err := request.Argument[bool](cfg, detailsArgument)
		if err != nil || !ok {
			details = true
		}
		return a.route(details)
	}
	return manilaown.ListFor[ShareGroupSnapshot](ctx, a.client, MinMicroversion, "share_group_snapshots", route, resource.ListControl{}, options, detailsArgument)
}

func (a *API) Get(ctx context.Context, id string) (*ShareGroupSnapshot, error) {
	if err := a.check(ctx); err != nil {
		return nil, wrap("Get", err)
	}
	if err := resource.ID(id).Validate(); err != nil {
		return nil, wrap("Get", err)
	}
	resp, err := manilaown.Call{Client: a.client, Floor: MinMicroversion, Method: http.MethodGet, URL: a.client.ServiceURL("share-group-snapshots", id), OK: []int{http.StatusOK}}.Do(ctx)
	if err != nil {
		return nil, wrap("Get", err)
	}
	value, err := manilaown.One[ShareGroupSnapshot](resp, "share_group_snapshot")
	return value, wrap("Get", err)
}

// CreateOpts is the body of POST /share-group-snapshots. ShareGroupID is the
// group to snapshot and is required.
type CreateOpts struct {
	ShareGroupID string                   `json:"share_group_id"`
	Name         request.Optional[string] `json:"name,omitzero"`
	Description  request.Optional[string] `json:"description,omitzero"`
}

type CreateOption = request.Option[CreateOpts]

func WithCreateOptions(value CreateOpts) CreateOption { return request.WithOptions(value) }

func WithCreateField(key string, value any) CreateOption {
	return request.WithField[CreateOpts](key, value)
}

type protectedCreate struct {
	CreateOpts
	ID        string `json:"id"`
	Status    string `json:"status"`
	ProjectID string `json:"project_id"`
	CreatedAt string `json:"created_at"`
	Members   any    `json:"members"`
	Envelope  any    `json:"share_group_snapshot"`
}

// Create sends one POST accepting 202 (or 200). Acknowledgement is not
// availability: use WaitFor. A failure is never replayed.
func (a *API) Create(ctx context.Context, opts CreateOpts, options ...CreateOption) (*ShareGroupSnapshot, error) {
	if err := a.check(ctx); err != nil {
		return nil, wrap("Create", err)
	}
	cfg, err := request.Apply(opts, options...)
	if err == nil {
		err = request.ValidateCapabilities(cfg, true, false, false)
	}
	if err == nil {
		err = resource.ID(cfg.Options.ShareGroupID).Validate()
	}
	var body map[string]any
	if err == nil {
		body, err = manilaown.Envelope("share_group_snapshot", cfg.Options, cfg.Fields, protectedCreate{})
	}
	if err != nil {
		return nil, wrap("Create", err)
	}
	resp, err := manilaown.Call{Client: a.client, Floor: MinMicroversion, Method: http.MethodPost, URL: a.client.ServiceURL("share-group-snapshots"), Body: body, OK: []int{http.StatusOK, http.StatusAccepted}}.Do(ctx)
	if err != nil {
		return nil, wrap("Create", err)
	}
	value, err := manilaown.One[ShareGroupSnapshot](resp, "share_group_snapshot")
	return value, wrap("Create", err)
}

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
	ID           string `json:"id"`
	Status       string `json:"status"`
	ShareGroupID string `json:"share_group_id"`
	Envelope     any    `json:"share_group_snapshot"`
}

func (a *API) Update(ctx context.Context, id string, opts UpdateOpts, options ...UpdateOption) (*ShareGroupSnapshot, error) {
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
		body, err = manilaown.Envelope("share_group_snapshot", cfg.Options, cfg.Fields, protectedUpdate{})
	}
	if err != nil {
		return nil, wrap("Update", err)
	}
	resp, err := manilaown.Call{Client: a.client, Floor: MinMicroversion, Method: http.MethodPut, URL: a.client.ServiceURL("share-group-snapshots", id), Body: body, OK: []int{http.StatusOK}}.Do(ctx)
	if err != nil {
		return nil, wrap("Update", err)
	}
	value, err := manilaown.One[ShareGroupSnapshot](resp, "share_group_snapshot")
	return value, wrap("Update", err)
}

func (a *API) Delete(ctx context.Context, id string) error {
	if err := a.check(ctx); err != nil {
		return wrap("Delete", err)
	}
	if err := resource.ID(id).Validate(); err != nil {
		return wrap("Delete", err)
	}
	_, err := manilaown.Call{Client: a.client, Floor: MinMicroversion, Method: http.MethodDelete, URL: a.client.ServiceURL("share-group-snapshots", id), OK: []int{200, 202, 204}}.Do(ctx)
	return wrap("Delete", err)
}

// ResetStatus posts the reset_status action for one snapshot (administrator
// action). It performs no preceding GET, requires a non-empty status and
// reports every non-2xx response; success is 200, 202 or 204.
func (a *API) ResetStatus(ctx context.Context, id, status string) error {
	if err := a.check(ctx); err != nil {
		return wrap("ResetStatus", err)
	}
	if err := resource.ID(id).Validate(); err != nil {
		return wrap("ResetStatus", err)
	}
	if strings.TrimSpace(status) == "" {
		return wrap("ResetStatus", fmt.Errorf("%w: status is required", resource.ErrInvalidOption))
	}
	body := map[string]any{"reset_status": map[string]any{"status": status}}
	_, err := manilaown.Call{Client: a.client, Floor: MinMicroversion, Method: http.MethodPost, URL: a.client.ServiceURL("share-group-snapshots", id, "action"), Body: body, OK: []int{200, 202, 204}}.Do(ctx)
	return wrap("ResetStatus", err)
}

// ListMembers reads GET .../members. Only 200 succeeds; the whole response is
// retained in Evidence.Body.
func (a *API) ListMembers(ctx context.Context, id string) (*Members, error) {
	if err := a.check(ctx); err != nil {
		return nil, wrap("ListMembers", err)
	}
	if err := resource.ID(id).Validate(); err != nil {
		return nil, wrap("ListMembers", err)
	}
	resp, err := manilaown.Call{Client: a.client, Floor: MinMicroversion, Method: http.MethodGet, URL: a.client.ServiceURL("share-group-snapshots", id, "members"), OK: []int{http.StatusOK}}.Do(ctx)
	if err != nil {
		return nil, wrap("ListMembers", err)
	}
	value, err := manilaown.Whole[Members](resp)
	return value, wrap("ListMembers", err)
}

func (a *API) Find(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) (*ShareGroupSnapshot, error) {
	return a.Resources.Find(ctx, ref, options...)
}

func (a *API) FindIdentity(ctx context.Context, identity string, options ...resource.IdentityFindOption) (*ShareGroupSnapshot, error) {
	return a.Resources.FindIdentity(ctx, identity, options...)
}

func (a *API) All(ctx context.Context, options ...resource.ListOption) ([]*ShareGroupSnapshot, error) {
	return a.Resources.All(ctx, options...)
}

func (a *API) Remove(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) error {
	return a.Resources.Delete(ctx, ref, options...)
}

func (a *API) WaitFor(ctx context.Context, ref resource.Ref, status string, options ...resource.WaitOption) (*ShareGroupSnapshot, error) {
	return a.Resources.Wait(ctx, ref, status, options...)
}

func (a *API) WaitForDeletion(ctx context.Context, ref resource.Ref, options ...resource.WaitOption) error {
	return a.Resources.WaitDeleted(ctx, ref, options...)
}

// Package shareinstances owns Manila's administrator share-instance API
// (microversion 2.3 and later), absent from Gophercloud v2.15.0: list, get,
// reset status and force delete.
package shareinstances

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

const MinMicroversion = 3

type Evidence = manilaown.Evidence

// ShareInstance is the typed view of one instance of a share.
type ShareInstance struct {
	ID                  string   `json:"id"`
	AccessRulesStatus   string   `json:"access_rules_status"`
	AvailabilityZone    string   `json:"availability_zone"`
	CastRulesToReadOnly *bool    `json:"cast_rules_to_readonly"`
	CreatedAt           string   `json:"created_at"`
	Host                string   `json:"host"`
	Progress            string   `json:"progress"`
	ReplicaState        string   `json:"replica_state"`
	ShareID             string   `json:"share_id"`
	ShareNetworkID      string   `json:"share_network_id"`
	ShareServerID       string   `json:"share_server_id"`
	Status              string   `json:"status"`
	Evidence            Evidence `json:"-"`
}

type API struct {
	client    *gophercloud.ServiceClient
	Resources *resource.Collection[ShareInstance]
}

func New(client *gophercloud.ServiceClient) *API {
	a := &API{client: client}
	a.Resources = resource.NewCollection(resource.Adapter[ShareInstance]{
		Kind:         "shareinstances",
		IdentityFind: true,
		Get:          a.Get,
		ID:           func(v *ShareInstance) string { return v.ID },
		Name:         func(*ShareInstance) string { return "" },
		Status:       func(v *ShareInstance) string { return v.Status },
		Failed:       func(s string) bool { return strings.HasPrefix(strings.ToLower(s), "error") },
		Delete:       a.ForceDelete,
		IterateControlled: func(ctx context.Context, q url.Values, control resource.ListControl) iter.Seq2[*ShareInstance, error] {
			call := manilaown.Call{Client: a.client, Floor: MinMicroversion, Method: http.MethodGet, URL: manilaown.WithQuery(a.client.ServiceURL("share_instances"), q), OK: []int{http.StatusOK}}
			return manilaown.Iterate[ShareInstance](ctx, call, "share_instances", control)
		},
	})
	return a
}

func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

func (a *API) check(ctx context.Context) error {
	if a == nil || a.client == nil {
		return fmt.Errorf("%w: share instance API is required", resource.ErrInvalidOption)
	}
	return ctx.Err()
}

func wrap(op string, err error) error { return request.Wrap(op, "shareinstances", err) }

// ListOpts are the server filters documented for GET /share_instances. Python
// accepts them in its docstring but silently drops them before the request.
type ListOpts struct {
	ExportLocationID   string `q:"export_location_id"`
	ExportLocationPath string `q:"export_location_path"`
	Limit              *int   `q:"limit"`
	Offset             *int   `q:"offset"`
}

type ListOption = request.Option[ListOpts]

func WithListOptions(value ListOpts) ListOption {
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

func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*ShareInstance, error] {
	if err := a.check(ctx); err != nil {
		return func(yield func(*ShareInstance, error) bool) { yield(nil, wrap("List", err)) }
	}
	route := func(request.Config[ListOpts]) string { return a.client.ServiceURL("share_instances") }
	return manilaown.ListFor[ShareInstance](ctx, a.client, MinMicroversion, "share_instances", route, resource.ListControl{}, options)
}

func (a *API) Get(ctx context.Context, id string) (*ShareInstance, error) {
	if err := a.check(ctx); err != nil {
		return nil, wrap("Get", err)
	}
	if err := resource.ID(id).Validate(); err != nil {
		return nil, wrap("Get", err)
	}
	resp, err := manilaown.Call{Client: a.client, Floor: MinMicroversion, Method: http.MethodGet, URL: a.client.ServiceURL("share_instances", id), OK: []int{http.StatusOK}}.Do(ctx)
	if err != nil {
		return nil, wrap("Get", err)
	}
	value, err := manilaown.One[ShareInstance](resp, "share_instance")
	return value, wrap("Get", err)
}

func (a *API) action(ctx context.Context, op, id string, body map[string]any) error {
	if err := a.check(ctx); err != nil {
		return wrap(op, err)
	}
	if err := resource.ID(id).Validate(); err != nil {
		return wrap(op, err)
	}
	_, err := manilaown.Call{Client: a.client, Floor: MinMicroversion, Method: http.MethodPost, URL: a.client.ServiceURL("share_instances", id, "action"), Body: body, OK: []int{200, 202, 204}}.Do(ctx)
	return wrap(op, err)
}

// ResetStatus posts reset_status for one instance and reports every non-2xx
// response (Python only checks it via raise_from_response as well). A status
// is required. It is never replayed.
func (a *API) ResetStatus(ctx context.Context, id, status string) error {
	if strings.TrimSpace(status) == "" {
		return wrap("ResetStatus", fmt.Errorf("%w: status is required", resource.ErrInvalidOption))
	}
	return a.action(ctx, "ResetStatus", id, map[string]any{"reset_status": map[string]any{"status": status}})
}

// ForceDelete posts {"force_delete":null}. A 404 stays the native error; use
// Remove for the ignore-missing default. Acknowledgement is not completion:
// use WaitForDeletion.
func (a *API) ForceDelete(ctx context.Context, id string) error {
	return a.action(ctx, "ForceDelete", id, map[string]any{"force_delete": nil})
}

func (a *API) Find(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) (*ShareInstance, error) {
	return a.Resources.Find(ctx, ref, options...)
}

func (a *API) All(ctx context.Context, options ...resource.ListOption) ([]*ShareInstance, error) {
	return a.Resources.All(ctx, options...)
}

// Remove force-deletes by reference; a missing instance is ignored unless
// resource.WithMissingError is passed.
func (a *API) Remove(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) error {
	return a.Resources.Delete(ctx, ref, options...)
}

func (a *API) WaitFor(ctx context.Context, ref resource.Ref, status string, options ...resource.WaitOption) (*ShareInstance, error) {
	return a.Resources.Wait(ctx, ref, status, options...)
}

func (a *API) WaitForDeletion(ctx context.Context, ref resource.Ref, options ...resource.WaitOption) error {
	return a.Resources.WaitDeleted(ctx, ref, options...)
}

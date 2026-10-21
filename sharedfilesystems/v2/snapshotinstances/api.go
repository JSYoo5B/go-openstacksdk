// Package snapshotinstances owns Manila's read-only share snapshot instance
// API (microversion 2.19 and later), absent from Gophercloud v2.15.0.
package snapshotinstances

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

const MinMicroversion = 19

type Evidence = manilaown.Evidence

// SnapshotInstance is the typed view of one snapshot instance. The summary list
// only carries id, status and links.
type SnapshotInstance struct {
	ID               string   `json:"id"`
	CreatedAt        string   `json:"created_at"`
	Progress         string   `json:"progress"`
	ProviderLocation string   `json:"provider_location"`
	ShareID          string   `json:"share_id"`
	ShareInstanceID  string   `json:"share_instance_id"`
	SnapshotID       string   `json:"snapshot_id"`
	Status           string   `json:"status"`
	UpdatedAt        string   `json:"updated_at"`
	Evidence         Evidence `json:"-"`
}

type API struct {
	client    *gophercloud.ServiceClient
	Resources *resource.Collection[SnapshotInstance]
}

func New(client *gophercloud.ServiceClient) *API {
	a := &API{client: client}
	a.Resources = resource.NewCollection(resource.Adapter[SnapshotInstance]{
		Kind:         "snapshotinstances",
		IdentityFind: true,
		Get:          a.Get,
		ID:           func(v *SnapshotInstance) string { return v.ID },
		Name:         func(*SnapshotInstance) string { return "" },
		Status:       func(v *SnapshotInstance) string { return v.Status },
		Failed:       func(s string) bool { return strings.HasPrefix(strings.ToLower(s), "error") },
		IterateControlled: func(ctx context.Context, q url.Values, control resource.ListControl) iter.Seq2[*SnapshotInstance, error] {
			call := manilaown.Call{Client: a.client, Floor: MinMicroversion, Method: http.MethodGet, URL: manilaown.WithQuery(a.route(true), q), OK: []int{http.StatusOK}}
			return manilaown.Iterate[SnapshotInstance](ctx, call, "snapshot_instances", control)
		},
	})
	return a
}

func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

func (a *API) route(details bool) string {
	if details {
		return a.client.ServiceURL("snapshot-instances", "detail")
	}
	return a.client.ServiceURL("snapshot-instances")
}

func (a *API) check(ctx context.Context) error {
	if a == nil || a.client == nil {
		return fmt.Errorf("%w: snapshot instance API is required", resource.ErrInvalidOption)
	}
	return ctx.Err()
}

func wrap(op string, err error) error { return request.Wrap(op, "snapshotinstances", err) }

// ListOpts are the server filters. Python's docstring names snapshot_id and
// project_id, but its query mapping drops both before the request.
type ListOpts struct {
	SnapshotID string `q:"snapshot_id"`
	ProjectID  string `q:"project_id"`
	Limit      *int   `q:"limit"`
	Offset     *int   `q:"offset"`
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

const detailsArgument = "snapshotinstances.details"

// WithListDetails selects /snapshot-instances/detail (true, the default, as
// Python's details=True) or the summary route /snapshot-instances.
func WithListDetails(value bool) ListOption {
	return request.WithArgument[ListOpts](detailsArgument, value)
}

func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*SnapshotInstance, error] {
	if err := a.check(ctx); err != nil {
		return func(yield func(*SnapshotInstance, error) bool) { yield(nil, wrap("List", err)) }
	}
	route := func(cfg request.Config[ListOpts]) string {
		details, ok, err := request.Argument[bool](cfg, detailsArgument)
		if err != nil || !ok {
			details = true
		}
		return a.route(details)
	}
	return manilaown.ListFor[SnapshotInstance](ctx, a.client, MinMicroversion, "snapshot_instances", route, resource.ListControl{}, options, detailsArgument)
}

func (a *API) Get(ctx context.Context, id string) (*SnapshotInstance, error) {
	if err := a.check(ctx); err != nil {
		return nil, wrap("Get", err)
	}
	if err := resource.ID(id).Validate(); err != nil {
		return nil, wrap("Get", err)
	}
	resp, err := manilaown.Call{Client: a.client, Floor: MinMicroversion, Method: http.MethodGet, URL: a.client.ServiceURL("snapshot-instances", id), OK: []int{http.StatusOK}}.Do(ctx)
	if err != nil {
		return nil, wrap("Get", err)
	}
	value, err := manilaown.One[SnapshotInstance](resp, "snapshot_instance")
	return value, wrap("Get", err)
}

func (a *API) Find(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) (*SnapshotInstance, error) {
	return a.Resources.Find(ctx, ref, options...)
}

func (a *API) All(ctx context.Context, options ...resource.ListOption) ([]*SnapshotInstance, error) {
	return a.Resources.All(ctx, options...)
}

func (a *API) WaitFor(ctx context.Context, ref resource.Ref, status string, options ...resource.WaitOption) (*SnapshotInstance, error) {
	return a.Resources.Wait(ctx, ref, status, options...)
}

// Package resourcelocks owns Manila's resource-lock API (microversion 2.81),
// absent from Gophercloud v2.15.0. Locks prevent an action such as deletion or
// field visibility on a share or access rule until the lock is removed.
package resourcelocks

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

// MinMicroversion is the minor version at which Manila introduced the routes.
const MinMicroversion = 81

// Evidence is the actual wire body, headers and status of a response.
type Evidence = manilaown.Evidence

// ResourceLock is the typed view of one lock. Absent and null members decode to
// empty strings; Evidence.Body distinguishes them.
type ResourceLock struct {
	ID             string   `json:"id"`
	ResourceID     string   `json:"resource_id"`
	ResourceType   string   `json:"resource_type"`
	ResourceAction string   `json:"resource_action"`
	LockReason     string   `json:"lock_reason"`
	LockContext    string   `json:"lock_context"`
	UserID         string   `json:"user_id"`
	ProjectID      string   `json:"project_id"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
	Evidence       Evidence `json:"-"`
}

// API owns typed lock operations and the shared Collection policies.
type API struct {
	client    *gophercloud.ServiceClient
	Resources *resource.Collection[ResourceLock]
}

func New(client *gophercloud.ServiceClient) *API {
	a := &API{client: client}
	a.Resources = resource.NewCollection(resource.Adapter[ResourceLock]{
		Kind:         "resourcelocks",
		IdentityFind: true,
		Get:          a.Get,
		ID:           func(v *ResourceLock) string { return v.ID },
		Name:         func(*ResourceLock) string { return "" },
		Delete:       a.Delete,
		IterateControlled: func(ctx context.Context, q url.Values, control resource.ListControl) iter.Seq2[*ResourceLock, error] {
			call := manilaown.Call{Client: a.client, Floor: MinMicroversion, Method: http.MethodGet, URL: manilaown.WithQuery(a.client.ServiceURL("resource-locks"), q), OK: []int{http.StatusOK}}
			return manilaown.Iterate[ResourceLock](ctx, call, "resource_locks", control)
		},
	})
	return a
}

func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

func (a *API) check(ctx context.Context) error {
	if a == nil || a.client == nil {
		return fmt.Errorf("%w: resource lock API is required", resource.ErrInvalidOption)
	}
	return ctx.Err()
}

func wrap(op string, err error) error { return request.Wrap(op, "resourcelocks", err) }

// ListOpts are the server-side filters of GET /resource-locks. Strings are
// omitted when empty; pointers preserve zero and false.
type ListOpts struct {
	ProjectID      string `q:"project_id"`
	CreatedSince   string `q:"created_since"`
	CreatedBefore  string `q:"created_before"`
	Limit          *int   `q:"limit"`
	Offset         *int   `q:"offset"`
	ID             string `q:"id"`
	ResourceID     string `q:"resource_id"`
	ResourceType   string `q:"resource_type"`
	ResourceAction string `q:"resource_action"`
	UserID         string `q:"user_id"`
	LockContext    string `q:"lock_context"`
	LockReason     string `q:"lock_reason"`
	LockReasonLike string `q:"lock_reason~"`
	SortKey        string `q:"sort_key"`
	SortDir        string `q:"sort_dir"`
	WithCount      *bool  `q:"with_count"`
	AllProjects    *bool  `q:"all_projects"`
}

type ListOption = request.Option[ListOpts]

func WithListOptions(value ListOpts) ListOption {
	copy := value
	copy.Limit, copy.Offset = clonePtr(value.Limit), clonePtr(value.Offset)
	copy.WithCount, copy.AllProjects = clonePtr(value.WithCount), clonePtr(value.AllProjects)
	return request.WithOptions(copy)
}

func clonePtr[T any](p *T) *T {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

// WithListQuery adds a query parameter that no typed field owns.
func WithListQuery(key, value string) ListOption { return request.WithQuery[ListOpts](key, value) }

// WithListMaxItems caps consumed rows locally (zero is unlimited, negative is
// rejected before HTTP). It never becomes a wire limit.
func WithListMaxItems(value int) ListOption {
	return manilaown.WithListControl[ListOpts](func(c *resource.ListControl) { c.MaxItems = value })
}

// List yields every lock of one GET /resource-locks response. No marker or
// offset continuation is followed; pass WithListOptions Limit/Offset for pages.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*ResourceLock, error] {
	if err := a.check(ctx); err != nil {
		return func(yield func(*ResourceLock, error) bool) { yield(nil, wrap("List", err)) }
	}
	route := func(request.Config[ListOpts]) string { return a.client.ServiceURL("resource-locks") }
	return manilaown.ListFor[ResourceLock](ctx, a.client, MinMicroversion, "resource_locks", route, resource.ListControl{}, options)
}

// Get fetches one lock by ID; only HTTP 200 succeeds.
func (a *API) Get(ctx context.Context, id string) (*ResourceLock, error) {
	if err := a.check(ctx); err != nil {
		return nil, wrap("Get", err)
	}
	if err := resource.ID(id).Validate(); err != nil {
		return nil, wrap("Get", err)
	}
	resp, err := manilaown.Call{Client: a.client, Floor: MinMicroversion, Method: http.MethodGet, URL: a.client.ServiceURL("resource-locks", id), OK: []int{http.StatusOK}}.Do(ctx)
	if err != nil {
		return nil, wrap("Get", err)
	}
	value, err := manilaown.One[ResourceLock](resp, "resource_lock")
	return value, wrap("Get", err)
}

// CreateOpts is the body of POST /resource-locks. ResourceID and
// ResourceType are required; Optional values preserve empty strings and null.
type CreateOpts struct {
	ResourceID     string                   `json:"resource_id"`
	ResourceType   string                   `json:"resource_type"`
	ResourceAction request.Optional[string] `json:"resource_action,omitzero"`
	LockReason     request.Optional[string] `json:"lock_reason,omitzero"`
}

type CreateOption = request.Option[CreateOpts]

func WithCreateOptions(value CreateOpts) CreateOption { return request.WithOptions(value) }

// WithCreateField adds a JSON member the typed options do not own.
func WithCreateField(key string, value any) CreateOption {
	return request.WithField[CreateOpts](key, value)
}

type protectedFields struct {
	CreateOpts
	ID          string `json:"id"`
	LockContext string `json:"lock_context"`
	UserID      string `json:"user_id"`
	ProjectID   string `json:"project_id"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	Envelope    any    `json:"resource_lock"`
}

// Create sends one POST. The lock exists after a 200; an uncertain failure is
// never replayed, so check List/Get before trying again.
func (a *API) Create(ctx context.Context, opts CreateOpts, options ...CreateOption) (*ResourceLock, error) {
	if err := a.check(ctx); err != nil {
		return nil, wrap("Create", err)
	}
	cfg, err := request.Apply(opts, options...)
	if err == nil {
		err = request.ValidateCapabilities(cfg, true, false, false)
	}
	if err == nil && (strings.TrimSpace(cfg.Options.ResourceID) == "" || strings.TrimSpace(cfg.Options.ResourceType) == "") {
		err = fmt.Errorf("%w: resource_id and resource_type are required", resource.ErrInvalidOption)
	}
	var body map[string]any
	if err == nil {
		body, err = manilaown.Envelope("resource_lock", cfg.Options, cfg.Fields, protectedFields{})
	}
	if err != nil {
		return nil, wrap("Create", err)
	}
	resp, err := manilaown.Call{Client: a.client, Floor: MinMicroversion, Method: http.MethodPost, URL: a.client.ServiceURL("resource-locks"), Body: body, OK: []int{http.StatusOK}}.Do(ctx)
	if err != nil {
		return nil, wrap("Create", err)
	}
	value, err := manilaown.One[ResourceLock](resp, "resource_lock")
	return value, wrap("Create", err)
}

// UpdateOpts changes the two mutable lock members. Null clears lock_reason.
type UpdateOpts struct {
	ResourceAction request.Optional[string] `json:"resource_action,omitzero"`
	LockReason     request.Optional[string] `json:"lock_reason,omitzero"`
}

type UpdateOption = request.Option[UpdateOpts]

func WithUpdateOptions(value UpdateOpts) UpdateOption { return request.WithOptions(value) }

func WithUpdateField(key string, value any) UpdateOption {
	return request.WithField[UpdateOpts](key, value)
}

type protectedUpdate struct {
	UpdateOpts
	ID           string `json:"id"`
	ResourceID   string `json:"resource_id"`
	ResourceType string `json:"resource_type"`
	LockContext  string `json:"lock_context"`
	UserID       string `json:"user_id"`
	ProjectID    string `json:"project_id"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	Envelope     any    `json:"resource_lock"`
}

// Update sends PUT with only the supplied members. An update that supplies
// nothing is rejected before HTTP.
func (a *API) Update(ctx context.Context, id string, opts UpdateOpts, options ...UpdateOption) (*ResourceLock, error) {
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
	if err == nil && cfg.Options.ResourceAction.IsZero() && cfg.Options.LockReason.IsZero() && len(cfg.Fields) == 0 {
		err = fmt.Errorf("%w: update supplies no field", resource.ErrInvalidOption)
	}
	var body map[string]any
	if err == nil {
		body, err = manilaown.Envelope("resource_lock", cfg.Options, cfg.Fields, protectedUpdate{})
	}
	if err != nil {
		return nil, wrap("Update", err)
	}
	resp, err := manilaown.Call{Client: a.client, Floor: MinMicroversion, Method: http.MethodPut, URL: a.client.ServiceURL("resource-locks", id), Body: body, OK: []int{http.StatusOK}}.Do(ctx)
	if err != nil {
		return nil, wrap("Update", err)
	}
	value, err := manilaown.One[ResourceLock](resp, "resource_lock")
	return value, wrap("Update", err)
}

// Delete sends one DELETE; a 404 stays the native error (use Remove to apply
// the ignore-missing default). Success is 200, 202 or 204.
func (a *API) Delete(ctx context.Context, id string) error {
	if err := a.check(ctx); err != nil {
		return wrap("Delete", err)
	}
	if err := resource.ID(id).Validate(); err != nil {
		return wrap("Delete", err)
	}
	_, err := manilaown.Call{Client: a.client, Floor: MinMicroversion, Method: http.MethodDelete, URL: a.client.ServiceURL("resource-locks", id), OK: []int{200, 202, 204}}.Do(ctx)
	return wrap("Delete", err)
}

// Find resolves a lock by ID reference (locks have no names).
func (a *API) Find(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) (*ResourceLock, error) {
	return a.Resources.Find(ctx, ref, options...)
}

// FindIdentity mirrors Resource.find: GET by ID first, then an ID match in the list.
func (a *API) FindIdentity(ctx context.Context, identity string, options ...resource.IdentityFindOption) (*ResourceLock, error) {
	return a.Resources.FindIdentity(ctx, identity, options...)
}

func (a *API) All(ctx context.Context, options ...resource.ListOption) ([]*ResourceLock, error) {
	return a.Resources.All(ctx, options...)
}

// Remove deletes by reference. Missing locks are ignored unless
// resource.WithMissingError is passed (ignore_missing=False).
func (a *API) Remove(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) error {
	return a.Resources.Delete(ctx, ref, options...)
}

// WaitForDeletion polls Get until 404. Lock deletion is not assumed complete
// from the acknowledgement.
func (a *API) WaitForDeletion(ctx context.Context, ref resource.Ref, options ...resource.WaitOption) error {
	return a.Resources.WaitDeleted(ctx, ref, options...)
}

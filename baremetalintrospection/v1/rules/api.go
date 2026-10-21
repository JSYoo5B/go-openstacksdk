// Package rules manages ironic-inspector introspection rules: POST/GET /rules
// and GET/DELETE /rules/{uuid}.
//
// The pinned Gophercloud release has no introspection rule binding, so this
// package is SDK-owned. The client microversion is used as configured.
//
// Python's IntrospectionRule marks create with requires_id=True, so its create
// would address POST /rules/{uuid} (and fail locally without a uuid). This
// package deliberately follows the documented route POST /rules and sends an
// optional uuid in the body; see README.md.
package rules

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

const kind = "introspection_rules"

type API struct {
	client *gophercloud.ServiceClient
	rest   ironicrest.Resource[Rule]
}

func New(client *gophercloud.ServiceClient) *API {
	return &API{client: client, rest: ironicrest.Resource[Rule]{
		Client: client, Kind: "introspection rule", Collection: "rules", ListKey: "rules", Decode: decode,
		Identify: func(r *Rule) (string, string) { return r.UUID, "" },
	}}
}

func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

// CreateOpts creates a rule. Nil pointers, nil slices are omitted; an empty
// description or scope, or an empty slice, is sent.
type CreateOpts struct {
	UUID        *string     `json:"uuid,omitempty"`
	Description *string     `json:"description,omitempty"`
	Scope       *string     `json:"scope,omitempty"`
	Conditions  []Condition `json:"conditions,omitzero"`
	Actions     []Action    `json:"actions,omitzero"`
}

type CreateOption = request.Option[CreateOpts]

// WithCreateField adds a JSON member this package does not model. It cannot
// replace a typed member.
func WithCreateField(key string, value any) CreateOption {
	return request.WithField[CreateOpts](key, value)
}
func WithCreateHeader(key, value string) CreateOption {
	return request.WithHeader[CreateOpts](key, value)
}

// Create sends POST /rules. Only 201 succeeds; a failed or uncertain call is
// never replayed.
func (a *API) Create(ctx context.Context, opts CreateOpts, options ...CreateOption) (*Rule, error) {
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

type GetOption = request.Option[struct{}]

func WithGetHeader(key, value string) GetOption { return request.WithHeader[struct{}](key, value) }

// Get sends GET /rules/{uuid}. Only 200 succeeds. The inspector has no name
// lookup and no fields query.
func (a *API) Get(ctx context.Context, id string, options ...GetOption) (*Rule, error) {
	config, err := request.Apply(struct{}{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	if err != nil {
		return nil, request.Wrap("Get", kind, err)
	}
	value, err := a.rest.Get(ctx, id, nil, config.Headers)
	return value, request.Wrap("Get", kind, err)
}

// ListOpts: MaxItems caps the rows yielded after one GET /rules; it is never
// sent as limit because the inspector has no paging.
type ListOpts struct{ MaxItems int }

type ListOption = request.Option[ListOpts]

func WithListMaxItems(n int) ListOption {
	return func(c *request.Config[ListOpts]) error { c.Options.MaxItems = n; return nil }
}
func WithListHeader(key, value string) ListOption { return request.WithHeader[ListOpts](key, value) }

// List sends one GET /rules and yields its rows. It never follows a next link
// and never sends limit or marker. Rows hold the brief representation.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*Rule, error] {
	config, err := request.Apply(ListOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	if err == nil && config.Options.MaxItems < 0 {
		err = fmt.Errorf("%w: max items must not be negative", resource.ErrInvalidOption)
	}
	if err != nil {
		return func(yield func(*Rule, error) bool) { yield(nil, request.Wrap("List", kind, err)) }
	}
	cfg := ironicrest.ListConfig{Headers: config.Headers, Paginated: false, MaxItems: config.Options.MaxItems, NoLimitHint: true}
	return func(yield func(*Rule, error) bool) {
		for value, err := range a.rest.List(ctx, cfg) {
			if !yield(value, request.Wrap("List", kind, err)) {
				return
			}
		}
	}
}

type DeleteOpts struct{ IgnoreMissing bool }

type DeleteOption = request.Option[DeleteOpts]

func WithDeleteIgnoreMissing(ignore bool) DeleteOption {
	return func(c *request.Config[DeleteOpts]) error { c.Options.IgnoreMissing = ignore; return nil }
}
func WithDeleteHeader(key, value string) DeleteOption {
	return request.WithHeader[DeleteOpts](key, value)
}

// Delete sends DELETE /rules/{uuid}. Only 204 succeeds. By default a missing
// rule is not an error and deleted is false.
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

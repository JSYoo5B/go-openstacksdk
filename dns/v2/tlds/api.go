// Package tlds binds Designate's top-level domains (/v2/tlds).
// Gophercloud v2.15.0 has no binding, so the SDK owns routes and bodies.
package tlds

import (
	"context"
	"iter"
	"net/http"
	"net/url"
	"strconv"

	"github.com/JSYoo5B/go-openstacksdk/dns/v2/internal/designate"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// TLD is a decoded Designate TLD. Body retains the exact object.
type TLD struct {
	designate.Evidence
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
	Links       map[string]string `json:"links"`
}

// Acknowledgement is the bodyless DELETE result.
type Acknowledgement = designate.Ack

// CreateOpts is the POST body. Nil-equivalent Description is omitted; explicit
// empty and null descriptions are preserved with request.Present / request.Null.
type CreateOpts struct {
	Name        string                   `json:"name"`
	Description request.Optional[string] `json:"description,omitzero"`
}

// UpdateOpts is the PATCH body: only supplied fields are sent.
type UpdateOpts struct {
	Name        request.Optional[string] `json:"name,omitzero"`
	Description request.Optional[string] `json:"description,omitzero"`
}

// ListOpts holds typed filters. Zero values are omitted; use WithListQuery to
// send an explicit empty value.
type ListOpts struct {
	Name        string
	Description string
	Limit       int
	Marker      string
	SortKey     string
	SortDir     string
}

func (o ListOpts) values() url.Values {
	q := url.Values{}
	set := func(k, v string) {
		if v != "" {
			q.Set(k, v)
		}
	}
	set("name", o.Name)
	set("description", o.Description)
	if o.Limit != 0 {
		q.Set("limit", strconv.Itoa(o.Limit))
	}
	set("marker", o.Marker)
	set("sort_key", o.SortKey)
	set("sort_dir", o.SortDir)
	return q
}

type CreateOption = request.Option[CreateOpts]
type UpdateOption = request.Option[UpdateOpts]
type ListOption = request.Option[ListOpts]

func WithCreateOptions(value CreateOpts) CreateOption { return request.WithOptions(value) }
func WithCreateField(key string, value any) CreateOption {
	return request.WithField[CreateOpts](key, value)
}
func WithUpdateOptions(value UpdateOpts) UpdateOption { return request.WithOptions(value) }
func WithUpdateField(key string, value any) UpdateOption {
	return request.WithField[UpdateOpts](key, value)
}
func WithListOptions(value ListOpts) ListOption  { return request.WithOptions(value) }
func WithListQuery(key, value string) ListOption { return request.WithQuery[ListOpts](key, value) }

// API owns one authenticated Designate client.
type API struct {
	Resources *resource.Collection[TLD]
	client    *gophercloud.ServiceClient
	route     designate.Route[TLD, *TLD]
}

func New(client *gophercloud.ServiceClient) *API {
	a := &API{client: client, route: designate.Route[TLD, *TLD]{Client: client, Segments: []string{"tlds"}, PluralKey: "tlds"}}
	a.Resources = resource.NewCollection(resource.Adapter[TLD]{
		Kind:         "tlds",
		Get:          func(ctx context.Context, id string) (*TLD, error) { return a.Get(ctx, id) },
		ID:           func(v *TLD) string { return v.ID },
		Name:         func(v *TLD) string { return v.Name },
		NameQuery:    func(name string) string { return name },
		IdentityFind: true,
		Delete: func(ctx context.Context, id string) error {
			_, err := a.Delete(ctx, id)
			return err
		},
		IterateControlled: func(ctx context.Context, q url.Values, control resource.ListControl) iter.Seq2[*TLD, error] {
			return a.route.List(ctx, q, nil, control)
		},
	})
	return a
}

func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

// Create sends POST /tlds and accepts only 201.
func (a *API) Create(ctx context.Context, opts CreateOpts, options ...CreateOption) (*TLD, error) {
	cfg, err := request.Apply(opts, options...)
	if err == nil {
		err = request.ValidateCapabilities(cfg, true, false, false)
	}
	var body map[string]any
	if err == nil {
		body, err = designate.Body(cfg.Options, cfg.Fields)
	}
	if err != nil {
		return nil, request.Wrap("Create", "tlds", err)
	}
	value, err := a.route.Send(ctx, http.MethodPost, "", body, nil, http.StatusCreated)
	return value, request.Wrap("Create", "tlds", err)
}

// Get sends GET /tlds/{id} and accepts only 200.
func (a *API) Get(ctx context.Context, id string) (*TLD, error) {
	value, err := a.route.Get(ctx, id, nil)
	return value, request.Wrap("Get", "tlds", designate.NotFound("tld", id, err))
}

// Update sends PATCH /tlds/{id}, accepts only 200 and never replays.
func (a *API) Update(ctx context.Context, id string, opts UpdateOpts, options ...UpdateOption) (*TLD, error) {
	cfg, err := request.Apply(opts, options...)
	if err == nil {
		err = request.ValidateCapabilities(cfg, true, false, false)
	}
	var body map[string]any
	if err == nil {
		body, err = designate.Body(cfg.Options, cfg.Fields)
	}
	if err != nil {
		return nil, request.Wrap("Update", "tlds", err)
	}
	value, err := a.route.Send(ctx, http.MethodPatch, id, body, nil, http.StatusOK)
	return value, request.Wrap("Update", "tlds", designate.NotFound("tld", id, err))
}

// Delete sends DELETE /tlds/{id} and accepts only 204. A missing
// tld is an error here; Remove ignores it by default.
func (a *API) Delete(ctx context.Context, id string) (*Acknowledgement, error) {
	ack, err := a.route.Delete(ctx, id, nil, http.StatusNoContent)
	return ack, request.Wrap("Delete", "tlds", designate.NotFound("tld", id, err))
}

// Remove deletes by reference; a missing tld is ignored unless
// resource.WithMissingError() is supplied (Python ignore_missing=True).
func (a *API) Remove(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) error {
	return a.Resources.Delete(ctx, ref, options...)
}

// List streams tlds across pages (links.next, then marker/limit).
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*TLD, error] {
	cfg, err := request.Apply(ListOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(cfg, false, true, false)
	}
	if err != nil {
		err = request.Wrap("List", "tlds", err)
		return func(yield func(*TLD, error) bool) { yield(nil, err) }
	}
	return a.route.List(ctx, designate.Query(cfg.Options.values(), cfg.Query), nil, resource.ListControl{})
}

// All collects List.
func (a *API) All(ctx context.Context, options ...resource.ListOption) ([]*TLD, error) {
	return a.Resources.All(ctx, options...)
}

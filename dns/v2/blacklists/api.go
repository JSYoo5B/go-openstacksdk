// Package blacklists binds Designate's zone-name blacklists (/v2/blacklists).
// Gophercloud v2.15.0 has no binding, so the SDK owns routes and bodies.
package blacklists

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

// Blacklist is a decoded Designate blacklist. Body retains the exact object.
type Blacklist struct {
	designate.Evidence
	ID          string            `json:"id"`
	Pattern     string            `json:"pattern"`
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
	Pattern     string                   `json:"pattern"`
	Description request.Optional[string] `json:"description,omitzero"`
}

// UpdateOpts is the PATCH body: only supplied fields are sent.
type UpdateOpts struct {
	Pattern     request.Optional[string] `json:"pattern,omitzero"`
	Description request.Optional[string] `json:"description,omitzero"`
}

// ListOpts holds typed filters. Zero values are omitted; use WithListQuery to
// send an explicit empty value.
type ListOpts struct {
	Pattern string
	Limit   int
	Marker  string
	SortKey string
	SortDir string
}

func (o ListOpts) values() url.Values {
	q := url.Values{}
	set := func(k, v string) {
		if v != "" {
			q.Set(k, v)
		}
	}
	set("pattern", o.Pattern)
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
	Resources *resource.Collection[Blacklist]
	client    *gophercloud.ServiceClient
	route     designate.Route[Blacklist, *Blacklist]
}

func New(client *gophercloud.ServiceClient) *API {
	a := &API{client: client, route: designate.Route[Blacklist, *Blacklist]{Client: client, Segments: []string{"blacklists"}, PluralKey: "blacklists"}}
	a.Resources = resource.NewCollection(resource.Adapter[Blacklist]{
		Kind: "blacklists",
		Get:  func(ctx context.Context, id string) (*Blacklist, error) { return a.Get(ctx, id) },
		ID:   func(v *Blacklist) string { return v.ID },
		Delete: func(ctx context.Context, id string) error {
			_, err := a.Delete(ctx, id)
			return err
		},
		IterateControlled: func(ctx context.Context, q url.Values, control resource.ListControl) iter.Seq2[*Blacklist, error] {
			return a.route.List(ctx, q, nil, control)
		},
	})
	return a
}

func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

// Create sends POST /blacklists and accepts only 201.
func (a *API) Create(ctx context.Context, opts CreateOpts, options ...CreateOption) (*Blacklist, error) {
	cfg, err := request.Apply(opts, options...)
	if err == nil {
		err = request.ValidateCapabilities(cfg, true, false, false)
	}
	var body map[string]any
	if err == nil {
		body, err = designate.Body(cfg.Options, cfg.Fields)
	}
	if err != nil {
		return nil, request.Wrap("Create", "blacklists", err)
	}
	value, err := a.route.Send(ctx, http.MethodPost, "", body, nil, http.StatusCreated)
	return value, request.Wrap("Create", "blacklists", err)
}

// Get sends GET /blacklists/{id} and accepts only 200.
func (a *API) Get(ctx context.Context, id string) (*Blacklist, error) {
	value, err := a.route.Get(ctx, id, nil)
	return value, request.Wrap("Get", "blacklists", designate.NotFound("blacklist", id, err))
}

// Update sends PATCH /blacklists/{id}, accepts only 200 and never replays.
func (a *API) Update(ctx context.Context, id string, opts UpdateOpts, options ...UpdateOption) (*Blacklist, error) {
	cfg, err := request.Apply(opts, options...)
	if err == nil {
		err = request.ValidateCapabilities(cfg, true, false, false)
	}
	var body map[string]any
	if err == nil {
		body, err = designate.Body(cfg.Options, cfg.Fields)
	}
	if err != nil {
		return nil, request.Wrap("Update", "blacklists", err)
	}
	value, err := a.route.Send(ctx, http.MethodPatch, id, body, nil, http.StatusOK)
	return value, request.Wrap("Update", "blacklists", designate.NotFound("blacklist", id, err))
}

// Delete sends DELETE /blacklists/{id} and accepts only 204. A missing
// blacklist is an error here; Remove ignores it by default.
func (a *API) Delete(ctx context.Context, id string) (*Acknowledgement, error) {
	ack, err := a.route.Delete(ctx, id, nil, http.StatusNoContent)
	return ack, request.Wrap("Delete", "blacklists", designate.NotFound("blacklist", id, err))
}

// Remove deletes by reference; a missing blacklist is ignored unless
// resource.WithMissingError() is supplied (Python ignore_missing=True).
func (a *API) Remove(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) error {
	return a.Resources.Delete(ctx, ref, options...)
}

// List streams blacklists across pages (links.next, then marker/limit).
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*Blacklist, error] {
	cfg, err := request.Apply(ListOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(cfg, false, true, false)
	}
	if err != nil {
		err = request.Wrap("List", "blacklists", err)
		return func(yield func(*Blacklist, error) bool) { yield(nil, err) }
	}
	return a.route.List(ctx, designate.Query(cfg.Options.values(), cfg.Query), nil, resource.ListControl{})
}

// All collects List.
func (a *API) All(ctx context.Context, options ...resource.ListOption) ([]*Blacklist, error) {
	return a.Resources.All(ctx, options...)
}

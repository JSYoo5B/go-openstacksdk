// Package floatingips binds Designate reverse DNS (PTR) records of floating
// IPs under /v2/reverse/floatingips. Gophercloud v2.15.0 has no binding.
// Identifiers use Designate's "<region>:<floatingip-id>" form.
package floatingips

import (
	"context"
	"iter"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/dns/v2/internal/designate"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// FloatingIP is a decoded PTR record view of one floating IP.
type FloatingIP struct {
	designate.Evidence
	ID          string         `json:"id"`
	Action      string         `json:"action"`
	Address     string         `json:"address"`
	Description string         `json:"description"`
	PTRDName    string         `json:"ptrdname"`
	Status      string         `json:"status"`
	TTL         int            `json:"ttl"`
	Links       map[string]any `json:"links"`
}

// SetOpts is the PATCH body. Omitted fields are not sent; request.Null clears a
// value, request.Present preserves false/zero/empty values.
type SetOpts struct {
	PTRDName    request.Optional[string] `json:"ptrdname,omitzero"`
	Description request.Optional[string] `json:"description,omitzero"`
	TTL         request.Optional[int]    `json:"ttl,omitzero"`
}

// ListOpts holds paging controls and scope headers.
type ListOpts struct {
	Limit       int
	Marker      string
	AllProjects *bool
	SudoProject string
}

func (o ListOpts) values() url.Values {
	q := url.Values{}
	if o.Limit != 0 {
		q.Set("limit", strconv.Itoa(o.Limit))
	}
	if o.Marker != "" {
		q.Set("marker", o.Marker)
	}
	return q
}

type SetOption = request.Option[SetOpts]
type ListOption = request.Option[ListOpts]

func WithSetOptions(value SetOpts) SetOption { return request.WithOptions(value) }
func WithSetField(key string, value any) SetOption {
	return request.WithField[SetOpts](key, value)
}
func WithSetHeader(key, value string) SetOption  { return request.WithHeader[SetOpts](key, value) }
func WithListOptions(value ListOpts) ListOption  { return request.WithOptions(value) }
func WithListQuery(key, value string) ListOption { return request.WithQuery[ListOpts](key, value) }

// API owns one authenticated Designate client.
type API struct {
	Resources *resource.Collection[FloatingIP]
	client    *gophercloud.ServiceClient
	route     designate.Route[FloatingIP, *FloatingIP]
}

func New(client *gophercloud.ServiceClient) *API {
	a := &API{client: client, route: designate.Route[FloatingIP, *FloatingIP]{Client: client, Segments: []string{"reverse", "floatingips"}, PluralKey: "floatingips"}}
	a.Resources = resource.NewCollection(resource.Adapter[FloatingIP]{
		Kind:   "floating IP PTR records",
		Get:    func(ctx context.Context, id string) (*FloatingIP, error) { return a.Get(ctx, id) },
		ID:     func(v *FloatingIP) string { return v.ID },
		Status: func(v *FloatingIP) string { return v.Status },
		Failed: func(status string) bool { return strings.EqualFold(status, "ERROR") },
		IterateControlled: func(ctx context.Context, q url.Values, control resource.ListControl) iter.Seq2[*FloatingIP, error] {
			return a.route.List(ctx, q, nil, control)
		},
	})
	return a
}

func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

// Get sends GET /reverse/floatingips/{region}:{id}, accepting only 200.
func (a *API) Get(ctx context.Context, id string) (*FloatingIP, error) {
	value, err := a.route.Get(ctx, id, nil)
	return value, request.Wrap("Get", "floating IP PTR records", designate.NotFound("floating IP PTR record", id, err))
}

// Set sends PATCH /reverse/floatingips/{id} with only the supplied fields and
// accepts 200 or 202. A 202 acknowledges the change; it does not mean the PTR
// record is ACTIVE. The body never carries the identifier.
func (a *API) Set(ctx context.Context, id string, opts SetOpts, options ...SetOption) (*FloatingIP, error) {
	cfg, err := request.Apply(opts, options...)
	if err == nil {
		err = request.ValidateCapabilities(cfg, true, false, true)
	}
	var body map[string]any
	var headers map[string]string
	if err == nil {
		body, err = designate.Body(cfg.Options, cfg.Fields)
	}
	if err == nil {
		headers, err = request.MergeHeadersFor(nil, cfg.Headers, cfg.Options)
	}
	if err != nil {
		return nil, request.Wrap("Set", "floating IP PTR records", err)
	}
	value, err := a.route.Send(ctx, http.MethodPatch, id, body, headers, http.StatusOK, http.StatusAccepted)
	return value, request.Wrap("Set", "floating IP PTR records", designate.NotFound("floating IP PTR record", id, err))
}

// Unset sends PATCH {"ptrdname": null}, the documented way to remove a PTR
// record, accepting 200 or 202.
func (a *API) Unset(ctx context.Context, id string) (*FloatingIP, error) {
	value, err := a.Set(ctx, id, SetOpts{PTRDName: request.Null[string]()})
	return value, err
}

// List streams PTR records across pages.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*FloatingIP, error] {
	cfg, err := request.Apply(ListOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(cfg, false, true, false)
	}
	if err != nil {
		err = request.Wrap("List", "floating IP PTR records", err)
		return func(yield func(*FloatingIP, error) bool) { yield(nil, err) }
	}
	headers := designate.Headers(cfg.Options.AllProjects, cfg.Options.SudoProject)
	return a.route.List(ctx, designate.Query(cfg.Options.values(), cfg.Query), headers, resource.ListControl{})
}

// Find resolves an explicit ID with one GET.
func (a *API) Find(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) (*FloatingIP, error) {
	return a.Resources.Find(ctx, ref, options...)
}

// All collects the PTR record list with common list options.
func (a *API) All(ctx context.Context, options ...resource.ListOption) ([]*FloatingIP, error) {
	return a.Resources.All(ctx, options...)
}

// WaitFor polls GET until status matches case-insensitively; ERROR fails.
func (a *API) WaitFor(ctx context.Context, ref resource.Ref, status string, options ...resource.WaitOption) (*FloatingIP, error) {
	return a.Resources.Wait(ctx, ref, status, options...)
}

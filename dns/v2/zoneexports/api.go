// Package zoneexports binds Designate zone export tasks. Gophercloud v2.15.0
// has no binding, so the SDK owns the documented routes: POST
// /zones/{zone}/tasks/export and /zones/tasks/exports[/{id}[/export]].
package zoneexports

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

// ZoneExport is a decoded export task. Body retains the exact JSON object.
type ZoneExport struct {
	designate.Evidence
	ID        string         `json:"id"`
	CreatedAt string         `json:"created_at"`
	UpdatedAt string         `json:"updated_at"`
	Links     map[string]any `json:"links"`
	Message   string         `json:"message"`
	ProjectID string         `json:"project_id"`
	Status    string         `json:"status"`
	Version   int            `json:"version"`
	ZoneID    string         `json:"zone_id"`
}

// Acknowledgement is the bodyless DELETE result.
type Acknowledgement = designate.Ack

// Text is the zone file returned by GetText, with its HTTP evidence.
type Text struct {
	Zone       string
	Header     http.Header
	StatusCode int
}

// ListOpts holds typed filters and scope headers. Zero values are omitted;
// AllProjects and SudoProject are sent only when set (explicit false is sent).
type ListOpts struct {
	ZoneID      string
	Message     string
	Status      string
	Limit       int
	Marker      string
	SortKey     string
	SortDir     string
	AllProjects *bool
	SudoProject string
}

func (o ListOpts) values() url.Values {
	q := url.Values{}
	for key, value := range map[string]string{"zone_id": o.ZoneID, "message": o.Message, "status": o.Status, "marker": o.Marker, "sort_key": o.SortKey, "sort_dir": o.SortDir} {
		if value != "" {
			q.Set(key, value)
		}
	}
	if o.Limit != 0 {
		q.Set("limit", strconv.Itoa(o.Limit))
	}
	return q
}

type ListOption = request.Option[ListOpts]

func WithListOptions(value ListOpts) ListOption  { return request.WithOptions(value) }
func WithListQuery(key, value string) ListOption { return request.WithQuery[ListOpts](key, value) }

// API owns one authenticated Designate client.
type API struct {
	Resources *resource.Collection[ZoneExport]
	client    *gophercloud.ServiceClient
	route     designate.Route[ZoneExport, *ZoneExport]
}

func New(client *gophercloud.ServiceClient) *API {
	a := &API{client: client, route: designate.Route[ZoneExport, *ZoneExport]{Client: client, Segments: []string{"zones", "tasks", "exports"}, PluralKey: "exports"}}
	a.Resources = resource.NewCollection(resource.Adapter[ZoneExport]{
		Kind:   "zone exports",
		Get:    func(ctx context.Context, id string) (*ZoneExport, error) { return a.Get(ctx, id) },
		ID:     func(v *ZoneExport) string { return v.ID },
		Status: func(v *ZoneExport) string { return v.Status },
		Failed: func(status string) bool { return strings.EqualFold(status, "ERROR") },
		Delete: func(ctx context.Context, id string) error { _, err := a.Delete(ctx, id); return err },
		IterateControlled: func(ctx context.Context, q url.Values, control resource.ListControl) iter.Seq2[*ZoneExport, error] {
			return a.route.List(ctx, q, nil, control)
		},
	})
	return a
}

func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

// Create starts an export with POST /zones/{zone}/tasks/export and no request
// body. Only 202 is accepted; the returned task is an acknowledgement, not a
// finished export. Wait with WaitFor(..., "COMPLETE").
func (a *API) Create(ctx context.Context, zoneID string) (*ZoneExport, error) {
	if err := designate.ValidateID(zoneID); err != nil {
		return nil, request.Wrap("Create", "zone exports", err)
	}
	if err := designate.Client(ctx, a.client); err != nil {
		return nil, request.Wrap("Create", "zone exports", err)
	}
	response, err := designate.Do(ctx, a.client, http.MethodPost, a.client.ServiceURL("zones", url.PathEscape(zoneID), "tasks", "export"), nil, nil, http.StatusAccepted)
	if err != nil {
		return nil, request.Wrap("Create", "zone exports", designate.NotFound("zone", zoneID, err))
	}
	value, err := designate.Decode[ZoneExport](response)
	return value, request.Wrap("Create", "zone exports", err)
}

// Get sends GET /zones/tasks/exports/{id}, accepting only 200.
func (a *API) Get(ctx context.Context, id string) (*ZoneExport, error) {
	value, err := a.route.Get(ctx, id, nil)
	return value, request.Wrap("Get", "zone exports", designate.NotFound("zone export", id, err))
}

// GetText sends GET /zones/tasks/exports/{id}/export with Accept: text/dns and
// returns the zone file text. Only 200 is accepted.
func (a *API) GetText(ctx context.Context, id string) (*Text, error) {
	wrap := func(err error) error {
		return request.Wrap("GetText", "zone exports", designate.NotFound("zone export", id, err))
	}
	if err := designate.ValidateID(id); err != nil {
		return nil, wrap(err)
	}
	response, err := designate.Do(ctx, a.client, http.MethodGet, a.client.ServiceURL("zones", "tasks", "exports", url.PathEscape(id), "export"), nil, map[string]string{"Accept": "text/dns"}, http.StatusOK)
	if err != nil {
		return nil, wrap(err)
	}
	return &Text{Zone: string(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}, nil
}

// Delete sends DELETE /zones/tasks/exports/{id}, accepting only 204. A missing
// export is an error here; Remove ignores it by default.
func (a *API) Delete(ctx context.Context, id string) (*Acknowledgement, error) {
	ack, err := a.route.Delete(ctx, id, nil, http.StatusNoContent)
	return ack, request.Wrap("Delete", "zone exports", designate.NotFound("zone export", id, err))
}

// List streams exports across pages.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*ZoneExport, error] {
	cfg, err := request.Apply(ListOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(cfg, false, true, false)
	}
	if err != nil {
		err = request.Wrap("List", "zone exports", err)
		return func(yield func(*ZoneExport, error) bool) { yield(nil, err) }
	}
	headers := designate.Headers(cfg.Options.AllProjects, cfg.Options.SudoProject)
	return a.route.List(ctx, designate.Query(cfg.Options.values(), cfg.Query), headers, resource.ListControl{})
}

// Package zoneimports binds Designate zone import tasks. Gophercloud v2.15.0
// has no binding, so the SDK owns the documented routes: POST
// /zones/tasks/imports (text/dns) and /zones/tasks/imports[/{id}].
package zoneimports

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

// ZoneImport is a decoded import task. Body retains the exact JSON object.
type ZoneImport struct {
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
	Resources *resource.Collection[ZoneImport]
	client    *gophercloud.ServiceClient
	route     designate.Route[ZoneImport, *ZoneImport]
}

func New(client *gophercloud.ServiceClient) *API {
	a := &API{client: client, route: designate.Route[ZoneImport, *ZoneImport]{Client: client, Segments: []string{"zones", "tasks", "imports"}, PluralKey: "imports"}}
	a.Resources = resource.NewCollection(resource.Adapter[ZoneImport]{
		Kind:   "zone imports",
		Get:    func(ctx context.Context, id string) (*ZoneImport, error) { return a.Get(ctx, id) },
		ID:     func(v *ZoneImport) string { return v.ID },
		Status: func(v *ZoneImport) string { return v.Status },
		Failed: func(status string) bool { return strings.EqualFold(status, "ERROR") },
		Delete: func(ctx context.Context, id string) error { _, err := a.Delete(ctx, id); return err },
		IterateControlled: func(ctx context.Context, q url.Values, control resource.ListControl) iter.Seq2[*ZoneImport, error] {
			return a.route.List(ctx, q, nil, control)
		},
	})
	return a
}

func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

// Create imports a zone file with POST /zones/tasks/imports, Content-Type
// text/dns and the file as the raw body. Only 202 is accepted; the returned
// task is an acknowledgement, not a finished import. Wait with
// WaitFor(..., "COMPLETE"). An empty zoneFile is sent as an empty body and
// left for the server to reject.
func (a *API) Create(ctx context.Context, zoneFile string) (*ZoneImport, error) {
	response, err := designate.DoRaw(ctx, a.client, http.MethodPost, a.client.ServiceURL("zones", "tasks", "imports"), []byte(zoneFile), map[string]string{"Content-Type": "text/dns"}, http.StatusAccepted)
	if err != nil {
		return nil, request.Wrap("Create", "zone imports", err)
	}
	value, err := designate.Decode[ZoneImport](response)
	return value, request.Wrap("Create", "zone imports", err)
}

// Get sends GET /zones/tasks/imports/{id}, accepting only 200.
func (a *API) Get(ctx context.Context, id string) (*ZoneImport, error) {
	value, err := a.route.Get(ctx, id, nil)
	return value, request.Wrap("Get", "zone imports", designate.NotFound("zone import", id, err))
}

// Delete sends DELETE /zones/tasks/imports/{id}, accepting only 204. A missing
// import is an error here; Remove ignores it by default.
func (a *API) Delete(ctx context.Context, id string) (*Acknowledgement, error) {
	ack, err := a.route.Delete(ctx, id, nil, http.StatusNoContent)
	return ack, request.Wrap("Delete", "zone imports", designate.NotFound("zone import", id, err))
}

// List streams imports across pages.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*ZoneImport, error] {
	cfg, err := request.Apply(ListOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(cfg, false, true, false)
	}
	if err != nil {
		err = request.Wrap("List", "zone imports", err)
		return func(yield func(*ZoneImport, error) bool) { yield(nil, err) }
	}
	headers := designate.Headers(cfg.Options.AllProjects, cfg.Options.SudoProject)
	return a.route.List(ctx, designate.Query(cfg.Options.values(), cfg.Query), headers, resource.ListControl{})
}

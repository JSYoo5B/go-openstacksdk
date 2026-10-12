// Package clusters implements the pinned openstacksdk block_storage v3
// cluster proxy calls over Cinder clusters, a route Gophercloud v2.15.0 does
// not bind.
package clusters

import (
	"context"
	"encoding/json"
	"iter"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/internal/cinderowned"
	"github.com/gophercloud/gophercloud/v2"
)

const collectionPath = "clusters"

var spec = &cinderowned.Spec{
	Kind: "clusters", ResourceKey: "cluster", ResourcesKey: "clusters", IDKey: "name",
	Fields: []string{"id", "name", "active_backend_id", "binary", "created_at", "disabled_reason", "frozen",
		"last_heartbeat", "num_down_hosts", "num_hosts", "replication_status", "state", "status", "updated_at"},
	Query: map[string]string{"name": "name", "binary": "binary", "is_up": "is_up", "disabled": "disabled",
		"num_hosts": "num_hosts", "num_down_hosts": "num_down_hosts", "replication_status": "replication_status",
		"frozen": "frozen", "active_backend_id": "active_backend_id"},
	Filters: map[string]string{"id": "id", "created_at": "created_at", "disabled_reason": "disabled_reason",
		"last_heartbeat": "last_heartbeat", "state": "state", "status": "status", "updated_at": "updated_at"},
}

// version is Cluster._get_microversion for fetch and list: the configured
// microversion, or min(3.26, server maximum) from discovery.
var version = cinderowned.Version{Ceiling: "3.26"}

// actionVersion is Cluster._action: session.default_microversion, that is the
// configured client microversion without discovery.
var actionVersion = cinderowned.Version{Default: true}

// API shares the selected authenticated Cinder client without owning it.
type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }
func (a *API) RawClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}

func (a *API) target() cinderowned.Target {
	var client func() *gophercloud.ServiceClient
	if a != nil {
		client = func() *gophercloud.ServiceClient { return a.client }
	}
	return cinderowned.Target{Spec: spec, Client: client, Path: collectionPath}
}

func readHeaders(v GetOpts) map[string]string { return v.Headers }

// Get implements get_cluster: GET clusters/{name}.
func (a *API) Get(ctx context.Context, name string, options ...GetOption) (*Cluster, error) {
	const op = "Get"
	t := a.target()
	if err := cinderowned.ValidateID("cluster name", name); err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	p, _, err := cinderowned.Begin(ctx, t, GetOpts{}, cinderowned.CopyRead, readHeaders, version, options)
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	return convert(t.Wrapper(ctx, op)(p.Get(name, nil)))
}

// List implements clusters(details=True, **query) as a lazy iterator over GET
// clusters/detail, or clusters with WithListDetails(false).
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*Cluster, error] {
	return func(yield func(*Cluster, error) bool) {
		const op = "List"
		t := a.target()
		p, v, err := cinderowned.Begin(ctx, t, ListOpts{}, copyList, func(v ListOpts) map[string]string { return v.Headers }, version, options)
		if err != nil {
			yield(nil, t.Wrap(ctx, op, err))
			return
		}
		path := collectionPath + "/detail"
		if v.Details != nil && !*v.Details {
			path = collectionPath
		}
		for record, err := range p.ListOptions(path, v.ListOpts) {
			if !yield(fromRecord(record), t.Wrap(ctx, op, err)) || err != nil {
				return
			}
		}
	}
}

// All collects List.
func (a *API) All(ctx context.Context, options ...ListOption) ([]*Cluster, error) {
	var values []*Cluster
	for value, err := range a.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// Find implements find_cluster: GET clusters/{name_or_id} and, after a clean
// 400, 403 or 404, the complete clusters/detail?name={name_or_id} list
// matched by exact ID or name. A missing cluster returns nil, nil unless
// WithFindIgnoreMissing(false) is given.
func (a *API) Find(ctx context.Context, nameOrID string, options ...FindOption) (*Cluster, error) {
	const op = "Find"
	t := a.target()
	p, v, err := cinderowned.Begin(ctx, t, FindOpts{}, cinderowned.CopyFind, func(FindOpts) map[string]string { return nil }, version, options)
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	listQuery := url.Values{"name": {nameOrID}}
	return convert(t.Wrapper(ctx, op)(p.Find(ctx, nameOrID, nil, collectionPath+"/detail", listQuery, v.IgnoreMissing == nil || *v.IgnoreMissing)))
}

// Enable implements enable_cluster: PUT clusters/enable with {"name": name}
// plus binary when it is nonempty, at the configured client microversion.
func (a *API) Enable(ctx context.Context, cluster Target, options ...EnableOption) (*Cluster, error) {
	return action(ctx, a, "Enable", "enable", cluster, options)
}

// Disable implements disable_cluster: PUT clusters/disable with
// {"name": name}, binary when nonempty and disabled_reason when the reason is
// nonempty, at the configured client microversion.
func (a *API) Disable(ctx context.Context, cluster Target, options ...DisableOption) (*Cluster, error) {
	return action(ctx, a, "Disable", "disable", cluster, options)
}

func action[O ~func(*ActionOpts) error](ctx context.Context, a *API, op, action string, cluster Target, options []O) (*Cluster, error) {
	t := a.target()
	p, v, err := cinderowned.Begin(ctx, t, ActionOpts{}, copyAction, func(v ActionOpts) map[string]string { return v.Headers }, actionVersion, options)
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	body := map[string]json.RawMessage{"name": json.RawMessage("null")}
	seed := map[string]json.RawMessage{}
	if cluster.Name != nil {
		body["name"], _ = json.Marshal(*cluster.Name)
		seed["name"] = body["name"]
	}
	if cluster.Binary != nil && *cluster.Binary != "" {
		body["binary"], _ = json.Marshal(*cluster.Binary)
		seed["binary"] = body["binary"]
	}
	if action == "disable" && v.Reason != "" {
		body["disabled_reason"], _ = json.Marshal(v.Reason)
	}
	response, err := p.Do(http.MethodPut, p.ServiceURL(collectionPath, action), body)
	if err != nil {
		return convert(p.Evidence(response, err), t.Wrap(ctx, op, err))
	}
	return convert(t.Wrapper(ctx, op)(p.Record(response, seed)))
}

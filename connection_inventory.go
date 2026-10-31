package openstack

import (
	"bytes"
	"context"
	"encoding/json"
	"slices"
	"sync"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudinventory"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudserver"
)

// CloudNames returns the cloud names of the first clouds.yaml found, in file
// order, using the same search rule as Connect: the files of WithCloudFiles,
// else OS_CLIENT_CONFIG_FILE, else the working directory, the user and the
// system configuration directories. Other options are validated but unused.
// Like Python, only the first file found is read and nothing is authenticated.
func CloudNames(options ...ConnectionOption) ([]string, error) {
	o, err := parseConnection(options)
	if err != nil {
		return nil, err
	}
	return cloudinventory.CloudNames(o.cloudFiles)
}

// InventoryCloud is one member of an Inventory. Supply a ready Connection, or a
// Name (with optional Options) to authenticate lazily with WithCloud(Name) the
// first time the inventory needs it. The Name labels hosts either way.
type InventoryCloud struct {
	Name       string
	Connection *Connection
	Options    []ConnectionOption
}

type inventoryEntry struct {
	name    string
	options []ConnectionOption
	mu      sync.Mutex
	conn    *Connection
}

// Inventory lists and searches compute servers across several clouds, the Go
// counterpart of OpenStackInventory. Connections supplied by the caller stay
// owned by the caller; lazily created ones are cached for later calls.
type Inventory struct{ entries []*inventoryEntry }

// NewInventory builds an inventory without any request. Names must be non-empty
// and unique, and each member needs a Connection or a cloud Name to connect.
func NewInventory(clouds ...InventoryCloud) (*Inventory, error) {
	seen := make(map[string]bool, len(clouds))
	inventory := &Inventory{}
	for _, cloud := range clouds {
		if cloud.Name == "" {
			return nil, invalid("inventory cloud name must not be empty")
		}
		if seen[cloud.Name] {
			return nil, invalid("duplicate inventory cloud %q", cloud.Name)
		}
		seen[cloud.Name] = true
		inventory.entries = append(inventory.entries, &inventoryEntry{name: cloud.Name, conn: cloud.Connection, options: slices.Clone(cloud.Options)})
	}
	return inventory, nil
}

// NewInventoryFromCloudFiles builds an inventory of the named clouds, or of
// every cloud of the first clouds.yaml when no name is given, applying options
// (for example WithCloudFiles) to each lazy connection.
func NewInventoryFromCloudFiles(names []string, options ...ConnectionOption) (*Inventory, error) {
	if len(names) == 0 {
		var err error
		if names, err = CloudNames(options...); err != nil {
			return nil, err
		}
	}
	clouds := make([]InventoryCloud, 0, len(names))
	for _, name := range names {
		clouds = append(clouds, InventoryCloud{Name: name, Options: options})
	}
	return NewInventory(clouds...)
}

func (e *inventoryEntry) connection(ctx context.Context) (*Connection, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.conn != nil {
		return e.conn, nil
	}
	options := append(slices.Clone(e.options), WithCloud(e.name))
	conn, err := Connect(ctx, options...)
	if err != nil {
		return nil, err
	}
	e.conn = conn
	return conn, nil
}

// InventoryHost is one server and the inventory cloud that returned it.
// Hostvars is set by the default expansion (Python expand=True); Interfaces
// is set by both expansions and is nil with WithInventoryBare. Record is the
// Source-shaped Proxy.servers row (servers.API.ListCloudRecords) the typed
// Server was decoded from.
type InventoryHost struct {
	Cloud      string
	Server     *servers.Server
	Record     *servers.ServerFindRecord
	Interfaces *compute.ServerAddressView
	Hostvars   *CloudServerHostvars
}

// InventoryExpansion selects how each listed server is expanded.
type InventoryExpansion int

const (
	// InventoryExpandHostvars is Python's default expand=True:
	// list_servers(detailed=True), the GetOpenStackVars hostvars.
	InventoryExpandHostvars InventoryExpansion = iota
	// InventoryExpandInterfaces is expand=False: list_servers(detailed=False)
	// still adds the interface expansion.
	InventoryExpandInterfaces
	// InventoryExpandNone is a Go extension that skips every expansion.
	InventoryExpandNone
)

// InventoryFailure records a cloud that could not be listed.
type InventoryFailure struct {
	Cloud string
	Err   error
}

// InventoryHosts is the result of ListHosts. Hosts are ordered by cloud and
// then by the server list order of that cloud. A cloud that failed contributes
// no hosts, even if some of its pages arrived before the failure.
type InventoryHosts struct {
	Hosts    []InventoryHost
	Failures []InventoryFailure
}

// InventoryOpts are the operation options of ListHosts, SearchHosts and GetHost.
type InventoryOpts struct {
	AllProjects      bool
	ContinueOnFailed bool
	Filters          *json.RawMessage
	Expansion        InventoryExpansion
	// Private maps OpenStackInventory(private=True): every cloud computes
	// interface addresses with compute.WithPrivateCloud(true).
	Private bool
}

// InventoryOption configures an inventory operation.
type InventoryOption func(*InventoryOpts) error

// WithInventoryAllProjects lists servers of all projects (all_tenants=true).
func WithInventoryAllProjects() InventoryOption {
	return func(o *InventoryOpts) error { o.AllProjects = true; return nil }
}

// WithInventoryExpand maps the expand argument: true (the default) builds
// hostvars, false builds only the interface expansion.
func WithInventoryExpand(expand bool) InventoryOption {
	return func(o *InventoryOpts) error {
		o.Expansion = InventoryExpandInterfaces
		if expand {
			o.Expansion = InventoryExpandHostvars
		}
		return nil
	}
}

// WithInventoryBare skips every per-server expansion (a Go extension).
func WithInventoryBare() InventoryOption {
	return func(o *InventoryOpts) error { o.Expansion = InventoryExpandNone; return nil }
}

// WithInventoryPrivate maps the private constructor flag for this call.
func WithInventoryPrivate(private bool) InventoryOption {
	return func(o *InventoryOpts) error { o.Private = private; return nil }
}

// WithInventoryContinueOnCloudError keeps going when one cloud cannot be
// connected or listed; the failure is recorded instead of returned. Without it
// the first failing cloud aborts the call (fail_on_cloud_config=True). Context
// cancellation and deadlines always abort.
func WithInventoryContinueOnCloudError() InventoryOption {
	return func(o *InventoryOpts) error { o.ContinueOnFailed = true; return nil }
}

// WithInventoryFilters supplies the local filters of SearchHosts and GetHost:
// a JSON object of exact (nested) matches or a JSON string JMESPath expression.
func WithInventoryFilters(filters json.RawMessage) InventoryOption {
	owned := bytes.Clone(filters)
	return func(o *InventoryOpts) error {
		o.Filters = nil
		if owned != nil {
			value := json.RawMessage(bytes.Clone(owned))
			o.Filters = &value
		}
		return nil
	}
}

func applyInventoryOptions(options []InventoryOption) (InventoryOpts, error) {
	var opts InventoryOpts
	for _, apply := range options {
		if apply == nil {
			return opts, invalid("nil inventory option")
		}
		if err := apply(&opts); err != nil {
			return opts, err
		}
	}
	return opts, nil
}

// dependencies exposes the inventory clouds to the cloudinventory helpers;
// Servers connects a lazy cloud on first use.
func (i *Inventory) dependencies() cloudinventory.Dependencies {
	names := make([]string, len(i.entries))
	for index, entry := range i.entries {
		names[index] = entry.name
	}
	return cloudinventory.Dependencies{
		Clouds: names,
		Servers: func(ctx context.Context, index int) (cloudserver.Dependencies, error) {
			conn, err := i.entries[index].connection(ctx)
			if err != nil {
				return cloudserver.Dependencies{}, err
			}
			return conn.cloudServerDependencies(), nil
		},
	}
}

func (o InventoryOpts) internal() cloudinventory.Opts {
	return cloudinventory.Opts{AllProjects: o.AllProjects, ContinueOnFailed: o.ContinueOnFailed, Filters: o.Filters, Expansion: cloudinventory.Expansion(o.Expansion), Private: o.Private}
}

func inventoryHosts(hosts []cloudinventory.Host) []InventoryHost {
	if hosts == nil {
		return nil
	}
	converted := make([]InventoryHost, len(hosts))
	for index, host := range hosts {
		converted[index] = InventoryHost(host)
	}
	return converted
}

func inventoryFailures(failures []cloudinventory.Failure) []InventoryFailure {
	if failures == nil {
		return nil
	}
	converted := make([]InventoryFailure, len(failures))
	for index, failure := range failures {
		converted[index] = InventoryFailure(failure)
	}
	return converted
}

// ListHosts lists the detailed servers of every cloud in order. Each server
// list is requested once and failures are never replayed. On a failure without
// WithInventoryContinueOnCloudError the hosts of earlier clouds are returned
// together with the error. Each server is then expanded as selected by
// WithInventoryExpand (hostvars by default) unless WithInventoryBare is set;
// an expansion failure fails that cloud like a listing failure.
func (i *Inventory) ListHosts(ctx context.Context, options ...InventoryOption) (*InventoryHosts, error) {
	opts, err := applyInventoryOptions(options)
	if err != nil {
		return nil, err
	}
	if i == nil {
		return nil, invalid("inventory is required")
	}
	listed, err := cloudinventory.ListHosts(ctx, i.dependencies(), opts.internal())
	return &InventoryHosts{Hosts: inventoryHosts(listed.Hosts), Failures: inventoryFailures(listed.Failures)}, err
}

// InventorySearch is the result of SearchHosts. Expression is true when a
// string filter produced arbitrary JSON in Value; Hosts is then empty.
type InventorySearch struct {
	Value      json.RawMessage
	Expression bool
	Hosts      []InventoryHost
	Failures   []InventoryFailure
}

// SearchHosts lists all hosts (with the failure policy of ListHosts) and
// applies the cloud identifier selection: exact or fnmatch-style glob on the
// server ID or name, then the optional filters over InventoryHostView. Filter
// keys are the JSON field names of the compute server record plus the
// expansion keys.
func (i *Inventory) SearchHosts(ctx context.Context, nameOrID string, options ...InventoryOption) (*InventorySearch, error) {
	opts, err := applyInventoryOptions(options)
	if err != nil {
		return nil, err
	}
	if i == nil {
		return nil, invalid("inventory is required")
	}
	found, err := cloudinventory.SearchHosts(ctx, i.dependencies(), nameOrID, opts.internal())
	if err != nil {
		return nil, err
	}
	return &InventorySearch{Value: found.Value, Expression: found.Expression, Hosts: inventoryHosts(found.Hosts), Failures: inventoryFailures(found.Failures)}, nil
}

// InventoryHostView is the JSON row SearchHosts filters: the hostvars
// mapping when expanded with hostvars, else the server JSON with the
// interface fields set over it, else the bare server JSON.
func InventoryHostView(host InventoryHost) (json.RawMessage, error) {
	return cloudinventory.HostView(cloudinventory.Host(host))
}

// GetHost returns the single host selected by SearchHosts, nil when nothing
// matches, and a resource.AmbiguousError when several do. A string filter that
// yields an expression value instead of rows is rejected.
func (i *Inventory) GetHost(ctx context.Context, nameOrID string, options ...InventoryOption) (*InventoryHost, error) {
	opts, err := applyInventoryOptions(options)
	if err != nil {
		return nil, err
	}
	if i == nil {
		return nil, invalid("inventory is required")
	}
	host, err := cloudinventory.GetHost(ctx, i.dependencies(), nameOrID, opts.internal())
	if host == nil || err != nil {
		return nil, err
	}
	converted := InventoryHost(*host)
	return &converted, nil
}

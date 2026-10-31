// Package cloudinventory implements the helpers of the root Inventory, the Go
// counterpart of OpenStackInventory: the clouds.yaml cloud name listing, the
// per-cloud server listing with its expansion and failure policy, the host
// search over the host JSON view, and the single-host ambiguity rule. The root
// keeps the Inventory type and its lazy Connections and supplies each cloud's
// server read dependencies through Dependencies, so this package never imports
// the root.
package cloudinventory

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudconfig"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudserver"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"gopkg.in/yaml.v2"
)

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

// CloudNames returns the cloud names of the first clouds.yaml found, in file
// order. cloudFiles are the WithCloudFiles locations; when empty the search
// falls back to OS_CLIENT_CONFIG_FILE, else the standard search paths.
func CloudNames(cloudFiles []string) ([]string, error) {
	locations := cloudFiles
	if len(locations) == 0 {
		if path := os.Getenv("OS_CLIENT_CONFIG_FILE"); path != "" {
			locations = []string{path}
		} else {
			var err error
			if locations, err = cloudconfig.SearchPaths("clouds.yaml"); err != nil {
				return nil, err
			}
		}
	}
	data, _, err := cloudconfig.ReadFirstFile(locations)
	if err != nil {
		return nil, err
	}
	var document struct {
		Clouds yaml.MapSlice `yaml:"clouds"`
	}
	if err := yaml.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("parse clouds.yaml: %w", err)
	}
	names := make([]string, 0, len(document.Clouds))
	for _, item := range document.Clouds {
		name, ok := item.Key.(string)
		if !ok {
			return nil, invalid("cloud name %v is not a string", item.Key)
		}
		names = append(names, name)
	}
	return names, nil
}

// Expansion mirrors the root InventoryExpansion; the values must stay in the
// same order.
type Expansion int

const (
	// ExpandHostvars builds the hostvars (Python expand=True).
	ExpandHostvars Expansion = iota
	// ExpandInterfaces builds only the interface expansion (expand=False).
	ExpandInterfaces
	// ExpandNone skips every expansion.
	ExpandNone
)

// Opts mirror the root InventoryOpts.
type Opts struct {
	AllProjects      bool
	ContinueOnFailed bool
	Filters          *json.RawMessage
	Expansion        Expansion
	Private          bool
}

// Host has the field set of the root InventoryHost so the two convert.
type Host struct {
	Cloud      string
	Server     *servers.Server
	Record     *servers.ServerFindRecord
	Interfaces *compute.ServerAddressView
	Hostvars   *compute.CloudServerHostvars
}

// Failure has the field set of the root InventoryFailure so the two convert.
type Failure struct {
	Cloud string
	Err   error
}

// Hosts is the result of ListHosts.
type Hosts struct {
	Hosts    []Host
	Failures []Failure
}

// Search is the result of SearchHosts.
type Search struct {
	Value      json.RawMessage
	Expression bool
	Hosts      []Host
	Failures   []Failure
}

// Dependencies are the Inventory facilities the helpers use.
type Dependencies struct {
	// Clouds are the inventory cloud names in inventory order.
	Clouds []string
	// Servers returns the cloud server read dependencies of Clouds[index],
	// connecting lazily when the cloud has no Connection yet.
	Servers func(ctx context.Context, index int) (cloudserver.Dependencies, error)
}

func listCloud(ctx context.Context, deps Dependencies, index int, opts Opts) ([]Host, error) {
	serverDeps, err := deps.Servers(ctx, index)
	if err != nil {
		return nil, err
	}
	// Python list_hosts calls cloud.list_servers(detailed=expand,
	// all_projects=all_projects), which is Proxy.servers(all_projects=...)
	// over GET /servers/detail followed by _expand_server for every row.
	cfg := compute.CloudServerReadOpts{
		List:     []servers.RecordFindOption{servers.WithRecordFindAllProjects(opts.AllProjects)},
		Bare:     opts.Expansion == ExpandNone,
		Detailed: opts.Expansion == ExpandHostvars,
	}
	if opts.Private {
		cfg.Address = append(cfg.Address, compute.WithPrivateCloud(true))
	}
	results, err := cloudserver.ListExpanded(ctx, serverDeps, cfg)
	if err != nil {
		return nil, err
	}
	hosts := make([]Host, len(results))
	for i, result := range results {
		hosts[i] = Host{Cloud: deps.Clouds[index], Server: result.Server, Record: result.Record, Interfaces: result.Interfaces, Hostvars: result.Hostvars}
	}
	return hosts, nil
}

// ListHosts lists every cloud in order. Without ContinueOnFailed the first
// failing cloud returns the hosts of the earlier clouds with the error; with
// it the failure is recorded. Context cancellation always aborts.
func ListHosts(ctx context.Context, deps Dependencies, opts Opts) (*Hosts, error) {
	result := &Hosts{}
	for index, name := range deps.Clouds {
		if err := ctx.Err(); err != nil {
			return result, fmt.Errorf("list hosts: %w", err)
		}
		hosts, err := listCloud(ctx, deps, index, opts)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return result, fmt.Errorf("list hosts of %s: %w", name, errors.Join(err, ctxErr))
			}
			if !opts.ContinueOnFailed {
				return result, fmt.Errorf("list hosts of %s: %w", name, err)
			}
			result.Failures = append(result.Failures, Failure{Cloud: name, Err: err})
			continue
		}
		result.Hosts = append(result.Hosts, hosts...)
	}
	return result, nil
}

// SearchHosts lists all hosts and applies the cloud identifier selection and
// the optional filters over HostView.
func SearchHosts(ctx context.Context, deps Dependencies, nameOrID string, opts Opts) (*Search, error) {
	listed, err := ListHosts(ctx, deps, opts)
	if err != nil {
		return nil, err
	}
	rows := make([]json.RawMessage, len(listed.Hosts))
	for index, host := range listed.Hosts {
		if rows[index], err = HostView(host); err != nil {
			return nil, fmt.Errorf("project server row: %w", err)
		}
	}
	selected, err := cloudfilter.Select(rows, nameOrID, opts.Filters, nil)
	if err != nil {
		return nil, errors.Join(invalid("local host search"), err)
	}
	result := &Search{Value: bytes.Clone(selected.Value), Expression: selected.Expression, Failures: listed.Failures}
	if !selected.Expression {
		for _, index := range selected.Indices {
			result.Hosts = append(result.Hosts, listed.Hosts[index])
		}
	}
	return result, nil
}

// HostView is the JSON row SearchHosts filters: the hostvars mapping when
// expanded with hostvars, else the server JSON with the interface fields set
// over it, else the bare server JSON.
func HostView(host Host) (json.RawMessage, error) {
	if host.Hostvars != nil {
		return json.Marshal(host.Hostvars)
	}
	if host.Interfaces == nil {
		return json.Marshal(host.Server)
	}
	return compute.CloudServerResultView(&compute.CloudServerResult{Server: host.Server, Interfaces: host.Interfaces})
}

// GetHost returns the single host selected by SearchHosts, nil when nothing
// matches, and a resource.AmbiguousError when several do.
func GetHost(ctx context.Context, deps Dependencies, nameOrID string, opts Opts) (*Host, error) {
	found, err := SearchHosts(ctx, deps, nameOrID, opts)
	if err != nil {
		return nil, err
	}
	if found.Expression {
		return nil, invalid("a JMESPath filter result is not a host record")
	}
	switch len(found.Hosts) {
	case 0:
		return nil, nil
	case 1:
		return &found.Hosts[0], nil
	}
	ids := make([]string, len(found.Hosts))
	for index, host := range found.Hosts {
		ids[index] = host.Server.ID
	}
	return nil, &resource.AmbiguousError{Resource: "hosts", Name: nameOrID, IDs: ids}
}

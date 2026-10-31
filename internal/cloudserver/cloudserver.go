// Package cloudserver implements the cloud-layer server reads and actions of
// Connection (openstack/cloud/_compute.py list/search/get/update/delete/
// rebuild_server) over the compute result and option types. Connection
// supplies its guard, lazy Nova service, location, expansions and floating-IP
// helpers through Dependencies, so this package never imports the root.
package cloudserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	computeapi "github.com/JSYoo5B/go-openstacksdk/compute/v2"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/serverview"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

// Dependencies are the Connection facilities the server reads and actions use.
type Dependencies struct {
	// Guard checks the context and that the Connection is usable before any
	// service is resolved.
	Guard func(context.Context) error
	// Compute is the lazy Nova service.
	Compute func(context.Context) (*computeapi.Service, error)
	// Location is the current cloud, region and project scope.
	Location func() (resource.CloudLocation, error)
	// ExpandInterfaces is the add_server_interfaces expansion.
	ExpandInterfaces func(context.Context, *servers.Server, ...compute.ServerAddressOption) (*compute.ServerAddressView, error)
	// Hostvars is the detailed get_hostvars_from_server expansion; the
	// record, when not nil, describes the same server snapshot.
	Hostvars func(context.Context, *servers.Server, *servers.ServerFindRecord, []compute.ServerAddressOption) (*compute.CloudServerHostvars, error)
	// FloatingIPSource is the configured floating_ip_source.
	FloatingIPSource func() compute.FloatingIPSource
	// GetFloatingIP and DeleteFloatingIP are the Connection floating-IP
	// helpers used by the delete_ips cleanup.
	GetFloatingIP    func(context.Context, compute.GetFloatingIPRequest, ...compute.FloatingIPQueryOption) (*compute.GetFloatingIPResult, error)
	DeleteFloatingIP func(context.Context, compute.DeleteFloatingIPRequest, ...compute.FloatingIPDeleteOption) (*compute.DeleteFloatingIPResult, error)
}

// ApplyReadOptions applies the read options in order and validates the filters.
func ApplyReadOptions(options []compute.CloudServerReadOption) (compute.CloudServerReadOpts, error) {
	var cfg compute.CloudServerReadOpts
	for _, apply := range slices.Clone(options) {
		if apply == nil {
			return cfg, fmt.Errorf("%w: nil cloud server option", resource.ErrInvalidOption)
		}
		if err := apply(&cfg); err != nil {
			return cfg, err
		}
	}
	if cfg.Filters != nil && !json.Valid(*cfg.Filters) {
		return cfg, fmt.Errorf("%w: cloud server filters require JSON", resource.ErrInvalidOption)
	}
	return cfg, nil
}

// ApplyActionOptions applies the action options over the 180-second default
// and validates the wait budget.
func ApplyActionOptions(options []compute.CloudServerActionOption) (compute.CloudServerActionOpts, error) {
	cfg := compute.CloudServerActionOpts{Timeout: 180 * time.Second}
	for _, apply := range slices.Clone(options) {
		if apply == nil {
			return cfg, fmt.Errorf("%w: nil cloud server option", resource.ErrInvalidOption)
		}
		if err := apply(&cfg); err != nil {
			return cfg, err
		}
	}
	if !cfg.Unlimited && cfg.Timeout <= 0 {
		return cfg, fmt.Errorf("%w: server wait timeout must be positive", resource.ErrInvalidOption)
	}
	return cfg, nil
}

func waitOptions(cfg compute.CloudServerActionOpts) []resource.WaitOption {
	first := resource.WithTimeout(cfg.Timeout)
	if cfg.Unlimited {
		first = resource.WithUnlimitedWait()
	}
	return append([]resource.WaitOption{first}, cfg.WaitOptions...)
}

// RecordString returns the string value of key in the record's Resource
// view, or "" when it is absent or not a string.
func RecordString(record *servers.ServerFindRecord, key string) string {
	if record == nil || record.Resource == nil {
		return ""
	}
	var text string
	_ = json.Unmarshal(record.Resource.Body[key], &text)
	return text
}

func list(ctx context.Context, deps Dependencies, cfg compute.CloudServerReadOpts) ([]*servers.ServerFindRecord, *servers.API, error) {
	if err := deps.Guard(ctx); err != nil {
		return nil, nil, err
	}
	service, err := deps.Compute(ctx)
	if err != nil {
		return nil, nil, err
	}
	location, err := deps.Location()
	if err != nil {
		return nil, nil, err
	}
	options := append([]servers.RecordFindOption{servers.WithRecordFindLocation(location)}, cfg.List...)
	var rows []*servers.ServerFindRecord
	for row, err := range service.Servers.ListCloudRecords(ctx, options...) {
		if err != nil {
			return rows, service.Servers, err
		}
		rows = append(rows, row)
	}
	return rows, service.Servers, nil
}

// ListRecords implements Connection.ListServerRecords.
func ListRecords(ctx context.Context, deps Dependencies, options []compute.CloudServerReadOption) ([]*servers.ServerFindRecord, error) {
	wrap := func(err error) error {
		return request.Wrap("ListServerRecords", "compute", cloudread.ContextError(ctx, err))
	}
	cfg, err := ApplyReadOptions(options)
	if err != nil {
		return nil, wrap(err)
	}
	rows, _, err := list(ctx, deps, cfg)
	return rows, wrap(err)
}

// SearchRecords implements Connection.SearchServerRecords.
func SearchRecords(ctx context.Context, deps Dependencies, nameOrID string, options []compute.CloudServerReadOption) (*compute.CloudServerSearchResult, error) {
	wrap := func(err error) error {
		return request.Wrap("SearchServerRecords", "compute", cloudread.ContextError(ctx, err))
	}
	cfg, err := ApplyReadOptions(options)
	if err != nil {
		return nil, wrap(err)
	}
	rows, _, err := list(ctx, deps, cfg)
	result := &compute.CloudServerSearchResult{Inventory: rows}
	if err != nil {
		return result, wrap(err)
	}
	views := make([]json.RawMessage, len(rows))
	for i, row := range rows {
		if row == nil || row.Resource == nil {
			return result, wrap(fmt.Errorf("%w: server list yielded no Resource", resource.ErrInvalidOption))
		}
		if views[i], err = json.Marshal(row.Resource); err != nil {
			return result, wrap(err)
		}
	}
	selected, err := cloudfilter.SelectSource(views, nameOrID, cfg.Filters, func() error { return cloudread.Context(ctx) })
	if err != nil {
		return result, wrap(fmt.Errorf("%w: local server filters: %w", resource.ErrInvalidOption, err))
	}
	result.Value, result.Expression = bytes.Clone(selected.Value), selected.Expression
	if !selected.Expression {
		result.Records = make([]*servers.ServerFindRecord, len(selected.Indices))
		for i, index := range selected.Indices {
			result.Records[i] = rows[index]
		}
	}
	return result, wrap(cloudread.Context(ctx))
}

func expand(ctx context.Context, deps Dependencies, result *compute.CloudServerResult, cfg compute.CloudServerReadOpts) error {
	return expandFrom(ctx, deps, result, cfg, nil)
}

// expandFrom expands result.Server; base is the record whose Resource view
// describes that same server snapshot, or nil.
func expandFrom(ctx context.Context, deps Dependencies, result *compute.CloudServerResult, cfg compute.CloudServerReadOpts, base *servers.ServerFindRecord) error {
	if cfg.Bare || result == nil || result.Server == nil {
		return nil
	}
	if cfg.Detailed {
		hostvars, err := deps.Hostvars(ctx, result.Server, base, cfg.Address)
		result.Hostvars = hostvars
		if hostvars != nil {
			result.Interfaces = hostvars.Interfaces
		}
		return err
	}
	view, err := deps.ExpandInterfaces(ctx, result.Server, cfg.Address...)
	result.Interfaces = view
	return err
}

// GetByID implements Connection.GetServerByID.
func GetByID(ctx context.Context, deps Dependencies, id string, options []compute.CloudServerReadOption) (*compute.CloudServerResult, error) {
	wrap := func(err error) error {
		return request.Wrap("GetServerByID", "compute", cloudread.ContextError(ctx, err))
	}
	cfg, err := ApplyReadOptions(options)
	if err != nil {
		return nil, wrap(err)
	}
	if cfg.Detailed {
		return nil, wrap(fmt.Errorf("%w: get_server_by_id has no detailed expansion", resource.ErrInvalidOption))
	}
	if err := deps.Guard(ctx); err != nil {
		return nil, wrap(err)
	}
	service, err := deps.Compute(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	server, err := service.Servers.Get(ctx, id)
	if err != nil {
		if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			return nil, nil
		}
		return nil, wrap(err)
	}
	result := &compute.CloudServerResult{Server: server, AdminPassword: server.AdminPass}
	return result, wrap(expand(ctx, deps, result, cfg))
}

// Update implements Connection.UpdateServer.
func Update(ctx context.Context, deps Dependencies, nameOrID string, update compute.CloudServerUpdate, options []compute.CloudServerReadOption) (*compute.CloudServerResult, error) {
	wrap := func(err error) error {
		return request.Wrap("UpdateServer", "compute", cloudread.ContextError(ctx, err))
	}
	cfg, err := ApplyReadOptions(options)
	if err != nil {
		return nil, wrap(err)
	}
	var base servers.UpdateOpts
	var updateOptions []servers.UpdateOption
	if update.Name.IsSet() {
		name, ok := update.Name.Get()
		if !ok || name == "" {
			return nil, wrap(fmt.Errorf("%w: server name must be a nonempty string", resource.ErrInvalidOption))
		}
		base.Name = name
	}
	if update.Description.IsSet() {
		if text, ok := update.Description.Get(); ok {
			updateOptions = append(updateOptions, servers.WithUpdateField("description", text))
		} else {
			updateOptions = append(updateOptions, servers.WithUpdateField("description", nil))
		}
	}
	if err := deps.Guard(ctx); err != nil {
		return nil, wrap(err)
	}
	service, err := deps.Compute(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	location, err := deps.Location()
	if err != nil {
		return nil, wrap(err)
	}
	found, err := service.Servers.FindRecord(ctx, nameOrID, servers.WithRecordFindIgnoreMissing(false), servers.WithRecordFindLocation(location))
	if err != nil {
		return nil, wrap(err)
	}
	if found == nil || found.Resource == nil {
		return nil, wrap(fmt.Errorf("%w: server %q", resource.ErrNotFound, nameOrID))
	}
	result := &compute.CloudServerResult{Record: found}
	if !update.Name.IsSet() && !update.Description.IsSet() {
		return result, nil
	}
	var id string
	if err := json.Unmarshal(found.Resource.Body["id"], &id); err != nil || id == "" {
		return result, wrap(fmt.Errorf("%w: found server has no string ID", resource.ErrInvalidOption))
	}
	result.Sent = true
	result.Server, err = service.Servers.Update(ctx, id, base, updateOptions...)
	if err != nil {
		return result, wrap(err)
	}
	return result, wrap(expand(ctx, deps, result, cfg))
}

// Get implements Connection.GetServer.
func Get(ctx context.Context, deps Dependencies, nameOrID string, options []compute.CloudServerReadOption) (*compute.CloudServerResult, error) {
	wrap := func(err error) error {
		return request.Wrap("GetServer", "compute", cloudread.ContextError(ctx, err))
	}
	cfg, err := ApplyReadOptions(options)
	if err != nil {
		return nil, wrap(err)
	}
	var record *servers.ServerFindRecord
	if cfg.Filters != nil {
		search, err := SearchRecords(ctx, deps, nameOrID, options)
		if err != nil {
			return nil, wrap(err)
		}
		if search.Expression {
			return nil, wrap(fmt.Errorf("%w: a JMESPath filter result is not a server record", resource.ErrInvalidOption))
		}
		switch len(search.Records) {
		case 0:
			return nil, nil
		case 1:
			record = search.Records[0]
		default:
			ids := make([]string, len(search.Records))
			for i, row := range search.Records {
				ids[i] = RecordString(row, "id")
			}
			return nil, wrap(&resource.AmbiguousError{Resource: "servers", Name: nameOrID, IDs: ids})
		}
	} else {
		if err := deps.Guard(ctx); err != nil {
			return nil, wrap(err)
		}
		service, err := deps.Compute(ctx)
		if err != nil {
			return nil, wrap(err)
		}
		location, err := deps.Location()
		if err != nil {
			return nil, wrap(err)
		}
		find := append([]servers.RecordFindOption{servers.WithRecordFindLocation(location)}, cfg.List...)
		find = append(find, servers.WithRecordFindDetails(true))
		record, err = service.Servers.FindRecord(ctx, nameOrID, find...)
		if err != nil {
			return nil, wrap(err)
		}
		if record == nil {
			return nil, nil
		}
	}
	server, err := serverview.FromRecord(record)
	if err != nil {
		return nil, wrap(err)
	}
	result := &compute.CloudServerResult{Server: server, Record: record, AdminPassword: server.AdminPass}
	return result, wrap(expandFrom(ctx, deps, result, cfg, record))
}

// List implements Connection.ListServers.
func List(ctx context.Context, deps Dependencies, options []compute.CloudServerReadOption) ([]*compute.CloudServerResult, error) {
	wrap := func(err error) error {
		return request.Wrap("ListServers", "compute", cloudread.ContextError(ctx, err))
	}
	cfg, err := ApplyReadOptions(options)
	if err != nil {
		return nil, wrap(err)
	}
	results, err := ListExpanded(ctx, deps, cfg)
	return results, wrap(err)
}

// ListExpanded consumes the server list once and expands every row in order
// as cfg selects. On a failure the results built before it are returned
// beside the unwrapped error.
func ListExpanded(ctx context.Context, deps Dependencies, cfg compute.CloudServerReadOpts) ([]*compute.CloudServerResult, error) {
	rows, _, err := list(ctx, deps, cfg)
	if err != nil {
		return nil, err
	}
	results := make([]*compute.CloudServerResult, 0, len(rows))
	for _, row := range rows {
		server, err := serverview.FromRecord(row)
		if err != nil {
			return results, err
		}
		result := &compute.CloudServerResult{Server: server, Record: row, AdminPassword: server.AdminPass}
		err = expandFrom(ctx, deps, result, cfg, row)
		results = append(results, result)
		if err != nil {
			return results, err
		}
	}
	return results, nil
}

// Search implements Connection.SearchServers.
func Search(ctx context.Context, deps Dependencies, nameOrID string, options []compute.CloudServerReadOption) (*compute.CloudServerSearch, error) {
	wrap := func(err error) error {
		return request.Wrap("SearchServers", "compute", cloudread.ContextError(ctx, err))
	}
	cfg, err := ApplyReadOptions(options)
	if err != nil {
		return nil, wrap(err)
	}
	listed, err := ListExpanded(ctx, deps, cfg)
	result := &compute.CloudServerSearch{Inventory: listed}
	if err != nil {
		return result, wrap(err)
	}
	views := make([]json.RawMessage, len(listed))
	for i, item := range listed {
		if views[i], err = compute.CloudServerResultView(item); err != nil {
			return result, wrap(err)
		}
	}
	selected, err := cloudfilter.SelectSource(views, nameOrID, cfg.Filters, func() error { return cloudread.Context(ctx) })
	if err != nil {
		return result, wrap(fmt.Errorf("%w: local server filters: %w", resource.ErrInvalidOption, err))
	}
	result.Value, result.Expression = bytes.Clone(selected.Value), selected.Expression
	if !selected.Expression {
		result.Servers = make([]*compute.CloudServerResult, len(selected.Indices))
		for i, index := range selected.Indices {
			result.Servers[i] = listed[index]
		}
	}
	return result, wrap(cloudread.Context(ctx))
}

// floatingAddresses mirrors meta.find_nova_interfaces(addresses,
// ext_tag='floating'): rows lacking OS-EXT-IPS:type are skipped and the
// original network and row order is kept.
func floatingAddresses(raw json.RawMessage) ([]string, error) {
	members, err := cloudfilter.ObjectMembers(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: server addresses must be an object: %w", resource.ErrInvalidOption, err)
	}
	var out []string
	for _, member := range members {
		var rows []map[string]json.RawMessage
		if err := json.Unmarshal(member.Value, &rows); err != nil {
			return nil, fmt.Errorf("%w: addresses of network %q must be a list of objects: %w", resource.ErrInvalidOption, member.Key, err)
		}
		for _, row := range rows {
			tag, ok := row["OS-EXT-IPS:type"]
			if !ok {
				continue
			}
			var kind string
			if json.Unmarshal(tag, &kind) != nil || kind != "floating" {
				continue
			}
			var addr string
			if err := json.Unmarshal(row["addr"], &addr); err != nil {
				return nil, fmt.Errorf("%w: floating address row of network %q has no string addr", resource.ErrInvalidOption, member.Key)
			}
			out = append(out, addr)
		}
	}
	return out, nil
}

func deleteFloatingIPs(ctx context.Context, deps Dependencies, record *servers.ServerFindRecord, cfg compute.CloudServerActionOpts) error {
	addresses := record.Resource.Body["addresses"]
	if truthy, err := cloudfilter.PythonJSONTruthy(bytes.Clone(addresses)); err != nil || !truthy {
		return err
	}
	floating, err := floatingAddresses(addresses)
	if err != nil {
		return err
	}
	retries := 1
	if cfg.DeleteIPRetry != nil {
		retries = *cfg.DeleteIPRetry
	}
	serverID := RecordString(record, "id")
	for _, address := range floating {
		filter, _ := json.Marshal(map[string]string{"floating_ip_address": address})
		query := append(slices.Clone(cfg.FloatingIPQuery), compute.WithFloatingIPQueryFilters(filter))
		found, err := deps.GetFloatingIP(ctx, compute.GetFloatingIPRequest{}, query...)
		if err != nil {
			if errors.Is(err, resource.ErrNotFound) || gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
				continue
			}
			return err
		}
		if found == nil || found.FloatingIP == nil || found.FloatingIP.Resource == nil {
			continue
		}
		var id string
		if err := json.Unmarshal(found.FloatingIP.Resource.Body["id"], &id); err != nil || id == "" {
			return fmt.Errorf("%w: floating IP %s has no string ID", resource.ErrInvalidOption, address)
		}
		deleteOptions := append(slices.Clone(cfg.FloatingIPDelete), compute.WithFloatingIPDeleteRetries(retries))
		deleted, err := deps.DeleteFloatingIP(ctx, compute.DeleteFloatingIPRequest{ID: id}, deleteOptions...)
		if err != nil {
			return err
		}
		if deleted == nil || !deleted.Deleted {
			return fmt.Errorf("tried to delete floating IP %s associated with server %s but it was not deleted; the server was not deleted", address, serverID)
		}
	}
	return nil
}

// Delete implements Connection.DeleteServer.
func Delete(ctx context.Context, deps Dependencies, nameOrID string, options []compute.CloudServerActionOption) (bool, error) {
	wrap := func(err error) error {
		return request.Wrap("DeleteServer", "compute", cloudread.ContextError(ctx, err))
	}
	cfg, err := ApplyActionOptions(options)
	if err != nil {
		return false, wrap(err)
	}
	if err := deps.Guard(ctx); err != nil {
		return false, wrap(err)
	}
	service, err := deps.Compute(ctx)
	if err != nil {
		return false, wrap(err)
	}
	location, err := deps.Location()
	if err != nil {
		return false, wrap(err)
	}
	record, err := service.Servers.FindRecord(ctx, nameOrID, servers.WithRecordFindLocation(location))
	if err != nil {
		return false, wrap(err)
	}
	if record == nil {
		return false, nil
	}
	id := RecordString(record, "id")
	if id == "" {
		return false, wrap(fmt.Errorf("%w: found server has no string ID", resource.ErrInvalidOption))
	}
	// _has_floating_ips: floating_ip_source none skips the floating cleanup.
	if cfg.DeleteIPs && deps.FloatingIPSource() != compute.FloatingIPNone {
		if err := deleteFloatingIPs(ctx, deps, record, cfg); err != nil {
			return false, wrap(err)
		}
	}
	if err := service.Servers.Delete(ctx, id); err != nil {
		if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			return false, nil
		}
		return false, wrap(err)
	}
	if !cfg.Wait {
		return true, nil
	}
	return true, wrap(service.Servers.WaitForDelete(ctx, resource.ID(id), waitOptions(cfg)...))
}

// Rebuild implements Connection.RebuildServer.
func Rebuild(ctx context.Context, deps Dependencies, serverID, imageID string, options []compute.CloudServerActionOption) (*compute.CloudServerResult, error) {
	wrap := func(err error) error {
		return request.Wrap("RebuildServer", "compute", cloudread.ContextError(ctx, err))
	}
	cfg, err := ApplyActionOptions(options)
	if err != nil {
		return nil, wrap(err)
	}
	if serverID == "" || imageID == "" {
		return nil, wrap(fmt.Errorf("%w: server ID and image ID are required", resource.ErrInvalidOption))
	}
	if err := deps.Guard(ctx); err != nil {
		return nil, wrap(err)
	}
	service, err := deps.Compute(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	opts := servers.RebuildOpts{ImageRef: imageID, AdminPass: cfg.AdminPassword}
	rebuilt, err := service.Servers.Rebuild(ctx, serverID, opts, cfg.Rebuild...)
	if err != nil {
		return nil, wrap(err)
	}
	result := &compute.CloudServerResult{Server: rebuilt, Sent: true, AdminPassword: rebuilt.AdminPass}
	read := compute.CloudServerReadOpts{Bare: cfg.Bare, Address: cfg.Address, Detailed: cfg.Detailed}
	if cfg.Wait {
		password := rebuilt.AdminPass
		if password == "" {
			password = cfg.AdminPassword
		}
		waitID := rebuilt.ID
		if waitID == "" {
			waitID = serverID
		}
		waited, err := service.Servers.WaitForServer(ctx, resource.ID(waitID), waitOptions(cfg)...)
		if err != nil {
			return result, wrap(err)
		}
		if waited.Status == "ACTIVE" {
			waited.AdminPass = password
		}
		result.Server, result.AdminPassword = waited, password
	}
	return result, wrap(expand(ctx, deps, result, read))
}

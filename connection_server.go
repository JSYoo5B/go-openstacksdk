package openstack

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
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/serverview"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

// CloudServerReadOpts owns the local controls of the cloud server reads.
// Filters follow the cloud filter rules: nil or null selects only the
// name-or-ID phase, a JSON object is an ordered dictionary filter and a JSON
// string is a JMESPath expression. List options reach servers.API.ListCloudRecords.
type CloudServerReadOpts struct {
	Filters *json.RawMessage
	List    []servers.RecordFindOption
	// Bare skips the address expansion of the typed server getters. Python
	// expands unless bare is set; the record readers never expand.
	Bare    bool
	Address []compute.ServerAddressOption
	// Detailed replaces the interface expansion with GetOpenStackVars
	// hostvars (Python detailed=True). Bare still skips every expansion.
	Detailed bool
}

// CloudServerReadOption mutates CloudServerReadOpts.
type CloudServerReadOption func(*CloudServerReadOpts) error

// WithCloudServerFilters sets owned JSON filters.
func WithCloudServerFilters(value json.RawMessage) CloudServerReadOption {
	owned := bytes.Clone(value)
	return func(o *CloudServerReadOpts) error {
		o.Filters = nil
		if owned != nil {
			raw := json.RawMessage(bytes.Clone(owned))
			o.Filters = &raw
		}
		return nil
	}
}

// WithCloudServerExpression sets a JMESPath filter expression.
func WithCloudServerExpression(expression string) CloudServerReadOption {
	raw, _ := json.Marshal(expression)
	return WithCloudServerFilters(raw)
}

// WithCloudServerListOptions forwards the Python-named list controls (all
// projects, details, query pushdown, paging caps, headers, microversion).
func WithCloudServerListOptions(options ...servers.RecordFindOption) CloudServerReadOption {
	owned := slices.Clone(options)
	return func(o *CloudServerReadOpts) error { o.List = append(o.List, owned...); return nil }
}

// WithCloudServerBare skips address expansion of typed getters.
func WithCloudServerBare(bare bool) CloudServerReadOption {
	return func(o *CloudServerReadOpts) error { o.Bare = bare; return nil }
}

// WithCloudServerAddressOptions forwards options to ExpandServerInterfaces.
func WithCloudServerAddressOptions(options ...compute.ServerAddressOption) CloudServerReadOption {
	owned := slices.Clone(options)
	return func(o *CloudServerReadOpts) error { o.Address = append(o.Address, owned...); return nil }
}

func applyCloudServerReadOptions(options []CloudServerReadOption) (CloudServerReadOpts, error) {
	var cfg CloudServerReadOpts
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

// CloudServerSearchResult separates the complete consumed inventory from the
// ordinary name/dictionary matches. Value is the filtered JSON array, or the
// arbitrary JMESPath output when Expression is true; an expression result has
// no source-row association and Records is then empty.
type CloudServerSearchResult struct {
	Value      json.RawMessage
	Records    []*servers.ServerFindRecord
	Inventory  []*servers.ServerFindRecord
	Expression bool
}

func (c *Connection) cloudServerList(ctx context.Context, cfg CloudServerReadOpts) ([]*servers.ServerFindRecord, *servers.API, error) {
	if err := c.recordListContext(ctx); err != nil {
		return nil, nil, err
	}
	service, err := c.ComputeV2(ctx)
	if err != nil {
		return nil, nil, err
	}
	location, err := c.CurrentLocation()
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

// ListServerRecords maps list_servers with bare=True: one eager consumption of
// GET /servers/detail (WithCloudServerListOptions selects details, all
// projects and query pushdown) without address expansion. On a failure the
// rows consumed before it are returned beside the error.
func (c *Connection) ListServerRecords(ctx context.Context, options ...CloudServerReadOption) ([]*servers.ServerFindRecord, error) {
	wrap := func(err error) error {
		return request.Wrap("ListServerRecords", "compute", cloudread.ContextError(ctx, err))
	}
	cfg, err := applyCloudServerReadOptions(options)
	if err != nil {
		return nil, wrap(err)
	}
	rows, _, err := c.cloudServerList(ctx, cfg)
	return rows, wrap(err)
}

// SearchServerRecords maps search_servers: list every server, then apply the
// exact/glob name-or-ID phase and the local dictionary or JMESPath filters.
// Nothing is pushed to Nova unless it is an explicit list option.
func (c *Connection) SearchServerRecords(ctx context.Context, nameOrID string, options ...CloudServerReadOption) (*CloudServerSearchResult, error) {
	wrap := func(err error) error {
		return request.Wrap("SearchServerRecords", "compute", cloudread.ContextError(ctx, err))
	}
	cfg, err := applyCloudServerReadOptions(options)
	if err != nil {
		return nil, wrap(err)
	}
	rows, _, err := c.cloudServerList(ctx, cfg)
	result := &CloudServerSearchResult{Inventory: rows}
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

// CloudServerResult is the outcome of the typed server getters and actions.
// Interfaces is nil when expansion was skipped or failed before a view existed.
type CloudServerResult struct {
	Server     *servers.Server
	Interfaces *compute.ServerAddressView
	// AdminPassword is the password Nova returned or the caller supplied.
	AdminPassword string
	// Record is the Source-shaped lookup row, when the operation found one.
	Record *servers.ServerFindRecord
	// Sent reports whether a mutating request was sent.
	Sent bool
	// Hostvars is the detailed expansion; nil unless Detailed was requested.
	Hostvars *CloudServerHostvars
}

func (c *Connection) cloudServerExpand(ctx context.Context, result *CloudServerResult, cfg CloudServerReadOpts) error {
	return c.cloudServerExpandFrom(ctx, result, cfg, nil)
}

// cloudServerExpandFrom expands result.Server; base is the record whose
// Resource view describes that same server snapshot, or nil.
func (c *Connection) cloudServerExpandFrom(ctx context.Context, result *CloudServerResult, cfg CloudServerReadOpts, base *servers.ServerFindRecord) error {
	if cfg.Bare || result == nil || result.Server == nil {
		return nil
	}
	if cfg.Detailed {
		hostvars, err := c.cloudServerHostvars(ctx, result.Server, base, cfg.Address)
		result.Hostvars = hostvars
		if hostvars != nil {
			result.Interfaces = hostvars.Interfaces
		}
		return err
	}
	view, err := c.ExpandServerInterfaces(ctx, result.Server, cfg.Address...)
	result.Interfaces = view
	return err
}

// GetServerByID maps get_server_by_id: GET /servers/{id}; a 404 is (nil, nil)
// as Python's None, and a found server is expanded with
// ExpandServerInterfaces unless WithCloudServerBare(true) is set. Any other
// failure, including an expansion failure beside the retained server, is returned.
func (c *Connection) GetServerByID(ctx context.Context, id string, options ...CloudServerReadOption) (*CloudServerResult, error) {
	wrap := func(err error) error {
		return request.Wrap("GetServerByID", "compute", cloudread.ContextError(ctx, err))
	}
	cfg, err := applyCloudServerReadOptions(options)
	if err != nil {
		return nil, wrap(err)
	}
	if cfg.Detailed {
		return nil, wrap(fmt.Errorf("%w: get_server_by_id has no detailed expansion", resource.ErrInvalidOption))
	}
	if err := c.recordListContext(ctx); err != nil {
		return nil, wrap(err)
	}
	service, err := c.ComputeV2(ctx)
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
	result := &CloudServerResult{Server: server, AdminPassword: server.AdminPass}
	return result, wrap(c.cloudServerExpand(ctx, result, cfg))
}

// CloudServerUpdate lists the only attributes update_server accepts (name and
// description). An unset attribute is omitted, an explicit Null description is
// sent as JSON null, and an empty or null Name is rejected before HTTP because
// the typed update builder cannot send it.
type CloudServerUpdate struct {
	Name        request.Optional[string]
	Description request.Optional[string]
}

// UpdateServer maps update_server: find the server strictly by ID or name
// (a missing server is resource.ErrNotFound and nothing is sent), then PUT only
// the supplied attributes. With no attribute Python's commit sends nothing; this
// returns the found record with Sent false and a nil Server.
func (c *Connection) UpdateServer(ctx context.Context, nameOrID string, update CloudServerUpdate, options ...CloudServerReadOption) (*CloudServerResult, error) {
	wrap := func(err error) error {
		return request.Wrap("UpdateServer", "compute", cloudread.ContextError(ctx, err))
	}
	cfg, err := applyCloudServerReadOptions(options)
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
	if err := c.recordListContext(ctx); err != nil {
		return nil, wrap(err)
	}
	service, err := c.ComputeV2(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	location, err := c.CurrentLocation()
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
	result := &CloudServerResult{Record: found}
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
	return result, wrap(c.cloudServerExpand(ctx, result, cfg))
}

// WithCloudServerDetailed selects Python's detailed=True expansion
// (GetOpenStackVars hostvars) instead of the interface expansion.
func WithCloudServerDetailed(detailed bool) CloudServerReadOption {
	return func(o *CloudServerReadOpts) error { o.Detailed = detailed; return nil }
}

// WithCloudServerActionDetailed selects the detailed=True expansion of the
// RebuildServer result.
func WithCloudServerActionDetailed(detailed bool) CloudServerActionOption {
	return func(o *CloudServerActionOpts) error { o.Detailed = detailed; return nil }
}

// GetServer maps get_server: find_server(name_or_id, details=True) through
// FindRecord (a missing server is (nil, nil)), then the default expansion:
// interfaces, hostvars with WithCloudServerDetailed(true), nothing with
// WithCloudServerBare(true). With WithCloudServerFilters (Python's deprecated
// filters argument) every server is listed bare and filtered locally; several
// matches are a resource.AmbiguousError and a JMESPath value that is not a
// row selection is rejected. Expansion failures are returned beside the found
// server.
func (c *Connection) GetServer(ctx context.Context, nameOrID string, options ...CloudServerReadOption) (*CloudServerResult, error) {
	wrap := func(err error) error {
		return request.Wrap("GetServer", "compute", cloudread.ContextError(ctx, err))
	}
	cfg, err := applyCloudServerReadOptions(options)
	if err != nil {
		return nil, wrap(err)
	}
	var record *servers.ServerFindRecord
	if cfg.Filters != nil {
		search, err := c.SearchServerRecords(ctx, nameOrID, options...)
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
				ids[i] = cloudServerRecordString(row, "id")
			}
			return nil, wrap(&resource.AmbiguousError{Resource: "servers", Name: nameOrID, IDs: ids})
		}
	} else {
		if err := c.recordListContext(ctx); err != nil {
			return nil, wrap(err)
		}
		service, err := c.ComputeV2(ctx)
		if err != nil {
			return nil, wrap(err)
		}
		location, err := c.CurrentLocation()
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
	result := &CloudServerResult{Server: server, Record: record, AdminPassword: server.AdminPass}
	return result, wrap(c.cloudServerExpandFrom(ctx, result, cfg, record))
}

// ListServers maps list_servers with its default bare=False: one eager
// consumption of the server list (as ListServerRecords), then every row is
// expanded in order with the interface expansion, the hostvars of
// WithCloudServerDetailed(true), or nothing with WithCloudServerBare(true).
// On a list or expansion failure the results built before it are returned
// beside the error; nothing is retried.
func (c *Connection) ListServers(ctx context.Context, options ...CloudServerReadOption) ([]*CloudServerResult, error) {
	wrap := func(err error) error {
		return request.Wrap("ListServers", "compute", cloudread.ContextError(ctx, err))
	}
	cfg, err := applyCloudServerReadOptions(options)
	if err != nil {
		return nil, wrap(err)
	}
	results, err := c.cloudServerListExpanded(ctx, cfg)
	return results, wrap(err)
}

func (c *Connection) cloudServerListExpanded(ctx context.Context, cfg CloudServerReadOpts) ([]*CloudServerResult, error) {
	rows, _, err := c.cloudServerList(ctx, cfg)
	if err != nil {
		return nil, err
	}
	results := make([]*CloudServerResult, 0, len(rows))
	for _, row := range rows {
		server, err := serverview.FromRecord(row)
		if err != nil {
			return results, err
		}
		result := &CloudServerResult{Server: server, Record: row, AdminPassword: server.AdminPass}
		err = c.cloudServerExpandFrom(ctx, result, cfg, row)
		results = append(results, result)
		if err != nil {
			return results, err
		}
	}
	return results, nil
}

// CloudServerSearch is the SearchServers result. Inventory is every expanded
// server; Value is the filtered JSON array of their views, or the arbitrary
// JMESPath output when Expression is true (Servers is then empty).
type CloudServerSearch struct {
	Value      json.RawMessage
	Servers    []*CloudServerResult
	Inventory  []*CloudServerResult
	Expression bool
}

// SearchServers maps search_servers with its default bare=False: ListServers,
// then the exact/glob name-or-ID phase and the local filters over each
// expanded view (CloudServerResultView). Filters never reach Nova.
func (c *Connection) SearchServers(ctx context.Context, nameOrID string, options ...CloudServerReadOption) (*CloudServerSearch, error) {
	wrap := func(err error) error {
		return request.Wrap("SearchServers", "compute", cloudread.ContextError(ctx, err))
	}
	cfg, err := applyCloudServerReadOptions(options)
	if err != nil {
		return nil, wrap(err)
	}
	listed, err := c.cloudServerListExpanded(ctx, cfg)
	result := &CloudServerSearch{Inventory: listed}
	if err != nil {
		return result, wrap(err)
	}
	views := make([]json.RawMessage, len(listed))
	for i, item := range listed {
		if views[i], err = CloudServerResultView(item); err != nil {
			return result, wrap(err)
		}
	}
	selected, err := cloudfilter.SelectSource(views, nameOrID, cfg.Filters, func() error { return cloudread.Context(ctx) })
	if err != nil {
		return result, wrap(fmt.Errorf("%w: local server filters: %w", resource.ErrInvalidOption, err))
	}
	result.Value, result.Expression = bytes.Clone(selected.Value), selected.Expression
	if !selected.Expression {
		result.Servers = make([]*CloudServerResult, len(selected.Indices))
		for i, index := range selected.Indices {
			result.Servers[i] = listed[index]
		}
	}
	return result, wrap(cloudread.Context(ctx))
}

// CloudServerResultView renders the JSON mapping Python filters and returns
// for one server: the hostvars mapping when detailed, else the record's
// Source Resource view (or the typed server JSON) with the interface
// fields of the address expansion set over it.
func CloudServerResultView(result *CloudServerResult) (json.RawMessage, error) {
	if result == nil {
		return nil, invalid("server result is required")
	}
	if result.Hostvars != nil {
		return json.Marshal(result.Hostvars)
	}
	return cloudServerView(result.Server, result.Record, resource.CloudLocation{}, result.Interfaces)
}

// CloudServerActionOpts owns the wait, address and floating-IP controls shared
// by DeleteServer and RebuildServer. Timeout defaults to Python's 180 seconds.
type CloudServerActionOpts struct {
	Wait      bool
	Timeout   time.Duration
	Unlimited bool
	// WaitOptions apply after the timeout (for example a poll interval).
	WaitOptions []resource.WaitOption

	// DeleteIPs and DeleteIPRetry map delete_ips and delete_ip_retry
	// (default one retry). FloatingIPQuery and FloatingIPDelete tune the two
	// floating-IP helpers used by DeleteIPs.
	DeleteIPs        bool
	DeleteIPRetry    *int
	FloatingIPQuery  []compute.FloatingIPQueryOption
	FloatingIPDelete []compute.FloatingIPDeleteOption

	// AdminPassword and Rebuild tune RebuildServer; Bare and Address expand
	// the result like the typed getters.
	AdminPassword string
	Rebuild       []servers.RebuildOption
	Bare          bool
	Address       []compute.ServerAddressOption
	// Detailed expands the rebuild result into GetOpenStackVars hostvars.
	Detailed bool
}

// CloudServerActionOption mutates CloudServerActionOpts.
type CloudServerActionOption func(*CloudServerActionOpts) error

// WithCloudServerWait enables the post-request wait.
func WithCloudServerWait(wait bool) CloudServerActionOption {
	return func(o *CloudServerActionOpts) error { o.Wait = wait; return nil }
}

// WithCloudServerTimeout replaces the 180-second wait budget.
func WithCloudServerTimeout(timeout time.Duration) CloudServerActionOption {
	return func(o *CloudServerActionOpts) error { o.Timeout, o.Unlimited = timeout, false; return nil }
}

// WithCloudServerUnlimitedWait maps a Python timeout of None.
func WithCloudServerUnlimitedWait() CloudServerActionOption {
	return func(o *CloudServerActionOpts) error { o.Unlimited = true; return nil }
}

// WithCloudServerWaitOptions appends wait options that apply after the timeout.
func WithCloudServerWaitOptions(options ...resource.WaitOption) CloudServerActionOption {
	owned := slices.Clone(options)
	return func(o *CloudServerActionOpts) error { o.WaitOptions = append(o.WaitOptions, owned...); return nil }
}

// WithCloudServerDeleteIPs deletes the floating IPs found in the server's
// addresses before the server is deleted.
func WithCloudServerDeleteIPs(enabled bool) CloudServerActionOption {
	return func(o *CloudServerActionOpts) error { o.DeleteIPs = enabled; return nil }
}

// WithCloudServerDeleteIPRetry sets the floating-IP delete retry count.
func WithCloudServerDeleteIPRetry(retries int) CloudServerActionOption {
	return func(o *CloudServerActionOpts) error { x := retries; o.DeleteIPRetry = &x; return nil }
}

// WithCloudServerFloatingIPOptions forwards options to the floating-IP lookup and delete.
func WithCloudServerFloatingIPOptions(query []compute.FloatingIPQueryOption, remove []compute.FloatingIPDeleteOption) CloudServerActionOption {
	q, d := slices.Clone(query), slices.Clone(remove)
	return func(o *CloudServerActionOpts) error {
		o.FloatingIPQuery = append(o.FloatingIPQuery, q...)
		o.FloatingIPDelete = append(o.FloatingIPDelete, d...)
		return nil
	}
}

// WithCloudServerAdminPassword sets the rebuild admin password; an empty value is omitted.
func WithCloudServerAdminPassword(password string) CloudServerActionOption {
	return func(o *CloudServerActionOpts) error { o.AdminPassword = password; return nil }
}

// WithCloudServerRebuildOptions forwards extension options to servers.API.Rebuild.
func WithCloudServerRebuildOptions(options ...servers.RebuildOption) CloudServerActionOption {
	owned := slices.Clone(options)
	return func(o *CloudServerActionOpts) error { o.Rebuild = append(o.Rebuild, owned...); return nil }
}

// WithCloudServerActionBare skips address expansion of the rebuild result.
func WithCloudServerActionBare(bare bool) CloudServerActionOption {
	return func(o *CloudServerActionOpts) error { o.Bare = bare; return nil }
}

// WithCloudServerActionAddressOptions forwards options to ExpandServerInterfaces.
func WithCloudServerActionAddressOptions(options ...compute.ServerAddressOption) CloudServerActionOption {
	owned := slices.Clone(options)
	return func(o *CloudServerActionOpts) error { o.Address = append(o.Address, owned...); return nil }
}

func applyCloudServerActionOptions(options []CloudServerActionOption) (CloudServerActionOpts, error) {
	cfg := CloudServerActionOpts{Timeout: 180 * time.Second}
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

func (cfg CloudServerActionOpts) waitOptions() []resource.WaitOption {
	first := resource.WithTimeout(cfg.Timeout)
	if cfg.Unlimited {
		first = resource.WithUnlimitedWait()
	}
	return append([]resource.WaitOption{first}, cfg.WaitOptions...)
}

// cloudServerFloatingAddresses mirrors meta.find_nova_interfaces(addresses,
// ext_tag='floating'): rows lacking OS-EXT-IPS:type are skipped and the
// original network and row order is kept.
func cloudServerFloatingAddresses(raw json.RawMessage) ([]string, error) {
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

func cloudServerRecordString(record *servers.ServerFindRecord, key string) string {
	if record == nil || record.Resource == nil {
		return ""
	}
	var text string
	_ = json.Unmarshal(record.Resource.Body[key], &text)
	return text
}

func (c *Connection) cloudServerDeleteFloatingIPs(ctx context.Context, record *servers.ServerFindRecord, cfg CloudServerActionOpts) error {
	addresses := record.Resource.Body["addresses"]
	if truthy, err := cloudfilter.PythonJSONTruthy(bytes.Clone(addresses)); err != nil || !truthy {
		return err
	}
	floating, err := cloudServerFloatingAddresses(addresses)
	if err != nil {
		return err
	}
	retries := 1
	if cfg.DeleteIPRetry != nil {
		retries = *cfg.DeleteIPRetry
	}
	serverID := cloudServerRecordString(record, "id")
	for _, address := range floating {
		filter, _ := json.Marshal(map[string]string{"floating_ip_address": address})
		query := append(slices.Clone(cfg.FloatingIPQuery), compute.WithFloatingIPQueryFilters(filter))
		found, err := c.GetFloatingIP(ctx, compute.GetFloatingIPRequest{}, query...)
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
		deleted, err := c.DeleteFloatingIP(ctx, compute.DeleteFloatingIPRequest{ID: id}, deleteOptions...)
		if err != nil {
			return err
		}
		if deleted == nil || !deleted.Deleted {
			return fmt.Errorf("tried to delete floating IP %s associated with server %s but it was not deleted; the server was not deleted", address, serverID)
		}
	}
	return nil
}

// DeleteServer maps delete_server. The server is found like find_server with
// its defaults (details, current project). A missing server is (false, nil)
// with no DELETE. With WithCloudServerDeleteIPs each floating address found in
// the server's addresses is looked up and deleted first, through the Neutron or
// Nova floating-IP backend the Connection selects; an address whose lookup
// reports not found or finds nothing is skipped, and a floating IP that is not
// deleted aborts before the server DELETE. When the Connection's
// floating_ip_source is none (ServerAddressPolicy or clouds.yaml), the
// floating-IP cleanup is skipped without any request, as Python's
// _has_floating_ips gate does. A 404 on the DELETE is (false, nil).
// With WithCloudServerWait the call then observes absence for the timeout
// (default 180 seconds). The DELETE is never replayed; once it was accepted a
// wait failure returns (true, err) because the remote delete was requested.
func (c *Connection) DeleteServer(ctx context.Context, nameOrID string, options ...CloudServerActionOption) (bool, error) {
	wrap := func(err error) error {
		return request.Wrap("DeleteServer", "compute", cloudread.ContextError(ctx, err))
	}
	cfg, err := applyCloudServerActionOptions(options)
	if err != nil {
		return false, wrap(err)
	}
	if err := c.recordListContext(ctx); err != nil {
		return false, wrap(err)
	}
	service, err := c.ComputeV2(ctx)
	if err != nil {
		return false, wrap(err)
	}
	location, err := c.CurrentLocation()
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
	id := cloudServerRecordString(record, "id")
	if id == "" {
		return false, wrap(fmt.Errorf("%w: found server has no string ID", resource.ErrInvalidOption))
	}
	// _has_floating_ips: floating_ip_source none skips the floating cleanup.
	if cfg.DeleteIPs && c.options.serverAddresses.FloatingIPSource() != compute.FloatingIPNone {
		if err := c.cloudServerDeleteFloatingIPs(ctx, record, cfg); err != nil {
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
	return true, wrap(service.Servers.WaitForDelete(ctx, resource.ID(id), cfg.waitOptions()...))
}

// RebuildServer maps rebuild_server: POST the rebuild action with imageRef and,
// when nonempty, adminPass. An empty image ID is rejected before HTTP (Python
// raises TypeError without it). With WithCloudServerWait the server is polled
// until ACTIVE (ERROR fails) for the timeout, and the admin password returned by
// the action, else the supplied one, is restored on the waited server because a
// later GET never repeats it. Without wait the action response is returned.
// A failed wait returns the action result beside the error and never replays
// the rebuild. The result is expanded unless WithCloudServerActionBare(true).
func (c *Connection) RebuildServer(ctx context.Context, serverID, imageID string, options ...CloudServerActionOption) (*CloudServerResult, error) {
	wrap := func(err error) error {
		return request.Wrap("RebuildServer", "compute", cloudread.ContextError(ctx, err))
	}
	cfg, err := applyCloudServerActionOptions(options)
	if err != nil {
		return nil, wrap(err)
	}
	if serverID == "" || imageID == "" {
		return nil, wrap(fmt.Errorf("%w: server ID and image ID are required", resource.ErrInvalidOption))
	}
	if err := c.recordListContext(ctx); err != nil {
		return nil, wrap(err)
	}
	service, err := c.ComputeV2(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	opts := servers.RebuildOpts{ImageRef: imageID, AdminPass: cfg.AdminPassword}
	rebuilt, err := service.Servers.Rebuild(ctx, serverID, opts, cfg.Rebuild...)
	if err != nil {
		return nil, wrap(err)
	}
	result := &CloudServerResult{Server: rebuilt, Sent: true, AdminPassword: rebuilt.AdminPass}
	read := CloudServerReadOpts{Bare: cfg.Bare, Address: cfg.Address, Detailed: cfg.Detailed}
	if cfg.Wait {
		password := rebuilt.AdminPass
		if password == "" {
			password = cfg.AdminPassword
		}
		waitID := rebuilt.ID
		if waitID == "" {
			waitID = serverID
		}
		waited, err := service.Servers.WaitForServer(ctx, resource.ID(waitID), cfg.waitOptions()...)
		if err != nil {
			return result, wrap(err)
		}
		if waited.Status == "ACTIVE" {
			waited.AdminPass = password
		}
		result.Server, result.AdminPassword = waited, password
	}
	return result, wrap(c.cloudServerExpand(ctx, result, read))
}

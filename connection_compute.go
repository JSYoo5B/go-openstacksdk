package openstack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/limits"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/usage"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

// UpdateAggregate finds a host aggregate by Nova ID-or-name (strict: a missing
// aggregate is resource.ErrNotFound and no update is sent) and then sends one
// PUT with only the attributes supplied through the update options.
func (c *Connection) UpdateAggregate(ctx context.Context, nameOrID string, options ...aggregates.UpdateRecordOption) (*aggregates.AggregateRecord, error) {
	fail := func(err error) (*aggregates.AggregateRecord, error) {
		return nil, request.Wrap("UpdateAggregate", "aggregate", err)
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	service, err := c.ComputeV2(ctx)
	if err != nil {
		return fail(err)
	}
	found, err := service.Aggregates.FindIdentity(ctx, nameOrID, resource.WithIdentityFindIgnoreMissing(false))
	if err != nil {
		return fail(err)
	}
	record, err := service.Aggregates.UpdateRecord(ctx, found.ID, options...)
	if err != nil {
		return fail(err)
	}
	return record, nil
}

// DeleteAggregate deletes a host aggregate and reports whether a delete
// happened. An identity made only of digits is taken as an aggregate ID and
// deleted without any lookup; any other identity is first found by name or ID.
// A missing aggregate, including a 404 on the DELETE itself, returns
// (false, nil). Every other failure is returned and the DELETE is never replayed.
func (c *Connection) DeleteAggregate(ctx context.Context, nameOrID string) (bool, error) {
	fail := func(err error) (bool, error) {
		return false, request.Wrap("DeleteAggregate", "aggregate", err)
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	if nameOrID == "" {
		return fail(fmt.Errorf("%w: aggregate name or ID is required", resource.ErrInvalidOption))
	}
	service, err := c.ComputeV2(ctx)
	if err != nil {
		return fail(err)
	}
	var id int
	if aggregateDigitsOnly(nameOrID) {
		id, err = strconv.Atoi(nameOrID)
		if err != nil {
			return fail(fmt.Errorf("%w: aggregate ID %q is not a valid integer", resource.ErrInvalidOption, nameOrID))
		}
	} else {
		found, err := service.Aggregates.FindIdentity(ctx, nameOrID)
		if err != nil {
			return fail(err)
		}
		if found == nil {
			return false, nil
		}
		id = found.ID
	}
	if err := service.Aggregates.Delete(ctx, id); err != nil {
		if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			return false, nil
		}
		return fail(err)
	}
	return true, nil
}

func aggregateDigitsOnly(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r > unicode.MaxASCII || !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// CloudAggregateSearchResult carries the ordinary matches and the JSON view the
// filters ran over. Value is the filtered JSON array or, for a JMESPath
// expression, arbitrary output with Expression true and no Aggregates.
type CloudAggregateSearchResult struct {
	Value      json.RawMessage
	Aggregates []*aggregates.Aggregate
	Inventory  []*aggregates.Aggregate
	Expression bool
}

// cloudAggregateView builds the Source Aggregate attribute view the cloud
// filters see from the row Nova returned. It keeps Python attribute names
// (deleted as is_deleted), turns absent members into null, keeps null and the
// original timestamp strings, applies the hosts, metadata and is_deleted
// conversions of openstack/fields.py, and adds the computed location, whose
// zone is the aggregate's availability_zone.
func cloudAggregateView(raw json.RawMessage, location resource.CloudLocation) (json.RawMessage, error) {
	var members map[string]json.RawMessage
	if err := json.Unmarshal(raw, &members); err != nil {
		return nil, err
	}
	field := func(name string) json.RawMessage {
		if value, ok := members[name]; ok {
			return bytes.TrimSpace(value)
		}
		return json.RawMessage("null")
	}
	view := make(map[string]json.RawMessage, 11)
	for _, name := range []string{"id", "name", "uuid", "availability_zone", "created_at", "updated_at", "deleted_at"} {
		view[name] = field(name)
	}
	// A non-list hosts value becomes a one-element list and a non-dict
	// metadata value an empty dict, as _convert_type does; null stays null.
	switch hosts := field("hosts"); {
	case bytes.Equal(hosts, []byte("null")) || hosts[0] == '[':
		view["hosts"] = hosts
	default:
		view["hosts"] = append(append(json.RawMessage("["), hosts...), ']')
	}
	switch metadata := field("metadata"); {
	case bytes.Equal(metadata, []byte("null")) || metadata[0] == '{':
		view["metadata"] = metadata
	default:
		view["metadata"] = json.RawMessage("{}")
	}
	deleted, err := jsonfilter.BooleanJSON(field("deleted"))
	if err != nil {
		return nil, err
	}
	view["is_deleted"] = deleted
	if view["location"], err = location.ForResource(nil, view["availability_zone"]); err != nil {
		return nil, err
	}
	return json.Marshal(view)
}

// SearchAggregates maps search_aggregates: list every host aggregate, then apply
// the exact/glob name-or-ID phase and the local dictionary or JMESPath filters.
// No filter is sent to Nova. The filters run over the Source attribute view
// built from each row Nova returned (see cloudAggregateView). Only the filter
// options apply; list, bare and address options are rejected before HTTP.
func (c *Connection) SearchAggregates(ctx context.Context, nameOrID string, options ...CloudServerReadOption) (*CloudAggregateSearchResult, error) {
	wrap := func(err error) error {
		return request.Wrap("SearchAggregates", "aggregates", cloudread.ContextError(ctx, err))
	}
	cfg, err := applyCloudServerReadOptions(options)
	if err != nil {
		return nil, wrap(err)
	}
	if len(cfg.List) > 0 || cfg.Bare || len(cfg.Address) > 0 {
		return nil, wrap(fmt.Errorf("%w: only the filter options apply to aggregates", resource.ErrInvalidOption))
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
	result := &CloudAggregateSearchResult{}
	var views []json.RawMessage
	for row, err := range service.Aggregates.ListRows(ctx) {
		if err != nil {
			return result, wrap(err)
		}
		view, err := cloudAggregateView(row.Raw, location)
		if err != nil {
			return result, wrap(err)
		}
		aggregate := row.Aggregate
		result.Inventory = append(result.Inventory, &aggregate)
		views = append(views, view)
	}
	selected, err := cloudfilter.SelectSource(views, nameOrID, cfg.Filters, func() error { return cloudread.Context(ctx) })
	if err != nil {
		return result, wrap(fmt.Errorf("%w: local aggregate filters: %w", resource.ErrInvalidOption, err))
	}
	result.Value, result.Expression = bytes.Clone(selected.Value), selected.Expression
	if !selected.Expression {
		for _, index := range selected.Indices {
			result.Aggregates = append(result.Aggregates, result.Inventory[index])
		}
	}
	return result, wrap(cloudread.Context(ctx))
}

// GetComputeLimits reads Nova absolute limits. An empty nameOrID reads the
// limits of the authenticated project without a tenant query and without any
// Identity request. A non-empty value is resolved once by Keystone ID-or-name
// lookup (strict) and sent as the tenant_id query. The whole LimitsResource is
// returned; its embedded Absolute data is the part a cloud helper exposes.
func (c *Connection) GetComputeLimits(ctx context.Context, nameOrID string, options ...limits.GetOption) (*limits.LimitsResource, error) {
	if err := ctx.Err(); err != nil {
		return nil, request.Wrap("GetComputeLimits", "compute limits", err)
	}
	if nameOrID == "" {
		service, err := c.ComputeV2(ctx)
		if err != nil {
			return nil, request.Wrap("GetComputeLimits", "compute limits", err)
		}
		return service.Limits.Fetch(ctx, options...)
	}
	id, err := c.findQuotaProjectID(ctx, "GetComputeLimits", nameOrID)
	if err != nil {
		return nil, err
	}
	scope, err := c.ProjectLimits(ctx, resource.ID(id))
	if err != nil {
		return nil, err
	}
	return scope.Get(ctx, options...)
}

// ComputeUsageRange bounds a tenant usage query. A zero Start or End is not
// sent. Values are converted to UTC before they are formatted.
type ComputeUsageRange struct {
	Start, End time.Time
}

// ParseComputeUsageDate parses an ISO 8601 date for ComputeUsageRange. It
// accepts a date, a date-time with a "T" or space separator, optional
// fractional seconds, and an optional "Z" or numeric offset. A value without an
// offset is taken as UTC. Anything else is rejected with resource.ErrInvalidOption.
func ParseComputeUsageDate(value string) (time.Time, error) {
	for _, layout := range []string{
		time.RFC3339Nano, "2006-01-02T15:04:05.999999999", "2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05.999999999", "2006-01-02T15:04Z07:00", "2006-01-02T15:04", "2006-01-02",
	} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("%w: date given, %s, is invalid; pass an ISO 8601 date string (YYYY-MM-DDTHH:MM:SS)", resource.ErrInvalidOption, strings.TrimSpace(value))
}

// GetComputeUsage resolves the project by strict Keystone ID-or-name lookup and
// reads its tenant usage with one request: the first value of the usage stream.
// Pagination links are not followed. The date range, when given, is sent as
// UTC start/end query values.
func (c *Connection) GetComputeUsage(ctx context.Context, nameOrID string, dates ComputeUsageRange, options ...usage.SingleTenantOption) (*usage.TenantUsage, error) {
	fail := func(err error) (*usage.TenantUsage, error) {
		return nil, request.Wrap("GetComputeUsage", "compute usage", err)
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	if nameOrID == "" {
		return fail(fmt.Errorf("%w: project name or ID is required", resource.ErrInvalidOption))
	}
	id, err := c.findQuotaProjectID(ctx, "GetComputeUsage", nameOrID)
	if err != nil {
		return nil, err
	}
	service, err := c.ComputeV2(ctx)
	if err != nil {
		return fail(err)
	}
	var opts usage.SingleTenantOpts
	if !dates.Start.IsZero() {
		start := dates.Start.UTC()
		opts.Start = &start
	}
	if !dates.End.IsZero() {
		end := dates.End.UTC()
		opts.End = &end
	}
	for value, err := range service.Usage.SingleTenant(ctx, id, opts, options...) {
		if err != nil {
			return fail(err)
		}
		return value, nil
	}
	return fail(fmt.Errorf("%w: no usage returned for project %q", resource.ErrNotFound, id))
}

func (c *Connection) cloudGroupContext(ctx context.Context) (*compute.Service, *resource.CloudLocation, error) {
	if err := c.recordListContext(ctx); err != nil {
		return nil, nil, err
	}
	service, err := c.Compute(ctx)
	if err != nil {
		return nil, nil, err
	}
	location, err := c.CurrentLocation()
	if err != nil {
		return nil, nil, err
	}
	return service, &location, nil
}

func (c *Connection) SearchServerGroups(ctx context.Context, identity string, options ...compute.ServerGroupCloudReadOption) (*compute.ServerGroupCloudSearchResult, error) {
	wrap := func(err error) error {
		return request.Wrap("SearchServerGroups", "compute", cloudread.ContextError(ctx, err))
	}
	service, location, err := c.cloudGroupContext(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	owned := append(slices.Clone(options), func(cfg *compute.ServerGroupCloudReadOpts) error {
		if cfg.CurrentLocation == nil {
			v := location.Clone()
			cfg.CurrentLocation = &v
		}
		return nil
	})
	value, err := service.SearchServerGroups(ctx, identity, owned...)
	return value, wrap(err)
}

func (c *Connection) GetServerGroup(ctx context.Context, identity string, options ...compute.ServerGroupCloudReadOption) (*compute.ServerGroupCloudGetResult, error) {
	wrap := func(err error) error {
		return request.Wrap("GetServerGroup", "compute", cloudread.ContextError(ctx, err))
	}
	service, location, err := c.cloudGroupContext(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	owned := append(slices.Clone(options), func(cfg *compute.ServerGroupCloudReadOpts) error {
		if cfg.CurrentLocation == nil {
			v := location.Clone()
			cfg.CurrentLocation = &v
		}
		return nil
	})
	value, err := service.GetServerGroup(ctx, identity, owned...)
	return value, wrap(err)
}

func (c *Connection) GetServerConsole(ctx context.Context, identity string, options ...compute.ServerConsoleOption) (*compute.ServerConsoleResult, error) {
	wrap := func(err error) error {
		return request.Wrap("GetServerConsole", "compute", cloudread.ContextError(ctx, err))
	}
	if err := c.recordListContext(ctx); err != nil {
		return nil, wrap(err)
	}
	service, err := c.Compute(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	value, err := service.GetServerConsole(ctx, identity, slices.Clone(options)...)
	return value, wrap(err)
}

// cloudSecurityGroupActionName mirrors sec_group.name or sec_group.id.
func cloudSecurityGroupActionName(body map[string]json.RawMessage) (string, error) {
	var name string
	if raw, ok := body["name"]; ok && json.Unmarshal(raw, &name) == nil && name != "" {
		return name, nil
	}
	var id any
	if err := json.Unmarshal(body["id"], &id); err != nil || id == nil || id == "" {
		return "", fmt.Errorf("%w: security group has neither a name nor an ID", resource.ErrInvalidOption)
	}
	return fmt.Sprint(id), nil
}

// cloudServerSecurityGroupNames mirrors _get_server_security_groups: it looks
// the server up with find_server (bare), then resolves every group in order
// before any action is sent. found is false when the server or any group is
// missing, in which case later groups are not looked up.
func (c *Connection) cloudServerSecurityGroupNames(ctx context.Context, server string, groups []string) (id string, names []string, found bool, err error) {
	service, err := c.ComputeV2(ctx)
	if err != nil {
		return "", nil, false, err
	}
	location, err := c.CurrentLocation()
	if err != nil {
		return "", nil, false, err
	}
	record, err := service.Servers.FindRecord(ctx, server, servers.WithRecordFindLocation(location))
	if err != nil || record == nil {
		return "", nil, false, err
	}
	id = cloudServerRecordString(record, "id")
	if id == "" {
		return "", nil, false, fmt.Errorf("%w: found server has no string ID", resource.ErrInvalidOption)
	}
	for _, group := range groups {
		result, err := c.GetSecurityGroupRecord(ctx, group)
		if err != nil {
			return id, nil, false, err
		}
		if result == nil || result.Group == nil || result.Group.Resource == nil {
			return id, nil, false, nil
		}
		name, err := cloudSecurityGroupActionName(result.Group.Resource.Body)
		if err != nil {
			return id, nil, false, err
		}
		names = append(names, name)
	}
	return id, names, true, nil
}

func (c *Connection) cloudSecurityGroupPrepare(ctx context.Context, op, server string, groups []string) (id string, names []string, found bool, err error) {
	fail := func(err error) (string, []string, bool, error) {
		return "", nil, false, request.Wrap(op, "compute", cloudread.ContextError(ctx, err))
	}
	if err := c.recordListContext(ctx); err != nil {
		return fail(err)
	}
	if server == "" {
		return fail(fmt.Errorf("%w: server name or ID is required", resource.ErrInvalidOption))
	}
	for _, group := range groups {
		if group == "" {
			return fail(fmt.Errorf("%w: security group name or ID must be nonempty", resource.ErrInvalidOption))
		}
	}
	// _has_secgroups runs before the server lookup.
	if !c.hasSecurityGroups() {
		return fail(network.ErrUnavailableSecurityGroups)
	}
	id, names, found, err = c.cloudServerSecurityGroupNames(ctx, server, groups)
	if err != nil {
		return fail(err)
	}
	return id, names, found, nil
}

// AddServerSecurityGroups maps add_server_security_groups for identities. It
// finds the server, resolves every group (Neutron or Nova per the Connection's
// SecurityGroupCloud selection), and only then sends one addSecurityGroup action
// per group, in order, using the group's name or else its ID. A missing server
// or group returns (false, nil) before any action. An empty group list sends
// nothing and returns false, as Python's empty-list guard does. A failing action stops the
// sequence (earlier groups stay added) and no action is replayed.
// When secgroup_source (WithSecurityGroupSource or clouds.yaml) is neither
// neutron nor nova, network.ErrUnavailableSecurityGroups is returned before any
// request, as Python raises OpenStackCloudUnavailableFeature. Passing
// already-resolved group objects is not modelled.
func (c *Connection) AddServerSecurityGroups(ctx context.Context, server string, groups ...string) (bool, error) {
	id, names, found, err := c.cloudSecurityGroupPrepare(ctx, "AddServerSecurityGroups", server, groups)
	if err != nil || !found || len(names) == 0 {
		return false, err
	}
	service, err := c.ComputeV2(ctx)
	if err != nil {
		return false, request.Wrap("AddServerSecurityGroups", "compute", cloudread.ContextError(ctx, err))
	}
	for _, name := range names {
		if err := service.SecurityGroups.AddServer(ctx, id, name); err != nil {
			return false, request.Wrap("AddServerSecurityGroups", "compute", cloudread.ContextError(ctx, err))
		}
	}
	return true, nil
}

// RemoveServerSecurityGroups maps remove_server_security_groups with the same
// lookup phase as AddServerSecurityGroups. A 404 from one removeSecurityGroup
// action is tolerated: the remaining groups are still attempted and the result
// is false. Any other failure stops the sequence.
func (c *Connection) RemoveServerSecurityGroups(ctx context.Context, server string, groups ...string) (bool, error) {
	id, names, found, err := c.cloudSecurityGroupPrepare(ctx, "RemoveServerSecurityGroups", server, groups)
	if err != nil || !found || len(names) == 0 {
		return false, err
	}
	service, err := c.ComputeV2(ctx)
	if err != nil {
		return false, request.Wrap("RemoveServerSecurityGroups", "compute", cloudread.ContextError(ctx, err))
	}
	all := true
	for _, name := range names {
		if err := service.SecurityGroups.RemoveServer(ctx, id, name); err != nil {
			if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
				all = false
				continue
			}
			return false, request.Wrap("RemoveServerSecurityGroups", "compute", cloudread.ContextError(ctx, err))
		}
	}
	return all, nil
}

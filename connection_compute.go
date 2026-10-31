package openstack

import (
	"context"
	"encoding/json"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/limits"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/usage"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudcompute"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudserver"
)

// cloudComputeDependencies supplies the Connection facilities of the cloud
// compute helpers.
func (c *Connection) cloudComputeDependencies() cloudcompute.Dependencies {
	return cloudcompute.Dependencies{
		Guard:         c.recordListContext,
		ComputeV2:     c.ComputeV2,
		Compute:       c.Compute,
		Location:      c.CurrentLocation,
		FindProjectID: c.findQuotaProjectID,
		ProjectLimits: c.ProjectLimits,
	}
}

// UpdateAggregate finds a host aggregate by Nova ID-or-name (strict: a missing
// aggregate is resource.ErrNotFound and no update is sent) and then sends one
// PUT with only the attributes supplied through the update options.
func (c *Connection) UpdateAggregate(ctx context.Context, nameOrID string, options ...aggregates.UpdateRecordOption) (*aggregates.AggregateRecord, error) {
	return cloudcompute.UpdateAggregate(ctx, c.cloudComputeDependencies(), nameOrID, options)
}

// DeleteAggregate deletes a host aggregate and reports whether a delete
// happened. An identity made only of digits is taken as an aggregate ID and
// deleted without any lookup; any other identity is first found by name or ID.
// A missing aggregate, including a 404 on the DELETE itself, returns
// (false, nil). Every other failure is returned and the DELETE is never replayed.
func (c *Connection) DeleteAggregate(ctx context.Context, nameOrID string) (bool, error) {
	return cloudcompute.DeleteAggregate(ctx, c.cloudComputeDependencies(), nameOrID)
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

// SearchAggregates maps search_aggregates: list every host aggregate, then apply
// the exact/glob name-or-ID phase and the local dictionary or JMESPath filters.
// No filter is sent to Nova. The filters run over the Source attribute view
// built from each row Nova returned: Python attribute names (deleted as
// is_deleted), null for absent members, the hosts, metadata and is_deleted
// conversions of openstack/fields.py, and the computed location. Only the
// filter options apply; list, bare and address options are rejected before
// HTTP.
func (c *Connection) SearchAggregates(ctx context.Context, nameOrID string, options ...CloudServerReadOption) (*CloudAggregateSearchResult, error) {
	result, err := cloudcompute.SearchAggregates(ctx, c.cloudComputeDependencies(), nameOrID, options)
	return (*CloudAggregateSearchResult)(result), err
}

// GetComputeLimits reads Nova absolute limits. An empty nameOrID reads the
// limits of the authenticated project without a tenant query and without any
// Identity request. A non-empty value is resolved once by Keystone ID-or-name
// lookup (strict) and sent as the tenant_id query. The whole LimitsResource is
// returned; its embedded Absolute data is the part a cloud helper exposes.
func (c *Connection) GetComputeLimits(ctx context.Context, nameOrID string, options ...limits.GetOption) (*limits.LimitsResource, error) {
	return cloudcompute.GetLimits(ctx, c.cloudComputeDependencies(), nameOrID, options)
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
	return cloudcompute.ParseUsageDate(value)
}

// GetComputeUsage resolves the project by strict Keystone ID-or-name lookup and
// reads its tenant usage with one request: the first value of the usage stream.
// Pagination links are not followed. The date range, when given, is sent as
// UTC start/end query values.
func (c *Connection) GetComputeUsage(ctx context.Context, nameOrID string, dates ComputeUsageRange, options ...usage.SingleTenantOption) (*usage.TenantUsage, error) {
	return cloudcompute.GetUsage(ctx, c.cloudComputeDependencies(), nameOrID, dates.Start, dates.End, options)
}

func (c *Connection) SearchServerGroups(ctx context.Context, identity string, options ...compute.ServerGroupCloudReadOption) (*compute.ServerGroupCloudSearchResult, error) {
	return cloudcompute.SearchServerGroups(ctx, c.cloudComputeDependencies(), identity, options)
}

func (c *Connection) GetServerGroup(ctx context.Context, identity string, options ...compute.ServerGroupCloudReadOption) (*compute.ServerGroupCloudGetResult, error) {
	return cloudcompute.GetServerGroup(ctx, c.cloudComputeDependencies(), identity, options)
}

func (c *Connection) GetServerConsole(ctx context.Context, identity string, options ...compute.ServerConsoleOption) (*compute.ServerConsoleResult, error) {
	return cloudcompute.GetServerConsole(ctx, c.cloudComputeDependencies(), identity, options)
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
	return cloudserver.AddSecurityGroups(ctx, c.cloudServerDependencies(), server, groups)
}

// RemoveServerSecurityGroups maps remove_server_security_groups with the same
// lookup phase as AddServerSecurityGroups. A 404 from one removeSecurityGroup
// action is tolerated: the remaining groups are still attempted and the result
// is false. Any other failure stops the sequence.
func (c *Connection) RemoveServerSecurityGroups(ctx context.Context, server string, groups ...string) (bool, error) {
	return cloudserver.RemoveSecurityGroups(ctx, c.cloudServerDependencies(), server, groups)
}

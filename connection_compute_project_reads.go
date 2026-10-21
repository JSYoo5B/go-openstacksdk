package openstack

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/limits"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/usage"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

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

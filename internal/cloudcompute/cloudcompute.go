// Package cloudcompute implements the cloud-layer compute helpers of
// Connection that are not server reads or actions: host aggregate update,
// delete and search, the Nova limits and tenant usage reads of a project, and
// the cloud server group and console reads. Connection supplies its guard,
// lazy Nova services, location and project lookups through Dependencies, so
// this package never imports the root.
package cloudcompute

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
	computeapi "github.com/JSYoo5B/go-openstacksdk/compute/v2"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/limits"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/usage"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudserver"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

// Dependencies are the Connection facilities the compute helpers use.
type Dependencies struct {
	// Guard checks the context and that the Connection is usable before any
	// service is resolved.
	Guard func(context.Context) error
	// ComputeV2 is the lazy Nova v2 service.
	ComputeV2 func(context.Context) (*computeapi.Service, error)
	// Compute is the lazy high-level compute service.
	Compute func(context.Context) (*compute.Service, error)
	// Location is the current cloud, region and project scope.
	Location func() (resource.CloudLocation, error)
	// FindProjectID is the strict Keystone ID-or-name project lookup; its
	// errors are already wrapped with the operation name.
	FindProjectID func(ctx context.Context, operation, nameOrID string) (string, error)
	// ProjectLimits opens the Nova limits scope of a project.
	ProjectLimits func(context.Context, resource.Ref) (*limits.ProjectLimitsScope, error)
}

// UpdateAggregate implements Connection.UpdateAggregate.
func UpdateAggregate(ctx context.Context, deps Dependencies, nameOrID string, options []aggregates.UpdateRecordOption) (*aggregates.AggregateRecord, error) {
	fail := func(err error) (*aggregates.AggregateRecord, error) {
		return nil, request.Wrap("UpdateAggregate", "aggregate", err)
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	service, err := deps.ComputeV2(ctx)
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

// DeleteAggregate implements Connection.DeleteAggregate.
func DeleteAggregate(ctx context.Context, deps Dependencies, nameOrID string) (bool, error) {
	fail := func(err error) (bool, error) {
		return false, request.Wrap("DeleteAggregate", "aggregate", err)
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	if nameOrID == "" {
		return fail(fmt.Errorf("%w: aggregate name or ID is required", resource.ErrInvalidOption))
	}
	service, err := deps.ComputeV2(ctx)
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

// AggregateSearchResult has the fields of the root CloudAggregateSearchResult
// in the same order, so a pointer converts to it directly.
type AggregateSearchResult struct {
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

// SearchAggregates implements Connection.SearchAggregates.
func SearchAggregates(ctx context.Context, deps Dependencies, nameOrID string, options []compute.CloudServerReadOption) (*AggregateSearchResult, error) {
	wrap := func(err error) error {
		return request.Wrap("SearchAggregates", "aggregates", cloudread.ContextError(ctx, err))
	}
	cfg, err := cloudserver.ApplyReadOptions(options)
	if err != nil {
		return nil, wrap(err)
	}
	if len(cfg.List) > 0 || cfg.Bare || len(cfg.Address) > 0 {
		return nil, wrap(fmt.Errorf("%w: only the filter options apply to aggregates", resource.ErrInvalidOption))
	}
	if err := deps.Guard(ctx); err != nil {
		return nil, wrap(err)
	}
	service, err := deps.ComputeV2(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	location, err := deps.Location()
	if err != nil {
		return nil, wrap(err)
	}
	result := &AggregateSearchResult{}
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

// GetLimits implements Connection.GetComputeLimits.
func GetLimits(ctx context.Context, deps Dependencies, nameOrID string, options []limits.GetOption) (*limits.LimitsResource, error) {
	if err := ctx.Err(); err != nil {
		return nil, request.Wrap("GetComputeLimits", "compute limits", err)
	}
	if nameOrID == "" {
		service, err := deps.ComputeV2(ctx)
		if err != nil {
			return nil, request.Wrap("GetComputeLimits", "compute limits", err)
		}
		return service.Limits.Fetch(ctx, options...)
	}
	id, err := deps.FindProjectID(ctx, "GetComputeLimits", nameOrID)
	if err != nil {
		return nil, err
	}
	scope, err := deps.ProjectLimits(ctx, resource.ID(id))
	if err != nil {
		return nil, err
	}
	return scope.Get(ctx, options...)
}

// ParseUsageDate implements ParseComputeUsageDate.
func ParseUsageDate(value string) (time.Time, error) {
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

// GetUsage implements Connection.GetComputeUsage; a zero start or end is not
// sent.
func GetUsage(ctx context.Context, deps Dependencies, nameOrID string, start, end time.Time, options []usage.SingleTenantOption) (*usage.TenantUsage, error) {
	fail := func(err error) (*usage.TenantUsage, error) {
		return nil, request.Wrap("GetComputeUsage", "compute usage", err)
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	if nameOrID == "" {
		return fail(fmt.Errorf("%w: project name or ID is required", resource.ErrInvalidOption))
	}
	id, err := deps.FindProjectID(ctx, "GetComputeUsage", nameOrID)
	if err != nil {
		return nil, err
	}
	service, err := deps.ComputeV2(ctx)
	if err != nil {
		return fail(err)
	}
	var opts usage.SingleTenantOpts
	if !start.IsZero() {
		start := start.UTC()
		opts.Start = &start
	}
	if !end.IsZero() {
		end := end.UTC()
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

func groupContext(ctx context.Context, deps Dependencies) (*compute.Service, *resource.CloudLocation, error) {
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
	return service, &location, nil
}

// withGroupLocation appends an option that fills the current location when
// the caller did not choose one.
func withGroupLocation(options []compute.ServerGroupCloudReadOption, location *resource.CloudLocation) []compute.ServerGroupCloudReadOption {
	return append(slices.Clone(options), func(cfg *compute.ServerGroupCloudReadOpts) error {
		if cfg.CurrentLocation == nil {
			v := location.Clone()
			cfg.CurrentLocation = &v
		}
		return nil
	})
}

// SearchServerGroups implements Connection.SearchServerGroups.
func SearchServerGroups(ctx context.Context, deps Dependencies, identity string, options []compute.ServerGroupCloudReadOption) (*compute.ServerGroupCloudSearchResult, error) {
	wrap := func(err error) error {
		return request.Wrap("SearchServerGroups", "compute", cloudread.ContextError(ctx, err))
	}
	service, location, err := groupContext(ctx, deps)
	if err != nil {
		return nil, wrap(err)
	}
	value, err := service.SearchServerGroups(ctx, identity, withGroupLocation(options, location)...)
	return value, wrap(err)
}

// GetServerGroup implements Connection.GetServerGroup.
func GetServerGroup(ctx context.Context, deps Dependencies, identity string, options []compute.ServerGroupCloudReadOption) (*compute.ServerGroupCloudGetResult, error) {
	wrap := func(err error) error {
		return request.Wrap("GetServerGroup", "compute", cloudread.ContextError(ctx, err))
	}
	service, location, err := groupContext(ctx, deps)
	if err != nil {
		return nil, wrap(err)
	}
	value, err := service.GetServerGroup(ctx, identity, withGroupLocation(options, location)...)
	return value, wrap(err)
}

// GetServerConsole implements Connection.GetServerConsole.
func GetServerConsole(ctx context.Context, deps Dependencies, identity string, options []compute.ServerConsoleOption) (*compute.ServerConsoleResult, error) {
	wrap := func(err error) error {
		return request.Wrap("GetServerConsole", "compute", cloudread.ContextError(ctx, err))
	}
	if err := deps.Guard(ctx); err != nil {
		return nil, wrap(err)
	}
	service, err := deps.Compute(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	value, err := service.GetServerConsole(ctx, identity, slices.Clone(options)...)
	return value, wrap(err)
}

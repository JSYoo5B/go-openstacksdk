package openstack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
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
}

func (c *Connection) cloudServerExpand(ctx context.Context, result *CloudServerResult, cfg CloudServerReadOpts) error {
	if cfg.Bare || result == nil || result.Server == nil {
		return nil
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

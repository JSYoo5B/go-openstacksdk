package openstack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

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
	server, err := cloudServerFromRecord(record)
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
		server, err := cloudServerFromRecord(row)
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
	location := resource.CloudLocation{}
	body, err := cloudServerBase(result.Server, result.Record, location)
	if err != nil {
		return nil, err
	}
	if err := cloudServerOverlayInterfaces(body, result.Interfaces); err != nil {
		return nil, err
	}
	return json.Marshal(body)
}

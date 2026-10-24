package openstack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

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

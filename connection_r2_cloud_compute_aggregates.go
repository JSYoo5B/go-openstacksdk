package openstack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
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

func cloudAggregateTime(value time.Time) json.RawMessage {
	if value.IsZero() {
		return json.RawMessage("null")
	}
	raw, _ := json.Marshal(value.UTC().Format("2006-01-02T15:04:05.000000"))
	return raw
}

// cloudAggregateView builds the Source attribute view the cloud filters see:
// Python attribute names, deleted as is_deleted, and null for unset times.
func cloudAggregateView(value *aggregates.Aggregate) (json.RawMessage, error) {
	hosts := value.Hosts
	if hosts == nil {
		hosts = []string{}
	}
	metadata := value.Metadata
	if metadata == nil {
		metadata = map[string]string{}
	}
	view := map[string]any{
		"id": value.ID, "name": value.Name, "uuid": value.UUID, "availability_zone": value.AvailabilityZone,
		"hosts": hosts, "metadata": metadata, "is_deleted": value.Deleted,
		"created_at": cloudAggregateTime(value.CreatedAt), "updated_at": cloudAggregateTime(value.UpdatedAt),
		"deleted_at": cloudAggregateTime(value.DeletedAt),
	}
	return json.Marshal(view)
}

// SearchAggregates maps search_aggregates: list every host aggregate, then apply
// the exact/glob name-or-ID phase and the local dictionary or JMESPath filters.
// No filter is sent to Nova. The filtered view is rebuilt from the typed Go
// aggregate (see the package notes for what that loses). Only the filter
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
	result := &CloudAggregateSearchResult{}
	var views []json.RawMessage
	for row, err := range service.Aggregates.List(ctx) {
		if err != nil {
			return result, wrap(err)
		}
		view, err := cloudAggregateView(row)
		if err != nil {
			return result, wrap(err)
		}
		result.Inventory = append(result.Inventory, row)
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

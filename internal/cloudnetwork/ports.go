package cloudnetwork

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ListPorts implements Connection.ListPorts.
func ListPorts(ctx context.Context, deps Dependencies, options ...ports.RecordListOption) ([]*ports.PortRecord, error) {
	wrap := func(err error) error { return request.Wrap("ListPorts", "ports", cloudread.ContextError(ctx, err)) }
	if err := deps.Guard(ctx); err != nil {
		return nil, wrap(err)
	}
	service, err := deps.NetworkV2(ctx)
	if err != nil {
		if cause := cloudread.Context(ctx); cause != nil {
			return nil, wrap(errors.Join(err, cause))
		}
		if deps.MissingCatalogEndpoint(err) {
			return make([]*ports.PortRecord, 0), nil
		}
		return nil, wrap(err)
	}
	location, err := deps.Location()
	if err != nil {
		return nil, wrap(err)
	}
	if _, err := location.ForResource(nil, nil); err != nil {
		return nil, wrap(err)
	}
	records, err := service.Ports.AllRecords(ctx, slices.Clone(options)...)
	if err != nil {
		return nil, wrap(err)
	}
	for _, record := range records {
		if err := cloudread.Context(ctx); err != nil {
			return nil, wrap(err)
		}
		actual, err := location.ForResource(record.Resource.Body["project_id"], record.Resource.Body["availability_zone"])
		if err != nil {
			return nil, wrap(err)
		}
		record.Resource.Body["location"] = actual
	}
	return records, wrap(cloudread.Context(ctx))
}

// SearchPorts implements Connection.SearchPorts.
func SearchPorts(ctx context.Context, deps Dependencies, nameOrID string, options ...ports.RecordSearchOption) (*ports.SearchResult, error) {
	wrap := func(err error) error { return request.Wrap("SearchPorts", "ports", cloudread.ContextError(ctx, err)) }
	if err := deps.Guard(ctx); err != nil {
		return nil, wrap(err)
	}
	service, err := deps.NetworkV2(ctx)
	if err != nil {
		if cause := cloudread.Context(ctx); cause != nil {
			return nil, wrap(errors.Join(err, cause))
		}
		if deps.MissingCatalogEndpoint(err) {
			return searchAbsentPorts(ctx, nameOrID, slices.Clone(options))
		}
		return nil, wrap(err)
	}
	location, err := deps.Location()
	if err != nil {
		return nil, wrap(err)
	}
	owned := append(slices.Clone(options), ports.WithRecordSearchCloudLocation(location))
	result, err := service.Ports.SearchRecords(ctx, nameOrID, owned...)
	return result, wrap(err)
}

func searchAbsentPorts(ctx context.Context, nameOrID string, options []ports.RecordSearchOption) (*ports.SearchResult, error) {
	wrap := func(err error) error { return request.Wrap("SearchPorts", "ports", cloudread.ContextError(ctx, err)) }
	cfg := ports.RecordSearchOpts{}
	for _, apply := range options {
		if err := cloudread.Context(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(fmt.Errorf("%w: nil port search option", resource.ErrInvalidOption))
		}
		var next ports.RecordSearchOpts
		_ = ports.WithRecordSearchOpts(cfg)(&next)
		if err := errors.Join(apply(&next), cloudread.Context(ctx)); err != nil {
			return nil, wrap(err)
		}
		_ = ports.WithRecordSearchOpts(next)(&cfg)
	}
	result := &ports.SearchResult{Inventory: make([]*ports.PortRecord, 0)}
	selected, err := cloudfilter.SelectLoaded(make([]json.RawMessage, 0), nameOrID, cfg.Filters, func() error { return cloudread.Context(ctx) })
	if err != nil {
		return result, wrap(fmt.Errorf("%w: local port search: %w", resource.ErrInvalidOption, err))
	}
	result.Value = bytes.Clone(selected.Value)
	if !selected.Expression {
		result.Ports = make([]*ports.PortRecord, 0)
	}
	return result, wrap(cloudread.Context(ctx))
}

// ListPortValues implements Connection.ListPortValues.
func ListPortValues(ctx context.Context, deps Dependencies, options ...ports.SourceCloudListOption) (*ports.SourceCloudListResult, error) {
	wrap := func(err error) error {
		return request.Wrap("ListPortValues", "ports", cloudread.ContextError(ctx, err))
	}
	if err := deps.Guard(ctx); err != nil {
		return nil, wrap(err)
	}
	service, err := deps.NetworkV2(ctx)
	if err != nil {
		if cause := cloudread.Context(ctx); cause != nil {
			return nil, wrap(errors.Join(err, cause))
		}
		if deps.MissingCatalogEndpoint(err) {
			return &ports.SourceCloudListResult{Values: make([]any, 0), JSON: json.RawMessage("[]"), Ports: make([]*ports.PortRecord, 0), Inventory: make([]*ports.PortRecord, 0)}, nil
		}
		return nil, wrap(err)
	}
	location, err := deps.Location()
	if err != nil {
		return nil, wrap(err)
	}
	owned := append(slices.Clone(options), ports.WithSourceCloudListLocation(location))
	result, err := service.Ports.ListSourceValues(ctx, owned...)
	return result, wrap(err)
}

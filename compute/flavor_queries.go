package compute

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// FlavorQueryOpts controls Cloud flavor operations. Nil GetExtra means false
// for AllFlavors and true for SearchFlavors/GetFlavor. Filters are local Cloud
// selection for SearchFlavors and member/find arguments for GetFlavor;
// AllFlavors has no filters argument.
type FlavorQueryOpts struct {
	GetExtra     *bool
	Filters      *json.RawMessage
	Microversion *string
}
type FlavorQueryOption = request.Option[FlavorQueryOpts]

func cloneFlavorQueryOptions(value FlavorQueryOpts) FlavorQueryOpts {
	if value.GetExtra != nil {
		owned := *value.GetExtra
		value.GetExtra = &owned
	}
	if value.Filters != nil {
		owned := json.RawMessage(bytes.Clone(*value.Filters))
		value.Filters = &owned
	}
	if value.Microversion != nil {
		owned := *value.Microversion
		value.Microversion = &owned
	}
	return value
}
func WithFlavorQueryOptions(value FlavorQueryOpts) FlavorQueryOption {
	owned := cloneFlavorQueryOptions(value)
	return func(config *request.Config[FlavorQueryOpts]) error {
		config.Options = cloneFlavorQueryOptions(owned)
		return nil
	}
}
func WithFlavorQueryExtraSpecs(value bool) FlavorQueryOption {
	return func(config *request.Config[FlavorQueryOpts]) error {
		owned := value
		config.Options.GetExtra = &owned
		return nil
	}
}

// WithFlavorQueryFilters owns raw JSON. Nil clears the option; explicit null
// remains present. Search preserves dictionary order and expression values.
func WithFlavorQueryFilters(value json.RawMessage) FlavorQueryOption {
	owned := bytes.Clone(value)
	return func(config *request.Config[FlavorQueryOpts]) error {
		config.Options.Filters = nil
		if owned != nil {
			raw := json.RawMessage(bytes.Clone(owned))
			config.Options.Filters = &raw
		}
		return nil
	}
}
func WithFlavorQueryExpression(value string) FlavorQueryOption {
	raw, _ := json.Marshal(value)
	return WithFlavorQueryFilters(raw)
}
func WithFlavorQueryMicroversion(value string) FlavorQueryOption {
	return func(config *request.Config[FlavorQueryOpts]) error {
		owned := value
		config.Options.Microversion = &owned
		return nil
	}
}
func WithFlavorQueryHeader(key, value string) FlavorQueryOption {
	return request.WithHeader[FlavorQueryOpts](key, value)
}

// FlavorQueryResult retains consumed records, including partial work on error.
// Value and Flavors represent completed collection/selection only. Expressions
// return arbitrary JSON in Value without inventing a Flavors association.
type FlavorQueryResult struct {
	Value     json.RawMessage
	Flavors   []*flavors.FlavorRecord
	Inventory []*flavors.FlavorRecord
}

func (p *flavorRecordWorkflow) prepareQuery(options []FlavorQueryOption) (request.Config[FlavorQueryOpts], error) {
	config, err := cloudread.ApplyReadOptions(p.ctx, FlavorQueryOpts{}, options, func(c *request.Config[FlavorQueryOpts]) {
		c.Options = cloneFlavorQueryOptions(c.Options)
		cloudread.OwnReadConfig(c)
	}, p.check)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	if err == nil && config.Options.Filters != nil {
		raw := *config.Options.Filters
		if !utf8.Valid(raw) || !json.Valid(raw) {
			err = invalid("flavor filters must be UTF-8 JSON")
		}
	}
	return config, errors.Join(err, p.check(p.ctx))
}
func flavorQueryExtra(value FlavorQueryOpts, defaultValue bool) bool {
	if value.GetExtra != nil {
		return *value.GetExtra
	}
	return defaultValue
}
func flavorQueryValues(rows []*flavors.FlavorRecord) ([]json.RawMessage, json.RawMessage, error) {
	views := make([]json.RawMessage, len(rows))
	for index, row := range rows {
		if row == nil || row.Resource == nil {
			return nil, nil, invalid("flavor inventory resource is required")
		}
		raw, err := json.Marshal(row.Resource)
		if err != nil {
			return nil, nil, err
		}
		views[index] = raw
	}
	value, err := json.Marshal(views)
	return views, value, err
}
func (p *flavorRecordWorkflow) queryInventory(config request.Config[FlavorQueryOpts], defaultExtra bool) (*FlavorQueryResult, error) {
	result := &FlavorQueryResult{Inventory: []*flavors.FlavorRecord{}}
	options := []flavors.FlavorListOption{flavors.WithFlavorListDetails(true), flavors.WithFlavorListExtraSpecs(flavorQueryExtra(config.Options, defaultExtra))}
	if config.Options.Microversion != nil {
		options = append(options, flavors.WithFlavorListMicroversion(*config.Options.Microversion))
	}
	for key, value := range config.Headers {
		options = append(options, flavors.WithFlavorListHeader(key, value))
	}
	for record, readErr := range p.api.ListRecords(p.ctx, options...) {
		if record != nil {
			result.Inventory = append(result.Inventory, p.locate(record))
		}
		if err := errors.Join(readErr, p.check(p.ctx)); err != nil {
			return result, err
		}
	}
	return result, p.check(p.ctx)
}
func flavorCloudFailure(operation string, err error) error {
	return request.Wrap(operation, "flavors", err)
}

// AllFlavors implements Cloud list_flavors: eagerly collect all detailed
// flavors, with extra specs disabled by default. ListFlavors remains lazy.
func (s *Service) AllFlavors(ctx context.Context, options ...FlavorQueryOption) (*FlavorQueryResult, error) {
	owned := slices.Clone(options)
	p, err := s.captureFlavorRecords(ctx)
	if err != nil {
		return nil, flavorCloudFailure("AllFlavors", err)
	}
	config, err := p.prepareQuery(owned)
	if err == nil && config.Options.Filters != nil {
		err = invalid("Cloud flavor list has no filters argument")
	}
	if err != nil {
		return nil, flavorCloudFailure("AllFlavors", err)
	}
	result, err := p.queryInventory(config, false)
	if err == nil {
		_, result.Value, err = flavorQueryValues(result.Inventory)
	}
	if err == nil {
		err = p.check(p.ctx)
	}
	if err == nil {
		result.Flavors = slices.Clone(result.Inventory)
	} else {
		result.Value = nil
	}
	return result, flavorCloudFailure("AllFlavors", err)
}

// SearchFlavors finishes the complete detailed inventory and requested specs
// enrichment before applying name/glob and local dictionary/JMESPath filters.
// Extra specs default true, including rows later excluded by selection.
func (s *Service) SearchFlavors(ctx context.Context, nameOrID string, options ...FlavorQueryOption) (*FlavorQueryResult, error) {
	owned := slices.Clone(options)
	p, err := s.captureFlavorRecords(ctx)
	if err != nil {
		return nil, flavorCloudFailure("SearchFlavors", err)
	}
	config, err := p.prepareQuery(owned)
	if err != nil {
		return nil, flavorCloudFailure("SearchFlavors", err)
	}
	result, err := p.queryInventory(config, true)
	if err != nil {
		return result, flavorCloudFailure("SearchFlavors", err)
	}
	views, _, err := flavorQueryValues(result.Inventory)
	if err != nil {
		return result, flavorCloudFailure("SearchFlavors", err)
	}
	selected, err := cloudfilter.Select(views, nameOrID, config.Options.Filters, func() error { return p.check(p.ctx) })
	if err != nil {
		return result, flavorCloudFailure("SearchFlavors", fmt.Errorf("%w: flavor local search: %w", resource.ErrInvalidOption, err))
	}
	if err = p.check(p.ctx); err != nil {
		return result, flavorCloudFailure("SearchFlavors", err)
	}
	result.Value = bytes.Clone(selected.Value)
	if !selected.Expression {
		result.Flavors = make([]*flavors.FlavorRecord, len(selected.Indices))
		for index, source := range selected.Indices {
			result.Flavors[index] = result.Inventory[source]
		}
	}
	return result, nil
}

// GetFlavor implements Cloud's direct find delegation, defaulting enrichment
// and ignored missing to true. Its deprecated filters are find arguments, not
// Cloud expressions: falsey values mean no filters, truthy nonobjects fail.
func (s *Service) GetFlavor(ctx context.Context, nameOrID string, options ...FlavorQueryOption) (*flavors.FlavorRecord, error) {
	owned := slices.Clone(options)
	fail := func(record *flavors.FlavorRecord, err error) (*flavors.FlavorRecord, error) {
		return record, flavorCloudFailure("GetFlavor", err)
	}
	p, err := s.captureFlavorRecords(ctx)
	if err != nil {
		return fail(nil, err)
	}
	config, err := p.prepareQuery(owned)
	if err != nil {
		return fail(nil, err)
	}
	findOptions := []flavors.FlavorFindOption{flavors.WithFlavorFindIgnoreMissing(true), flavors.WithFlavorFindExtraSpecs(flavorQueryExtra(config.Options, true))}
	if config.Options.Microversion != nil {
		findOptions = append(findOptions, flavors.WithFlavorFindMicroversion(*config.Options.Microversion))
	}
	for key, value := range config.Headers {
		findOptions = append(findOptions, flavors.WithFlavorFindHeader(key, value))
	}
	if config.Options.Filters != nil {
		raw := *config.Options.Filters
		truthy, err := cloudfilter.PythonTruthy(raw)
		if err != nil {
			return fail(nil, err)
		}
		if truthy {
			members, err := cloudfilter.ObjectMembers(raw)
			if err != nil {
				return fail(nil, fmt.Errorf("%w: Cloud get flavor filters must be a dictionary: %w", resource.ErrInvalidOption, err))
			}
			for _, member := range members {
				switch member.Key {
				case "name_or_id", "ignore_missing", "get_extra_specs":
					return fail(nil, invalid("Cloud flavor filter conflicts with an explicit find argument"))
				}
				findOptions = append(findOptions, flavors.WithFlavorFindFilter(member.Key, json.RawMessage(bytes.Clone(member.Value))))
			}
		}
	}
	record, err := p.api.FindFlavor(p.ctx, nameOrID, findOptions...)
	return fail(p.locate(record), errors.Join(err, p.check(p.ctx)))
}

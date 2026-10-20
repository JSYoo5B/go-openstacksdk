package endpointgroups

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// CloudCreate implements the cloud layer create_endpoint_group(name, filters,
// description=None): name and filters are always sent and description only
// when it is nonempty. Further CreateOption values are applied first.
func (a *API) CloudCreate(ctx context.Context, name string, filters any, description string, options ...CreateOption) (*Record, error) {
	owned := slices.Clone(options)
	owned = append(owned, WithCreateField("name", name), WithCreateField("filters", filters))
	if description != "" {
		owned = append(owned, WithCreateField("description", description))
	}
	return a.create(ctx, "CloudCreate", owned)
}

// CloudDelete implements the cloud layer delete_endpoint_group(id): Find with
// ignore_missing first, false without DELETE when nothing matches, and
// otherwise delete_endpoint_group(id) with its default ignore_missing, true on
// success. Python logs a failed DELETE and returns False; Go returns false
// together with that error.
func (a *API) CloudDelete(ctx context.Context, id string, options ...ReadOption) (bool, error) {
	const op = "CloudDelete"
	p, err := a.capture(ctx)
	if err != nil {
		return false, wrap(ctx, op, err)
	}
	policy, err := applyOptions(ctx, p.guard, ReadOpts{}, copyRead, slices.Clone(options))
	if err != nil {
		return false, wrap(ctx, op, err)
	}
	found, err := a.Find(ctx, id)
	if err != nil {
		return false, wrap(ctx, op, err)
	}
	if found == nil {
		return false, nil
	}
	// Python deletes the literal argument, not the ID of the found row.
	if err := a.Delete(ctx, id, WithDeleteHeaders(policy.Headers)); err != nil {
		return false, wrap(ctx, op, err)
	}
	return true, nil
}

// SearchResult is a completed cloud search. Value is the selected JSON array
// of Python views, or the arbitrary JMESPath result. EndpointGroups keeps the
// selected records in order and is nil for an expression. Inventory is every
// listed record, including the rows read before a failure.
type SearchResult struct {
	Value          json.RawMessage
	EndpointGroups []*Record
	Inventory      []*Record
}

// CloudSearch implements the cloud layer search_endpoint_groups(name_or_id,
// filters): the complete unfiltered List, then _utils._filter_list with exact
// or glob ID/name matching and nested dictionary or JMESPath filters on the
// Python views.
func (a *API) CloudSearch(ctx context.Context, nameOrID string, options ...SearchOption) (*SearchResult, error) {
	const op = "CloudSearch"
	p, err := a.capture(ctx)
	if err != nil {
		return nil, wrap(ctx, op, err)
	}
	policy, err := applyOptions(ctx, p.guard, SearchOpts{}, copySearch, slices.Clone(options))
	if err != nil {
		return nil, wrap(ctx, op, err)
	}
	result := &SearchResult{}
	var rows []json.RawMessage
	for v, err := range a.List(ctx, WithListHeaders(policy.Headers)) {
		if err != nil {
			return result, wrap(ctx, op, err)
		}
		row, err := json.Marshal(v.Resource.Body)
		if err != nil {
			return result, wrap(ctx, op, err)
		}
		result.Inventory = append(result.Inventory, v)
		rows = append(rows, row)
	}
	selected, err := cloudfilter.Select(rows, nameOrID, policy.Filters, func() error { return p.guard(ctx) })
	if err != nil {
		return result, wrap(ctx, op, fmt.Errorf("%w: local endpoint group search: %w", resource.ErrInvalidOption, err))
	}
	if !selected.Expression {
		result.EndpointGroups = make([]*Record, len(selected.Indices))
		for i, index := range selected.Indices {
			result.EndpointGroups[i] = result.Inventory[index]
		}
	}
	result.Value = bytes.Clone(selected.Value)
	return result, wrap(ctx, op, p.guard(ctx))
}

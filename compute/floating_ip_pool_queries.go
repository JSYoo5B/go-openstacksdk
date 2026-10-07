package compute

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"gophercloudsdk/internal/cloudfilter"
	"gophercloudsdk/resource"
)

// ListFloatingIPPools always reads legacy Nova, independently of IP source.
// Only name is exposed in each logical pool; Pages retain the full responses.
func (s *Service) ListFloatingIPPools(ctx context.Context, options ...FloatingIPQueryOption) (*FloatingIPPoolQueryResult, error) {
	p, err := s.captureFloatingIPQuery(ctx, options)
	if err != nil {
		return nil, err
	}
	defer p.cancel()
	if p.options.Filters != nil {
		return nil, invalid("pool list has no filters; use SearchFloatingIPPools")
	}
	return p.pools()
}
func (s *Service) SearchFloatingIPPools(ctx context.Context, input SearchFloatingIPPoolsRequest, options ...FloatingIPQueryOption) (*FloatingIPPoolQueryResult, error) {
	p, err := s.captureFloatingIPQuery(ctx, options)
	if err != nil {
		return nil, err
	}
	defer p.cancel()
	result, err := p.pools()
	if err != nil {
		return result, err
	}
	views := make([]json.RawMessage, len(result.Pools))
	for i, pool := range result.Pools {
		views[i], err = json.Marshal(pool)
		if err != nil {
			return result, err
		}
	}
	selected, err := cloudfilter.Select(views, input.Name, p.options.Filters, func() error { return p.state.check(p.ctx) })
	if err != nil {
		result.Value = nil
		result.Pools = nil
		return result, floatingIPQueryInputError("pool search", err)
	}
	pools := result.Pools
	result.Pools = nil
	if !selected.Expression {
		result.Pools = make([]*resource.RawResource, len(selected.Indices))
		for i, index := range selected.Indices {
			result.Pools[i] = pools[index]
		}
	}
	result.Value = bytes.Clone(selected.Value)
	return result, p.state.check(p.ctx)
}
func (p *floatingIPQueryState) pools() (*FloatingIPPoolQueryResult, error) {
	result := &FloatingIPPoolQueryResult{}
	client, err := p.novaClient()
	if err != nil {
		return result, err
	}
	rows, pages, err := p.collect(FloatingIPNova, client, "os-floating-ip-pools", "floating_ip_pools", floatingIPQueryParameters{})
	result.Pages = pages
	if err != nil {
		result.Failure = queryFailure(FloatingIPNova, err)
		return result, err
	}
	pools := make([]*resource.RawResource, len(rows))
	for i, row := range rows {
		if err := p.state.check(p.ctx); err != nil {
			return result, err
		}
		name, present := row.wire.Body["name"]
		if !present {
			err = row.origin.Fail(invalid("Nova floating IP pool lacks a name field"))
			result.Failure = queryFailure(FloatingIPNova, err)
			return result, err
		}
		pools[i] = row.wire.Clone()
		pools[i].Body = map[string]json.RawMessage{"name": bytes.Clone(name)}
	}
	result.Value, err = json.Marshal(pools)
	if err == nil {
		result.Pools = pools
	}
	return result, errors.Join(err, p.state.check(p.ctx))
}

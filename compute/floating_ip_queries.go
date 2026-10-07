package compute

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"regexp"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudfilter"
	"github.com/JSYoo5B/gophercloudsdk/internal/cloudread"
	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// ListFloatingIPs follows cloud's filter-specific 404 policy. Dictionary
// filters are Neutron parameters; Nova accepts only an unfiltered list.
func (s *Service) ListFloatingIPs(ctx context.Context, options ...FloatingIPQueryOption) (*FloatingIPQueryResult, error) {
	p, err := s.captureFloatingIPQuery(ctx, options)
	if err != nil {
		return nil, err
	}
	defer p.cancel()
	return p.list(p.options.Filters)
}
func (s *Service) SearchFloatingIPs(ctx context.Context, input SearchFloatingIPsRequest, options ...FloatingIPQueryOption) (*FloatingIPQueryResult, error) {
	p, err := s.captureFloatingIPQuery(ctx, options)
	if err != nil {
		return nil, err
	}
	defer p.cancel()
	return p.search(input.ID)
}
func (p *floatingIPQueryState) list(filters *json.RawMessage) (*FloatingIPQueryResult, error) {
	raw, object, truthy, err := floatingIPQueryFilter(filters)
	if err != nil {
		return nil, err
	}
	if p.source != FloatingIPNeutron && truthy {
		return nil, invalid("Nova floating IP list does not accept server filters; use SearchFloatingIPs")
	}
	backend, client, err := p.backend()
	result := &FloatingIPQueryResult{Backend: backend}
	if err != nil {
		return result, err
	}
	if backend == FloatingIPNeutron {
		if truthy && !object {
			return result, invalid("Neutron floating IP list filters must be a dictionary")
		}
		if !truthy {
			raw = nil
		}
		params, err := prepareFloatingIPParameters(raw)
		if err != nil {
			return result, err
		}
		rows, pages, err := p.collect(backend, client, "floatingips", "floatingips", params)
		result.Pages = pages
		if err == nil {
			result.FloatingIPs, err = p.records(backend, client, rows, params.local)
			if err == nil {
				result.Value, err = queryValues(result.FloatingIPs)
			}
			result.Failure = queryFailure(backend, err)
			return result, errors.Join(err, p.state.check(p.ctx))
		}
		result.Failure = queryFailure(backend, err)
		if !queryNotFound(err) {
			return result, err
		}
		if truthy {
			result.Value = json.RawMessage("[]")
			result.FloatingIPs = []*FloatingIPRecord{}
			result.SuppressedNotFound = err
			return result, nil
		}
		result.FallbackError = err
		backend = FloatingIPNova
	}
	result.Backend = backend
	if truthy {
		return result, invalid("Nova floating IP list does not accept server filters; use SearchFloatingIPs")
	}
	client, err = p.novaClient()
	if err != nil {
		return result, err
	}
	rows, pages, err := p.collect(backend, client, "os-floating-ips", "floating_ips", floatingIPQueryParameters{})
	result.Pages = append(result.Pages, pages...)
	if err != nil {
		result.Failure = queryFailure(backend, err)
		if queryNotFound(err) {
			result.Value = json.RawMessage("[]")
			result.FloatingIPs = []*FloatingIPRecord{}
			result.SuppressedNotFound = err
			return result, nil
		}
		return result, err
	}
	result.FloatingIPs, err = p.records(backend, client, rows, nil)
	if err == nil {
		result.Value, err = queryValues(result.FloatingIPs)
	}
	if err != nil {
		result.Failure = queryFailure(backend, err)
	}
	return result, errors.Join(err, p.state.check(p.ctx))
}

func (p *floatingIPQueryState) search(id string) (*FloatingIPQueryResult, error) {
	raw, object, _, err := floatingIPQueryFilter(p.options.Filters)
	if err != nil {
		return nil, err
	}
	backend, client, err := p.backend()
	if err != nil {
		return &FloatingIPQueryResult{Backend: backend}, err
	}
	if backend == FloatingIPNeutron && object {
		params, err := prepareFloatingIPParameters(raw)
		if err != nil {
			return &FloatingIPQueryResult{Backend: backend}, err
		}
		rows, pages, err := p.collect(backend, client, "floatingips", "floatingips", params)
		result := &FloatingIPQueryResult{Backend: backend, Pages: pages, Failure: queryFailure(backend, err)}
		if err != nil {
			return result, err
		}
		result.FloatingIPs, err = p.records(backend, client, rows, params.local)
		if err == nil {
			result.Value, err = queryValues(result.FloatingIPs)
		}
		if err != nil {
			result.Failure = queryFailure(backend, err)
		}
		return result, errors.Join(err, p.state.check(p.ctx))
	}
	result, err := p.list(nil)
	if err != nil {
		return result, err
	}
	views := make([]json.RawMessage, len(result.FloatingIPs))
	for i, record := range result.FloatingIPs {
		views[i], err = json.Marshal(record.Resource)
		if err != nil {
			return result, err
		}
	}
	selected, err := cloudfilter.Select(views, id, p.options.Filters, func() error { return p.state.check(p.ctx) })
	if err != nil {
		result.Value = nil
		result.FloatingIPs = nil
		return result, floatingIPQueryInputError("local search", err)
	}
	rows := result.FloatingIPs
	result.FloatingIPs = nil
	if !selected.Expression {
		result.FloatingIPs = make([]*FloatingIPRecord, len(selected.Indices))
		for i, index := range selected.Indices {
			result.FloatingIPs[i] = rows[index]
		}
	}
	result.Value = bytes.Clone(selected.Value)
	return result, p.state.check(p.ctx)
}

var floatingIPUUID = regexp.MustCompile(`(?i)^(?:urn:uuid:)?\{?[0-9a-f]{8}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{12}\}?$`)

func (s *Service) GetFloatingIP(ctx context.Context, input GetFloatingIPRequest, options ...FloatingIPQueryOption) (*GetFloatingIPResult, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if input.Existing != nil {
		if input.Existing.Resource == nil {
			return nil, invalid("existing floating IP resource is required")
		}
		if _, present := input.Existing.Resource.Body["id"]; !present {
			return nil, invalid("existing floating IP resource lacks an ID field")
		}
		value, err := json.Marshal(input.Existing.Resource)
		return &GetFloatingIPResult{Backend: input.Existing.Backend, FloatingIP: input.Existing, Value: value}, err
	}
	p, err := s.captureFloatingIPQuery(ctx, options)
	if err != nil {
		return nil, err
	}
	defer p.cancel()
	return p.get(input.ID)
}

func (p *floatingIPQueryState) get(id string) (*GetFloatingIPResult, error) {
	if p.options.DirectGet && floatingIPUUID.MatchString(id) {
		return p.read(id)
	}
	search, err := p.search(id)
	if search == nil {
		return nil, err
	}
	result := &GetFloatingIPResult{Backend: search.Backend, Pages: search.Pages, FallbackError: search.FallbackError, SuppressedNotFound: search.SuppressedNotFound, Failure: search.Failure}
	if err != nil {
		return result, err
	}
	selected, err := cloudfilter.First(search.Value)
	if err != nil {
		var multiple *cloudfilter.MultipleError
		if errors.As(err, &multiple) {
			err = &FloatingIPSelectionError{ID: id, Length: multiple.Length}
		} else {
			err = floatingIPQueryInputError("selection", err)
		}
		return result, err
	}
	result.Value = bytes.Clone(selected)
	if selected != nil && len(search.FloatingIPs) == 1 {
		result.FloatingIP = search.FloatingIPs[0]
	}
	return result, p.state.check(p.ctx)
}
func (s *Service) GetFloatingIPByID(ctx context.Context, input GetFloatingIPByIDRequest, options ...FloatingIPQueryOption) (*GetFloatingIPResult, error) {
	p, err := s.captureFloatingIPQuery(ctx, options)
	if err != nil {
		return nil, err
	}
	defer p.cancel()
	return p.read(input.ID)
}
func (p *floatingIPQueryState) read(id string) (*GetFloatingIPResult, error) {
	if err := resource.ID(id).Validate(); err != nil {
		return nil, err
	}
	backend, client, err := p.backend()
	result := &GetFloatingIPResult{Backend: backend}
	if err != nil {
		return result, err
	}
	path, key := "floatingips", "floatingip"
	if backend == FloatingIPNova {
		client, err = p.novaClient()
		if err != nil {
			return result, err
		}
		path, key = "os-floating-ips", "floating_ip"
	}
	response, requestErr := rest.DoJSONGuarded(p.ctx, client, p.state.check, http.MethodGet, client.ServiceURL(path, url.PathEscape(id)), nil, nil, 200)
	if response == nil {
		result.Failure = queryFailure(backend, requestErr)
		return result, requestErr
	}
	result.Observed = queryResponse(backend, response)
	wire, decodeErr := rest.Decode[resource.RawResource](response, key, func(row *resource.RawResource) *resource.Metadata { return &row.Metadata })
	if decodeErr == nil {
		result.FloatingIP = &FloatingIPRecord{Backend: backend, Wire: wire.Clone()}
		var record *FloatingIPRecord
		record, decodeErr = p.record(backend, client, wire, true)
		if record != nil {
			result.FloatingIP = record
			if backend == FloatingIPNeutron {
				if _, present := wire.Body["id"]; !present {
					record.Resource.Body["id"], _ = json.Marshal(id)
				}
			}
		}
		if decodeErr != nil {
			decodeErr = response.Fail(decodeErr)
		}
		if result.FloatingIP.Resource != nil {
			result.Value, decodeErr = json.Marshal(result.FloatingIP.Resource)
			if decodeErr != nil {
				decodeErr = response.Fail(decodeErr)
			}
		}
	}
	err = errors.Join(requestErr, decodeErr, p.state.check(p.ctx))
	result.Failure = queryFailure(backend, err)
	return result, err
}

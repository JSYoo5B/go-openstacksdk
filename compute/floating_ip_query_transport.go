package compute

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudfilter"
	"github.com/JSYoo5B/gophercloudsdk/internal/cloudlocation"
	"github.com/JSYoo5B/gophercloudsdk/internal/cloudread"
	"github.com/JSYoo5B/gophercloudsdk/internal/jsonfilter"
	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/network"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type floatingIPQueryState struct {
	state       *automaticIPState
	options     FloatingIPQueryOpts
	source      FloatingIPSource
	neutronMode bool
	ctx         context.Context
	cancel      context.CancelFunc
}

func (s *Service) captureFloatingIPQuery(ctx context.Context, options []FloatingIPQueryOption) (*floatingIPQueryState, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if s == nil || s.API == nil || s.Servers == nil || s.Servers.Collection == nil {
		return nil, invalid("floating IP query facade is required")
	}
	base := s.captureAutomaticComputeSource()
	servers, source := s.Servers, s.Servers.addressPolicy.options.source
	policy, err := prepareFloatingIPQueryOptions(options, func() error { return base(ctx) })
	if err != nil {
		return nil, err
	}
	if policy.Source != nil {
		source = *policy.Source
	}
	cancel := func() {}
	if policy.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, policy.Timeout)
	}
	ctx = rest.WithOperationSources(ctx)
	state := &automaticIPState{service: s, address: &serverAddressState{servers: servers}, outerGuard: rest.OperationGuard(ctx)}
	state.computeGuard = func(ctx context.Context) error {
		var bound error
		if state.clientGuard != nil {
			bound = state.clientGuard(ctx)
		}
		return errors.Join(base(ctx), bound)
	}
	ctx = rest.WithOperationGuard(ctx, state.computeGuard)
	return &floatingIPQueryState{state: state, options: policy, source: source, ctx: ctx, cancel: cancel}, nil
}

func (p *floatingIPQueryState) backend() (FloatingIPSource, *gophercloud.ServiceClient, error) {
	if p.source == FloatingIPNeutron {
		service, err := p.state.loadNetwork(p.ctx)
		if err != nil {
			return FloatingIPNeutron, nil, err
		}
		p.neutronMode = service != nil
		if service != nil {
			return FloatingIPNeutron, service.RawClient(), p.state.check(p.ctx)
		}
	}
	return FloatingIPNova, nil, p.state.check(p.ctx)
}
func (p *floatingIPQueryState) novaClient() (*gophercloud.ServiceClient, error) {
	policy, err := network.PrepareEnsureFloatingIPOptions(p.ctx)
	if err != nil {
		return nil, err
	}
	p.state.policy = policy
	b, err := p.state.novaBackendFor(p.ctx, false)
	if err != nil {
		return nil, err
	}
	return b.client, nil
}
func (p *floatingIPQueryState) location(client *gophercloud.ServiceClient) (resource.CloudLocation, error) {
	if err := p.state.check(p.ctx); err != nil {
		return resource.CloudLocation{}, err
	}
	if p.options.Location != nil {
		return p.options.Location.Clone(), nil
	}
	if getter := p.state.address.servers.dependencies.CloudLocation; getter != nil {
		location, err := getter()
		return location.Clone(), errors.Join(err, p.state.check(p.ctx))
	}
	id, err := cloudlocation.ProjectID(client.ProviderClient)
	return resource.CloudLocation{Project: resource.CloudProject{ID: id}}, err
}
func queryResponse(backend FloatingIPSource, response *rest.Response) *FloatingIPQueryResponse {
	if response == nil {
		return nil
	}
	return &FloatingIPQueryResponse{Backend: backend, Envelope: bytes.Clone(response.Body), Metadata: resource.Metadata{Header: response.Header.Clone(), StatusCode: response.StatusCode}}
}
func queryFailure(backend FloatingIPSource, err error) *FloatingIPQueryResponse {
	var accepted *resource.ResponseError
	if errors.As(err, &accepted) {
		return &FloatingIPQueryResponse{Backend: backend, Envelope: bytes.Clone(accepted.Body), Metadata: resource.Metadata{Header: accepted.Header.Clone(), StatusCode: accepted.StatusCode}}
	}
	native := queryHTTPFailure(err)
	if native != nil {
		return &FloatingIPQueryResponse{Backend: backend, Envelope: bytes.Clone(native.Body), Metadata: resource.Metadata{Header: native.ResponseHeader.Clone(), StatusCode: native.Actual}}
	}
	return nil
}
func queryNotFound(err error) bool {
	response := queryHTTPFailure(err)
	return availableIPNotFound(err) && response != nil && response.Actual == 404
}

type floatingIPQueryRow struct {
	wire     *resource.RawResource
	origin   *rest.Response
	record   *FloatingIPRecord
	prepared bool
}

func (p *floatingIPQueryState) collect(backend FloatingIPSource, client *gophercloud.ServiceClient, path, plural string, params floatingIPQueryParameters) ([]floatingIPQueryRow, []*FloatingIPQueryResponse, error) {
	var origin *rest.Response
	var pages []*FloatingIPQueryResponse
	spec := rest.CollectionSpec[resource.RawResource]{Client: client, Path: path, Kind: "floating IP", PluralKey: plural,
		Validate: p.state.check, SourceGuard: p.state.check, ListCodes: []int{200}, Metadata: func(row *resource.RawResource) *resource.Metadata { return &row.Metadata },
		ValidateResponse: func(response *rest.Response) error {
			origin = response
			pages = append(pages, queryResponse(backend, response))
			return nil
		},
		Paging: rest.PagePolicy[resource.RawResource]{HTTPLink: true}}
	if backend == FloatingIPNeutron {
		spec.Paging.StopOnEmptyPage = true
		spec.Paging.MarkerFallback = true
		spec.Paging.MarkerOnShortPage = true
		spec.Paging.AllowFirstLimitReduction = true
		spec.Paging.Marker = func(row *resource.RawResource) (string, error) {
			values, err := cloudfilter.RequestQueryValues(row.Body["id"])
			if err != nil {
				return "", err
			}
			if len(values) != 1 {
				return "", fmt.Errorf("floating IP pagination ID must be a single query value")
			}
			return values[0], nil
		}
	}
	var rows []floatingIPQueryRow
	for row, err := range rest.ListWithControl(p.ctx, spec, params.query, params.control) {
		if err != nil {
			return rows, pages, errors.Join(err, p.state.check(p.ctx))
		}
		entry := floatingIPQueryRow{wire: row, origin: origin}
		if backend == FloatingIPNeutron {
			// Neutron constructs and filters each Resource before following
			// continuation. A later 404 must not hide this row's error.
			values, err := p.records(backend, client, []floatingIPQueryRow{entry}, params.local)
			if err != nil {
				return rows, pages, errors.Join(err, p.state.check(p.ctx))
			}
			entry.prepared = true
			if len(values) != 0 {
				entry.record = values[0]
			}
		}
		rows = append(rows, entry)
	}
	return rows, pages, p.state.check(p.ctx)
}

func (p *floatingIPQueryState) record(backend FloatingIPSource, client *gophercloud.ServiceClient, wire *resource.RawResource, member bool) (*FloatingIPRecord, error) {
	location, err := p.location(client)
	if err != nil {
		return nil, err
	}
	if backend == FloatingIPNova {
		return normalizeQueryNovaIP(wire, p.neutronMode, p.options.Strict, location)
	}
	view := wire.Clone()
	delete(view.Body, "self")
	for _, key := range floatingIPBodyKeys {
		if _, present := view.Body[key]; !present {
			view.Body[key] = json.RawMessage("null")
		}
	}
	view.Body["name"] = bytes.Clone(view.Body["floating_ip_address"])
	if _, present := wire.Body["project_id"]; !present {
		view.Body["project_id"] = bytes.Clone(view.Body["tenant_id"])
	}
	if _, present := wire.Body["tags"]; !present {
		view.Body["tags"] = json.RawMessage("[]")
	}
	for _, key := range []string{"port_details", "tags"} {
		raw := bytes.TrimSpace(view.Body[key])
		want := byte('{')
		empty := json.RawMessage("{}")
		if key == "tags" {
			want = '['
			empty = append(append(json.RawMessage("["), raw...), ']')
		}
		if len(raw) > 0 && !bytes.Equal(raw, []byte("null")) && raw[0] != want {
			view.Body[key] = empty
		}
	}
	view.Body["revision_number"], err = jsonfilter.DescriptorIntegerJSON(view.Body["revision_number"])
	if err != nil {
		return nil, err
	}
	view.Body["if_match"] = json.RawMessage("null")
	if member && len(wire.Header.Values("If-Match")) > 0 {
		view.Body["if_match"], _ = json.Marshal(wire.Header.Values("If-Match"))
	}
	view.Body["location"], err = location.ForResource(view.Body["project_id"], nil)
	if err != nil {
		return nil, err
	}
	return &FloatingIPRecord{Backend: FloatingIPNeutron, NormalizationSource: FloatingIPNeutron, Resource: view, Wire: wire.Clone()}, nil
}

func (p *floatingIPQueryState) records(backend FloatingIPSource, client *gophercloud.ServiceClient, rows []floatingIPQueryRow, local []cloudfilter.JSONMember) ([]*FloatingIPRecord, error) {
	values := make([]*FloatingIPRecord, 0, len(rows))
	for _, row := range rows {
		if err := p.state.check(p.ctx); err != nil {
			return nil, err
		}
		if row.prepared {
			if row.record != nil {
				values = append(values, row.record)
			}
			continue
		}
		record, err := p.record(backend, client, row.wire, false)
		if err != nil {
			return nil, row.origin.Fail(err)
		}
		match := true
		for _, filter := range local {
			equal, err := floatingIPLocalMatch(filter.Value, record.Resource.Body[filter.Key])
			if err != nil {
				return nil, row.origin.Fail(err)
			}
			if !equal {
				match = false
				break
			}
		}
		if match {
			values = append(values, record)
		}
	}
	return values, p.state.check(p.ctx)
}

func floatingIPLocalMatch(want, actual json.RawMessage) (bool, error) {
	if len(bytes.TrimSpace(want)) == 0 || bytes.TrimSpace(want)[0] != '{' {
		return jsonfilter.EqualPythonJSON(want, actual)
	}
	truthy, err := cloudfilter.PythonTruthy(actual)
	if err != nil || !truthy {
		return false, err
	}
	filters, err := cloudfilter.ObjectMembers(want)
	if err != nil {
		return false, err
	}
	if len(filters) == 0 {
		return true, nil
	}
	members, err := cloudfilter.ObjectMembers(actual)
	if err != nil {
		return false, err
	}
	fields := map[string]json.RawMessage{}
	for _, member := range members {
		fields[member.Key] = member.Value
	}
	for _, filter := range filters {
		raw := fields[filter.Key]
		if raw == nil {
			raw = json.RawMessage("null")
		}
		equal, err := floatingIPLocalMatch(filter.Value, raw)
		if err != nil || !equal {
			return false, err
		}
	}
	return true, nil
}

func queryValues(records []*FloatingIPRecord) (json.RawMessage, error) {
	rows := make([]*resource.RawResource, len(records))
	for i, record := range records {
		rows[i] = record.Resource
	}
	return json.Marshal(rows)
}

func queryHTTPFailure(err error) *gophercloud.ErrUnexpectedResponseCode {
	var value gophercloud.ErrUnexpectedResponseCode
	if errors.As(err, &value) {
		return &value
	}
	var pointer *gophercloud.ErrUnexpectedResponseCode
	if errors.As(err, &pointer) {
		return pointer
	}
	return nil
}

package compute

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/gophercloud/gophercloud/v2"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// CreateFloatingIP always allocates fresh. Only Neutron's selected port can
// activate the source wait policy; Nova only POSTs then performs a compatibility
// GET. An accepted allocation is never concealed by allocating on another backend.
func (s *Service) CreateFloatingIP(ctx context.Context, input CreateFloatingIPRequest, options ...FloatingIPCreateOption) (*CreateFloatingIPResult, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if input.Network != nil {
		value := *input.Network
		input.Network = &value
	}
	if s == nil || s.API == nil || s.Servers == nil || s.Servers.Collection == nil {
		return nil, invalid("floating IP create facade is required")
	}
	base := s.captureAutomaticComputeSource()
	policy, err := prepareFloatingIPCreateOptions(options, func() error { return base(ctx) })
	if err != nil {
		return nil, err
	}
	p, err := s.captureFloatingIPQuery(ctx, []FloatingIPQueryOption{WithFloatingIPQueryOptions(policy.query())})
	if err != nil {
		return nil, err
	}
	defer p.cancel()
	backend, client, err := p.backend()
	result := &CreateFloatingIPResult{Backend: backend}
	if err != nil {
		return result, err
	}
	if backend == FloatingIPNeutron {
		allocation, allocateErr := p.state.network.FloatingIPs.Allocate(p.ctx, network.AllocateFloatingIPRequest{Network: input.Network, Server: policy.Server, PortID: policy.PortID, FixedAddress: policy.FixedAddress, NATDestination: policy.NATDestination})
		err = p.adoptNeutronAllocation(result, client, allocation, allocateErr)
		if err == nil {
			return p.finishNeutronCreate(result, policy)
		}
		if result.Allocated || !availableIPNotFound(err) {
			return result, err
		}
		result.FallbackError = err
	}
	result.Backend = FloatingIPNova
	if policy.PortID != "" {
		return result, errors.Join(resource.ErrUnsupported, invalid("Nova cannot create an arbitrary port mapping"))
	}
	return p.createNova(result, input.Network)
}

// Preserve request-seeded Resource fields separately from actual allocation Wire.
func (p *floatingIPQueryState) adoptNeutronAllocation(result *CreateFloatingIPResult, client *gophercloud.ServiceClient, allocation *network.FloatingIPAllocation, prior error) error {
	err := errors.Join(prior, p.state.check(p.ctx))
	if allocation != nil {
		result.Selection, result.Allocated = allocation.Selection, allocation.Allocated
		if receipt := allocation.AllocationResponse; receipt != nil {
			result.AllocationResponse = &FloatingIPQueryResponse{Backend: FloatingIPNeutron, Metadata: receipt.Metadata, Envelope: append(json.RawMessage(nil), receipt.Envelope...)}
		}
		if allocation.Wire != nil {
			seeded := allocation.Wire.Clone()
			seed := map[string]string{"floating_network_id": allocation.Selection.NetworkID}
			if _, present := seeded.Body["id"]; !present {
				seeded.Body["id"] = json.RawMessage("null")
			}
			if allocation.Selection.PortID != "" {
				seed["port_id"] = allocation.Selection.PortID
			}
			if allocation.Selection.FixedIPv4 != "" {
				seed["fixed_ip_address"] = allocation.Selection.FixedIPv4
			}
			for key, value := range seed {
				if _, present := seeded.Body[key]; !present {
					seeded.Body[key], _ = json.Marshal(value)
				}
			}
			record, recordErr := p.record(FloatingIPNeutron, client, seeded, false)
			if record != nil {
				record.Wire = allocation.Wire.Clone()
			} else {
				record = &FloatingIPRecord{Backend: FloatingIPNeutron, Wire: allocation.Wire.Clone()}
			}
			result.Allocation, result.FloatingIP = record, cloneFloatingIPRecord(record)
			if recordErr != nil {
				recordErr = createResponseError(result.AllocationResponse, recordErr)
			}
			err = errors.Join(err, recordErr)
		}
	}
	result.Failure = queryFailure(FloatingIPNeutron, err)
	return err
}

func createResponseError(response *FloatingIPQueryResponse, cause error) error {
	if response == nil {
		return cause
	}
	return (&rest.Response{StatusCode: response.StatusCode, Header: response.Header.Clone(), Body: append([]byte(nil), response.Envelope...)}).Fail(cause)
}

func floatingIPExecutableID(record *FloatingIPRecord) (string, error) {
	if record == nil || record.Wire == nil {
		return "", invalid("floating IP allocation has no resource")
	}
	raw, present := record.Wire.Body["id"]
	if !present {
		return "", invalid("floating IP allocation lacks an executable ID")
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] == '{' || raw[0] == '[' {
		return "", invalid("floating IP allocation ID must be a scalar")
	}
	values, err := cloudfilter.RequestQueryValues(raw)
	if err != nil || len(values) != 1 {
		return "", invalid("floating IP allocation ID must be a scalar")
	}
	if err := resource.ID(values[0]).Validate(); err != nil {
		return "", err
	}
	return values[0], nil
}

func (p *floatingIPQueryState) createNova(result *CreateFloatingIPResult, network *string) (*CreateFloatingIPResult, error) {
	client, err := p.novaClient()
	if err != nil {
		return result, err
	}
	var pool any
	if network != nil {
		pool = *network
	} else {
		pools, poolErr := p.pools()
		result.PoolQuery = pools
		if poolErr != nil {
			if pools != nil {
				result.Failure = pools.Failure
			}
			return result, poolErr
		}
		if len(pools.Pools) == 0 {
			return result, &resource.NotFoundError{Resource: "floating IP pool"}
		}
		pool = append(json.RawMessage(nil), pools.Pools[0].Body["name"]...)
	}
	return p.createNovaInPool(result, client, pool)
}

// Reuse the accepted POST and compatibility GET with an already selected pool.
// Availability must not resolve the same default pool again before allocation.
func (p *floatingIPQueryState) createNovaInPool(result *CreateFloatingIPResult, client *gophercloud.ServiceClient, pool any) (*CreateFloatingIPResult, error) {
	response, requestErr := rest.DoJSONGuarded(p.ctx, client, p.state.check, http.MethodPost, client.ServiceURL("os-floating-ips"), map[string]any{"pool": pool}, nil, floatingIPDeleteCodes...)
	result.Allocated, result.AllocationResponse = response != nil, queryResponse(FloatingIPNova, response)
	if response == nil {
		result.Failure = queryFailure(FloatingIPNova, requestErr)
		return result, requestErr
	}
	wire, decodeErr := rest.Decode[resource.RawResource](response, "floating_ip", func(row *resource.RawResource) *resource.Metadata { return &row.Metadata })
	if wire != nil {
		result.Allocation = &FloatingIPRecord{Backend: FloatingIPNova, Wire: wire.Clone()}
		result.FloatingIP = cloneFloatingIPRecord(result.Allocation)
	}
	err := errors.Join(requestErr, decodeErr, p.state.check(p.ctx))
	if err != nil {
		result.Failure = queryFailure(FloatingIPNova, err)
		return result, err
	}
	id, err := floatingIPExecutableID(result.Allocation)
	if err != nil {
		err = response.Fail(err)
		result.Failure = queryFailure(FloatingIPNova, err)
		return result, err
	}
	observed, readErr := rest.DoJSONGuarded(p.ctx, client, p.state.check, http.MethodGet, client.ServiceURL("os-floating-ips", url.PathEscape(id)), nil, nil, floatingIPDeleteCodes...)
	compat := &GetFloatingIPResult{Backend: FloatingIPNova, Observed: queryResponse(FloatingIPNova, observed)}
	result.Compatibility = compat
	if observed != nil {
		row, rowErr := rest.Decode[resource.RawResource](observed, "floating_ip", func(row *resource.RawResource) *resource.Metadata { return &row.Metadata })
		if row != nil {
			compat.FloatingIP = &FloatingIPRecord{Backend: FloatingIPNova, Wire: row.Clone()}
			record, recordErr := p.record(FloatingIPNova, client, row, false)
			if record != nil {
				compat.FloatingIP = record
				result.FloatingIP = record
				compat.Value, rowErr = json.Marshal(record.Resource)
			}
			rowErr = errors.Join(rowErr, recordErr)
		}
		if rowErr != nil {
			rowErr = observed.Fail(rowErr)
		}
		readErr = errors.Join(readErr, rowErr)
	}
	err = errors.Join(readErr, p.state.check(p.ctx))
	compat.Failure = queryFailure(FloatingIPNova, err)
	result.Failure = compat.Failure
	return result, err
}

package compute

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudfilter"
	"github.com/JSYoo5B/gophercloudsdk/network"
)

func (p *floatingIPQueryState) availableNeutron(result *AvailableFloatingIPResult, input AvailableFloatingIPRequest, policy network.AvailableFloatingIPPolicy) (*AvailableFloatingIPResult, error) {
	child := *p
	if timeout := policy.Timeout(); timeout > 0 {
		ctx, cancel := context.WithTimeout(p.ctx, timeout)
		defer cancel()
		child.ctx = ctx
	}
	plan, err := p.state.planner.PrepareAvailability(child.ctx, input, policy)
	if err != nil {
		return result, err
	}
	inventory, err := child.list(nil)
	result.Inventory = inventory
	if err != nil {
		return result, err
	}
	networkID, _ := json.Marshal(plan.NetworkID())
	filter := append(json.RawMessage(`{"port_id":null,"floating_network_id":`), networkID...)
	filter = append(filter, []byte(`,"project_id":`)...)
	filter = append(filter, plan.ProjectID()...)
	filter = append(filter, '}')
	views := make([]json.RawMessage, len(inventory.FloatingIPs))
	for i, record := range inventory.FloatingIPs {
		views[i], err = json.Marshal(record.Resource)
		if err != nil {
			return result, err
		}
	}
	matched, err := cloudfilter.Select(views, "", &filter, func() error { return child.state.check(child.ctx) })
	if err != nil {
		return result, floatingIPQueryInputError("availability filter", err)
	}
	if len(matched.Indices) != 0 {
		record := inventory.FloatingIPs[matched.Indices[0]]
		result.Backend, result.Reused = record.Backend, true
		setAvailableRecord(result, record)
		return result, errors.Join(plan.Check(child.ctx), child.state.check(child.ctx))
	}
	allocation, allocateErr := plan.Allocate(child.ctx)
	created := &CreateFloatingIPResult{Backend: FloatingIPNeutron}
	err = child.adoptNeutronAllocation(created, child.state.network.RawClient(), allocation, allocateErr)
	if err == nil {
		created, err = child.finishNeutronCreate(created, FloatingIPCreateOpts{})
	}
	result.Creation, result.Allocated = created, created.Allocated
	if created.Allocated {
		result.Neutron = &network.FloatingIPAvailability{Allocated: true}
		if receipt := created.AllocationResponse; receipt != nil {
			result.Neutron.AllocationResponse = &network.FloatingIPAvailabilityResponse{Metadata: receipt.Metadata, Envelope: append(json.RawMessage(nil), receipt.Envelope...)}
			result.Neutron.AllocationResponse.Header = receipt.Header.Clone()
		}
		setAvailableRecord(result, created.FloatingIP)
	}
	return result, errors.Join(err, child.state.check(child.ctx))
}

// Typed convenience projections are separate from the authoritative cloud view.
func setAvailableRecord(result *AvailableFloatingIPResult, record *FloatingIPRecord) {
	result.FloatingIP = cloneFloatingIPRecord(record)
	if record == nil || record.Wire == nil {
		return
	}
	raw, err := json.Marshal(record.Wire)
	if err != nil {
		return
	}
	if record.Backend == FloatingIPNova {
		if result.Nova == nil {
			result.Nova = &NovaFloatingIPAvailability{Inventory: result.Inventory, Reused: result.Reused, Allocated: result.Allocated}
		}
		var typed NovaFloatingIP
		if json.Unmarshal(raw, &typed) == nil {
			typed.Header, typed.StatusCode = record.Wire.Header.Clone(), record.Wire.StatusCode
			result.Nova.FloatingIP = &typed
			result.ID, result.Address = typed.ID, typed.Address
		}
	} else {
		if result.Neutron == nil {
			result.Neutron = &network.FloatingIPAvailability{Reused: result.Reused, Allocated: result.Allocated}
		}
		result.Neutron.Metadata = record.Wire.Clone().Metadata
		var typed network.FloatingIP
		if json.Unmarshal(raw, &typed) == nil {
			result.Neutron.FloatingIP = &typed
			result.ID, result.Address = typed.ID, typed.FloatingIP
		}
	}
}

package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"slices"

	"gophercloudsdk/internal/cloudlocation"
	"gophercloudsdk/resource"
)

// FloatingIPAvailabilityPlan captures the chosen external network and nullable
// reuse project. Server/port discovery remains lazy until fresh allocation.
// Connection prepares and consumes this concrete plan without caller builders.
// Direct plan consumers supply the same bounded context to preparation and
// allocation; the plan does not restart the policy's timeout at each stage.
type FloatingIPAvailabilityPlan struct {
	planner   *FloatingIPPlanner
	input     AvailableFloatingIPRequest
	options   availableFloatingIPOptions
	networkID string
	projectID json.RawMessage
}

func (p *FloatingIPAvailabilityPlan) NetworkID() string          { return p.networkID }
func (p *FloatingIPAvailabilityPlan) ProjectID() json.RawMessage { return bytes.Clone(p.projectID) }
func (p *FloatingIPAvailabilityPlan) Check(ctx context.Context) error {
	if p == nil || p.planner == nil {
		return floatingIPInvalid("a prepared availability plan is required")
	}
	return p.planner.Check(ctx)
}

func (f *FloatingIPs) PrepareAvailability(ctx context.Context, input AvailableFloatingIPRequest, policy AvailableFloatingIPPolicy) (*FloatingIPAvailabilityPlan, error) {
	planner, err := f.NewPlanner(ctx)
	if err != nil {
		return nil, err
	}
	return planner.PrepareAvailability(ctx, input, policy)
}

func (p *FloatingIPPlanner) PrepareAvailability(ctx context.Context, input AvailableFloatingIPRequest, policy AvailableFloatingIPPolicy) (*FloatingIPAvailabilityPlan, error) {
	if err := p.Check(ctx); err != nil {
		return nil, err
	}
	if !policy.prepared {
		return nil, floatingIPInvalid("a prepared availability policy is required")
	}
	input.Networks = slices.Clone(input.Networks)
	for _, ref := range input.Networks {
		if err := ref.Validate(); err != nil {
			return nil, err
		}
	}
	if input.Server != (resource.Ref{}) {
		if err := input.Server.Validate(); err != nil {
			return nil, err
		}
	}
	options := policy.options
	owner, err := cloudlocation.ProjectID(p.service.api.RawClient().ProviderClient)
	if options.projectID != nil {
		copy := *options.projectID
		options.projectID = &copy
		owner, err = json.Marshal(copy)
	}
	if err != nil {
		return nil, err
	}
	if owner == nil {
		owner = json.RawMessage("null")
	}
	id, err := p.service.availableExternalNetwork(ctx, input.Networks, p)
	if err = errors.Join(err, p.Check(ctx)); err != nil {
		return nil, err
	}
	return &FloatingIPAvailabilityPlan{planner: p, input: input, options: options, networkID: id, projectID: bytes.Clone(owner)}, nil
}

// Allocate uses the captured network directly; no ordinary network relookup,
// free-IP list, wait, association action or cleanup is introduced here.
func (p *FloatingIPAvailabilityPlan) Allocate(ctx context.Context) (*FloatingIPAllocation, error) {
	if err := p.Check(ctx); err != nil {
		return nil, err
	}
	f := p.planner.service
	selection := FloatingIPSelection{NetworkID: p.networkID}
	if p.input.Server != (resource.Ref{}) {
		serverID := p.input.Server.String()
		var err error
		if p.input.Server.IsName() {
			if f.dependencies.Server == nil {
				return nil, floatingIPInvalid("named availability server needs a Compute resolver")
			}
			serverID, err = f.dependencies.Server(ctx, p.input.Server)
			if err = errors.Join(err, p.Check(ctx)); err != nil {
				return nil, err
			}
		}
		if err := resource.ID(serverID).Validate(); err != nil {
			return nil, err
		}
		destination, err := f.allocateDestination(ctx, serverID, AllocateFloatingIPRequest{FixedAddress: p.options.fixed, NATDestination: p.options.nat.String()}, p.planner)
		if err != nil {
			return nil, err
		}
		selection.ServerID, selection.PortID, selection.PortNetworkID, selection.FixedIPv4 = serverID, destination.PortID, destination.PortNetworkID, destination.FixedIPv4
	}
	return f.allocateSelected(ctx, selection, p.planner)
}

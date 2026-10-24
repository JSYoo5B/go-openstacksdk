package compute

import (
	"context"
	"errors"

	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// The standalone address lane follows Python's _attach_ip_to_server for each
// selected IP: the short circuit and the conditional attached-IP refresh run
// after the exact address lookup and before any port lookup or mutation, and a
// Neutron NotFound during attachment is recorded instead of failing the call.

// sourceFloatingIPv4 returns the version-4 floating-tagged addresses of a
// server, the candidates of Python's get_server_ip(ext_tag="floating").
func sourceFloatingIPv4(view *ServerAddressView) []string {
	var values []string
	if view == nil {
		return nil
	}
	for _, rows := range view.Addresses {
		for _, row := range rows {
			if row.Version == 4 && row.Type == "floating" {
				values = append(values, row.Address)
			}
		}
	}
	return values
}

// sourceShortCircuit reports whether the current server model already shows
// target as its floating IPv4 address. When that model has no floating IPv4
// address and the selected IP is associated, the server is fetched once first,
// as Python refreshes it in that case.
//
// Python's get_server_ip probes reachability when several floating addresses
// exist. Go never opens such probes; a server with several floating IPv4
// addresses does not short-circuit and the normal attachment path decides. A
// supplied server without address metadata, which Python cannot evaluate,
// also takes the normal path.
func (state *automaticIPState) sourceShortCircuit(ctx context.Context, target string, associated bool) (bool, error) {
	if err := state.check(ctx); err != nil {
		return false, err
	}
	if state.last == nil || state.last.Addresses == nil {
		return false, nil
	}
	view, err := parseServerAddresses(state.last, state.address.options.networkOrder)
	if err != nil {
		return false, err
	}
	floating := sourceFloatingIPv4(view)
	if len(floating) == 0 && associated {
		view, err = state.rawServer(ctx)
		if err != nil {
			return false, err
		}
		floating = sourceFloatingIPv4(view)
	}
	return len(floating) == 1 && floating[0] == target, state.check(ctx)
}

// sourceNotFound accepts only a clean Neutron 404 for the fall-through; any
// joined cancellation or source failure remains an error.
func (state *automaticIPState) sourceNotFound(ctx context.Context, err error) bool {
	if err == nil || state.check(ctx) != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, resource.ErrInvalidOption) {
		return false
	}
	var pointer *gophercloud.ErrUnexpectedResponseCode
	if errors.As(err, &pointer) && pointer != nil {
		return pointer.Actual == 404
	}
	var value gophercloud.ErrUnexpectedResponseCode
	return errors.As(err, &value) && value.Actual == 404
}

// attachSourceIP selects one exact Neutron IP, evaluates the short circuit,
// then selects the destination and attaches. A 404 after selection is returned
// as notFound with the last known IP and a nil error.
func (state *automaticIPState) attachSourceIP(ctx context.Context, address string) (assignment *network.FloatingIPAssignment, shortCircuit bool, notFound error, err error) {
	var selected *network.FloatingIP
	check := func(ctx context.Context, ip network.FloatingIP) (bool, error) {
		selected = &ip
		return state.sourceShortCircuit(ctx, ip.FloatingIP, ip.PortID != "")
	}
	plan, err := state.planner.PrepareAttachChecked(ctx, network.AttachFloatingIPRequest{Server: resource.ID(state.serverID), IP: resource.Name(address)}, check, network.WithAttachFloatingIPPolicy(state.attachPolicy))
	if skipped, ok := err.(*network.AttachSkippedError); ok {
		ip := skipped.FloatingIP
		return &network.FloatingIPAssignment{FloatingIP: &ip, Reused: true}, true, nil, state.check(ctx)
	}
	if err == nil {
		state.decision.AttachmentSelections = append(state.decision.AttachmentSelections, plan.Selection())
		assignment, err = state.network.FloatingIPs.AttachPrepared(ctx, plan)
	}
	if selected != nil && state.sourceNotFound(ctx, err) {
		if assignment == nil || assignment.FloatingIP == nil {
			ip := *selected
			assignment = &network.FloatingIPAssignment{FloatingIP: &ip, Reused: true}
		}
		return assignment, false, err, state.check(ctx)
	}
	return assignment, false, nil, errors.Join(err, state.check(ctx))
}

// novaSourceAssignment is the legacy Nova address lane with the same short
// circuit. Python's Nova rows carry no port_id, so no refresh happens here,
// and Nova attachment errors are never swallowed.
func (state *automaticIPState) novaSourceAssignment(ctx context.Context, address string) (*NovaFloatingIPAssignment, bool, error) {
	b, err := state.novaBackend(ctx)
	if err != nil {
		return nil, false, err
	}
	selected, err := b.selectAddress(ctx, address)
	if err != nil {
		return nil, false, err
	}
	result := &NovaFloatingIPAssignment{FloatingIP: selected, Reused: true}
	state.decision.NovaSelections = append(state.decision.NovaSelections, NovaFloatingIPSelection{ID: selected.ID, Address: selected.Address, Pool: selected.Pool})
	shortCircuit, err := state.sourceShortCircuit(ctx, address, false)
	if err != nil || shortCircuit {
		return result, shortCircuit, errors.Join(err, state.check(ctx))
	}
	err = b.attach(ctx, result)
	return result, false, errors.Join(err, state.check(ctx))
}

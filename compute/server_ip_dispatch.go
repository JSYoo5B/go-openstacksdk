package compute

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Explicit requests bypass automatic needs/skip classification. Backend
// absence/configuration remains distinct from a disabled automatic decision.
func (state *automaticIPState) explicitBackend(ctx context.Context) error {
	state.decision.Needed = true
	state.decision.Reason = AutomaticIPAddressesRequested
	if state.options.dispatchMode() == ServerIPPool {
		state.decision.Reason = AutomaticIPPoolRequested
	}
	state.decision.Backend = FloatingIPNova
	if err := state.check(ctx); err != nil {
		return err
	}
	if state.address.options.source != FloatingIPNeutron {
		return nil
	}
	state.decision.Backend = FloatingIPNeutron
	service, err := state.loadNetwork(ctx)
	if err != nil {
		return err
	}
	if service == nil {
		state.decision.Backend = FloatingIPNova
	}
	return state.check(ctx)
}

func (state *automaticIPState) preparePool(ctx context.Context) (network.FloatingIPPlan, error) {
	plan, err := state.planner.PrepareEnsure(ctx, network.EnsureFloatingIPRequest{Server: resource.ID(state.serverID), Network: state.options.pool}, network.WithEnsureFloatingIPPolicy(state.policy))
	if err == nil {
		state.decision.Selection = plan.Selection()
	}
	return plan, errors.Join(err, state.check(ctx))
}

func (state *automaticIPState) prepareExplicitIP(ctx context.Context, address string) (network.FloatingIPAttachPlan, error) {
	plan, err := state.planner.PrepareAttach(ctx, network.AttachFloatingIPRequest{Server: resource.ID(state.serverID), IP: resource.Name(address)}, network.WithAttachFloatingIPPolicy(state.attachPolicy))
	if err == nil {
		state.decision.AttachmentSelections = append(state.decision.AttachmentSelections, plan.Selection())
	}
	return plan, errors.Join(err, state.check(ctx))
}

func (state *automaticIPState) planDispatch(ctx context.Context) error {
	if state.options.dispatchMode() == ServerIPAutomatic {
		return state.decide(ctx)
	}
	if err := state.explicitBackend(ctx); err != nil {
		return err
	}
	// Plan is diagnostic, including a Needed Nova backend with no executable
	// Neutron selection, as in the existing automatic Plan contract.
	if state.decision.Backend != FloatingIPNeutron {
		return state.check(ctx)
	}
	if state.options.dispatchMode() == ServerIPPool {
		_, err := state.preparePool(ctx)
		return err
	}
	for _, address := range state.options.requestedIPs {
		if _, err := state.prepareExplicitIP(ctx, address); err != nil {
			return err
		}
	}
	return state.check(ctx)
}

func (state *automaticIPState) finishExplicit(ctx context.Context, result *AutomaticServerIPResult, index int, cause error) (*AutomaticServerIPResult, error) {
	err := errors.Join(cause, state.check(ctx))
	result.Server, state.decision.Server = state.last, state.last
	if index >= 0 {
		result.Attempts[index].Error = err
	}
	result.Observed = err == nil && state.observeAssignment
	return result, err
}

func (state *automaticIPState) ensureExplicit(ctx context.Context) (*AutomaticServerIPResult, error) {
	result := state.newResult()
	if state.options.dispatchMode() == ServerIPExplicit && len(state.options.requestedIPs) == 0 {
		state.decision.Reason = AutomaticIPEmptyAddressList
		return result, state.check(ctx)
	}
	if err := state.explicitBackend(ctx); err != nil {
		return state.finishExplicit(ctx, result, -1, err)
	}
	if state.requireServerActive && !strings.EqualFold(state.last.Status, "ACTIVE") {
		return state.finishExplicit(ctx, result, -1, invalid("explicit floating IP assignment requires an ACTIVE server"))
	}
	if state.decision.Backend == FloatingIPNova {
		if _, err := state.novaBackend(ctx); err != nil {
			return state.finishExplicit(ctx, result, -1, err)
		}
	}
	owned, err := cloneAutomaticProgressServer(state.last)
	if err != nil {
		return state.finishExplicit(ctx, result, -1, err)
	}
	state.last = owned
	// Verify mandatory raw observation capability before the first mutation.
	if state.observeAssignment {
		client, err := state.rawClient(ctx)
		if err != nil {
			return state.finishExplicit(ctx, result, -1, err)
		}
		if err := rest.ValidateTarget(client, client.ServiceURL("servers", url.PathEscape(state.serverID))); err != nil {
			return state.finishExplicit(ctx, result, -1, err)
		}
	}
	addresses := state.options.requestedIPs
	if state.options.dispatchMode() == ServerIPPool {
		addresses = []string{""}
	}
	for index, address := range addresses {
		result.Attempts = append(result.Attempts, ServerFloatingIPAttempt{Index: index, RequestedAddress: address})
		if state.decision.Backend == FloatingIPNova {
			assignment, err := state.novaAssignment(ctx, address, state.options.pool)
			attempt := &result.Attempts[index]
			attempt.NovaAssignment = assignment
			if assignment != nil {
				result.NovaAssignment = assignment
			}
			if err = errors.Join(err, state.check(ctx)); err != nil {
				return state.finishExplicit(ctx, result, index, err)
			}
			if state.observeAssignment {
				if err := state.observe(ctx, assignment.FloatingIP.Address); err != nil {
					return state.finishExplicit(ctx, result, index, err)
				}
				attempt.Observed = true
			}
			attempt.Completed = true
			continue
		}
		var assignment *network.FloatingIPAssignment
		var err error
		if state.options.dispatchMode() == ServerIPPool {
			plan, planErr := state.preparePool(ctx)
			err = planErr
			if err == nil {
				assignment, err = state.network.FloatingIPs.EnsurePrepared(ctx, plan)
			}
		} else {
			plan, planErr := state.prepareExplicitIP(ctx, address)
			err = planErr
			if err == nil {
				assignment, err = state.network.FloatingIPs.AttachPrepared(ctx, plan)
			}
		}
		attempt := &result.Attempts[index]
		attempt.Assignment = assignment
		if assignment != nil {
			result.Assignment = assignment
		}
		if err = errors.Join(err, state.check(ctx)); err != nil {
			return state.finishExplicit(ctx, result, index, err)
		}
		if assignment == nil || assignment.FloatingIP == nil {
			return state.finishExplicit(ctx, result, index, invalid("floating IP assignment returned no address"))
		}
		if state.observeAssignment {
			if err := state.observe(ctx, assignment.FloatingIP.FloatingIP); err != nil {
				return state.finishExplicit(ctx, result, index, err)
			}
			attempt.Observed = true
		}
		attempt.Completed = true
	}
	return state.finishExplicit(ctx, result, -1, nil)
}

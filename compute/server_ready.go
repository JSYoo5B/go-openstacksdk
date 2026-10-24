package compute

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type serverReadyOptions struct {
	automatic   []AutomaticFloatingIPOption
	waitOptions []resource.WaitOption
	waitIP      bool

	withoutAddresses ServerWithoutAddressesPolicy
}

// ServerReadyOption configures existing-server readiness without an adapter.
type ServerReadyOption func(*serverReadyOptions) error

func WithServerReadyAutomaticIPOptions(options ...AutomaticFloatingIPOption) ServerReadyOption {
	options = append([]AutomaticFloatingIPOption(nil), options...)
	return func(o *serverReadyOptions) error { o.automatic = append(o.automatic, options...); return nil }
}

// WithServerReadyWaitOptions configures raw server polling. Defaults are an
// unlimited stage under the overall 180-second budget and a five-second poll.
func WithServerReadyWaitOptions(options ...resource.WaitOption) ServerReadyOption {
	options = append([]resource.WaitOption(nil), options...)
	return func(o *serverReadyOptions) error { o.waitOptions = append(o.waitOptions, options...); return nil }
}

// WithActiveServerWait enables raw Nova address convergence in GetActiveServer
// and actual IP ACTIVE for Neutron assignments. Its default is false.
// WaitForServer always requires raw server ACTIVE and address convergence;
// legacy Nova IPs have no ACTIVE status to wait for.
func WithActiveServerWait(enabled bool) ServerReadyOption {
	return func(o *serverReadyOptions) error { o.waitIP = enabled; return nil }
}

// GetActiveServer checks the supplied status without refreshing it. Non-ACTIVE
// returns nil,nil. ERROR returns ServerFaultError and ACTIVE without addresses
// returns ServerAddressesUnavailableError, both with the known Server. No
// DELETE is sent unless WithServerWithoutAddressesPolicy selects deletion. Conditional IP assignment returns its accepted
// result by default; WithActiveServerWait also verifies address convergence.
func (s *Service) GetActiveServer(ctx context.Context, input AutomaticFloatingIPRequest, options ...ServerReadyOption) (*AutomaticServerIPResult, error) {
	state, ctx, cancel, _, err := s.prepareServerReady(ctx, input, options, false)
	if err != nil {
		return nil, err
	}
	defer cancel()
	result := state.newResult()
	if strings.EqualFold(state.last.Status, "ERROR") {
		return result, errors.Join(newServerFaultError(state.last, nil), state.check(ctx))
	}
	if !strings.EqualFold(state.last.Status, "ACTIVE") {
		return nil, state.check(ctx)
	}
	view, err := parseServerAddresses(state.last, state.address.options.networkOrder)
	if err != nil {
		return result, errors.Join(err, state.check(ctx))
	}
	if !serverAddressRowsPresent(view) {
		err := state.applyWithoutAddresses(ctx, &ServerAddressesUnavailableError{ServerID: state.serverID, EmptyRows: len(state.last.Addresses) > 0}, state.withoutAddresses, state.deleteWait)
		return result, errors.Join(err, state.check(ctx))
	}
	// Detach valid ACTIVE input before callbacks or association; non-ACTIVE and
	// ERROR branches do not inspect unrelated address/model fields.
	owned, err := cloneAutomaticProgressServer(state.last)
	if err != nil {
		return result, errors.Join(err, state.check(ctx))
	}
	state.last = owned
	state.setView(view)
	// Python's get_active_server calls add_ips_to_server, whose ips branch is
	// add_ip_list: the address lane shares the standalone _attach_ip_to_server
	// steps (short circuit, attached-IP refresh, Neutron NotFound fall-through).
	// The pool and automatic lanes keep the shared Ensure dispatch, as in
	// AddIPsToServer.
	state.sourceAttach = state.options.dispatchMode() == ServerIPExplicit
	return state.ensure(ctx)
}

// WaitForServer fixes the supplied server ID and fetches current raw metadata
// even when the supplied model says ACTIVE or ERROR. One 180-second budget
// covers polling, conditional assignment, Neutron IP ACTIVE and Nova convergence.
// Known raw Server, backend assignment and original causes survive failures.
//
// Polling uses the direct GET by ID, as Python's cloud get_server does first
// through find_server. The ACTIVE poll result then receives the default
// (non-bare) get_server interface expansion, returned as Interfaces, and the
// IP stage decides on that expanded server as get_active_server does. The IP
// stage inherits the GetActiveServer lane choices.
func (s *Service) WaitForServer(ctx context.Context, input AutomaticFloatingIPRequest, options ...ServerReadyOption) (*AutomaticServerIPResult, error) {
	state, ctx, cancel, policy, err := s.prepareServerReady(ctx, input, options, true)
	if err != nil {
		return nil, err
	}
	defer cancel()
	result := state.newResult()
	if err := state.waitReadyServer(ctx, policy); err != nil {
		err = state.applyWithoutAddresses(ctx, err, state.withoutAddresses, state.deleteWait)
		result.Server = state.last
		result.Decision.Server = state.last
		return result, errors.Join(err, state.check(ctx))
	}
	interfaces, err := state.expandReadyServer(ctx)
	if err != nil {
		result.Server, result.Decision.Server, result.Interfaces = state.last, state.last, interfaces
		return result, errors.Join(err, state.check(ctx))
	}
	// As in GetActiveServer, the address lane follows _attach_ip_to_server.
	state.sourceAttach = state.options.dispatchMode() == ServerIPExplicit
	result, err = state.ensure(ctx)
	if result != nil {
		result.Interfaces = interfaces
	}
	return result, err
}

// expandReadyServer applies meta.add_server_interfaces, the default expansion
// of Python's cloud get_server, to the ACTIVE poll result: Neutron (or Nova
// for the Nova source) supplemental floating rows, then public, private,
// default-interface and access addresses. The workflow's lazy Network and raw
// Compute bindings serve the lookups; a clean HTTP failure of the optional
// supplement is recorded in SupplementalError, as Python ignores it.
func (state *automaticIPState) expandReadyServer(ctx context.Context) (*ServerAddressView, error) {
	view, err := parseServerAddresses(state.last, state.address.options.networkOrder)
	if err != nil {
		return nil, err
	}
	state.setView(view)
	reader := *state.address.servers
	reader.dependencies.AddressNetworks = state.loadNetwork
	reader.dependencies.AddressCompute = state.rawClient
	if err := reader.supplementServerAddresses(ctx, state.last.Status, state.address); err != nil {
		return view, wrapServerAddress("supplement", err)
	}
	view, err = state.address.interfaceView(ctx)
	if err != nil {
		return view, err
	}
	state.expanded = view
	return view, state.check(ctx)
}

func (s *Service) prepareServerReady(ctx context.Context, input AutomaticFloatingIPRequest, options []ServerReadyOption, wait bool) (*automaticIPState, context.Context, context.CancelFunc, resource.WaitPolicy, error) {
	var policy resource.WaitPolicy
	if ctx == nil {
		return nil, nil, nil, policy, invalid("context is required")
	}
	if err := errors.Join(ctx.Err(), context.Cause(ctx)); err != nil {
		return nil, nil, nil, policy, err
	}
	if s == nil || s.API == nil || s.Servers == nil || input.Server == nil {
		return nil, nil, nil, policy, invalid("compute service and server are required")
	}
	snapshot := *input.Server
	if err := resource.ID(snapshot.ID).Validate(); err != nil {
		return nil, nil, nil, policy, err
	}
	input.Server = &snapshot
	guard := s.captureAutomaticComputeSource()
	o := serverReadyOptions{}
	for _, apply := range options {
		if apply == nil {
			return nil, nil, nil, policy, invalid("nil server readiness option")
		}
		if err := apply(&o); err != nil {
			return nil, nil, nil, policy, err
		}
	}
	if err := guard(ctx); err != nil {
		return nil, nil, nil, policy, err
	}
	prepared, err := resource.PrepareWaitOptionsFor[Server](append([]resource.WaitOption{resource.WithUnlimitedWait(), resource.WithPollInterval(5 * time.Second)}, o.waitOptions...)...)
	if err != nil {
		return nil, nil, nil, policy, err
	}
	if err := prepared.ValidateFixedStatus(); err != nil {
		return nil, nil, nil, policy, err
	}
	if err := guard(ctx); err != nil {
		return nil, nil, nil, policy, err
	}
	ctx = rest.WithOperationGuard(ctx, guard)
	automatic := append([]AutomaticFloatingIPOption{WithAutomaticIPTimeout(180 * time.Second), WithAutomaticIPPollInterval(5 * time.Second)}, o.automatic...)
	state, ctx, cancel, err := s.prepareAutomaticIPReadiness(ctx, input, automatic, wait || o.waitIP)
	if err != nil {
		return nil, nil, nil, policy, err
	}
	state.retainAcceptedServer = true
	state.withoutAddresses, state.deleteWait = o.withoutAddresses, wait || o.waitIP
	return state, ctx, cancel, prepared, nil
}

func serverAddressRowsPresent(view *ServerAddressView) bool {
	for _, rows := range view.Addresses {
		if len(rows) > 0 {
			return true
		}
	}
	return false
}

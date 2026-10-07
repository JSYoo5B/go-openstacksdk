package compute

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/network"
	"gophercloudsdk/resource"
)

type automaticIPState struct {
	service              *Service
	address              *serverAddressState
	decision             *ServerFloatingIPDecision
	options              automaticFloatingIPOptions
	policy               network.EnsureFloatingIPPolicy
	input                AutomaticFloatingIPRequest
	serverID             string
	computeGuard         func(context.Context) error
	outerGuard           func(context.Context) error
	computeClient        *gophercloud.ServiceClient
	clientGuard          func(context.Context) error
	networkLoaded        bool
	network              *network.Service
	planner              *network.FloatingIPPlanner
	plan                 network.FloatingIPPlan
	last                 *Server
	retainAcceptedServer bool
	observeAssignment    bool
}

// PlanServerFloatingIP applies lazy automatic need/skip policy and, when
// Neutron is available, prepares a fixed target without scope or mutation.
// A Nova backend can be Needed even though its mutation is not implemented.
func (s *Service) PlanServerFloatingIP(ctx context.Context, input AutomaticFloatingIPRequest, options ...AutomaticFloatingIPOption) (*ServerFloatingIPDecision, error) {
	state, ctx, cancel, err := s.prepareAutomaticIP(ctx, input, options)
	if err != nil {
		return nil, err
	}
	defer cancel()
	err = state.decide(ctx)
	return state.decision, errors.Join(err, state.check(ctx))
}

// EnsureServerFloatingIP assigns only when needed, using the same prepared
// target, then observes the exact floating IPv4 in raw Nova responses. Known
// Server/Assignment survive every later error. No fallback or cleanup is sent.
func (s *Service) EnsureServerFloatingIP(ctx context.Context, input AutomaticFloatingIPRequest, options ...AutomaticFloatingIPOption) (*AutomaticServerIPResult, error) {
	state, ctx, cancel, err := s.prepareAutomaticIP(ctx, input, options)
	if err != nil {
		return nil, err
	}
	defer cancel()
	return state.ensure(ctx)
}

func (state *automaticIPState) ensure(ctx context.Context) (*AutomaticServerIPResult, error) {
	result := &AutomaticServerIPResult{Server: state.last, Decision: state.decision}
	if err := state.decide(ctx); err != nil {
		result.Server = state.last
		return result, errors.Join(err, state.check(ctx))
	}
	result.Server = state.last
	if !state.decision.Needed {
		return result, state.check(ctx)
	}
	if state.decision.Backend != FloatingIPNeutron {
		return result, fmt.Errorf("%w: automatic Nova floating IP assignment is not implemented", resource.ErrUnsupported)
	}
	if !strings.EqualFold(state.last.Status, "ACTIVE") {
		return result, invalid("automatic floating IP assignment requires an ACTIVE server")
	}
	// A synchronous workflow verifies mandatory observation before side effects.
	if state.observeAssignment {
		client, err := state.rawClient(ctx)
		if err != nil {
			return result, err
		}
		if err := rest.ValidateTarget(client, client.ServiceURL("servers", url.PathEscape(state.serverID))); err != nil {
			return result, err
		}
	}
	assignment, err := state.network.FloatingIPs.EnsurePrepared(ctx, state.plan)
	result.Assignment = assignment
	if err = errors.Join(err, state.check(ctx)); err != nil {
		return result, err
	}
	if assignment == nil || assignment.FloatingIP == nil {
		return result, invalid("floating IP assignment returned no address")
	}
	target := assignment.FloatingIP.FloatingIP
	if ip, err := netip.ParseAddr(target); err != nil || !ip.Is4() {
		return result, invalid("assigned floating IP is not IPv4")
	}
	if !state.observeAssignment {
		return result, state.check(ctx)
	}
	err = state.observe(ctx, target)
	result.Server = state.last
	result.Observed = err == nil
	return result, errors.Join(err, state.check(ctx))
}

func (s *Service) prepareAutomaticIP(ctx context.Context, input AutomaticFloatingIPRequest, options []AutomaticFloatingIPOption) (*automaticIPState, context.Context, context.CancelFunc, error) {
	return s.prepareAutomaticIPReadiness(ctx, input, options, true)
}

func (s *Service) prepareAutomaticIPReadiness(ctx context.Context, input AutomaticFloatingIPRequest, options []AutomaticFloatingIPOption, ready bool) (*automaticIPState, context.Context, context.CancelFunc, error) {
	if ctx == nil {
		return nil, nil, nil, invalid("context is required")
	}
	if ctx.Err() != nil {
		return nil, nil, nil, errors.Join(ctx.Err(), context.Cause(ctx))
	}
	if s == nil || s.API == nil || s.Servers == nil {
		return nil, nil, nil, invalid("compute service is required")
	}
	ctx = rest.WithOperationSources(ctx)
	baseGuard := s.captureAutomaticComputeSource()
	o := automaticFloatingIPOptions{enabled: true, timeout: 5 * time.Minute, interval: 2 * time.Second}
	for _, apply := range options {
		if apply == nil {
			return nil, nil, nil, invalid("nil automatic floating IP option")
		}
		if err := apply(&o); err != nil {
			return nil, nil, nil, err
		}
	}
	if input.Network != (resource.Ref{}) {
		if err := input.Network.Validate(); err != nil {
			return nil, nil, nil, err
		}
	}
	if err := baseGuard(ctx); err != nil {
		return nil, nil, nil, err
	}
	final := network.WithEnsureNoWait()
	if ready {
		final = network.WithEnsureActive()
	}
	policy, err := network.PrepareEnsureFloatingIPOptions(ctx, append(o.ips, final)...)
	if err != nil {
		return nil, nil, nil, errors.Join(err, ctx.Err(), context.Cause(ctx))
	}
	address, err := s.prepareAddressView(ctx, input.Server, o.addresses, false)
	if err != nil {
		return nil, nil, nil, err
	}
	supplied := *input.Server
	state := &automaticIPState{service: s, address: address, options: o, policy: policy, input: input, last: &supplied, serverID: supplied.ID}
	state.observeAssignment = ready
	state.outerGuard = rest.OperationGuard(ctx)
	state.decision = &ServerFloatingIPDecision{Reason: AutomaticIPUndetermined, Server: state.last, Addresses: address.view}
	client := s.client
	previous := address.sourceGuard
	// This outer guard is Compute-only; Network may invoke it under its lock.
	state.computeGuard = func(ctx context.Context) error {
		var bound error
		if state.clientGuard != nil {
			bound = state.clientGuard(ctx)
		}
		return errors.Join(baseGuard(ctx), previous(ctx), bound)
	}
	if client != nil {
		state.bindRawClient(client)
	}
	address.sourceGuard = state.check
	address.loadRoles = func(ctx context.Context) (*network.NetworkRoleSnapshot, error) {
		service, err := state.loadNetwork(ctx)
		if err != nil {
			return nil, err
		}
		if service == nil {
			return &network.NetworkRoleSnapshot{}, state.check(ctx)
		}
		return state.planner.NetworkRoles(ctx)
	}
	cancel := func() {}
	if o.timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, o.timeout)
	}
	ctx = rest.WithOperationGuard(ctx, state.computeGuard)
	if err := state.check(ctx); err != nil {
		cancel()
		return nil, nil, nil, err
	}
	return state, ctx, cancel, nil
}

func (s *Service) captureAutomaticComputeSource() func(context.Context) error {
	api, servers, collection, client := s.API, s.Servers, s.Servers.Collection, s.client
	apiServers, flavors, serverFlavors := api.Servers, s.Flavors, servers.flavors
	var provider *gophercloud.ProviderClient
	var endpoint, base, version, kind string
	if client != nil {
		provider, endpoint, base, version, kind = client.ProviderClient, client.Endpoint, client.ResourceBase, client.Microversion, client.Type
	}
	var changed error
	var mu sync.Mutex
	return func(ctx context.Context) error {
		mu.Lock()
		defer mu.Unlock()
		if changed == nil && (s.API != api || s.Servers != servers || servers.Collection != collection || s.client != client || servers.client != client || api.RawClient() != client || api.Servers != apiServers || s.Flavors != flavors || servers.flavors != serverFlavors) {
			changed = invalid("compute service source changed during automatic IP workflow")
		}
		if changed == nil && client != nil && (client.ProviderClient != provider || client.Endpoint != endpoint || client.ResourceBase != base || client.Microversion != version || client.Type != kind) {
			changed = invalid("compute client source changed during automatic IP workflow")
		}
		return errors.Join(changed, ctx.Err(), context.Cause(ctx))
	}
}

func (state *automaticIPState) check(ctx context.Context) error {
	var outer error
	if state.outerGuard != nil {
		outer = state.outerGuard(ctx)
	}
	if err := errors.Join(outer, state.computeGuard(ctx)); err != nil {
		return err
	}
	if state.planner != nil {
		return state.planner.Check(ctx)
	}
	return nil
}

func (state *automaticIPState) loadNetwork(ctx context.Context) (*network.Service, error) {
	if err := state.check(ctx); err != nil {
		return nil, err
	}
	if !state.networkLoaded {
		if getter := state.address.servers.dependencies.AddressNetworks; getter != nil {
			service, err := getter(ctx)
			if err = errors.Join(err, state.computeGuard(ctx)); err != nil {
				return nil, err
			}
			state.network = service
		}
		if state.network != nil {
			planner, err := state.network.FloatingIPs.NewPlanner(ctx)
			if err != nil {
				return nil, err
			}
			state.planner = planner
		}
		state.networkLoaded = true
	}
	return state.network, state.check(ctx)
}

func (state *automaticIPState) rawClient(ctx context.Context) (*gophercloud.ServiceClient, error) {
	if err := state.check(ctx); err != nil {
		return nil, err
	}
	if state.computeClient == nil {
		client := state.service.client
		if client == nil && state.address.servers.dependencies.AddressCompute != nil {
			var err error
			client, err = state.address.servers.dependencies.AddressCompute(ctx)
			if err = errors.Join(err, state.check(ctx)); err != nil {
				return nil, err
			}
		}
		if client == nil || client.ProviderClient == nil {
			return nil, fmt.Errorf("%w: authenticated raw Compute client is unavailable", resource.ErrUnsupported)
		}
		state.bindRawClient(client)
	}
	if state.computeClient.ProviderClient == nil {
		return nil, invalid("authenticated raw Compute client is required")
	}
	return state.computeClient, state.check(ctx)
}

func (state *automaticIPState) bindRawClient(client *gophercloud.ServiceClient) {
	provider, endpoint, base, version, kind := client.ProviderClient, client.Endpoint, client.ResourceBase, client.Microversion, client.Type
	var changed error
	state.computeClient = client
	state.clientGuard = func(ctx context.Context) error {
		if changed == nil && (client.ProviderClient != provider || client.Endpoint != endpoint || client.ResourceBase != base || client.Microversion != version || client.Type != kind) {
			changed = invalid("raw Compute client source changed during automatic IP workflow")
		}
		return errors.Join(changed, ctx.Err(), context.Cause(ctx))
	}
}

func (state *automaticIPState) rawServer(ctx context.Context) (*ServerAddressView, error) {
	client, err := state.rawClient(ctx)
	if err != nil {
		return nil, err
	}
	id := state.serverID
	if err := resource.ID(id).Validate(); err != nil {
		return nil, err
	}
	response, requestErr := rest.DoJSONGuarded(ctx, client, state.check, "GET", client.ServiceURL("servers", url.PathEscape(id)), nil, nil, 200, 203)
	if response == nil {
		return nil, requestErr
	}
	if requestErr != nil && !state.retainAcceptedServer {
		return nil, requestErr
	}
	if response.StatusCode != 200 && response.StatusCode != 203 {
		return nil, requestErr
	}
	server, decodeErr := rest.Decode[Server](response, "server", func(*Server) *resource.Metadata { return &resource.Metadata{} })
	if decodeErr != nil {
		return nil, errors.Join(requestErr, decodeErr, state.check(ctx))
	}
	if server.ID != id {
		return nil, errors.Join(requestErr, response.Fail(invalid("Nova response does not match server %q", id)), state.check(ctx))
	}
	state.last = server
	if strings.EqualFold(server.Status, "ERROR") {
		return nil, errors.Join(requestErr, response.Fail(&resource.FailedStateError{Resource: "server", ID: id, Status: server.Status}), state.check(ctx))
	}
	view, err := parseServerAddresses(server, state.address.options.networkOrder)
	if err != nil {
		err = response.Fail(err)
	}
	return view, errors.Join(requestErr, err, state.check(ctx))
}

func (state *automaticIPState) skip(ctx context.Context, reason AutomaticIPReason) error {
	state.decision.Reason = reason
	return state.check(ctx)
}

func (state *automaticIPState) setView(view *ServerAddressView) {
	state.address.view = view
	state.address.accessIPv4, state.address.accessIPv6 = state.last.AccessIPv4, state.last.AccessIPv6
	state.decision.Server, state.decision.Addresses = state.last, view
}

func addressHasType(view *ServerAddressView, kind string) bool {
	for _, rows := range view.Addresses {
		for _, row := range rows {
			if row.Type == kind {
				return true
			}
		}
	}
	return false
}

func (state *automaticIPState) decide(ctx context.Context) error {
	if err := state.check(ctx); err != nil {
		return err
	}
	if !state.options.enabled || state.address.options.source == FloatingIPNone {
		return state.skip(ctx, AutomaticIPDisabled)
	}
	if state.address.options.private {
		return state.skip(ctx, AutomaticIPPrivateCloud)
	}
	if state.address.servers.dependencies.NetworkPolicy.UseExternalNetwork() && state.address.accessIPv4 != "" {
		state.address.view.PublicIPv4 = state.address.accessIPv4
		return state.skip(ctx, AutomaticIPExistingPublicIPv4)
	}
	var view *ServerAddressView
	var err error
	if state.last.Addresses == nil {
		view, err = state.rawServer(ctx)
	} else {
		view, err = parseServerAddresses(state.last, state.address.options.networkOrder)
	}
	if err != nil {
		state.decision.Server = state.last
		return err
	}
	state.setView(view)
	if state.address.servers.dependencies.NetworkPolicy.UseExternalNetwork() && state.address.accessIPv4 != "" {
		view.PublicIPv4 = state.address.accessIPv4
		return state.skip(ctx, AutomaticIPExistingPublicIPv4)
	}
	if addressHasType(view, "floating") {
		return state.skip(ctx, AutomaticIPExistingFloating)
	}
	empty := true
	for _, rows := range view.Addresses {
		empty = empty && len(rows) == 0
	}
	if empty {
		return state.skip(ctx, AutomaticIPNoFixedAddress)
	}
	service, err := state.loadNetwork(ctx)
	if err != nil {
		return err
	}
	// Reuse the owned supplement helper with bound lazy services; do not mutate
	// the original collection or run full/default/IPv6 expansion twice.
	reader := *state.address.servers
	reader.dependencies.AddressNetworks = state.loadNetwork
	reader.dependencies.AddressCompute = state.rawClient
	if err := reader.supplementServerAddresses(ctx, state.last.Status, state.address); err != nil {
		return err
	}
	if err := state.address.view.SupplementalError; err != nil {
		return err
	}
	if addressHasType(view, "floating") {
		return state.skip(ctx, AutomaticIPExistingFloating)
	}
	view.PublicIPv4, err = state.address.publicIPv4(ctx, state.address.servers.dependencies.NetworkPolicy.UseExternalNetwork())
	if err != nil {
		return err
	}
	if view.PublicIPv4 != "" {
		return state.skip(ctx, AutomaticIPExistingPublicIPv4)
	}
	view.PrivateIPv4, err = state.address.privateIPv4(ctx, state.address.servers.dependencies.NetworkPolicy.UseInternalNetwork())
	if err != nil {
		return err
	}
	if view.PrivateIPv4 == "" && !addressHasType(view, "fixed") {
		return state.skip(ctx, AutomaticIPNoFixedAddress)
	}
	state.decision.Backend = state.address.options.source
	if service == nil {
		state.decision.Backend = FloatingIPNova
	} else {
		plan, err := state.planner.PrepareEnsure(ctx, network.EnsureFloatingIPRequest{Server: resource.ID(state.serverID), Network: state.input.Network}, network.WithEnsureFloatingIPPolicy(state.policy))
		if err != nil {
			if check := state.check(ctx); check != nil {
				return errors.Join(err, check)
			}
			if absence := cleanFloatingIPAbsence(err); absence != nil {
				return state.skip(ctx, AutomaticIPReason(absence.Reason))
			}
			return err
		}
		state.plan = plan
		state.decision.Selection = plan.Selection()
	}
	state.decision.Needed, state.decision.Reason = true, AutomaticIPNeeded
	return state.check(ctx)
}

// Only a clean, completed semantic absence may become a skip. A joined HTTP,
// cancellation, callback or source error remains a failure.
func cleanFloatingIPAbsence(err error) *network.FloatingIPPlanUnavailableError {
	switch e := err.(type) {
	case *network.FloatingIPPlanUnavailableError:
		return e
	case *resource.ResponseError:
		return nil
	case interface{ Unwrap() []error }:
		if causes := e.Unwrap(); len(causes) == 1 {
			return cleanFloatingIPAbsence(causes[0])
		}
	case interface{ Unwrap() error }:
		return cleanFloatingIPAbsence(e.Unwrap())
	}
	return nil
}

func (state *automaticIPState) observe(ctx context.Context, target string) error {
	for {
		view, err := state.rawServer(ctx)
		if err != nil {
			return err
		}
		if strings.EqualFold(state.last.Status, "ACTIVE") {
			for _, rows := range view.Addresses {
				for _, row := range rows {
					if row.Version == 4 && row.Type == "floating" && row.Address == target {
						return state.check(ctx)
					}
				}
			}
		}
		if state.options.progress != nil {
			progress, err := cloneAutomaticProgressServer(state.last)
			if err != nil {
				return err
			}
			if err := state.options.progress(progress); err != nil {
				return errors.Join(err, state.check(ctx))
			}
		}
		if err := state.check(ctx); err != nil {
			return err
		}
		timer := time.NewTimer(state.options.interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return state.check(ctx)
		case <-timer.C:
		}
	}
}

func cloneAutomaticProgressServer(server *Server) (*Server, error) {
	data, err := json.Marshal(server)
	if err != nil {
		return nil, err
	}
	var copy Server
	if err := json.Unmarshal(data, &copy); err != nil {
		return nil, err
	}
	// Native models omit these fields when marshaling, despite decoding them.
	copy.LaunchedAt, copy.TerminatedAt, copy.ConfigDrive = server.LaunchedAt, server.TerminatedAt, server.ConfigDrive
	image, err := json.Marshal(server.Image)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(image, &copy.Image); err != nil {
		return nil, err
	}
	return &copy, nil
}

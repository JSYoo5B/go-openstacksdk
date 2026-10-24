package compute

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// AutomaticServerCreateOptions reuses existing server and automatic-IP options.
// The workflow always waits for actual ACTIVE and address metadata.
type AutomaticServerCreateOptions struct {
	Server            []CreateServerOption
	AutomaticIP       []AutomaticFloatingIPOption
	FloatingIPNetwork resource.Ref
	// ServerWithoutAddresses defaults to KeepServerWithoutAddresses. The
	// delete policy sends one DELETE and waits for 404 inside the budget.
	ServerWithoutAddresses ServerWithoutAddressesPolicy
}

// AutomaticServerCreateResult retains the original POST separately from the
// latest matching raw GET. Creation may contain creation-only AdminPass.
type AutomaticServerCreateResult struct {
	Creation  *Server
	Server    *Server
	Automatic *AutomaticServerIPResult
}

var ErrServerAddressesUnavailable = errors.New("server addresses unavailable")

// ServerAddressesUnavailableError is an ACTIVE server with explicit empty
// address metadata. The SDK returns the server and never deletes it implicitly;
// Deletion is set only when DeleteServerWithoutAddresses was selected.
type ServerAddressesUnavailableError struct {
	ServerID string
	// EmptyRows reports a non-empty address map whose rows are all empty or
	// null. Python treats such a map as truthy and never deletes the server,
	// so DeleteServerWithoutAddresses leaves it in place as well.
	EmptyRows bool
	Deletion  *ServerDeletion
}

func (e *ServerAddressesUnavailableError) Error() string {
	if e.Deletion != nil && e.Deletion.Err != nil {
		return fmt.Sprintf("ACTIVE server %q: %v AND then could not be deleted", e.ServerID, ErrServerAddressesUnavailable)
	}
	return fmt.Sprintf("ACTIVE server %q: %v", e.ServerID, ErrServerAddressesUnavailable)
}
func (e *ServerAddressesUnavailableError) Unwrap() error { return ErrServerAddressesUnavailable }

// CreateWithAutomaticFloatingIP creates, waits, conditionally assigns and
// observes Nova addresses under one automatic-IP timeout. All policies are
// prepared before POST; known resources survive failure without cleanup.
func (s *Service) CreateWithAutomaticFloatingIP(ctx context.Context, request CreateServerRequest, options AutomaticServerCreateOptions) (*AutomaticServerCreateResult, error) {
	if ctx == nil {
		return nil, invalid("context is required")
	}
	if err := errors.Join(ctx.Err(), context.Cause(ctx)); err != nil {
		return nil, err
	}
	if s == nil || s.API == nil || s.Servers == nil || s.client == nil || s.client.ProviderClient == nil {
		return nil, invalid("authenticated compute service is required")
	}
	if err := options.ServerWithoutAddresses.validate(); err != nil {
		return nil, err
	}
	state, ctx, cancel, err := s.prepareAutomaticIP(ctx, AutomaticFloatingIPRequest{Server: &Server{ID: "pending-creation"}, Network: options.FloatingIPNetwork}, options.AutomaticIP)
	if err != nil {
		return nil, err
	}
	defer cancel()
	// Validate a known legacy backend's version and destination capabilities
	// before creating the server. Discovery of an absent Network remains lazy.
	if state.options.dispatchMode() != ServerIPAutomatic && state.address.options.source != FloatingIPNeutron {
		if _, err := state.novaBackend(ctx); err != nil {
			return nil, err
		}
	}
	state.retainAcceptedServer = true
	reader := *s.Servers
	serverOptions, err := reader.prepareCreateServerOptions(request, options.Server...)
	if err = errors.Join(err, state.check(ctx)); err != nil {
		return nil, err
	}
	policy := serverOptions.waitPolicy
	if policy == nil {
		prepared, err := resource.PrepareWaitOptionsFor[Server]()
		if err != nil {
			return nil, err
		}
		policy = &prepared
	}
	if err := policy.ValidateFixedStatus(); err != nil {
		return nil, err
	}
	client, err := state.rawClient(ctx)
	if err != nil {
		return nil, err
	}
	if err := rest.ValidateTarget(client, client.ServiceURL("servers")); err != nil {
		return nil, err
	}
	if reader.dependencies.DefaultNetworkUsesRoles {
		reader.dependencies.DefaultNetwork = func(ctx context.Context) (resource.Ref, error) {
			if !reader.dependencies.NetworkPolicy.UseExternalNetwork() && !reader.dependencies.NetworkPolicy.UseInternalNetwork() {
				return resource.Ref{}, state.check(ctx)
			}
			roles, err := state.address.loadRoles(ctx)
			if err != nil {
				return resource.Ref{}, err
			}
			if roles.DefaultNetwork == nil {
				return resource.Ref{}, state.check(ctx)
			}
			return resource.ID(roles.DefaultNetwork.ID), state.check(ctx)
		}
	}
	body, err := reader.prepareServerCreateBody(ctx, request, serverOptions)
	if err != nil {
		return nil, errors.Join(err, state.check(ctx))
	}
	encoded, err := body.ToServerCreateMap()
	if err != nil {
		return nil, errors.Join(err, state.check(ctx))
	}
	response, postErr := rest.DoJSONGuarded(ctx, client, state.check, http.MethodPost, client.ServiceURL("servers"), encoded, nil, 200, 202)
	if response == nil {
		return nil, postErr
	}
	result := &AutomaticServerCreateResult{}
	if response.StatusCode != 200 && response.StatusCode != 202 {
		return result, postErr
	}
	created, decodeErr := rest.Decode[Server](response, "server", func(*Server) *resource.Metadata { return &resource.Metadata{} })
	result.Creation, result.Server = created, created
	if err := errors.Join(postErr, decodeErr, state.check(ctx)); err != nil {
		return result, err
	}
	if err := resource.ID(created.ID).Validate(); err != nil {
		return result, response.Fail(err)
	}
	state.serverID, state.last, state.input.Server = created.ID, created, created
	state.address.view.ServerID = created.ID
	state.address.accessIPv4, state.address.accessIPv6 = created.AccessIPv4, created.AccessIPv6
	state.decision.Server = created
	err = state.waitReadyServer(ctx, *policy)
	err = state.applyWithoutAddresses(ctx, err, options.ServerWithoutAddresses, true)
	result.Server = state.last
	copyCreationAdminPass(result)
	if err != nil {
		return result, errors.Join(err, state.check(ctx))
	}
	state.address.accessIPv4, state.address.accessIPv6 = state.last.AccessIPv4, state.last.AccessIPv6
	result.Automatic, err = state.ensure(ctx)
	result.Server = state.last
	copyCreationAdminPass(result)
	return result, errors.Join(err, state.check(ctx))
}

func (state *automaticIPState) waitReadyServer(ctx context.Context, policy resource.WaitPolicy) error {
	collection := resource.NewCollection(resource.Adapter[Server]{Kind: "server", FixedWaitStatus: true, WaitGuard: state.check,
		ID: func(server *Server) string { return server.ID },
		Get: func(ctx context.Context, id string) (*Server, error) {
			if id != state.serverID {
				return nil, invalid("server wait identity changed")
			}
			view, err := state.rawServer(ctx)
			if err != nil {
				return nil, err
			}
			if strings.EqualFold(state.last.Status, "ACTIVE") && state.last.Addresses != nil {
				rows := 0
				for _, addresses := range view.Addresses {
					rows += len(addresses)
				}
				if rows == 0 {
					return nil, &ServerAddressesUnavailableError{ServerID: state.serverID, EmptyRows: len(state.last.Addresses) > 0}
				}
			}
			return state.last, nil
		},
		Status: func(server *Server) string {
			if strings.EqualFold(server.Status, "ACTIVE") && server.Addresses == nil {
				return "address_metadata_pending"
			}
			return server.Status
		},
	})
	_, err := collection.Wait(ctx, resource.ID(state.serverID), "ACTIVE", resource.WithWaitPolicy(policy))
	var failed *resource.FailedStateError
	var fault *ServerFaultError
	if errors.As(err, &failed) && !errors.As(err, &fault) && strings.EqualFold(failed.Status, "ERROR") && state.last != nil && strings.EqualFold(state.last.Status, "ERROR") {
		err = newServerFaultError(state.last, state.lastFault)
	}
	return errors.Join(err, state.check(ctx))
}

// copyCreationAdminPass mirrors Python setting admin_password on the returned
// server: a creation-only adminPass is copied onto the latest Server when the
// GET model has none. The copy is a detached Server, so Automatic evidence
// keeps the raw GET model; Creation keeps its own value.
func copyCreationAdminPass(result *AutomaticServerCreateResult) {
	if result.Creation == nil || result.Server == nil || result.Server == result.Creation || result.Creation.AdminPass == "" || result.Server.AdminPass != "" {
		return
	}
	copied := *result.Server
	copied.AdminPass = result.Creation.AdminPass
	result.Server = &copied
}

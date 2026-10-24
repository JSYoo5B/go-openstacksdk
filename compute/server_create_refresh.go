package compute

import (
	"context"
)

// RefreshedServerCreateResult is Python's cloud create_server wait=False
// result split into typed parts. Creation is the POST response, Server is the
// single refreshed GET with the creation adminPass copied onto it, and
// Addresses is the add_server_interfaces equivalent computed for that Server.
type RefreshedServerCreateResult struct {
	Creation  *Server
	Server    *Server
	Addresses *ServerAddressView
}

// CreateRefreshedServer follows Python's cloud create_server with
// wait=False: POST, one GET, the ERROR fault check, then interface expansion
// through ExpandServerInterfaces with the given address options. When the
// refreshed Server lacks adminPass, the creation adminPass is copied onto it,
// as Python sets admin_password on the returned server. Any failure returns
// the parts already known; the POST is never repeated and nothing is deleted.
// WithWait is rejected; use CreateWithAutomaticFloatingIP to wait.
func (s *Service) CreateRefreshedServer(ctx context.Context, request CreateServerRequest, serverOptions []CreateServerOption, addressOptions ...ServerAddressOption) (*RefreshedServerCreateResult, error) {
	if s == nil || s.Servers == nil {
		return nil, invalid("compute service is required")
	}
	if ctx == nil {
		return nil, invalid("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	o, err := s.Servers.prepareCreateServerOptions(request, append(append([]CreateServerOption(nil), serverOptions...), WithCreateRefresh())...)
	if err != nil {
		return nil, err
	}
	created, err := s.Servers.createServerPrepared(ctx, request, o)
	if err != nil {
		return nil, err
	}
	result := &RefreshedServerCreateResult{Creation: created, Server: created}
	refreshed, err := s.Servers.refreshCreatedServer(ctx, created)
	if refreshed != created {
		copied := *refreshed
		if copied.AdminPass == "" {
			copied.AdminPass = created.AdminPass
		}
		result.Server = &copied
		if fault, ok := err.(*ServerFaultError); ok {
			fault.Server = result.Server
		}
	}
	if err != nil {
		return result, err
	}
	result.Addresses, err = s.ExpandServerInterfaces(ctx, result.Server, addressOptions...)
	return result, err
}

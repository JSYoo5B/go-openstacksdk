package compute

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/network"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// CreateWithFloatingIP creates a server, waits for its actual ACTIVE status,
// then reuses/allocates and associates a floating IPv4 address and waits for its
// actual ACTIVE status. All stages share one overall deadline. Failures preserve
// known resources. Connection supplies services; no app builder is required.
// This explicit workflow does not apply Python's automatic need/skip policy or
// wait for Nova's addresses to reflect the Neutron assignment.
func (s *Servers) CreateWithFloatingIP(ctx context.Context, request CreateServerWithFloatingIPRequest, options ...CreateServerWithFloatingIPOption) (*ServerFloatingIPResult, error) {
	if ctx == nil {
		return nil, invalid("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	o := createServerWithFloatingIPOptions{timeout: 5 * time.Minute}
	for _, apply := range options {
		if apply == nil {
			return nil, invalid("nil server floating IP workflow option")
		}
		if err := apply(&o); err != nil {
			return nil, err
		}
	}
	serverOptions, err := s.prepareCreateServerOptions(request.Server, o.serverOptions...)
	if err != nil {
		return nil, err
	}
	policy, err := network.PrepareEnsureFloatingIPOptions(ctx, o.ipOptions...)
	if err != nil {
		return nil, err
	}
	if request.FloatingIPNetwork != (resource.Ref{}) {
		if err := request.FloatingIPNetwork.Validate(); err != nil {
			return nil, fmt.Errorf("external network: %w", err)
		}
	}
	if s.dependencies.FloatingIPs == nil {
		return nil, s.wrap("prepare floating IP", fmt.Errorf("%w: floating IP service is unavailable", resource.ErrUnsupported))
	}
	if o.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.timeout)
		defer cancel()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ips, err := s.dependencies.FloatingIPs(ctx)
	if err != nil {
		return nil, s.wrap("prepare floating IP", err)
	}
	if ips == nil {
		return nil, s.wrap("prepare floating IP", fmt.Errorf("%w: floating IP service is unavailable", resource.ErrUnsupported))
	}
	policy, err = ips.PrepareEnsureActive(ctx, network.WithEnsureFloatingIPPolicy(policy))
	if err != nil {
		return nil, s.wrap("prepare floating IP", err)
	}
	created, err := s.createServerPrepared(ctx, request.Server, serverOptions)
	if err != nil {
		return nil, err
	}
	result := &ServerFloatingIPResult{Server: created}
	if created == nil {
		return result, s.wrap("verify creation", fmt.Errorf("Nova returned no server"))
	}
	if err := resource.ID(created.ID).Validate(); err != nil {
		return result, s.wrap("verify creation", err)
	}
	ready, err := s.Wait(ctx, resource.ID(created.ID), "ACTIVE", serverOptions.waitOptions...)
	if err != nil {
		return result, s.wrap("create/wait", err)
	}
	if ready == nil || ready.ID != created.ID || !strings.EqualFold(ready.Status, "ACTIVE") {
		return result, s.wrap("verify ACTIVE server", fmt.Errorf("Nova response does not match ACTIVE server %q", created.ID))
	}
	result.Server = ready
	if err := ctx.Err(); err != nil {
		return result, err
	}
	assignment, err := ips.Ensure(ctx, network.EnsureFloatingIPRequest{
		Server: resource.ID(created.ID), Network: request.FloatingIPNetwork,
	}, network.WithEnsureFloatingIPPolicy(policy))
	result.Assignment = assignment
	if err != nil {
		return result, s.wrap("create/floating IP", err)
	}
	return result, ctx.Err()
}

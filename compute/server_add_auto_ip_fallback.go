package compute

import (
	"context"
	"errors"

	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// attachAutoIPFallbackRow mirrors the Neutron branch of the pinned
// _attach_ip_to_server for a reused row that AvailableFloatingIP obtained from
// Nova after a Neutron NotFound. Python passes that row to
// _neutron_attach_ip_to_server and ignores a NotFoundException, without
// falling through to a Nova addFloatingIp action. Other failures, such as a
// server without ports, are returned.
func (s *Service) attachAutoIPFallbackRow(ctx context.Context, result *AddAutoIPResult) error {
	if s.Servers == nil || s.Servers.dependencies.AddressNetworks == nil {
		return invalid("the Neutron attach of a Nova fallback row needs a network service")
	}
	service, err := s.Servers.dependencies.AddressNetworks(ctx)
	if err != nil {
		return err
	}
	if service == nil || service.FloatingIPs == nil {
		return invalid("the Neutron attach of a Nova fallback row needs a network service")
	}
	update, err := service.FloatingIPs.AttachByID(ctx, network.AttachFloatingIPByIDRequest{Server: resource.ID(result.Server.ID), FloatingIPID: result.FloatingIPID})
	result.NeutronAttach = update
	if err != nil && ctx.Err() == nil && neutronAttachNotFound(err) {
		result.NeutronNotFound = err
		return nil
	}
	return err
}

// neutronAttachNotFound reports a single HTTP 404 cause, the only failure the
// pinned helper ignores here. Joined causes, such as a source change or a
// cancellation next to the 404, are not ignored.
func neutronAttachNotFound(err error) bool {
	switch e := err.(type) {
	case *resource.NotFoundError:
		return true
	case gophercloud.ErrUnexpectedResponseCode:
		return e.Actual == 404
	case *gophercloud.ErrUnexpectedResponseCode:
		return e != nil && e.Actual == 404
	case *resource.ResponseError:
		return e.StatusCode == 404
	case interface{ Unwrap() []error }:
		causes := e.Unwrap()
		return len(causes) == 1 && neutronAttachNotFound(causes[0])
	case interface{ Unwrap() error }:
		return neutronAttachNotFound(e.Unwrap())
	default:
		return errors.Is(err, resource.ErrNotFound)
	}
}

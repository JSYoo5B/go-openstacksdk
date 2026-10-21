package openstack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	baremetal "github.com/JSYoo5B/go-openstacksdk/baremetal/v1"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes"
	neutronports "github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// The helpers below combine the Bare Metal and Network services like the
// pinned openstacksdk cloud layer (attach_port_to_machine and friends). No
// mutation is replayed: the Python default retry on HTTP 409/503 is not
// performed, and an acknowledgement is not treated as completion.

// machineNodeID finds the machine by name or UUID and returns its UUID. A
// missing machine is an error wrapping resource.ErrNotFound.
func machineNodeID(ctx context.Context, operation string, bm *baremetal.Service, machine string) (string, error) {
	record, err := bm.GetMachine(ctx, machine)
	if err != nil {
		return "", request.Wrap(operation, "machines", err)
	}
	if record == nil {
		return "", request.Wrap(operation, "machines", fmt.Errorf("%w: machine not found: %s", resource.ErrNotFound, machine))
	}
	var id string
	if raw, ok := record.Resource.Body["id"]; !ok || json.Unmarshal(raw, &id) != nil || id == "" {
		return "", request.Wrap(operation, "machines", fmt.Errorf("machine %q response has no usable id", machine))
	}
	return id, nil
}

func (c *Connection) machineServices(ctx context.Context, operation string) (*baremetal.Service, *neutronService, error) {
	bm, err := c.BareMetalV1(ctx)
	if err != nil {
		return nil, nil, request.Wrap(operation, "machines", err)
	}
	net, err := c.NetworkV2(ctx)
	if err != nil {
		return nil, nil, request.Wrap(operation, "machines", err)
	}
	return bm, &neutronService{ports: net.Ports}, nil
}

type neutronService struct{ ports *neutronports.API }

func (n *neutronService) findPort(ctx context.Context, operation, identity string) (*neutronports.Port, error) {
	port, err := n.ports.FindIdentity(ctx, identity, resource.WithIdentityFindIgnoreMissing(false))
	if err != nil {
		return nil, request.Wrap(operation, "machines", err)
	}
	return port, nil
}

// AttachPortToMachine attaches the Network service port (not a bare metal
// port) to the machine as a VIF. The machine is found by name or UUID; a
// missing machine fails with resource.ErrNotFound. The port is found by ID,
// then by exact name, and must exist. Success is the service's acknowledgement.
func (c *Connection) AttachPortToMachine(ctx context.Context, machine, port string) error {
	bm, net, err := c.machineServices(ctx, "AttachPortToMachine")
	if err != nil {
		return err
	}
	nodeID, err := machineNodeID(ctx, "AttachPortToMachine", bm, machine)
	if err != nil {
		return err
	}
	found, err := net.findPort(ctx, "AttachPortToMachine", port)
	if err != nil {
		return err
	}
	return request.Wrap("AttachPortToMachine", "machines", bm.Nodes.AttachVirtualInterface(ctx, nodeID, nodes.VirtualInterfaceOpts{ID: found.ID}))
}

// DetachPortFromMachine detaches the Network service port from the machine.
// Like the pinned source's default ignore_missing=True, an HTTP 400 answer
// means the VIF was not attached: the result is (false, nil). A successful
// detach returns (true, nil). Every other failure is returned.
func (c *Connection) DetachPortFromMachine(ctx context.Context, machine, port string) (bool, error) {
	bm, net, err := c.machineServices(ctx, "DetachPortFromMachine")
	if err != nil {
		return false, err
	}
	nodeID, err := machineNodeID(ctx, "DetachPortFromMachine", bm, machine)
	if err != nil {
		return false, err
	}
	found, err := net.findPort(ctx, "DetachPortFromMachine", port)
	if err != nil {
		return false, err
	}
	err = bm.Nodes.DetachVirtualInterface(ctx, nodeID, found.ID)
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if errors.As(err, &unexpected) && unexpected.Actual == http.StatusBadRequest {
		return false, nil
	}
	if err != nil {
		return false, request.Wrap("DetachPortFromMachine", "machines", err)
	}
	return true, nil
}

// ListPortsAttachedToMachine lists the machine's VIF IDs and finds each as a
// Network service port by ID or exact name; a VIF that cannot be found fails
// the whole call with resource.ErrNotFound and no partial result.
func (c *Connection) ListPortsAttachedToMachine(ctx context.Context, machine string) ([]*neutronports.Port, error) {
	bm, net, err := c.machineServices(ctx, "ListPortsAttachedToMachine")
	if err != nil {
		return nil, err
	}
	nodeID, err := machineNodeID(ctx, "ListPortsAttachedToMachine", bm, machine)
	if err != nil {
		return nil, err
	}
	vifs, err := bm.Nodes.ListVirtualInterfaces(ctx, nodeID)
	if err != nil {
		return nil, request.Wrap("ListPortsAttachedToMachine", "machines", err)
	}
	out := make([]*neutronports.Port, 0, len(vifs))
	for _, vif := range vifs {
		port, err := net.findPort(ctx, "ListPortsAttachedToMachine", vif.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, port)
	}
	return out, nil
}

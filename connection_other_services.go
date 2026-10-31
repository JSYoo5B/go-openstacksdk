package openstack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	baremetal "github.com/JSYoo5B/go-openstacksdk/baremetal/v1"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes"
	dnsv2 "github.com/JSYoo5B/go-openstacksdk/dns/v2"
	"github.com/JSYoo5B/go-openstacksdk/dns/v2/recordsets"
	"github.com/JSYoo5B/go-openstacksdk/dns/v2/zones"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudobjectstorage"
	neutronports "github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

// objectStorageDependencies supplies the Connection guard, lazy Swift service
// and location to the cloud-layer Swift helpers.
func (c *Connection) objectStorageDependencies() cloudobjectstorage.Dependencies {
	return cloudobjectstorage.Dependencies{
		Guard:         c.recordListContext,
		ObjectStorage: c.ObjectStorageV1,
		Location:      c.CurrentLocation,
	}
}

// SearchContainers shares the cached Swift service and supplies configured
// Connection location before identifier/mapping/expression selection. Explicit
// per-call typed location overrides remain owned; a reset DTO uses the Connection
// default without requiring callers to construct a resource or builder.
func (c *Connection) SearchContainers(ctx context.Context, options ...objectstorage.ContainerSearchOption) (*objectstorage.ContainerSearchResult, error) {
	return cloudobjectstorage.SearchContainers(ctx, c.objectStorageDependencies(), options...)
}

// SearchObjects shares the cached Swift service, collects its container listing
// and supplies the configured Connection location before cloud filtering.
// Explicit per-call location overrides remain owned; a reset DTO returns to the
// Connection default without asking users to build individual resource objects.
func (c *Connection) SearchObjects(ctx context.Context, container string, options ...objectstorage.ObjectSearchOption) (*objectstorage.ObjectSearchResult, error) {
	return cloudobjectstorage.SearchObjects(ctx, c.objectStorageDependencies(), container, options...)
}

// GetObjectText supplies the cached Swift facade for eager owned text or output handling.
func (c *Connection) GetObjectText(ctx context.Context, container, object string, options ...objectstorage.ObjectGetOption) (*objectstorage.ObjectGetResult, error) {
	return cloudobjectstorage.GetObjectText(ctx, c.objectStorageDependencies(), container, object, options...)
}

// StreamObjectChunks supplies the cached Swift facade when the SDK-owned
// iterator is first consumed. Close and Err preserve cleanup after early break.
func (c *Connection) StreamObjectChunks(ctx context.Context, container, object string, options ...objectstorage.ObjectStreamOption) objectstorage.ObjectChunkStream {
	return cloudobjectstorage.StreamObjectChunks(ctx, c.objectStorageDependencies(), container, object, options...)
}

// The DNS cloud helpers authenticate through Connection.DNS and then run the
// dns/v2 Service helpers of the same names; see dns/v2/cloud.go for contracts.

func (c *Connection) ListDNSZones(ctx context.Context, options ...zones.ListOption) ([]*zones.Zone, error) {
	s, err := c.DNS(ctx)
	if err != nil {
		return nil, err
	}
	return s.ListDNSZones(ctx, options...)
}

func (c *Connection) GetDNSZone(ctx context.Context, nameOrID string, options ...resource.IdentityFindOption) (*zones.Zone, error) {
	s, err := c.DNS(ctx)
	if err != nil {
		return nil, err
	}
	return s.GetDNSZone(ctx, nameOrID, options...)
}

func (c *Connection) SearchDNSZones(ctx context.Context, nameOrID string, options ...dnsv2.DNSSearchOption) (*dnsv2.DNSSearchResult[zones.Zone], error) {
	s, err := c.DNS(ctx)
	if err != nil {
		return nil, err
	}
	return s.SearchDNSZones(ctx, nameOrID, options...)
}

func (c *Connection) CreateDNSZone(ctx context.Context, name string, options ...dnsv2.DNSZoneCreateOption) (*zones.Zone, error) {
	s, err := c.DNS(ctx)
	if err != nil {
		return nil, err
	}
	return s.CreateDNSZone(ctx, name, options...)
}

func (c *Connection) UpdateDNSZone(ctx context.Context, nameOrID string, opts zones.UpdateOpts, options ...zones.UpdateOption) (*zones.Zone, error) {
	s, err := c.DNS(ctx)
	if err != nil {
		return nil, err
	}
	return s.UpdateDNSZone(ctx, nameOrID, opts, options...)
}

func (c *Connection) DeleteDNSZone(ctx context.Context, nameOrID string) (bool, error) {
	s, err := c.DNS(ctx)
	if err != nil {
		return false, err
	}
	return s.DeleteDNSZone(ctx, nameOrID)
}

func (c *Connection) ListDNSRecordSets(ctx context.Context, zone string) ([]*recordsets.RecordSet, error) {
	s, err := c.DNS(ctx)
	if err != nil {
		return nil, err
	}
	return s.ListDNSRecordSets(ctx, zone)
}

func (c *Connection) GetDNSRecordSet(ctx context.Context, zone, nameOrID string, options ...resource.IdentityFindOption) (*recordsets.RecordSet, error) {
	s, err := c.DNS(ctx)
	if err != nil {
		return nil, err
	}
	return s.GetDNSRecordSet(ctx, zone, nameOrID, options...)
}

func (c *Connection) SearchDNSRecordSets(ctx context.Context, zone, nameOrID string, options ...dnsv2.DNSSearchOption) (*dnsv2.DNSSearchResult[recordsets.RecordSet], error) {
	s, err := c.DNS(ctx)
	if err != nil {
		return nil, err
	}
	return s.SearchDNSRecordSets(ctx, zone, nameOrID, options...)
}

func (c *Connection) CreateDNSRecordSet(ctx context.Context, zone, name, recordsetType string, records []string, description string, ttl int) (*recordsets.RecordSet, error) {
	s, err := c.DNS(ctx)
	if err != nil {
		return nil, err
	}
	return s.CreateDNSRecordSet(ctx, zone, name, recordsetType, records, description, ttl)
}

func (c *Connection) UpdateDNSRecordSet(ctx context.Context, zone, nameOrID string, opts recordsets.UpdateOpts, options ...recordsets.UpdateOption) (*recordsets.RecordSet, error) {
	s, err := c.DNS(ctx)
	if err != nil {
		return nil, err
	}
	return s.UpdateDNSRecordSet(ctx, zone, nameOrID, opts, options...)
}

func (c *Connection) DeleteDNSRecordSet(ctx context.Context, zone, nameOrID string) (bool, error) {
	s, err := c.DNS(ctx)
	if err != nil {
		return false, err
	}
	return s.DeleteDNSRecordSet(ctx, zone, nameOrID)
}

func (c *Connection) orchestrationStackClient(ctx context.Context, operation string) (*gophercloud.ServiceClient, error) {
	if ctx == nil {
		return nil, request.Wrap(operation, "stacks", fmt.Errorf("%w: context is required", resource.ErrInvalidOption))
	}
	if c == nil {
		return nil, request.Wrap(operation, "stacks", fmt.Errorf("%w: Connection is required", resource.ErrInvalidOption))
	}
	service, err := c.OrchestrationV1(ctx)
	if err != nil {
		return nil, request.Wrap(operation, "stacks", err)
	}
	return service.RawClient(), nil
}

// SearchStacks maps the Python cloud search_stacks; see stackcloud.SearchStacks.
func (c *Connection) SearchStacks(ctx context.Context, input stackcloud.SearchRequest, options ...stackcloud.Option) (*stackcloud.SearchResult, error) {
	heat, err := c.orchestrationStackClient(ctx, "search_stacks")
	if err != nil {
		return nil, err
	}
	return stackcloud.SearchStacks(ctx, heat, input, options...)
}

// CreateStack maps the Python cloud create_stack; see stackcloud.CreateStack.
func (c *Connection) CreateStack(ctx context.Context, input stackcloud.CreateRequest, options ...stackcloud.Option) (*stackcloud.MutationResult, error) {
	heat, err := c.orchestrationStackClient(ctx, "create_stack")
	if err != nil {
		return nil, err
	}
	return stackcloud.CreateStack(ctx, heat, input, options...)
}

// UpdateStack maps the Python cloud update_stack; see stackcloud.UpdateStack.
func (c *Connection) UpdateStack(ctx context.Context, input stackcloud.UpdateRequest, options ...stackcloud.Option) (*stackcloud.MutationResult, error) {
	heat, err := c.orchestrationStackClient(ctx, "update_stack")
	if err != nil {
		return nil, err
	}
	return stackcloud.UpdateStack(ctx, heat, input, options...)
}

// DeleteStack maps the Python cloud delete_stack; see stackcloud.DeleteStack.
func (c *Connection) DeleteStack(ctx context.Context, input stackcloud.DeleteRequest, options ...stackcloud.Option) (*stackcloud.DeleteResult, error) {
	heat, err := c.orchestrationStackClient(ctx, "delete_stack")
	if err != nil {
		return nil, err
	}
	return stackcloud.DeleteStack(ctx, heat, input, options...)
}

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

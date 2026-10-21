package v1

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/ports"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/baremetal/v1/nodes"
)

// This file provides the single-service "machine" helpers that the pinned
// openstacksdk exposes on its cloud layer (BaremetalCloudMixin). Each helper is
// a thin, explicit composition of the node and port APIs. None of them replays
// a mutation, waits implicitly or infers completion from an acknowledgement.
// Operations that need other services (Neutron port lookup) or provision-state
// waits are intentionally not provided here.

func machineIdentity(operation, identity string) error {
	if identity == "" {
		return request.Wrap(operation, "machines", fmt.Errorf("%w: machine name or UUID must not be empty", resource.ErrInvalidOption))
	}
	return nil
}

// ListMachines lists every node with the basic (non-detail) collection route
// and follows all pages, like list(baremetal.nodes()).
func (s *Service) ListMachines(ctx context.Context) ([]*nodes.NodeRecord, error) {
	values, err := s.Nodes.AllRecords(ctx)
	return values, request.Wrap("ListMachines", "machines", err)
}

// GetMachine finds a node by name or UUID with detailed fields. A missing node
// is reported as (nil, nil), like find_node(..., ignore_missing=True). Other
// failures, including cancellation, are returned unchanged.
func (s *Service) GetMachine(ctx context.Context, nameOrID string) (*nodes.NodeRecord, error) {
	if err := machineIdentity("GetMachine", nameOrID); err != nil {
		return nil, err
	}
	value, err := s.Nodes.FindRecord(ctx, nameOrID, nodes.WithRecordFindIgnoreMissing(true))
	return value, request.Wrap("GetMachine", "machines", err)
}

// ListNICs lists every bare metal port with details.
func (s *Service) ListNICs(ctx context.Context) ([]*ports.Port, error) {
	return s.collectPorts(ctx, "ListNICs")
}

// ListNICsForMachine lists detailed ports whose node_uuid equals uuid. Only a
// UUID is accepted, as in the pinned source; an empty value is rejected rather
// than silently listing every port.
func (s *Service) ListNICsForMachine(ctx context.Context, uuid string) ([]*ports.Port, error) {
	if err := machineIdentity("ListNICsForMachine", uuid); err != nil {
		return nil, err
	}
	return s.collectPorts(ctx, "ListNICsForMachine", ports.WithListDetailOptions(ports.ListOpts{NodeUUID: uuid}))
}

// GetNICByMAC returns the first detailed port with the given address, or nil
// when none matches. Like the pinned source it consumes the complete listing
// before choosing the first row. An empty address is rejected.
func (s *Service) GetNICByMAC(ctx context.Context, mac string) (*ports.Port, error) {
	if mac == "" {
		return nil, request.Wrap("GetNICByMAC", "machines", fmt.Errorf("%w: MAC address must not be empty", resource.ErrInvalidOption))
	}
	values, err := s.collectPorts(ctx, "GetNICByMAC", ports.WithListDetailOptions(ports.ListOpts{Address: mac}))
	if err != nil || len(values) == 0 {
		return nil, err
	}
	return values[0], nil
}

// GetMachineByMAC finds the port by address and then finds its node by the
// port's node_uuid. It returns nil when no port matches or the node is missing.
func (s *Service) GetMachineByMAC(ctx context.Context, mac string) (*nodes.NodeRecord, error) {
	nic, err := s.GetNICByMAC(ctx, mac)
	if err != nil || nic == nil {
		return nil, err
	}
	return s.GetMachine(ctx, nic.NodeUUID)
}

func (s *Service) collectPorts(ctx context.Context, operation string, options ...ports.ListDetailOption) ([]*ports.Port, error) {
	var out []*ports.Port
	for value, err := range s.Ports.ListDetail(ctx, options...) {
		if err != nil {
			return nil, request.Wrap(operation, "machines", err)
		}
		out = append(out, value)
	}
	return out, nil
}

// SetMachinePowerOn requests "power on". Success is only the service's
// acknowledgement; callers must read power_state afterwards.
func (s *Service) SetMachinePowerOn(ctx context.Context, nameOrID string) error {
	return s.setMachinePower(ctx, "SetMachinePowerOn", nameOrID, nodes.PowerOn)
}

// SetMachinePowerOff requests "power off" (acknowledgement only).
func (s *Service) SetMachinePowerOff(ctx context.Context, nameOrID string) error {
	return s.setMachinePower(ctx, "SetMachinePowerOff", nameOrID, "power off")
}

// SetMachinePowerReboot requests "rebooting" (acknowledgement only).
func (s *Service) SetMachinePowerReboot(ctx context.Context, nameOrID string) error {
	return s.setMachinePower(ctx, "SetMachinePowerReboot", nameOrID, "rebooting")
}

func (s *Service) setMachinePower(ctx context.Context, operation, nameOrID string, target nodes.TargetPowerState) error {
	if err := machineIdentity(operation, nameOrID); err != nil {
		return err
	}
	return request.Wrap(operation, "machines", s.Nodes.ChangePowerState(ctx, nameOrID, nodes.PowerStateOpts{Target: target}))
}

// SetMachineMaintenance enables maintenance. A nil or empty reason sends no
// reason (the typed body omits it), which the service treats like the explicit
// JSON null the pinned source sends; a non-empty reason is sent unchanged. An
// explicit empty-string reason cannot be expressed. Success is only an
// acknowledgement and no follow-up read is made.
func (s *Service) SetMachineMaintenance(ctx context.Context, nameOrID string, reason *string) error {
	if err := machineIdentity("SetMachineMaintenance", nameOrID); err != nil {
		return err
	}
	var opts nodes.MaintenanceOpts
	if reason != nil {
		opts.Reason = *reason
	}
	return request.Wrap("SetMachineMaintenance", "machines", s.Nodes.SetMaintenance(ctx, nameOrID, opts))
}

// RemoveMachineFromMaintenance clears maintenance mode (acknowledgement only).
func (s *Service) RemoveMachineFromMaintenance(ctx context.Context, nameOrID string) error {
	if err := machineIdentity("RemoveMachineFromMaintenance", nameOrID); err != nil {
		return err
	}
	return request.Wrap("RemoveMachineFromMaintenance", "machines", s.Nodes.UnsetMaintenance(ctx, nameOrID))
}

// PatchMachine sends a JSON Patch to the node. A conflict is returned as an
// error; unlike the Python default the SDK never retries it.
func (s *Service) PatchMachine(ctx context.Context, nameOrID string, patch nodes.UpdateOpts) (*nodes.Node, error) {
	if err := machineIdentity("PatchMachine", nameOrID); err != nil {
		return nil, err
	}
	value, err := s.Nodes.Update(ctx, nameOrID, patch)
	return value, request.Wrap("PatchMachine", "machines", err)
}

// PurgeMachineInstanceInfo removes /instance_info with one JSON Patch.
func (s *Service) PurgeMachineInstanceInfo(ctx context.Context, nameOrID string) (*nodes.Node, error) {
	return s.PatchMachine(ctx, nameOrID, nodes.UpdateOpts{nodes.UpdateOperation{Op: nodes.RemoveOp, Path: "/instance_info"}})
}

// MachineValidationError reports required interfaces whose validation result
// was false, null or absent.
type MachineValidationError struct {
	Machine  string
	Failures []string
}

func (e *MachineValidationError) Error() string {
	return fmt.Sprintf("Validation failed for required interfaces of node %s: %s", e.Machine, strings.Join(e.Failures, ", "))
}

// ValidateMachine validates readiness for deployment (boot, deploy,
// management, power) or, with forDeploy false, only power.
func (s *Service) ValidateMachine(ctx context.Context, nameOrID string, forDeploy bool) error {
	required := []string{"power"}
	if forDeploy {
		required = []string{"boot", "deploy", "management", "power"}
	}
	return s.ValidateMachineInterfaces(ctx, nameOrID, required...)
}

// ValidateMachineInterfaces validates the node and fails with
// *MachineValidationError when a listed interface is present in the response
// without a true result. Interfaces missing from the response are ignored.
// With no required interfaces only the HTTP result is checked.
func (s *Service) ValidateMachineInterfaces(ctx context.Context, nameOrID string, required ...string) error {
	if err := machineIdentity("ValidateMachine", nameOrID); err != nil {
		return err
	}
	var body map[string]struct {
		Result *bool   `json:"result"`
		Reason *string `json:"reason"`
	}
	if err := upstream.Validate(ctx, s.client, nameOrID).ExtractInto(&body); err != nil {
		return request.Wrap("ValidateMachine", "machines", err)
	}
	var failed []string
	for _, key := range sortedKeys(body) {
		if !slices.Contains(required, key) {
			continue
		}
		value := body[key]
		if value.Result != nil && *value.Result {
			continue
		}
		reason := "no reason"
		if value.Reason != nil {
			reason = *value.Reason
		}
		failed = append(failed, fmt.Sprintf("%s (%s)", key, reason))
	}
	if len(failed) > 0 {
		return request.Wrap("ValidateMachine", "machines", &MachineValidationError{Machine: nameOrID, Failures: failed})
	}
	return nil
}

// WaitForMachineReservation polls GET /nodes/{id} until the node has no
// reservation. It always performs the first read (the pinned source skips the
// read when the caller passes a bare name or UUID string). Timeout, interval
// and cancellation use the shared wait options; the lock is best effort and a
// conductor may take it again after this returns.
func (s *Service) WaitForMachineReservation(ctx context.Context, nameOrID string, options ...resource.WaitOption) (*nodes.Node, error) {
	if err := machineIdentity("WaitForMachineReservation", nameOrID); err != nil {
		return nil, err
	}
	if err := resource.ValidateWaitOptions(options...); err != nil {
		return nil, request.Wrap("WaitForMachineReservation", "machines", err)
	}
	waiter := resource.NewCollection(resource.Adapter[nodes.Node]{
		Kind:            "machines",
		FixedWaitStatus: true,
		Get:             func(ctx context.Context, _ string) (*nodes.Node, error) { return s.Nodes.Get(ctx, nameOrID) },
		ID:              func(*nodes.Node) string { return nameOrID },
		Status: func(value *nodes.Node) string {
			if value.Reservation == "" {
				return "unlocked"
			}
			return "locked"
		},
	})
	value, err := waiter.Wait(ctx, resource.ID(nameOrID), "unlocked", options...)
	return value, request.Wrap("WaitForMachineReservation", "machines", err)
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Python iterates the JSON object in response order; Go maps have no
	// order, so failures are reported in stable alphabetical order.
	slices.Sort(keys)
	return keys
}

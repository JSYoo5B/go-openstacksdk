package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/ports"
	"github.com/JSYoo5B/go-openstacksdk/internal/ironicwire"
	"github.com/JSYoo5B/go-openstacksdk/internal/servicewait"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// ErrMachineState is matched by errors.Is when a machine workflow refuses to
// run because of the machine's provision state or association.
var ErrMachineState = errors.New("machine is in a state that does not allow the operation")

// MachineOption configures the machine workflows of this file.
type MachineOption func(*machineConfig)

type machineConfig struct {
	wait       bool
	timeout    *time.Duration
	unlimited  bool
	lock       *time.Duration
	interval   time.Duration
	configDriv string
	state      string
}

// WithMachineWait makes the workflow wait for the final provision state.
func WithMachineWait(wait bool) MachineOption { return func(c *machineConfig) { c.wait = wait } }

// WithMachineTimeout bounds each provision-state wait. Each workflow documents
// the pinned Python default (3600 s for node_set_provision_state, register and
// inspect, 1200 s for activate and deactivate, 600 s for unregister).
func WithMachineTimeout(d time.Duration) MachineOption {
	return func(c *machineConfig) { c.timeout = &d; c.unlimited = false }
}

// WithMachineUnlimitedWait removes the default timeout; only the caller's context
// bounds the wait.
func WithMachineUnlimitedWait() MachineOption { return func(c *machineConfig) { c.unlimited = true } }

// WithMachineLockTimeout bounds the wait for a reservation lock (default 600 s).
func WithMachineLockTimeout(d time.Duration) MachineOption {
	return func(c *machineConfig) { c.lock = &d }
}

// WithMachineWaitInterval sets the polling pause (default two seconds).
func WithMachineWaitInterval(d time.Duration) MachineOption {
	return func(c *machineConfig) { c.interval = d }
}

// WithMachineConfigDrive sends a config drive string as the provisioning
// configdrive. It is passed through unchanged; the SDK neither reads a path nor
// builds an image from a directory (the pinned source does not either).
func WithMachineConfigDrive(drive string) MachineOption {
	return func(c *machineConfig) { c.configDriv = drive }
}

// WithRegisterProvisionState selects enroll, manageable or available (default).
func WithRegisterProvisionState(state string) MachineOption {
	return func(c *machineConfig) { c.state = state }
}

func newMachineConfig(options []MachineOption) machineConfig {
	var c machineConfig
	for _, o := range options {
		if o != nil {
			o(&c)
		}
	}
	return c
}

func (c machineConfig) bound(ctx context.Context, def time.Duration) (context.Context, context.CancelFunc) {
	if c.unlimited {
		return ctx, func() {}
	}
	d := def
	if c.timeout != nil {
		d = *c.timeout
	}
	return context.WithTimeout(ctx, d)
}

func (c machineConfig) waitOptions() []nodes.ProvisionWaitOption {
	if c.interval > 0 {
		return []nodes.ProvisionWaitOption{nodes.WithProvisionWaitInterval(c.interval)}
	}
	return nil
}

func (c machineConfig) provision(ctx context.Context, s *Service, id, target string, def time.Duration, drive string) (*nodes.Node, error) {
	req := nodes.ProvisionStateRequest{Target: target}
	if drive != "" {
		req.ConfigDrive = drive
	}
	var opts []nodes.SetProvisionStateOption
	if c.wait {
		opts = append(opts, nodes.WithProvisionStateWait(c.waitOptions()...))
	}
	ctx, cancel := c.bound(ctx, def)
	defer cancel()
	return s.Nodes.SetProvisionState(ctx, id, req, opts...)
}

// NodeSetProvisionState runs one provisioning action like
// BaremetalCloudMixin.node_set_provision_state: it calls
// Nodes.SetProvisionState with the optional config drive string and, with
// WithMachineWait, waits for the expected stable state. The wait timeout
// defaults to 3600 seconds and applies to the whole call (the Python timer only
// covers the wait). The acknowledgement alone never proves completion.
func (s *Service) NodeSetProvisionState(ctx context.Context, nameOrID, state string, options ...MachineOption) (*nodes.Node, error) {
	if err := machineIdentity("NodeSetProvisionState", nameOrID); err != nil {
		return nil, err
	}
	c := newMachineConfig(options)
	node, err := c.provision(ctx, s, nameOrID, state, time.Hour, c.configDriv)
	return node, request.Wrap("NodeSetProvisionState", "machines", err)
}

// ActivateNode provisions the node ("active") with an optional config drive; the
// wait timeout defaults to 1200 seconds. It returns only an error, like Python.
func (s *Service) ActivateNode(ctx context.Context, uuid string, options ...MachineOption) error {
	if err := machineIdentity("ActivateNode", uuid); err != nil {
		return err
	}
	c := newMachineConfig(options)
	_, err := c.provision(ctx, s, uuid, "active", 1200*time.Second, c.configDriv)
	return request.Wrap("ActivateNode", "machines", err)
}

// DeactivateNode tears the node down ("deleted"); the wait timeout defaults to
// 1200 seconds.
func (s *Service) DeactivateNode(ctx context.Context, uuid string, options ...MachineOption) error {
	if err := machineIdentity("DeactivateNode", uuid); err != nil {
		return err
	}
	c := newMachineConfig(options)
	_, err := c.provision(ctx, s, uuid, "deleted", 1200*time.Second, "")
	return request.Wrap("DeactivateNode", "machines", err)
}

func machineStateError(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrMachineState, fmt.Sprintf(format, args...))
}

// InspectMachine inspects a machine like inspect_machine. An "available"
// machine without an instance is moved to manageable first and provided again
// afterwards; one with an instance is refused. The machine must then be
// manageable or "inspect failed". Every step waits (Python ignores its wait
// argument and always waits), each with its own timeout of 3600 seconds by
// default. A failure in the middle leaves the machine in the intermediate
// state; nothing is rolled back.
func (s *Service) InspectMachine(ctx context.Context, nameOrID string, options ...MachineOption) (*nodes.Node, error) {
	if err := machineIdentity("InspectMachine", nameOrID); err != nil {
		return nil, err
	}
	c := newMachineConfig(options)
	c.wait = true
	fail := func(err error) (*nodes.Node, error) { return nil, request.Wrap("InspectMachine", "machines", err) }
	node, err := s.Nodes.Get(ctx, nameOrID)
	if err != nil {
		return fail(err)
	}
	back := false
	if node.ProvisionState == "available" {
		if node.InstanceUUID != "" {
			return fail(machineStateError("refusing to inspect available machine %s which is associated with an instance (instance_uuid %s)", node.UUID, node.InstanceUUID))
		}
		back = true
		if node, err = c.provision(ctx, s, node.UUID, "manage", time.Hour, ""); err != nil {
			return fail(err)
		}
	}
	if node.ProvisionState != "manageable" && node.ProvisionState != "inspect failed" {
		return fail(machineStateError("machine %s must be in 'manageable', 'inspect failed' or 'available' provision state to start inspection, the current state is %s", node.UUID, node.ProvisionState))
	}
	if node, err = c.provision(ctx, s, node.UUID, "inspect", time.Hour, ""); err != nil {
		return fail(err)
	}
	if back {
		if node, err = c.provision(ctx, s, node.UUID, "provide", time.Hour, ""); err != nil {
			return fail(err)
		}
	}
	return node, nil
}

// RegisterError reports a failed RegisterMachine. Cause is the original failure.
// RolledBack is true when the node was deleted again; RollbackErr holds the
// deletion error otherwise. Python hides rollback failures.
type RegisterError struct {
	Cause       error
	NodeID      string
	RolledBack  bool
	RollbackErr error
}

func (e *RegisterError) Error() string { return e.Cause.Error() }
func (e *RegisterError) Unwrap() error { return e.Cause }

// NICAddresses builds port options from plain MAC addresses.
func NICAddresses(addresses ...string) []ports.CreateOpts {
	out := make([]ports.CreateOpts, len(addresses))
	for i, a := range addresses {
		out[i] = ports.CreateOpts{Address: a}
	}
	return out
}

// RegisterMachine registers a node like register_machine. The provision state
// (WithRegisterProvisionState: enroll, manageable or available, default
// available) other than available creates the node with the initial state
// "enroll" through Nodes.CreateWithProvisionState; available creates it with
// the server default. A node in enroll is then managed and waited for (timeout
// 3600 s) and its reservation lock is waited for (lock timeout 600 s), ports are
// created with the node UUID (a port that already names a node fails), and an
// available target runs "provide" (waiting only with WithMachineWait).
//
// Any failure after the node exists makes the workflow delete the ports it
// created and then the node, using a detached context bounded to 30 seconds so a
// cancelled or timed-out registration is also cleaned up. Rollback failures are
// reported in *RegisterError. The node creation POST is never replayed.
func (s *Service) RegisterMachine(ctx context.Context, node nodes.CreateOpts, nics []ports.CreateOpts, options ...MachineOption) (*nodes.Node, error) {
	const op = "RegisterMachine"
	c := newMachineConfig(options)
	state := c.state
	if state == "" {
		state = "available"
	}
	if state != "enroll" && state != "manageable" && state != "available" {
		return nil, request.Wrap(op, "machines", fmt.Errorf("%w: initial provision state must be enroll, manageable or available, got %s", resource.ErrInvalidOption, state))
	}
	for i, nic := range nics {
		if nic.NodeUUID != "" {
			return nil, request.Wrap(op, "machines", fmt.Errorf("%w: nic %d already sets node_uuid", resource.ErrInvalidOption, i))
		}
	}
	initial := nodes.CreateProvisionState("")
	if state != "available" {
		initial = nodes.CreateEnroll
	}
	machine, err := s.Nodes.CreateWithProvisionState(ctx, initial, node)
	if err != nil {
		return nil, request.Wrap(op, "machines", err)
	}
	var created []string
	fail := func(cause error) (*nodes.Node, error) {
		rb := &RegisterError{Cause: request.Wrap(op, "machines", cause), NodeID: machine.UUID}
		bg, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		for _, id := range created {
			_ = s.Ports.Delete(bg, id)
		}
		if err := s.Nodes.Delete(bg, machine.UUID); err != nil {
			rb.RollbackErr = err
		} else {
			rb.RolledBack = true
		}
		return nil, rb
	}
	if machine.ProvisionState == "enroll" && state != "enroll" {
		manage := c
		manage.wait = true
		if machine, err = manage.provision(ctx, s, machine.UUID, "manage", time.Hour, ""); err != nil {
			return fail(err)
		}
		lockCtx, cancel := context.WithTimeout(ctx, durationOr(c.lock, 600*time.Second))
		var wopts []nodes.WaitOption
		if c.interval > 0 {
			wopts = append(wopts, nodes.WithWaitInterval(c.interval))
		}
		machine, err = s.Nodes.WaitForReservation(lockCtx, machine, wopts...)
		cancel()
		if err != nil {
			return fail(err)
		}
	}
	for _, nic := range nics {
		nic.NodeUUID = machine.UUID
		port, err := s.Ports.Create(ctx, nic)
		if err != nil {
			return fail(err)
		}
		created = append(created, port.UUID)
	}
	if machine.ProvisionState != "available" && state == "available" {
		if machine, err = c.provision(ctx, s, machine.UUID, "provide", time.Hour, ""); err != nil {
			return fail(err)
		}
	}
	return machine, nil
}

func durationOr(d *time.Duration, def time.Duration) time.Duration {
	if d != nil {
		return *d
	}
	return def
}

func recordString(rec *nodes.NodeRecord, key string) string {
	var v string
	if raw, ok := rec.Resource.Body[key]; ok {
		_ = json.Unmarshal(raw, &v)
	}
	return v
}

// UnregisterMachine removes a machine like unregister_machine. The machine must
// exist and must not be active, cleaning, clean wait or clean failed. Its
// reservation lock is waited for (lock timeout, default 600 s), the first port of
// each listed MAC address is deleted (a missing port is skipped, a 404 on the
// delete is ignored) and the node is deleted (404 ignored, like the default
// ignore_missing). Nothing is rolled back if a later step fails.
func (s *Service) UnregisterMachine(ctx context.Context, addresses []string, uuid string, options ...MachineOption) error {
	const op = "UnregisterMachine"
	if err := machineIdentity(op, uuid); err != nil {
		return err
	}
	c := newMachineConfig(options)
	rec, err := s.GetMachine(ctx, uuid)
	if err != nil {
		return request.Wrap(op, "machines", err)
	}
	if rec == nil {
		return request.Wrap(op, "machines", fmt.Errorf("%w: machine unregister failed to find machine %s", resource.ErrNotFound, uuid))
	}
	state := recordString(rec, "provision_state")
	if slices.Contains([]string{"active", "cleaning", "clean wait", "clean failed"}, state) {
		return request.Wrap(op, "machines", machineStateError("error unregistering node '%s' due to current provision state '%s'", uuid, state))
	}
	snapshot := &nodes.Node{UUID: recordString(rec, "id"), Reservation: recordString(rec, "reservation")}
	lockCtx, cancel := context.WithTimeout(ctx, durationOr(c.lock, 600*time.Second))
	var wopts []nodes.WaitOption
	if c.interval > 0 {
		wopts = append(wopts, nodes.WithWaitInterval(c.interval))
	}
	_, err = s.Nodes.WaitForReservation(lockCtx, snapshot, wopts...)
	cancel()
	if err != nil {
		return request.Wrap(op, "machines", fmt.Errorf("error unregistering node '%s': exception occurred while waiting to be able to proceed: %w", uuid, err))
	}
	for _, address := range addresses {
		if address == "" {
			return request.Wrap(op, "machines", fmt.Errorf("%w: MAC address must not be empty", resource.ErrInvalidOption))
		}
		var first *ports.Port
		for p, err := range s.Ports.List(ctx, ports.WithListOptions(ports.ListOpts{Address: address})) {
			if err != nil {
				return request.Wrap(op, "machines", err)
			}
			first = p
			break
		}
		if first == nil {
			continue
		}
		if err := s.Ports.Delete(ctx, first.UUID); err != nil && !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			return request.Wrap(op, "machines", err)
		}
	}
	if err := s.Nodes.Delete(ctx, uuid); err != nil && !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		return request.Wrap(op, "machines", err)
	}
	return nil
}

// MachineUpdate is the result of UpdateMachine. Changes is nil when nothing
// differed; Node is nil then and Record holds the machine that was read.
type MachineUpdate struct {
	Record  *nodes.NodeRecord
	Node    *nodes.Node
	Changes []string
}

// UpdateMachine updates a machine like update_machine. It reads the machine
// (missing is an error), compares its declared attribute view with the view
// extended by attrs and returns Changes (sorted attribute-name paths). Without a
// difference nothing is written. Otherwise Nodes.UpdateFromOriginal sends the
// structural patch against the fields read, with maintenance routed as in
// Nodes.UpdateAttributes. Unknown attribute names fail before any write. The
// patch has no move operations and Changes are sorted, unlike the pinned
// jsonpatch output.
func (s *Service) UpdateMachine(ctx context.Context, nameOrID string, attrs map[string]any) (*MachineUpdate, error) {
	const op = "UpdateMachine"
	if err := machineIdentity(op, nameOrID); err != nil {
		return nil, err
	}
	rec, err := s.GetMachine(ctx, nameOrID)
	if err != nil {
		return nil, request.Wrap(op, "machines", err)
	}
	if rec == nil {
		return nil, request.Wrap(op, "machines", fmt.Errorf("%w: machine update failed to find machine %s", resource.ErrNotFound, nameOrID))
	}
	src, dst := map[string]json.RawMessage{}, map[string]json.RawMessage{}
	for key, value := range attrs {
		client, ok := nodes.NodeAttributeName(key)
		if !ok {
			return nil, request.Wrap(op, "machines", fmt.Errorf("%w: attribute %q is not a declared node field", resource.ErrInvalidOption, key))
		}
		raw, err := json.Marshal(value)
		if err != nil {
			return nil, request.Wrap(op, "machines", fmt.Errorf("%w: attribute %q: %w", resource.ErrInvalidOption, key, err))
		}
		src[client] = rec.Resource.Body[client]
		dst[client] = raw
	}
	ops := ironicwire.Diff(src, dst)
	if len(ops) == 0 {
		return &MachineUpdate{Record: rec}, nil
	}
	original := map[string]json.RawMessage{}
	for key, value := range rec.Wire.Body {
		if _, ok := nodes.NodeAttributeWireName(key); ok {
			original[key] = value
		}
	}
	node, err := s.Nodes.UpdateFromOriginal(ctx, recordString(rec, "id"), original, attrs)
	if err != nil {
		return nil, request.Wrap(op, "machines", err)
	}
	changes := ironicwire.ChangedPaths(ops)
	sort.Strings(changes)
	return &MachineUpdate{Record: rec, Node: node, Changes: changes}, nil
}

// WaitForStatus waits until the selected status attribute of the resource equals
// status, with the pinned wait_for_status defaults: a two second interval, no
// timeout and the single failure state ERROR (case-insensitive). Options from the
// resource package apply last (WithTimeout, WithPollInterval,
// WithFailureStates, WithStatusAttribute, WithProgressCallback). Python's
// default attribute "status" does not exist on bare metal resources; the
// collection's own status accessor (provision_state for nodes, state for
// allocations) is used unless WithStatusAttribute selects another one. A read
// error ends the wait and a deadline is the context.
func WaitForStatus[T any](ctx context.Context, collection *resource.Collection[T], ref resource.Ref, status string, options ...resource.WaitOption) (*T, error) {
	return servicewait.State(ctx, collection, ref, status, "ERROR", options...)
}

// WaitForDelete observes until the resource is gone (a 404 read or an empty
// list match), with the pinned wait_for_delete defaults: a two second interval and
// 120 seconds. It only reads. Options from the resource package apply last.
func WaitForDelete[T any](ctx context.Context, collection *resource.Collection[T], ref resource.Ref, options ...resource.WaitOption) error {
	return servicewait.Delete(ctx, collection, ref, options...)
}

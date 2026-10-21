package nodes

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/ironicwire"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// ExpectedProvisionStates maps a provisioning action to the stable state the
// node is expected to reach, like the pinned Python EXPECTED_STATES.
var expectedProvisionStates = map[string]string{
	"active": "active", "adopt": "active", "clean": "manageable", "deleted": "available",
	"inspect": "manageable", "manage": "manageable", "provide": "available",
	"rebuild": "active", "rescue": "rescue",
}

// provisioningVersions are the microversions that introduced a provisioning verb.
var provisioningVersions = map[string]string{
	"abort": "1.13", "adopt": "1.17", "clean": "1.15", "inspect": "1.6", "manage": "1.4",
	"provide": "1.4", "rescue": "1.38", "unrescue": "1.38", "unhold": "1.85", "service": "1.87",
}

var expectedPowerStates = map[string]string{
	"power on": "power on", "power off": "power off", "rebooting": "power on",
	"soft power off": "power off", "soft rebooting": "power on",
}

// ExpectedProvisionState returns the stable state expected after a provisioning
// action, and false for actions Python cannot wait for (abort, unrescue, unhold,
// service and unknown verbs).
func ExpectedProvisionState(target string) (string, bool) {
	state, ok := expectedProvisionStates[target]
	return state, ok
}

// ExpectedPowerState returns the power state expected after a power action.
func ExpectedPowerState(target string) (string, bool) {
	state, ok := expectedPowerStates[target]
	return state, ok
}

// ErrProvisionFailure is matched by errors.Is for a *ProvisionFailureError.
var ErrProvisionFailure = errors.New("node reached a provision failure state")

// ProvisionFailureError reports a node that entered a failure state while a
// provision-state wait was running. It also matches resource.ErrFailedState.
type ProvisionFailureError struct {
	NodeID    string
	State     string
	LastError string
	// Manage is true for the special enroll-after-manage failure.
	Manage bool
}

func (e *ProvisionFailureError) Error() string {
	if e.Manage {
		return fmt.Sprintf("Node %s could not reach state manageable: failed to verify management credentials; the last error is %s", e.NodeID, e.LastError)
	}
	return fmt.Sprintf("Node %s reached failure state '%s'; the last error is %s", e.NodeID, e.State, e.LastError)
}

// Is matches ErrProvisionFailure and resource.ErrFailedState.
func (e *ProvisionFailureError) Is(target error) bool {
	return target == ErrProvisionFailure || target == resource.ErrFailedState
}

// StateReached reports whether the node reached expected, mirroring
// Node._check_state_reached. An empty provision state stands for the JSON null
// of API 1.1 and counts as "available". With abort enabled a state ending in
// " failed" or equal to "error" fails, and so does "enroll" when "manageable"
// is expected and last_error is set.
func StateReached(node *Node, expected string, abort bool) (bool, error) {
	if node.ProvisionState == expected || (expected == "available" && node.ProvisionState == "") {
		return true, nil
	}
	if !abort {
		return false, nil
	}
	if strings.HasSuffix(node.ProvisionState, " failed") || node.ProvisionState == "error" {
		return false, &ProvisionFailureError{NodeID: node.UUID, State: node.ProvisionState, LastError: node.LastError}
	}
	if expected == "manageable" && node.ProvisionState == "enroll" && node.LastError != "" {
		return false, &ProvisionFailureError{NodeID: node.UUID, State: node.ProvisionState, LastError: node.LastError, Manage: true}
	}
	return false, nil
}

type provisionWait struct {
	interval time.Duration
	abort    bool
	keep     bool
}

// ProvisionWaitOption configures AwaitProvisionState and AwaitProvisionStates.
type ProvisionWaitOption func(*provisionWait)

// WithProvisionWaitInterval sets the pause between polls; non-positive values
// keep DefaultWaitInterval.
func WithProvisionWaitInterval(d time.Duration) ProvisionWaitOption {
	return func(c *provisionWait) {
		if d > 0 {
			c.interval = d
		}
	}
}

// WithAbortOnFailedState chooses whether a failure state ends the wait. The
// default is true.
func WithAbortOnFailedState(abort bool) ProvisionWaitOption {
	return func(c *provisionWait) { c.abort = abort }
}

// WithContinueOnFailure is the fail=false form of AwaitProvisionStates: failures
// and the context deadline are collected in the result instead of returned as an
// error.
func WithContinueOnFailure() ProvisionWaitOption { return func(c *provisionWait) { c.keep = true } }

func provisionWaitFrom(options []ProvisionWaitOption) provisionWait {
	cfg := provisionWait{interval: DefaultWaitInterval, abort: true}
	for _, option := range options {
		if option != nil {
			option(&cfg)
		}
	}
	return cfg
}

// ProvisionWaitResult is the Go form of the pinned WaitResult: nodes that
// reached the state, nodes that hit a failure and nodes still waiting when the
// context deadline ended the wait.
type ProvisionWaitResult struct {
	Success []*Node
	Failure []*Node
	Timeout []*Node
}

// AwaitProvisionState polls the node until StateReached is true. It reads
// immediately, then once per interval. A failure state returns
// *ProvisionFailureError (also matching resource.ErrFailedState) unless
// WithAbortOnFailedState(false) was given. The deadline is the context.
func (a *API) AwaitProvisionState(ctx context.Context, id, expected string, options ...ProvisionWaitOption) (*Node, error) {
	cfg := provisionWaitFrom(options)
	var failure error
	node, err := a.poll(ctx, "AwaitProvisionState", id, waitConfig{interval: cfg.interval}, func(n *Node) bool {
		reached, err := StateReached(n, expected, cfg.abort)
		if err != nil {
			failure = err
			return true
		}
		return reached
	})
	if err != nil {
		return nil, err
	}
	if failure != nil {
		return node, request.Wrap("AwaitProvisionState", "nodes", failure)
	}
	return node, nil
}

// AwaitProvisionStates waits for several nodes with the pinned
// wait_for_nodes_provision_state rules. Every round reads every node still
// waiting, in order, and then classifies them. By default the first failure
// returns *ProvisionFailureError together with the result so far, and a context
// deadline is returned as an error. With WithContinueOnFailure, failures go to
// Result.Failure and a context deadline puts the unfinished nodes into
// Result.Timeout with a nil error (also when the deadline ends a read; the nodes
// of the last completed round, or bare references in the first round, are
// reported); other errors, including cancellation without
// a deadline, are still returned. A read error ends the wait.
func (a *API) AwaitProvisionStates(ctx context.Context, ids []string, expected string, options ...ProvisionWaitOption) (ProvisionWaitResult, error) {
	cfg := provisionWaitFrom(options)
	var result ProvisionWaitResult
	refs := append([]string(nil), ids...)
	var remaining, last []*Node
	wrap := func(err error) error { return request.Wrap("AwaitProvisionStates", "nodes", err) }
	for {
		remaining = nil
		nodes := make([]*Node, 0, len(refs))
		for _, ref := range refs {
			node, err := a.Get(ctx, ref)
			if err != nil {
				if cfg.keep && errors.Is(err, context.DeadlineExceeded) {
					// The deadline ended a read: the nodes of the last completed
					// round (or bare references in the first round) are unfinished.
					result.Timeout = last
					if result.Timeout == nil {
						for _, r := range refs {
							result.Timeout = append(result.Timeout, &Node{UUID: r})
						}
					}
					return result, nil
				}
				return result, wrap(err)
			}
			nodes = append(nodes, node)
		}
		var next []string
		for _, node := range nodes {
			reached, err := StateReached(node, expected, cfg.abort)
			switch {
			case err != nil && !cfg.keep:
				return result, wrap(err)
			case err != nil:
				result.Failure = append(result.Failure, node)
			case reached:
				result.Success = append(result.Success, node)
			default:
				remaining = append(remaining, node)
				next = append(next, node.UUID)
			}
		}
		if len(remaining) == 0 {
			return result, nil
		}
		refs, last = next, remaining
		timer := time.NewTimer(cfg.interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			cause := context.Cause(ctx)
			if cfg.keep && errors.Is(ctx.Err(), context.DeadlineExceeded) {
				result.Timeout = remaining
				return result, nil
			}
			return result, wrap(cause)
		case <-timer.C:
		}
	}
}

// ProvisionStateRequest is the request of SetProvisionState. Slices keep
// omission apart from emptiness: a nil slice is omitted and an empty non-nil
// slice is sent as an empty list. A nil RescuePassword is omitted and an empty
// string is sent. ConfigDrive and Runbook are sent only when non-empty, like the
// pinned truthiness checks.
type ProvisionStateRequest struct {
	Target       string
	ConfigDrive  any // string, []byte or map[string]any
	CleanSteps   []map[string]any
	DeploySteps  []map[string]any
	ServiceSteps []map[string]any
	// RescuePassword is sent when non-nil, also when empty.
	RescuePassword *string
	Runbook        string
}

// SetProvisionStateOption configures SetProvisionState.
type SetProvisionStateOption func(*setProvisionConfig)

type setProvisionConfig struct {
	wait    bool
	options []ProvisionWaitOption
}

// WithProvisionStateWait waits for the stable state expected for the target
// (see ExpectedProvisionState) after the request is acknowledged. Targets with no
// known expected state fail before any request.
func WithProvisionStateWait(options ...ProvisionWaitOption) SetProvisionStateOption {
	return func(c *setProvisionConfig) { c.wait = true; c.options = append(c.options, options...) }
}

func truthyConfigDrive(value any) (any, bool, error) {
	switch v := value.(type) {
	case nil:
		return nil, false, nil
	case string:
		return v, v != "", nil
	case []byte:
		if len(v) == 0 {
			return nil, false, nil
		}
		if !utf8.Valid(v) {
			return nil, false, fmt.Errorf("%w: config drive must be a dictionary or a base64 encoded string", resource.ErrInvalidOption)
		}
		return string(v), true, nil
	case map[string]any:
		return v, len(v) > 0, nil
	}
	return nil, false, fmt.Errorf("%w: config drive must be a string, bytes or a map, got %T", resource.ErrInvalidOption, value)
}

func (r ProvisionStateRequest) build() (body map[string]any, required string, err error) {
	invalid := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
	}
	target := r.Target
	if v, ok := provisioningVersions[target]; ok {
		required = v
	}
	drive, hasDrive, err := truthyConfigDrive(r.ConfigDrive)
	if err != nil {
		return nil, "", err
	}
	if hasDrive {
		if _, isDict := drive.(map[string]any); isDict {
			required = "1.56"
		} else if target == "rebuild" {
			required = "1.35"
		}
	}
	if len(r.DeploySteps) > 0 {
		required = "1.69"
	}
	body = map[string]any{"target": target}
	if r.Runbook != "" {
		if r.CleanSteps != nil {
			return nil, "", invalid("please provide either clean steps or a runbook, but not both")
		}
		if r.ServiceSteps != nil {
			return nil, "", invalid("please provide either service steps or a runbook, but not both")
		}
		if target != "clean" && target != "service" {
			return nil, "", invalid(`a runbook can only be provided when setting target provision state to any of "[clean, service]"`)
		}
		body["runbook"] = r.Runbook
	}
	if hasDrive {
		if target != "active" && target != "rebuild" {
			return nil, "", invalid(`config drive can only be provided with "active" and "rebuild" targets`)
		}
		body["configdrive"] = drive
	}
	if r.CleanSteps != nil {
		if target != "clean" {
			return nil, "", invalid(`clean steps can only be provided with "clean" target`)
		}
		body["clean_steps"] = r.CleanSteps
	}
	if r.DeploySteps != nil {
		if target != "active" && target != "rebuild" {
			return nil, "", invalid(`deploy steps can only be provided with "deploy" and "rebuild" target`)
		}
		body["deploy_steps"] = r.DeploySteps
	}
	if r.ServiceSteps != nil {
		if target != "service" {
			return nil, "", invalid(`service steps can only be provided with "service" target`)
		}
		body["service_steps"] = r.ServiceSteps
	}
	if r.RescuePassword != nil {
		if target != "rescue" {
			return nil, "", invalid(`rescue password can only be provided with "rescue" target`)
		}
		body["rescue_password"] = *r.RescuePassword
	}
	return body, required, nil
}

// SetProvisionState runs one provisioning action like the pinned
// set_node_provision_state and returns the node read afterwards.
//
// The request is validated first (no HTTP): config drive only with active or
// rebuild, clean steps only with clean, deploy steps only with active or
// rebuild, service steps only with service, rescue password only with rescue, a
// runbook only with clean or service and never together with steps. The
// microversion must be at least the one that introduced the verb (config drive
// map 1.56, rebuild drive 1.35, deploy steps 1.69, runbook 1.92); a configured
// client microversion that is lower, or an endpoint that cannot reach it, fails
// with resource.ErrUnsupported. An unset client microversion is replaced by the
// highest version shared with the endpoint, at most 1.109.
//
// PUT /nodes/{id}/states/provision is sent once and only 202 succeeds. The
// call returns the node read after the acknowledgement; with
// WithProvisionStateWait it instead polls until the expected stable state and
// returns that node. The deadline of a wait is the context; a failure state
// returns *ProvisionFailureError. The acknowledgement alone never proves that
// the transition finished.
func (a *API) SetProvisionState(ctx context.Context, id string, req ProvisionStateRequest, options ...SetProvisionStateOption) (*Node, error) {
	const op = "SetProvisionState"
	if id == "" {
		return nil, nodeInvalid(op, "node ID or name must not be empty")
	}
	var cfg setProvisionConfig
	for _, option := range options {
		if option != nil {
			option(&cfg)
		}
	}
	body, required, err := req.build()
	if err != nil {
		return nil, request.Wrap(op, "nodes", err)
	}
	expected := ""
	if cfg.wait {
		var ok bool
		if expected, ok = expectedProvisionStates[req.Target]; !ok {
			return nil, nodeInvalid(op, "for target %q the expected state is not known, cannot wait for it", req.Target)
		}
	}
	version, err := ironicwire.Assert(ctx, a.client, required, ironicwire.NodeCeiling, "")
	if err == nil && req.Runbook != "" {
		version, err = ironicwire.Assert(ctx, a.client, "1.92", ironicwire.NodeCeiling, "")
	}
	if err != nil {
		return nil, request.Wrap(op, "nodes", err)
	}
	client := ironicwire.At(a.client, version)
	if _, err := client.Put(ctx, a.nodeURL(client, id, "states", "provision"), body, nil, &gophercloud.RequestOpts{OkCodes: []int{202}}); err != nil {
		return nil, request.Wrap(op, "nodes", err)
	}
	if cfg.wait {
		node, err := a.AwaitProvisionState(ctx, id, expected, cfg.options...)
		return node, request.Wrap(op, "nodes", err)
	}
	return a.getOrNil(ctx, op, id)
}

// SetPowerStateOption configures SetPowerState.
type SetPowerStateOption func(*setPowerConfig)

type setPowerConfig struct {
	wait    bool
	options []WaitOption
}

// WithPowerStateWait waits for the power state expected for the target (see
// ExpectedPowerState) after the request is acknowledged.
func WithPowerStateWait(options ...WaitOption) SetPowerStateOption {
	return func(c *setPowerConfig) { c.wait = true; c.options = append(c.options, options...) }
}

// SetPowerState sends PUT /nodes/{id}/states/power with {"target": target} once
// and accepts only 202. With WithPowerStateWait the target must be a known power
// action, otherwise the call fails before any request, and the call then polls
// until the expected power state. A soft target ("soft power off", "soft
// rebooting") needs microversion 1.27 under the same rule as SetProvisionState;
// other targets use the configured client microversion unchanged. The
// acknowledgement alone never proves the power change.
func (a *API) SetPowerState(ctx context.Context, id, target string, options ...SetPowerStateOption) error {
	const op = "SetPowerState"
	if id == "" {
		return nodeInvalid(op, "node ID or name must not be empty")
	}
	var cfg setPowerConfig
	for _, option := range options {
		if option != nil {
			option(&cfg)
		}
	}
	expected := ""
	if cfg.wait {
		var ok bool
		if expected, ok = expectedPowerStates[target]; !ok {
			return nodeInvalid(op, "cannot use target power state %q with wait, the expected state is not known", target)
		}
	}
	client := a.client
	if strings.HasPrefix(target, "soft ") {
		version, err := ironicwire.Assert(ctx, a.client, "1.27", ironicwire.NodeCeiling, "")
		if err != nil {
			return request.Wrap(op, "nodes", err)
		}
		client = ironicwire.At(a.client, version)
	}
	if _, err := client.Put(ctx, a.nodeURL(client, id, "states", "power"), map[string]any{"target": target}, nil, &gophercloud.RequestOpts{OkCodes: []int{202}}); err != nil {
		return request.Wrap(op, "nodes", err)
	}
	if cfg.wait {
		_, err := a.WaitForPowerState(ctx, id, expected, cfg.options...)
		return request.Wrap(op, "nodes", err)
	}
	return nil
}

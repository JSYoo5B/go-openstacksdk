package nodes

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/baremetal/v1/nodes"
)

// DefaultValidationInterfaces lists the interfaces that must pass validation
// for provisioning, matching the pinned Python default of boot, deploy and power.
func DefaultValidationInterfaces() []string { return []string{"boot", "deploy", "power"} }

// DefaultWaitInterval is the pause between polls of the wait helpers, matching
// the two second cadence of the pinned Python wait loop.
const DefaultWaitInterval = 2 * time.Second

// ErrValidationFailed is matched by errors.Is for any *ValidationError.
var ErrValidationFailed = errors.New("node validation failed")

// InterfaceValidation is the validation entry of one interface. A nil field
// means the server sent null or omitted the key.
type InterfaceValidation struct {
	Result *bool
	Reason *string
}

// InterfaceFailure names a required interface that did not validate.
type InterfaceFailure struct {
	Interface string
	Reason    string
}

// ValidationError reports required interfaces that failed validation.
type ValidationError struct {
	Node     string
	Failures []InterfaceFailure
}

func (e *ValidationError) Error() string {
	parts := make([]string, len(e.Failures))
	for i, f := range e.Failures {
		parts[i] = fmt.Sprintf("%s (%s)", f.Interface, f.Reason)
	}
	return fmt.Sprintf("Validation failed for required interfaces of node %s: %s", e.Node, strings.Join(parts, ", "))
}

// Is reports ErrValidationFailed.
func (e *ValidationError) Is(target error) bool { return target == ErrValidationFailed }

// ValidateRequired validates the node and fails with a *ValidationError when a
// required interface is present in the response with a falsy result. Required
// interfaces absent from the response are not checked. A nil or empty required
// list checks nothing. The returned map holds every interface the server
// reported, also alongside a *ValidationError. Failures are ordered by
// interface name.
func (a *API) ValidateRequired(ctx context.Context, id string, required []string) (map[string]InterfaceValidation, error) {
	result := upstream.Validate(ctx, a.client, id)
	if result.Err != nil {
		return nil, request.Wrap("ValidateRequired", "nodes", result.Err)
	}
	raw, ok := result.Body.(map[string]any)
	if !ok {
		return nil, request.Wrap("ValidateRequired", "nodes", fmt.Errorf("validate response is %T, not an object", result.Body))
	}
	out := make(map[string]InterfaceValidation, len(raw))
	var failures []InterfaceFailure
	for key, value := range raw {
		entry, _ := value.(map[string]any)
		var v InterfaceValidation
		reason := "no reason"
		if r, present := entry["result"]; present {
			if b, isBool := r.(bool); isBool {
				v.Result = &b
			}
		}
		if r, present := entry["reason"]; present {
			if s, isString := r.(string); isString {
				v.Reason = &s
				reason = s
			} else if r == nil {
				reason = "None"
			}
		}
		out[key] = v
		if slices.Contains(required, key) && !validationTruthy(entry["result"]) {
			failures = append(failures, InterfaceFailure{Interface: key, Reason: reason})
		}
	}
	if len(failures) > 0 {
		slices.SortFunc(failures, func(x, y InterfaceFailure) int { return strings.Compare(x.Interface, y.Interface) })
		return out, request.Wrap("ValidateRequired", "nodes", &ValidationError{Node: id, Failures: failures})
	}
	return out, nil
}

func validationTruthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	case float64:
		return x != 0
	case []any:
		return len(x) > 0
	case map[string]any:
		return len(x) > 0
	}
	return true
}

// SetMaintenanceAndGet enables maintenance mode and then reads the node. A nil
// reason sends an explicit JSON null; a non-nil reason is sent verbatim, also
// when empty. Only HTTP 202 is accepted for the PUT. The node is read only
// after the PUT was acknowledged; a failed PUT performs no read and a failed
// read leaves the maintenance change in effect remotely.
func (a *API) SetMaintenanceAndGet(ctx context.Context, id string, reason *string) (*Node, error) {
	var value any
	if reason != nil {
		value = *reason
	}
	_, err := a.client.Put(ctx, a.client.ServiceURL("nodes", id, "maintenance"), map[string]any{"reason": value}, nil, &gophercloud.RequestOpts{OkCodes: []int{202}})
	if err != nil {
		return nil, request.Wrap("SetMaintenanceAndGet", "nodes", err)
	}
	return a.getOrNil(ctx, "SetMaintenanceAndGet", id)
}

// UnsetMaintenanceAndGet disables maintenance mode and then reads the node
// with the same ordering and failure semantics as SetMaintenanceAndGet.
func (a *API) UnsetMaintenanceAndGet(ctx context.Context, id string) (*Node, error) {
	if err := a.UnsetMaintenance(ctx, id); err != nil {
		return nil, request.Wrap("UnsetMaintenanceAndGet", "nodes", err)
	}
	return a.getOrNil(ctx, "UnsetMaintenanceAndGet", id)
}

// WaitOption configures the polling helpers.
type WaitOption func(*waitConfig)

type waitConfig struct{ interval time.Duration }

// WithWaitInterval sets the pause between polls. Non-positive values keep
// DefaultWaitInterval.
func WithWaitInterval(d time.Duration) WaitOption {
	return func(c *waitConfig) {
		if d > 0 {
			c.interval = d
		}
	}
}

func waitConfigFrom(options []WaitOption) waitConfig {
	cfg := waitConfig{interval: DefaultWaitInterval}
	for _, o := range options {
		if o != nil {
			o(&cfg)
		}
	}
	return cfg
}

// poll reads the node immediately and then after each interval until done
// reports true. Context expiry ends the wait and stays observable through
// errors.Is. A read error ends the wait unchanged.
func (a *API) poll(ctx context.Context, op, id string, cfg waitConfig, done func(*Node) bool) (*Node, error) {
	for {
		node, err := a.Get(ctx, id)
		if err != nil {
			return nil, request.Wrap(op, "nodes", err)
		}
		if done(node) {
			return node, nil
		}
		timer := time.NewTimer(cfg.interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, request.Wrap(op, "nodes", context.Cause(ctx))
		case <-timer.C:
		}
	}
}

// WaitForPowerState polls the node until PowerState equals expected. It reads
// immediately, waits the configured interval between reads, and returns the
// node that matched. The deadline is the context; there is no separate
// timeout value.
func (a *API) WaitForPowerState(ctx context.Context, id string, expected string, options ...WaitOption) (*Node, error) {
	return a.poll(ctx, "WaitForPowerState", id, waitConfigFrom(options), func(n *Node) bool { return n.PowerState == expected })
}

// WaitForReservation waits until the node's reservation lock is released. The
// given snapshot decides whether to wait: a snapshot without a reservation is
// returned at once without HTTP, mirroring the pinned Python behavior that
// only consults the passed object. Otherwise the node is re-read until its
// Reservation is empty. The release is best effort; a conductor may lock the
// node again immediately.
func (a *API) WaitForReservation(ctx context.Context, snapshot *Node, options ...WaitOption) (*Node, error) {
	if snapshot == nil {
		return nil, request.Wrap("WaitForReservation", "nodes", errors.New("node snapshot is required"))
	}
	if snapshot.Reservation == "" {
		return snapshot, nil
	}
	return a.poll(ctx, "WaitForReservation", snapshot.UUID, waitConfigFrom(options), func(n *Node) bool { return n.Reservation == "" })
}

// getOrNil reads the node and returns a nil node together with any error.
func (a *API) getOrNil(ctx context.Context, op, id string) (*Node, error) {
	node, err := a.Get(ctx, id)
	if err != nil {
		return nil, request.Wrap(op, "nodes", err)
	}
	return node, nil
}

package allocations

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/ironicwire"
	"github.com/JSYoo5B/go-openstacksdk/internal/pollschedule"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// PatchOp is one JSON Patch operation; Value is omitted only when nil.
type PatchOp = ironicwire.PatchOp

func allocationNames() ironicwire.Names {
	names := ironicwire.Names{Attributes: map[string]string{}, ID: "uuid"}
	for _, f := range listRecordProfile().Body {
		names.Attributes[f.Name] = f.Wire
	}
	return names
}

func allocationInvalid(op, format string, args ...any) error {
	return request.Wrap(op, "allocations", fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...)))
}

// PatchResult holds the allocation returned by PatchAllocation. Allocation is
// nil and Sent false when the patch was empty and nothing was sent.
type PatchResult struct {
	Allocation *Allocation
	Sent       bool
}

// PatchAllocation sends PATCH /allocations/{id} (microversion 1.57 or later is
// selected by the client; the SDK does not raise it). Path first segments are
// converted from attribute names to wire names, an operation without op or with
// an empty path fails before any request, and an empty patch sends nothing. Only
// 200 succeeds and a conflict is never retried.
func (a *API) PatchAllocation(ctx context.Context, id string, patch []PatchOp) (*PatchResult, error) {
	const op = "PatchAllocation"
	if id == "" {
		return nil, allocationInvalid(op, "allocation ID must not be empty")
	}
	ops, err := ironicwire.ConvertPatch(allocationNames(), patch)
	if err != nil {
		return nil, request.Wrap(op, "allocations", err)
	}
	if len(ops) == 0 {
		return &PatchResult{}, nil
	}
	return a.send(ctx, op, id, ops)
}

// UpdateAttributes derives the patch of the pinned update_allocation(id, **attrs):
// every attribute becomes an "add" operation sorted by path, because only the
// identity is known as the original body. Unknown names and the identity field
// fail before any request. An empty attribute map sends nothing.
func (a *API) UpdateAttributes(ctx context.Context, id string, attrs map[string]any) (*PatchResult, error) {
	const op = "UpdateAttributes"
	if id == "" {
		return nil, allocationInvalid(op, "allocation ID must not be empty")
	}
	resolved, err := ironicwire.ResolveAttributes(allocationNames(), attrs)
	if err != nil {
		return nil, request.Wrap(op, "allocations", err)
	}
	if len(resolved) == 0 {
		return &PatchResult{}, nil
	}
	return a.send(ctx, op, id, ironicwire.AttributePatch(resolved))
}

func (a *API) send(ctx context.Context, op, id string, ops []PatchOp) (*PatchResult, error) {
	var body struct {
		Allocation
	}
	if _, err := a.client.Patch(ctx, a.client.ServiceURL("allocations", url.PathEscape(id)), ops, &body.Allocation, &gophercloud.RequestOpts{OkCodes: []int{200}}); err != nil {
		return nil, request.Wrap(op, "allocations", err)
	}
	value := body.Allocation
	return &PatchResult{Allocation: &value, Sent: true}, nil
}

// ErrAllocationFailed is matched by errors.Is when an allocation reaches the
// error state; it also matches resource.ErrFailedState.
var ErrAllocationFailed = errors.New("allocation failed")

// AllocationFailedError carries the last_error of a failed allocation.
type AllocationFailedError struct {
	ID        string
	LastError string
}

func (e *AllocationFailedError) Error() string {
	return fmt.Sprintf("Allocation %s failed: %s", e.ID, e.LastError)
}
func (e *AllocationFailedError) Is(t error) bool {
	return t == ErrAllocationFailed || t == resource.ErrFailedState
}

// WaitOption configures WaitForActive.
type WaitOption func(*waitConfig)

type waitConfig struct {
	wait   []resource.WaitOption
	ignore bool
}

// allocationWaitDefaults are the WaitForActive defaults: no SDK timeout (the
// context is the deadline) and two seconds between reads. Failure states, a
// status attribute and progress are not honored.
var allocationWaitDefaults = resource.WaitDefaults{Unlimited: true, Strategy: resource.FixedWait(2 * time.Second)}

// WithAllocationWait sets the wait controls of WaitForActive:
// resource.WithPollInterval or resource.WithWaitStrategy (default two
// seconds) and resource.WithTimeout or resource.WithUnlimitedWait (default: no
// SDK timeout, the deadline is the context). Repeated calls append; the later
// option wins per control. Failure states, a status attribute, progress and
// invalid options fail with resource.ErrInvalidOption before the first read.
func WithAllocationWait(options ...resource.WaitOption) WaitOption {
	options = append([]resource.WaitOption(nil), options...)
	return func(c *waitConfig) { c.wait = append(c.wait, options...) }
}

// WithIgnoreError makes the error state a normal end of the wait.
func WithIgnoreError(ignore bool) WaitOption { return func(c *waitConfig) { c.ignore = ignore } }

// WaitForActive reads the allocation immediately and then once per interval
// until its state is not "allocating", like Allocation.wait on a bare
// reference. The state "error" returns *AllocationFailedError unless
// WithIgnoreError(true) was given; every other state (including a null or
// empty one) ends the wait and returns the allocation. The deadline is the
// context unless WithAllocationWait selects resource.WithTimeout.
func (a *API) WaitForActive(ctx context.Context, id string, options ...WaitOption) (*Allocation, error) {
	var cfg waitConfig
	for _, o := range options {
		if o != nil {
			o(&cfg)
		}
	}
	settings, err := resource.ResolveWait(allocationWaitDefaults, cfg.wait...)
	if err != nil {
		return nil, request.Wrap("WaitForActive", "allocations", err)
	}
	if !settings.Unlimited {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, settings.Timeout)
		defer cancel()
	}
	pacer := pollschedule.NewPacer(settings.Strategy.Delay)
	for {
		value, err := a.Get(ctx, id)
		if err != nil {
			return nil, request.Wrap("WaitForActive", "allocations", err)
		}
		if value.State == "error" && !cfg.ignore {
			return nil, request.Wrap("WaitForActive", "allocations", &AllocationFailedError{ID: value.UUID, LastError: value.LastError})
		}
		if value.State != "allocating" {
			return value, nil
		}
		timer := time.NewTimer(pacer.Next())
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, request.Wrap("WaitForActive", "allocations", context.Cause(ctx))
		case <-timer.C:
		}
	}
}

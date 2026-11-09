package compute

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/sourcewait"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type addAutoIPOptions struct {
	wait      bool
	waitOpts  []resource.WaitOption
	reuse     bool
	cleanup   bool
	timeout   time.Duration
	unlimited bool
	strategy  *resource.WaitStrategy
	available []AvailableFloatingIPOption
	create    []FloatingIPCreateOption
	attach    []ServerIPOption
	delete    []FloatingIPDeleteOption
	address   []ServerAddressOption
}

// AddAutoIPOption configures AddAutoIP. Defaults match the pinned helper:
// reuse available IPs, do not wait, and a 60-second budget for waits.
type AddAutoIPOption func(*addAutoIPOptions) error

// WithAddAutoIPWait waits until the server shows the floating address. Without
// it the helper returns after the attach request is accepted.
func WithAddAutoIPWait(enabled bool) AddAutoIPOption {
	return func(o *addAutoIPOptions) error { o.wait = enabled; return nil }
}

// WithAddAutoIPWaitOptions tunes the waits with the shared resource wait
// options; repeated calls append and the later option wins per control.
// resource.WithTimeout replaces the 60-second budget for waits. Like the pinned
// helper, the budget is spent by the create wait first and the rest bounds the
// server address wait; plain HTTP requests are bounded only by ctx. A zero or
// negative remaining budget makes a requested wait expire before its first
// poll. resource.WithUnlimitedWait removes the budget; ctx still applies.
// resource.WithPollInterval or resource.WithWaitStrategy (a Go-only extension)
// replaces the server address wait's interval, which otherwise is
// min(5 seconds, remaining budget) as in the pinned helper (5 seconds when
// unlimited). Failure states, a status attribute and progress are unsupported
// and fail with resource.ErrInvalidOption before HTTP.
func WithAddAutoIPWaitOptions(options ...resource.WaitOption) AddAutoIPOption {
	owned := slices.Clone(options)
	return func(o *addAutoIPOptions) error { o.waitOpts = append(o.waitOpts, owned...); return nil }
}

// WithAddAutoIPReuse selects between choosing an available floating IP (the
// default) and always allocating a new one.
func WithAddAutoIPReuse(reuse bool) AddAutoIPOption {
	return func(o *addAutoIPOptions) error { o.reuse = reuse; return nil }
}

// WithAddAutoIPCleanup controls deleting a floating IP that this call created
// on Neutron when the address wait times out. The default is true, matching
// the pinned helper. Cleanup is one DeleteFloatingIP call on a context that
// ignores the parent's cancellation; a failure is reported, never retried.
func WithAddAutoIPCleanup(enabled bool) AddAutoIPOption {
	return func(o *addAutoIPOptions) error { o.cleanup = enabled; return nil }
}

// WithAddAutoIPAvailableOptions passes options to the reuse lookup.
func WithAddAutoIPAvailableOptions(options ...AvailableFloatingIPOption) AddAutoIPOption {
	options = append([]AvailableFloatingIPOption(nil), options...)
	return func(o *addAutoIPOptions) error { o.available = append(o.available, options...); return nil }
}

// WithAddAutoIPCreateOptions passes options to the allocation used when reuse
// is disabled. The helper itself sets the server, wait and wait timeout first.
func WithAddAutoIPCreateOptions(options ...FloatingIPCreateOption) AddAutoIPOption {
	options = append([]FloatingIPCreateOption(nil), options...)
	return func(o *addAutoIPOptions) error { o.create = append(o.create, options...); return nil }
}

// WithAddAutoIPAttachOptions passes options to the explicit attach request.
func WithAddAutoIPAttachOptions(options ...ServerIPOption) AddAutoIPOption {
	options = append([]ServerIPOption(nil), options...)
	return func(o *addAutoIPOptions) error { o.attach = append(o.attach, options...); return nil }
}

// WithAddAutoIPDeleteOptions passes options to the timeout cleanup delete.
func WithAddAutoIPDeleteOptions(options ...FloatingIPDeleteOption) AddAutoIPOption {
	options = append([]FloatingIPDeleteOption(nil), options...)
	return func(o *addAutoIPOptions) error { o.delete = append(o.delete, options...); return nil }
}

// WithAddAutoIPAddressOptions passes options to the final interface IP view.
func WithAddAutoIPAddressOptions(options ...ServerAddressOption) AddAutoIPOption {
	options = append([]ServerAddressOption(nil), options...)
	return func(o *addAutoIPOptions) error { o.address = append(o.address, options...); return nil }
}

// AddAutoIPResult retains every accepted step even when a later step fails.
// InterfaceIP is the pinned helper's return value: the server's interface IP
// computed from Server, which is the refreshed server only when a refresh or
// wait happened and otherwise the caller's snapshot. Empty is the helper's None.
//
// NeutronFloating is Python _use_neutron_floating for this call: the
// allocation or reuse step used Neutron, or fell back to Nova after a Neutron
// 404 (Backend is then FloatingIPNova). It selects skip-attach and timeout
// cleanup exactly as the pinned helper does.
//
// NeutronAttach is the Neutron port update sent for a reused row that came from
// that Nova fallback, and NeutronNotFound the NotFound it produced, which the
// helper ignores like Python.
//
// TimeoutRemaining is the wait budget left for the server address wait; it is
// zero when resource.WithUnlimitedWait removed the budget.
type AddAutoIPResult struct {
	Backend          FloatingIPSource
	NeutronFloating  bool
	FloatingIPID     string
	FloatingIPAddr   string
	Reused           bool
	Created          bool
	AttachSkipped    bool
	AlreadyAttached  bool
	Observed         bool
	InterfaceIP      string
	Server           *Server
	Available        *AvailableFloatingIPResult
	Creation         *CreateFloatingIPResult
	Attach           *AutomaticServerIPResult
	NeutronAttach    *network.FloatingIPPortUpdate
	NeutronNotFound  error
	Cleanup          *DeleteFloatingIPResult
	CleanupError     error
	TimeoutRemaining time.Duration
}

// AddAutoIPTimeoutError reports that the floating address did not appear on the
// server within the budget. It matches context.DeadlineExceeded.
type AddAutoIPTimeoutError struct {
	Address string
	Timeout time.Duration
}

func (e *AddAutoIPTimeoutError) Error() string {
	return fmt.Sprintf("floating IP %s was not attached to the server within %s", e.Address, e.Timeout)
}
func (e *AddAutoIPTimeoutError) Unwrap() error { return context.DeadlineExceeded }

// AddAutoIP implements NetworkCommonCloudMixin.add_auto_ip for basic use.
//
// With reuse (default) it calls AvailableFloatingIP for the server; otherwise
// it allocates with CreateFloatingIP, passing the server, wait and the whole
// timeout. Unless the server snapshot already shows that floating address, it
// then attaches by exact address through AddIPList, except that a freshly
// allocated Neutron IP skips this step as the allocation already targeted the
// server. With the Neutron source this also holds for an allocation that fell
// back to Nova after a Neutron 404: like Python, no attach is sent for it. A
// reused row from that Nova fallback is attached as Python does: by one Neutron
// PUT floatingips/{id} on the server's NAT destination port, where a NotFound
// from the port list or the PUT is ignored (recorded in NeutronNotFound) and no
// Nova action follows. A Nova-source allocation is always attached. With wait, the server is polled
// until the address appears or the remaining budget ends with
// *AddAutoIPTimeoutError; on that timeout an IP created by this call under the
// Neutron source (including the Nova fallback) is deleted unless
// WithAddAutoIPCleanup(false). Mutations are never replayed.
func (s *Service) AddAutoIP(ctx context.Context, server *Server, options ...AddAutoIPOption) (*AddAutoIPResult, error) {
	if ctx == nil {
		return nil, invalid("context is required")
	}
	if err := errors.Join(ctx.Err(), context.Cause(ctx)); err != nil {
		return nil, err
	}
	if s == nil || s.API == nil || s.Servers == nil || server == nil {
		return nil, invalid("compute service and server are required")
	}
	snapshot := *server
	if err := resource.ID(snapshot.ID).Validate(); err != nil {
		return nil, err
	}
	o := addAutoIPOptions{reuse: true, cleanup: true}
	for _, apply := range options {
		if apply == nil {
			return nil, invalid("nil add auto IP option")
		}
		if err := apply(&o); err != nil {
			return nil, err
		}
	}
	if err := o.resolveWait(); err != nil {
		return nil, err
	}
	result := &AddAutoIPResult{Server: &snapshot}
	start := time.Now()
	remaining := o.timeout
	var port string
	if o.reuse {
		available, err := s.AvailableFloatingIP(ctx, network.AvailableFloatingIPRequest{Server: resource.ID(snapshot.ID)}, o.available...)
		result.Available = available
		if err != nil {
			return result, err
		}
		result.Backend, result.FloatingIPID, result.FloatingIPAddr, result.Reused = available.Backend, available.ID, available.Address, available.Reused
		result.NeutronFloating = available.Backend == FloatingIPNeutron || available.FallbackError != nil
		port, _ = floatingIPRowString(available.FloatingIP, "port_id")
	} else {
		create := append([]FloatingIPCreateOption{func(c *FloatingIPCreateOpts) error { c.Server = resource.ID(snapshot.ID); return nil }, WithFloatingIPCreateWait(o.wait), o.createWaitBudget()}, o.create...)
		created, err := s.CreateFloatingIP(ctx, CreateFloatingIPRequest{}, create...)
		result.Creation = created
		if err != nil {
			return result, err
		}
		result.Backend, result.Created = created.Backend, true
		result.NeutronFloating = created.Backend == FloatingIPNeutron || created.FallbackError != nil
		result.FloatingIPID, _ = floatingIPRowString(created.FloatingIP, "id")
		result.FloatingIPAddr, _ = floatingIPRowString(created.FloatingIP, "floating_ip_address")
		port, _ = floatingIPRowString(created.FloatingIP, "port_id")
		remaining = o.remainingAfter(start)
	}
	result.TimeoutRemaining = remaining
	if result.FloatingIPAddr == "" {
		return result, invalid("floating IP %q has no floating_ip_address", result.FloatingIPID)
	}
	// Python sets skip_attach for every allocation and honours it only in its
	// Neutron-source branch, including an allocation that fell back to Nova.
	skipAttach := result.Created && result.NeutronFloating
	attached, err := serverShowsFloatingAddress(&snapshot, result.FloatingIPAddr)
	if err != nil {
		return result, err
	}
	if !attached && port != "" {
		refreshed, err := s.refreshAutoIPServer(ctx, &snapshot, result.FloatingIPAddr, remaining, o)
		if err != nil {
			return result, err
		}
		result.Server = refreshed
		attached, err = serverShowsFloatingAddress(refreshed, result.FloatingIPAddr)
		if err != nil {
			return result, err
		}
	}
	if attached {
		result.AlreadyAttached = true
		return s.finishAutoIP(ctx, result, o)
	}
	if skipAttach {
		result.AttachSkipped = true
	} else if result.Reused && result.Available != nil && result.Available.FallbackError != nil {
		if err := s.attachAutoIPFallbackRow(ctx, result); err != nil {
			return result, err
		}
	} else {
		attach, err := s.AddIPList(ctx, result.Server, []string{result.FloatingIPAddr}, o.attach...)
		result.Attach = attach
		if attach != nil && attach.Server != nil {
			result.Server = attach.Server
		}
		if err != nil {
			return result, err
		}
	}
	if o.wait {
		if err := s.waitAutoIPAddress(ctx, result, remaining, o); err != nil {
			return result, err
		}
	}
	return s.finishAutoIP(ctx, result, o)
}

func (s *Service) autoIPObserver(ctx context.Context, server *Server, address string, remaining time.Duration, o addAutoIPOptions) (*automaticIPState, context.Context, context.CancelFunc, error) {
	budget := resource.WithUnlimitedWait()
	interval := 5 * time.Second
	if !o.unlimited {
		interval = min(interval, remaining)
		budget = resource.WithTimeout(max(remaining, time.Nanosecond))
	}
	if interval <= 0 {
		interval = time.Nanosecond
	}
	pacing := resource.WithPollInterval(interval)
	if o.strategy != nil {
		pacing = resource.WithWaitStrategy(*o.strategy)
	}
	return s.prepareServerIPWorkflow(ctx, AutomaticFloatingIPRequest{Server: server}, []AutomaticFloatingIPOption{WithAutomaticIPWait(budget, pacing)}, serverIPReadiness{observe: true, explicitList: true, addresses: []string{address}})
}

func (s *Service) refreshAutoIPServer(ctx context.Context, server *Server, address string, remaining time.Duration, o addAutoIPOptions) (*Server, error) {
	// A used-up create budget must not make this plain refresh fail.
	state, ctx, cancel, err := s.autoIPObserver(ctx, server, address, max(remaining, time.Minute), o)
	if err != nil {
		return nil, err
	}
	defer cancel()
	if _, err := state.rawServer(ctx); err != nil {
		return nil, err
	}
	return state.last, nil
}

func (s *Service) waitAutoIPAddress(ctx context.Context, result *AddAutoIPResult, remaining time.Duration, o addAutoIPOptions) error {
	expired := &AddAutoIPTimeoutError{Address: result.FloatingIPAddr, Timeout: o.timeout}
	var err error
	if !o.unlimited && remaining <= 0 {
		err = expired
	} else {
		state, waitCtx, cancel, prepareErr := s.autoIPObserver(ctx, result.Server, result.FloatingIPAddr, remaining, o)
		if prepareErr != nil {
			return prepareErr
		}
		err = state.observe(waitCtx, result.FloatingIPAddr)
		cancel()
		if state.last != nil {
			result.Server = state.last
		}
		if err == nil {
			result.Observed = true
			return nil
		}
		if ctx.Err() != nil || !errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		err = expired
	}
	if result.Created && result.NeutronFloating && o.cleanup {
		cleanup := context.WithoutCancel(ctx)
		deleted, deleteErr := s.DeleteFloatingIP(cleanup, DeleteFloatingIPRequest{ID: result.FloatingIPID}, o.delete...)
		result.Cleanup, result.CleanupError = deleted, deleteErr
		err = errors.Join(err, deleteErr)
	}
	return err
}

func (s *Service) finishAutoIP(ctx context.Context, result *AddAutoIPResult, o addAutoIPOptions) (*AddAutoIPResult, error) {
	view, err := s.ExpandServerInterfaces(ctx, result.Server, o.address...)
	if view != nil {
		result.InterfaceIP = view.InterfaceIP
	}
	return result, err
}

// serverShowsFloatingAddress reports whether the supplied server model has the
// exact floating IPv4, ignoring any other address type.
func serverShowsFloatingAddress(server *Server, address string) (bool, error) {
	view, err := parseServerAddresses(server, nil)
	if err != nil {
		return false, err
	}
	for _, rows := range view.Addresses {
		for _, row := range rows {
			if row.Version == 4 && row.Type == "floating" && row.Address == address {
				return true, nil
			}
		}
	}
	return false, nil
}

// resolveWait resolves the wait options over the 60-second budget. The address
// wait's pacing stays nil unless the options choose it, keeping the
// min(5 seconds, remaining budget) default.
func (o *addAutoIPOptions) resolveWait() error {
	settings, err := sourcewait.Resolve(0, o.waitOpts)
	if err != nil {
		return err
	}
	o.timeout, o.unlimited, o.strategy = 60*time.Second, false, nil
	if settings.TimeoutSet {
		o.timeout, o.unlimited = settings.Timeout, settings.Unlimited
	}
	if settings.StrategySet {
		strategy := settings.Strategy
		o.strategy = &strategy
	}
	return nil
}

// createWaitBudget passes the whole budget to the allocation's wait.
func (o addAutoIPOptions) createWaitBudget() FloatingIPCreateOption {
	if o.unlimited {
		return WithFloatingIPCreateWaitOptions(resource.WithUnlimitedWait())
	}
	return WithFloatingIPCreateWaitTimeout(o.timeout)
}

// remainingAfter is the budget left after a step that started at start; it is
// zero (unused) when the budget is unlimited.
func (o addAutoIPOptions) remainingAfter(start time.Time) time.Duration {
	if o.unlimited {
		return 0
	}
	return o.timeout - time.Since(start)
}

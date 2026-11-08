package resource

import (
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Ref explicitly distinguishes IDs from names, including UUID-shaped names.
// The zero value is invalid; there is no heuristic or fallback between kinds.
type Ref struct {
	value  string
	byName bool
}

func ID(value string) Ref    { return Ref{value: value} }
func Name(value string) Ref  { return Ref{value: value, byName: true} }
func (r Ref) String() string { return r.value }

// IsName reports whether this is an explicit name reference.
func (r Ref) IsName() bool { return r.byName }
func (r Ref) Validate() error {
	if strings.TrimSpace(r.value) == "" {
		return invalid("reference must not be empty")
	}
	if !r.byName && (r.value == "." || r.value == ".." || strings.ContainsAny(r.value, "/\\?#% \t\r\n")) {
		return invalid("ID must be a single unescaped URL path segment")
	}
	return nil
}

type listOptions struct {
	name        *string
	query       url.Values
	status      bool
	control     ListControl
	bodyFilters []bodyFilterUpdate
	filters     []filterUpdate
}

// ListOption configures a typed resource iterator. Later options win.
type ListOption func(*listOptions) error

// WithMaxItems caps rows before local name/status/Body filtering. Zero is unlimited.
// It is separate from WithPageSize, which controls the server's requested page.
func WithMaxItems(maximum int) ListOption {
	return func(o *listOptions) error {
		o.control.MaxItems = maximum
		return nil
	}
}

// WithPaginated(false) reads only the first page. Bindings with opaque custom
// iterators must explicitly support page control; native pagers already do.
func WithPaginated(paginated bool) ListOption {
	return func(o *listOptions) error {
		o.control.SinglePage = !paginated
		return nil
	}
}

// WithName uses literal, exact name matching, including for Nova regex filters.
func WithName(name string) ListOption {
	return func(o *listOptions) error { o.name = &name; return nil }
}

func WithStatus(status string) ListOption {
	return func(o *listOptions) error {
		if status == "" {
			return invalid("status must not be empty")
		}
		o.query.Set("status", status)
		o.status = true
		return nil
	}
}

// WithPageSize controls the server-side page size, not a total result limit.
func WithPageSize(size int) ListOption {
	return func(o *listOptions) error {
		if size < 1 {
			return invalid("page size must be positive")
		}
		o.query.Set("limit", strconv.Itoa(size))
		return nil
	}
}

// WithQuery passes a service-specific query field without a custom builder.
// The server validates its semantics. WithName remains an exact local filter.
// Local controls max_items and paginated require their dedicated options.
func WithQuery(key, value string) ListOption {
	return func(o *listOptions) error {
		if strings.TrimSpace(key) == "" {
			return invalid("query key must not be empty")
		}
		if key == "max_items" || key == "paginated" {
			return invalid("local list controls require WithMaxItems or WithPaginated")
		}
		o.query.Set(key, value)
		return nil
	}
}

type lookupOptions struct{ ignoreMissing bool }
type LookupOption func(*lookupOptions) error

// WithIgnoreMissing makes Find return nil, nil and Delete succeed on absence.
func WithIgnoreMissing() LookupOption {
	return func(o *lookupOptions) error { o.ignoreMissing = true; return nil }
}

// WithMissingError requires existence. Find defaults to this behavior;
// Delete defaults to ignoring missing resources for idempotence.
func WithMissingError() LookupOption {
	return func(o *lookupOptions) error { o.ignoreMissing = false; return nil }
}

type waitOptions struct {
	timeout, interval time.Duration
	strategy          WaitStrategy
	failureStates     []string
	failureStatesSet  bool
	statusAttribute   string
	progressCallback  func(int)
	progress          chan<- int
}
type WaitOption func(*waitOptions) error

func WithTimeout(timeout time.Duration) WaitOption {
	return func(o *waitOptions) error {
		if timeout <= 0 {
			return invalid("timeout must be positive")
		}
		o.timeout = timeout
		return nil
	}
}

// WithUnlimitedWait removes the SDK timeout. The parent context still controls
// cancellation and deadlines. A later WithTimeout reinstates a bounded wait.
func WithUnlimitedWait() WaitOption {
	return func(o *waitOptions) error { o.timeout = 0; return nil }
}

// WithFailureStates replaces the service's failure predicate with exact,
// case-insensitive states. With no states, failure-state detection is disabled.
// It applies to Wait; WaitDeleted preserves service-specific deletion errors.
func WithFailureStates(states ...string) WaitOption {
	states = append([]string(nil), states...)
	return func(o *waitOptions) error {
		for _, state := range states {
			if strings.TrimSpace(state) == "" {
				return invalid("failure state must not be empty")
			}
		}
		o.failureStates = states
		o.failureStatesSet = true
		return nil
	}
}

// WithStatusAttribute selects an exported string field by its JSON tag or Go
// name. The SDK owns field access; callers do not implement a status adapter.
func WithStatusAttribute(attribute string) WaitOption {
	return func(o *waitOptions) error {
		if strings.TrimSpace(attribute) == "" || strings.ContainsAny(attribute, ". /\\\t\r\n") {
			return invalid("status attribute must be a single non-empty field name")
		}
		o.statusAttribute = attribute
		return nil
	}
}

// WithProgressCallback reports model progress after each nonterminal response,
// including the initial lookup. Missing or nil progress is reported as zero.
// The callback runs synchronously and is not called for success or failure.
// With WithProgress also set, the callback runs first and the value is then
// sent on the channel.
func WithProgressCallback(callback func(int)) WaitOption {
	return func(o *waitOptions) error {
		if callback == nil {
			return invalid("progress callback must not be nil")
		}
		o.progressCallback = callback
		return nil
	}
}

// WithProgress sends model progress on progress after each nonterminal
// response, including the initial lookup. Missing or nil progress is sent as
// zero; nothing is sent for success or failure. The wait sends from its own
// goroutine and blocks until the value is received or its context ends, so the
// caller must drain progress; a context that ends during a send stops the wait
// with its cancellation error. The SDK never closes progress and sends nothing
// after the wait returns. With WithProgressCallback also set, the callback runs
// first and the value is then sent.
func WithProgress(progress chan<- int) WaitOption {
	return func(o *waitOptions) error {
		if progress == nil {
			return invalid("progress channel must not be nil")
		}
		o.progress = progress
		return nil
	}
}

// WithPollInterval pauses for interval before every poll. It is equivalent
// to WithWaitStrategy(FixedWait(interval)); the later of the two wins.
func WithPollInterval(interval time.Duration) WaitOption {
	return func(o *waitOptions) error {
		if interval <= 0 {
			return invalid("poll interval must be positive")
		}
		o.interval = interval
		o.strategy = WaitStrategy{}
		return nil
	}
}

// WithWaitStrategy chooses the pause before each poll for this call. Without
// it the wait keeps its fixed interval. An invalid strategy fails with
// ErrInvalidOption before any request. WithWaitStrategy and WithPollInterval
// replace each other; the later one wins.
func WithWaitStrategy(strategy WaitStrategy) WaitOption {
	return func(o *waitOptions) error {
		if err := strategy.Validate(); err != nil {
			return err
		}
		o.strategy = strategy
		return nil
	}
}

func parseWait(opts []WaitOption) (waitOptions, error) {
	o := waitOptions{timeout: 5 * time.Minute, interval: 2 * time.Second}
	for _, apply := range opts {
		if apply == nil {
			return o, invalid("nil wait option")
		}
		if err := apply(&o); err != nil {
			return o, err
		}
	}
	return o, nil
}

func parseLookup(ignoreMissing bool, opts []LookupOption) (lookupOptions, error) {
	o := lookupOptions{ignoreMissing: ignoreMissing}
	for _, apply := range opts {
		if apply == nil {
			return o, invalid("nil lookup option")
		}
		if err := apply(&o); err != nil {
			return o, err
		}
	}
	return o, nil
}

// ValidateWaitOptions validates a reusable waiter policy before a mutating request.
func ValidateWaitOptions(opts ...WaitOption) error {
	_, err := parseWait(opts)
	return err
}

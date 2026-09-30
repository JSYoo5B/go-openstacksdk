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
	if !r.byName && strings.ContainsAny(r.value, "/\\?#% \t\r\n") {
		return invalid("ID must be a single unescaped URL path segment")
	}
	return nil
}

type listOptions struct {
	name   *string
	query  url.Values
	status bool
}

// ListOption configures a typed resource iterator. Later options win.
type ListOption func(*listOptions) error

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
func WithQuery(key, value string) ListOption {
	return func(o *listOptions) error {
		if strings.TrimSpace(key) == "" {
			return invalid("query key must not be empty")
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

type waitOptions struct{ timeout, interval time.Duration }
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

func WithPollInterval(interval time.Duration) WaitOption {
	return func(o *waitOptions) error {
		if interval <= 0 {
			return invalid("poll interval must be positive")
		}
		o.interval = interval
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

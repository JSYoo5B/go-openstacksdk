package resource

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"maps"
	"net/http"
	"net/url"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

// Adapter supplies service operations to the shared collection implementation.
// SDK service packages construct it; normal SDK callers do not implement adapters.
type Adapter[T any] struct {
	Kind string
	// IdentityFind opts an audited SDK binding into GET-first identity lookup.
	// Numeric IDs, opaque keys and hierarchical controllers require a separate
	// binding audit; Get/List availability alone does not enable this capability.
	IdentityFind bool
	// ValidateID is supplied by SDK bindings whose identifiers are not ordinary
	// URL segments, such as Swift object keys. Such bindings also own URL escaping.
	ValidateID func(string) error
	Get        func(context.Context, string) (*T, error)
	// GetIdentityQuery applies a frozen service query to the direct GET phase of
	// FindIdentity. An audited binding must supply it when that phase has query
	// fields; ordinary Get and query-free FindIdentity keep using Get.
	GetIdentityQuery func(context.Context, string, url.Values) (*T, error)
	List             func(url.Values) pagination.Pager
	Iterate          func(context.Context, url.Values) iter.Seq2[*T, error]
	// IterateIdentity selects a detailed or summary fallback for FindIdentity.
	// It is used only by audited bindings; ordinary List keeps its own iterator.
	IterateIdentity func(context.Context, url.Values, bool) iter.Seq2[*T, error]
	// IdentityAllProjectsQuery opts a binding into the typed, list-only
	// AllProjects policy and identifies its wire query key, such as all_tenants.
	IdentityAllProjectsQuery string
	// IdentityMissingListQuery requests one additional list after a complete,
	// successful empty identity search. SDK bindings own the fixed query overlay;
	// it never retries HTTP errors or changes ordinary Get/List/Ref lookup.
	IdentityMissingListQuery url.Values
	// IdentityListQueryDefaults supplies audited list-only defaults for identity
	// searches. Explicit caller keys, including empty values, take precedence.
	IdentityListQueryDefaults url.Values
	// IdentityExtraSpecs enriches a validated, uniquely resolved resource when
	// requested. Audited bindings own the native fetch and model snapshot.
	IdentityExtraSpecs func(context.Context, *T) (*T, error)
	// IterateControlled lets SDK bindings apply row/page controls inside their
	// transport iterator, before filtering and continuation processing.
	IterateControlled func(context.Context, url.Values, ListControl) iter.Seq2[*T, error]
	Extract           func(pagination.Page) ([]T, error)
	Delete            func(context.Context, string) error
	ID                func(*T) string
	Name              func(*T) string
	NameQuery         func(string) string
	NameQueryKey      string
	Status            func(*T) string
	// BodyFilterFields maps SDK-owned aliases to canonical response fields.
	// The canonical field must also map to itself. Only audited bindings opt in.
	BodyFilterFields map[string]string
	// BodyFilterValue projects one canonical field from the native typed model.
	// This does not restore original wire presence or numeric precision.
	BodyFilterValue func(*T, string) (json.RawMessage, error)
	// IterateBodyControlled pairs native models with original row fields when
	// an audited Body filter needs wire presence or values lost by that model.
	// It is selected only for a nonempty prepared Body filter set.
	IterateBodyControlled func(context.Context, url.Values, ListControl) iter.Seq2[*BodyRecord[T], error]
	BodyFilterRecordValue func(*BodyRecord[T], string) (json.RawMessage, error)
	// FixedWaitStatus prevents replacing a specialized waiter's completion
	// condition, such as Inspector's Finished boolean, with another attribute.
	FixedWaitStatus bool
	// LocalStatus keeps WithStatus out of the wire query for bindings whose
	// controllers expose a state field but no generic status query parameter.
	LocalStatus bool
	Failed      func(string) bool
}

// Collection shares lookup, streaming, deletion and wait policies across services.
// Instances are obtained through a Connection; their zero value is not usable.
type Collection[T any] struct{ binding Adapter[T] }

func NewCollection[T any](adapter Adapter[T]) *Collection[T] {
	adapter.BodyFilterFields = maps.Clone(adapter.BodyFilterFields)
	if adapter.IdentityMissingListQuery != nil {
		adapter.IdentityMissingListQuery = cloneIdentityFindOptions(IdentityFindOpts{Query: adapter.IdentityMissingListQuery}).Query
	}
	if adapter.IdentityListQueryDefaults != nil {
		adapter.IdentityListQueryDefaults = cloneIdentityFindOptions(IdentityFindOpts{Query: adapter.IdentityListQueryDefaults}).Query
	}
	return &Collection[T]{binding: adapter}
}

func (c *Collection[T]) validateID(id string) error {
	if c.binding.ValidateID != nil {
		return c.binding.ValidateID(id)
	}
	return ID(id).Validate()
}

func (c *Collection[T]) validateRef(ref Ref) error {
	if ref.IsName() {
		return ref.Validate()
	}
	return c.validateID(ref.String())
}

func (c *Collection[T]) wrap(op string, err error) error {
	return &OperationError{Operation: op, Resource: c.binding.Kind, Cause: err}
}

// Get fetches by ID. Use Find with Name for an exact name lookup.
func (c *Collection[T]) Get(ctx context.Context, id string) (*T, error) {
	if err := c.validateID(id); err != nil {
		return nil, err
	}
	if c.binding.Get == nil {
		return nil, c.wrap("get", ErrUnsupported)
	}
	v, err := c.binding.Get(ctx, id)
	if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		err = &NotFoundError{Resource: c.binding.Kind, Reference: id, Cause: err}
	}
	if err != nil {
		return nil, c.wrap("get", err)
	}
	return v, nil
}

// List lazily streams resources across all pages. Break stops further fetches.
// On failure it yields nil, error once, then stops. Every iteration is a fresh request.
// The option slice is retained at construction; caller slice replacement cannot
// change an existing iterator. Options are still applied lazily on each iteration.
func (c *Collection[T]) List(ctx context.Context, opts ...ListOption) iter.Seq2[*T, error] {
	opts = append([]ListOption(nil), opts...)
	return func(yield func(*T, error) bool) {
		o := listOptions{query: make(url.Values)}
		for _, apply := range opts {
			if apply == nil {
				yield(nil, invalid("nil list option"))
				return
			}
			if err := apply(&o); err != nil {
				yield(nil, err)
				return
			}
		}
		if o.control.MaxItems < 0 {
			yield(nil, invalid("maximum items must be non-negative"))
			return
		}
		bodyFilters, err := c.prepareBodyFilters(o.bodyFilters)
		if err != nil {
			yield(nil, err)
			return
		}
		if o.status && c.binding.Status == nil {
			yield(nil, c.wrap("list", ErrUnsupported))
			return
		}
		if o.name != nil && c.binding.Name == nil {
			yield(nil, c.wrap("list", ErrUnsupported))
			return
		}
		if o.name != nil && c.binding.NameQuery != nil {
			key := c.binding.NameQueryKey
			if key == "" {
				key = "name"
			}
			o.query.Set(key, c.binding.NameQuery(*o.name))
		}
		wireQuery := o.query
		if o.status && c.binding.LocalStatus {
			wireQuery = maps.Clone(o.query)
			delete(wireQuery, "status")
		}
		if len(bodyFilters) > 0 && c.binding.BodyFilterRecordValue != nil {
			for record, err := range c.binding.IterateBodyControlled(ctx, wireQuery, o.control) {
				if err != nil {
					yield(nil, c.wrap("list", err))
					return
				}
				if record == nil {
					yield(nil, c.wrap("list", invalid("nil Body filter record")))
					return
				}
				matched, err := c.matchBodyRecordFilters(record, bodyFilters)
				if err != nil {
					yield(nil, c.wrap("list", err))
					return
				}
				if !matched {
					continue
				}
				value := &record.Value
				if o.name != nil && c.binding.Name(value) != *o.name {
					continue
				}
				if o.status && !strings.EqualFold(c.binding.Status(value), o.query.Get("status")) {
					continue
				}
				if !yield(value, nil) {
					return
				}
			}
			return
		}
		var iterator iter.Seq2[*T, error]
		if c.binding.IterateControlled != nil {
			iterator = c.binding.IterateControlled(ctx, wireQuery, o.control)
		} else if c.binding.Iterate != nil {
			if o.control.SinglePage {
				yield(nil, c.wrap("list", ErrUnsupported))
				return
			}
			iterator = capIterator(ctx, c.binding.Iterate(ctx, wireQuery), o.control.MaxItems)
		}
		if iterator != nil {
			for value, err := range iterator {
				if err != nil {
					yield(nil, c.wrap("list", err))
					return
				}
				matched, err := c.matchBodyFilters(value, bodyFilters)
				if err != nil {
					yield(nil, c.wrap("list", err))
					return
				}
				if !matched {
					continue
				}
				if o.name != nil && c.binding.Name(value) != *o.name {
					continue
				}
				if o.status && !strings.EqualFold(c.binding.Status(value), o.query.Get("status")) {
					continue
				}
				if !yield(value, nil) {
					return
				}
			}
			return
		}
		if c.binding.List == nil || c.binding.Extract == nil {
			yield(nil, c.wrap("list", ErrUnsupported))
			return
		}
		stopped := false
		count := 0
		err = eachPage(ctx, c.binding.List(wireQuery), func(_ context.Context, page pagination.Page) (bool, error) {
			items, err := c.binding.Extract(page)
			if err != nil {
				return false, err
			}
			for i := range items {
				if err := ctx.Err(); err != nil {
					return false, err
				}
				count++
				matched, err := c.matchBodyFilters(&items[i], bodyFilters)
				if err != nil {
					return false, err
				}
				matched = matched && (o.name == nil || c.binding.Name(&items[i]) == *o.name)
				if o.status && !strings.EqualFold(c.binding.Status(&items[i]), o.query.Get("status")) {
					matched = false
				}
				if matched && !yield(&items[i], nil) {
					stopped = true
					return false, nil
				}
				if err := ctx.Err(); err != nil {
					return false, err
				}
				if o.control.MaxItems > 0 && count >= o.control.MaxItems {
					return false, nil
				}
			}
			return !o.control.SinglePage, nil
		})
		if err != nil && !stopped {
			yield(nil, c.wrap("list", err))
		}
	}
}

// All collects List into memory. The empty result is a non-nil slice.
func (c *Collection[T]) All(ctx context.Context, opts ...ListOption) ([]*T, error) {
	all := make([]*T, 0)
	for v, err := range c.List(ctx, opts...) {
		if err != nil {
			return nil, err
		}
		all = append(all, v)
	}
	return all, nil
}

// Find resolves an explicit ID or exact name. Duplicate names always fail.
func (c *Collection[T]) Find(ctx context.Context, ref Ref, opts ...LookupOption) (*T, error) {
	if err := c.validateRef(ref); err != nil {
		return nil, err
	}
	o, err := parseLookup(false, opts)
	if err != nil {
		return nil, err
	}
	if !ref.byName {
		v, err := c.Get(ctx, ref.value)
		if o.ignoreMissing && errors.Is(err, ErrNotFound) {
			return nil, nil
		}
		return v, err
	}
	var found *T
	for v, err := range c.List(ctx, WithName(ref.value)) {
		if err != nil {
			return nil, err
		}
		if found != nil {
			return nil, &AmbiguousError{Resource: c.binding.Kind, Name: ref.value, IDs: []string{c.binding.ID(found), c.binding.ID(v)}}
		}
		found = v
	}
	if found == nil && !o.ignoreMissing {
		return nil, &NotFoundError{Resource: c.binding.Kind, Reference: ref.value}
	}
	return found, nil
}

// Delete ignores missing resources by default, but never hides other errors.
// ID references are deleted directly; names are resolved first.
func (c *Collection[T]) Delete(ctx context.Context, ref Ref, opts ...LookupOption) error {
	if err := c.validateRef(ref); err != nil {
		return err
	}
	o, err := parseLookup(true, opts)
	if err != nil {
		return err
	}
	if c.binding.Delete == nil {
		return c.wrap("delete", ErrUnsupported)
	}
	id := ref.value
	if ref.byName {
		v, err := c.Find(ctx, ref, opts...)
		if errors.Is(err, ErrNotFound) && o.ignoreMissing {
			return nil
		}
		if err != nil {
			return err
		}
		if v == nil {
			return nil
		}
		id = c.binding.ID(v)
		if err := c.validateID(id); err != nil {
			return c.wrap("delete", err)
		}
	}
	err = c.binding.Delete(ctx, id)
	if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		if o.ignoreMissing {
			return nil
		}
		err = &NotFoundError{Resource: c.binding.Kind, Reference: id, Cause: err}
	}
	if err != nil {
		return c.wrap("delete", err)
	}
	return nil
}

// Wait resolves the reference once, then polls the same ID. It defaults to a
// five-minute timeout and two-second interval; parent cancellation takes priority.
func (c *Collection[T]) Wait(ctx context.Context, ref Ref, status string, opts ...WaitOption) (*T, error) {
	if err := c.validateRef(ref); err != nil {
		return nil, err
	}
	if strings.TrimSpace(status) == "" {
		return nil, invalid("target status must not be empty")
	}
	o, err := parseWait(opts)
	if err != nil {
		return nil, err
	}
	statusField, progressField, err := waitFields[T](o)
	if err != nil {
		return nil, c.wrap("wait", err)
	}
	if c.binding.FixedWaitStatus && o.statusAttribute != "" {
		return nil, c.wrap("wait", fmt.Errorf("%w: this waiter has a fixed completion condition", ErrUnsupported))
	}
	if c.binding.Status == nil && statusField == nil {
		return nil, c.wrap("wait", ErrUnsupported)
	}
	ctx, cancel := o.context(ctx)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, c.wrap("wait", err)
	}
	v, err := c.Find(ctx, ref)
	if err != nil {
		return nil, c.wrap("wait", err)
	}
	if v == nil {
		return nil, &FailedStateError{Resource: c.binding.Kind, ID: ref.String(), Status: "missing"}
	}
	id := ref.String()
	if ref.IsName() {
		id = c.binding.ID(v)
	}
	if err := c.validateID(id); err != nil {
		return nil, c.wrap("wait", err)
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil, c.wrap("wait", err)
		}
		if v == nil {
			return nil, &FailedStateError{Resource: c.binding.Kind, ID: id, Status: "missing"}
		}
		current, err := waitStatus(v, statusField, c.binding.Status)
		if err != nil {
			return nil, c.wrap("wait", err)
		}
		if strings.EqualFold(current, status) {
			return v, nil
		}
		if o.failed(current, c.binding.Failed) {
			return nil, &FailedStateError{Resource: c.binding.Kind, ID: id, Status: current}
		}
		if err := reportWaitProgress(o, v, progressField); err != nil {
			return nil, c.wrap("wait", err)
		}
		if err := o.pause(ctx); err != nil {
			return nil, c.wrap("wait", err)
		}
		v, err = c.Get(ctx, id)
		if err != nil {
			return nil, c.wrap("wait", err)
		}
	}
}

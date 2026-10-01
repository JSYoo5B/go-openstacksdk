package resource

import (
	"context"
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
	// ValidateID is supplied by SDK bindings whose identifiers are not ordinary
	// URL segments, such as Swift object keys. Such bindings also own URL escaping.
	ValidateID   func(string) error
	Get          func(context.Context, string) (*T, error)
	List         func(url.Values) pagination.Pager
	Iterate      func(context.Context, url.Values) iter.Seq2[*T, error]
	Extract      func(pagination.Page) ([]T, error)
	Delete       func(context.Context, string) error
	ID           func(*T) string
	Name         func(*T) string
	NameQuery    func(string) string
	NameQueryKey string
	Status       func(*T) string
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
func (c *Collection[T]) List(ctx context.Context, opts ...ListOption) iter.Seq2[*T, error] {
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
		if c.binding.Iterate != nil {
			for value, err := range c.binding.Iterate(ctx, wireQuery) {
				if err != nil {
					yield(nil, c.wrap("list", err))
					return
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
		err := eachPage(ctx, c.binding.List(wireQuery), func(_ context.Context, page pagination.Page) (bool, error) {
			items, err := c.binding.Extract(page)
			if err != nil {
				return false, err
			}
			for i := range items {
				if o.name != nil && c.binding.Name(&items[i]) != *o.name {
					continue
				}
				if o.status && !strings.EqualFold(c.binding.Status(&items[i]), o.query.Get("status")) {
					continue
				}
				if !yield(&items[i], nil) {
					stopped = true
					return false, nil
				}
			}
			return true, nil
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

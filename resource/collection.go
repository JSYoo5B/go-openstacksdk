package resource

import (
	"context"
	"errors"
	"iter"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

// Adapter supplies service operations to the shared collection implementation.
// SDK service packages construct it; normal SDK callers do not implement adapters.
type Adapter[T any] struct {
	Kind      string
	Get       func(context.Context, string) (*T, error)
	List      func(url.Values) pagination.Pager
	Iterate   func(context.Context, url.Values) iter.Seq2[*T, error]
	Extract   func(pagination.Page) ([]T, error)
	Delete    func(context.Context, string) error
	ID        func(*T) string
	Name      func(*T) string
	NameQuery func(string) string
	Status    func(*T) string
	Failed    func(string) bool
}

// Collection shares lookup, streaming, deletion and wait policies across services.
// Instances are obtained through a Connection; their zero value is not usable.
type Collection[T any] struct{ binding Adapter[T] }

func NewCollection[T any](adapter Adapter[T]) *Collection[T] {
	return &Collection[T]{binding: adapter}
}

func (c *Collection[T]) wrap(op string, err error) error {
	return &OperationError{Operation: op, Resource: c.binding.Kind, Cause: err}
}

// Get fetches by ID. Use Find with Name for an exact name lookup.
func (c *Collection[T]) Get(ctx context.Context, id string) (*T, error) {
	if err := ID(id).Validate(); err != nil {
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
			o.query.Set("name", c.binding.NameQuery(*o.name))
		}
		if c.binding.Iterate != nil {
			for value, err := range c.binding.Iterate(ctx, o.query) {
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
		err := c.binding.List(o.query).EachPage(ctx, func(_ context.Context, page pagination.Page) (bool, error) {
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
	if err := ref.Validate(); err != nil {
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
	if err := ref.Validate(); err != nil {
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
	if err := ref.Validate(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(status) == "" {
		return nil, invalid("target status must not be empty")
	}
	o, err := parseWait(opts)
	if err != nil {
		return nil, err
	}
	if c.binding.Status == nil {
		return nil, c.wrap("wait", ErrUnsupported)
	}
	ctx, cancel := context.WithTimeout(ctx, o.timeout)
	defer cancel()
	v, err := c.Find(ctx, ref)
	if err != nil {
		return nil, c.wrap("wait", err)
	}
	id := c.binding.ID(v)
	for {
		current := c.binding.Status(v)
		if strings.EqualFold(current, status) {
			return v, nil
		}
		if c.binding.Failed != nil && c.binding.Failed(current) {
			return nil, &FailedStateError{Resource: c.binding.Kind, ID: id, Status: current}
		}
		timer := time.NewTimer(o.interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, c.wrap("wait", ctx.Err())
		case <-timer.C:
		}
		v, err = c.Get(ctx, id)
		if err != nil {
			return nil, c.wrap("wait", err)
		}
	}
}

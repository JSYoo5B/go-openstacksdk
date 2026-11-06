package osfederation

import (
	"context"
	"iter"
	"net/url"
	"slices"
)

// Get implements a Python get_* proxy call (Resource.fetch).
func Get[O ~func(*ReadOpts) error](ctx context.Context, t Target, id string, options []O) (*Record, error) {
	const op = "Get"
	p, err := t.Prepare(ctx)
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	policy, err := Apply(ctx, p.guard, ReadOpts{}, CopyRead, slices.Clone(options))
	if err == nil {
		err = ValidateID("ID", id)
	}
	if err == nil {
		err = p.Headers(policy.Headers)
	}
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	value, err := p.Get(id, nil)
	return value, t.Wrap(ctx, op, err)
}

// Create implements a Python create_* proxy call (Resource.create with PUT).
func Create[O ~func(*WriteOpts) error](ctx context.Context, t Target, id string, options []O) (*Record, error) {
	const op = "Create"
	p, err := t.Prepare(ctx)
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	policy, err := Apply(ctx, p.guard, WriteOpts{}, CopyWrite, slices.Clone(options))
	if err == nil {
		err = ValidateID("ID", id)
	}
	if err == nil {
		err = p.Headers(policy.Headers)
	}
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	value, err := p.Create(id, policy.Fields)
	return value, t.Wrap(ctx, op, err)
}

// Update implements a Python update_* proxy call (Resource.commit with PATCH).
func Update[O ~func(*WriteOpts) error](ctx context.Context, t Target, id string, options []O) (*Record, error) {
	const op = "Update"
	p, err := t.Prepare(ctx)
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	policy, err := Apply(ctx, p.guard, WriteOpts{}, CopyWrite, slices.Clone(options))
	if err == nil {
		err = ValidateID("ID", id)
	}
	if err == nil {
		err = p.Headers(policy.Headers)
	}
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	value, err := p.Update(id, policy.Fields)
	return value, t.Wrap(ctx, op, err)
}

// Delete implements a Python delete_* proxy call with ignore_missing=True.
func Delete[O ~func(*DeleteOpts) error](ctx context.Context, t Target, id string, options []O) error {
	const op = "Delete"
	p, err := t.Prepare(ctx)
	if err != nil {
		return t.Wrap(ctx, op, err)
	}
	policy, err := Apply(ctx, p.guard, DeleteOpts{}, CopyDelete, slices.Clone(options))
	if err == nil {
		err = ValidateID("ID", id)
	}
	if err == nil {
		err = p.Headers(policy.Headers)
	}
	if err != nil {
		return t.Wrap(ctx, op, err)
	}
	return t.Wrap(ctx, op, p.Delete(id, policy.IgnoreMissing == nil || *policy.IgnoreMissing))
}

// List implements a Python list proxy call. The iterator is lazy and every
// iteration captures the client and sends the requests again.
func List[O ~func(*ListOpts) error](ctx context.Context, t Target, options []O) iter.Seq2[*Record, error] {
	owned := slices.Clone(options)
	return func(yield func(*Record, error) bool) {
		const op = "List"
		p, err := t.Prepare(ctx)
		if err != nil {
			yield(nil, t.Wrap(ctx, op, err))
			return
		}
		policy, err := Apply(ctx, p.guard, ListOpts{}, CopyList, owned)
		if err != nil {
			yield(nil, t.Wrap(ctx, op, err))
			return
		}
		for value, err := range p.ListOptions(policy) {
			if !yield(value, t.Wrap(ctx, op, err)) || err != nil {
				return
			}
		}
	}
}

// Find implements a Python find_* proxy call. getQuery holds the URI
// attributes that Resource.find forwards to fetch as query parameters.
func Find[O ~func(*FindOpts) error](ctx context.Context, t Target, nameOrID string, getQuery url.Values, options []O) (*Record, error) {
	const op = "Find"
	p, err := t.Prepare(ctx)
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	policy, err := Apply(ctx, p.guard, FindOpts{}, CopyFind, slices.Clone(options))
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	value, err := p.Find(ctx, nameOrID, getQuery, policy.IgnoreMissing == nil || *policy.IgnoreMissing)
	return value, t.Wrap(ctx, op, err)
}

// WriteSnapshot replaces the whole write policy with an owned copy.
func WriteSnapshot(v WriteOpts) func(*WriteOpts) error {
	owned := CopyWrite(v)
	return func(c *WriteOpts) error { *c = CopyWrite(owned); return nil }
}

// ListSnapshot replaces the whole list policy with an owned copy.
func ListSnapshot(v ListOpts) func(*ListOpts) error {
	owned := CopyList(v)
	return func(c *ListOpts) error { *c = CopyList(owned); return nil }
}

// ListQuery records one Python **query name. Names that are neither server
// query parameters nor local Body filters are ignored before marshaling.
func (s *Spec) ListQuery(key string, value any) func(*ListOpts) error {
	_, query := s.Query[key]
	_, filter := s.Filters[key]
	if !query && !filter {
		return func(*ListOpts) error { return nil }
	}
	raw := Marshal(value)
	return func(c *ListOpts) error { return SetField(&c.Query, key, raw) }
}

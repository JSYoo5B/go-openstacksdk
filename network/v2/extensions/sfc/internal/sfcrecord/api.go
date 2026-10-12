package sfcrecord

import (
	"context"
	"iter"
	"maps"
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
	value, err := p.Get(id, nil, nil)
	return value, t.Wrap(ctx, op, err)
}

// Create implements a Python create_* proxy call (Resource.create with POST).
func Create[O ~func(*WriteOpts) error](ctx context.Context, t Target, options []O) (*Record, error) {
	const op = "Create"
	p, err := t.Prepare(ctx)
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	policy, err := Apply(ctx, p.guard, WriteOpts{}, CopyWrite, slices.Clone(options))
	if err == nil {
		err = p.Headers(policy.Headers)
	}
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	value, err := p.Create(policy.Fields)
	return value, t.Wrap(ctx, op, err)
}

// Update implements a Python update_* proxy call (Resource.commit with PUT).
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

// Find implements a Python find_* proxy call.
func Find[O ~func(*FindOpts) error](ctx context.Context, t Target, nameOrID string, options []O) (*Record, error) {
	const op = "Find"
	p, err := t.Prepare(ctx)
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	policy, err := Apply(ctx, p.guard, FindOpts{}, CopyFind, slices.Clone(options))
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	value, err := p.Find(ctx, nameOrID, policy)
	return value, t.Wrap(ctx, op, err)
}

// ReadHeaders sets per-request headers.
func ReadHeaders(values map[string]string) func(*ReadOpts) error {
	owned := maps.Clone(values)
	return func(c *ReadOpts) error { c.Headers = maps.Clone(owned); return nil }
}

// WriteSnapshot replaces the whole write policy with an owned copy.
func WriteSnapshot(v WriteOpts) func(*WriteOpts) error {
	owned := CopyWrite(v)
	return func(c *WriteOpts) error { *c = CopyWrite(owned); return nil }
}

// WriteField records one Python attribute. Unknown names are ignored before
// marshaling, as Resource.new drops them.
func (s *Spec) WriteField(key string, value any) func(*WriteOpts) error {
	if _, known := s.Attrs[key]; !known {
		return func(*WriteOpts) error { return nil }
	}
	raw := Marshal(value)
	return func(c *WriteOpts) error { return SetField(&c.Fields, key, raw) }
}

// WriteHeaders sets per-request headers.
func WriteHeaders(values map[string]string) func(*WriteOpts) error {
	owned := maps.Clone(values)
	return func(c *WriteOpts) error { c.Headers = maps.Clone(owned); return nil }
}

// DeleteIgnoreMissing selects Python ignore_missing.
func DeleteIgnoreMissing(v bool) func(*DeleteOpts) error {
	return func(c *DeleteOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}

// DeleteHeaders sets per-request headers.
func DeleteHeaders(values map[string]string) func(*DeleteOpts) error {
	owned := maps.Clone(values)
	return func(c *DeleteOpts) error { c.Headers = maps.Clone(owned); return nil }
}

// FindIgnoreMissing selects Python ignore_missing.
func FindIgnoreMissing(v bool) func(*FindOpts) error {
	return func(c *FindOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}

// FindQuery records one Python find **query name. Validation happens when
// Find runs, so reserved names fail before HTTP.
func FindQuery(key string, value any) func(*FindOpts) error {
	raw := Marshal(value)
	return func(c *FindOpts) error { return SetField(&c.Query, key, raw) }
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

func ListLimit(v int) func(*ListOpts) error {
	return func(c *ListOpts) error { x := v; c.Limit = &x; return nil }
}
func ListMarker(v string) func(*ListOpts) error {
	return func(c *ListOpts) error { c.Marker = v; return nil }
}
func ListMaxItems(v int) func(*ListOpts) error {
	return func(c *ListOpts) error { c.MaxItems = v; return nil }
}
func ListPaginated(v bool) func(*ListOpts) error {
	return func(c *ListOpts) error { x := v; c.Paginated = &x; return nil }
}
func ListHeaders(values map[string]string) func(*ListOpts) error {
	owned := maps.Clone(values)
	return func(c *ListOpts) error { c.Headers = maps.Clone(owned); return nil }
}

// NewSpec builds a Spec from the Python Body attribute table. attrs maps each
// Python attribute name to its Body key; query lists the QueryParameters
// names (limit and marker are added). Body attributes outside the query
// mapping become local filters, as Resource.list does.
func NewSpec(kind, path, resourceKey, resourcesKey string, attrs map[string]string, types map[string]string, query []string) *Spec {
	spec := &Spec{Kind: kind, Path: path, ResourceKey: resourceKey, ResourcesKey: resourcesKey,
		Types: maps.Clone(types), Attrs: make(map[string]string), Query: map[string]string{"limit": "limit", "marker": "marker"}, Filters: make(map[string]string)}
	for _, name := range query {
		spec.Query[name] = name
	}
	all := map[string]string{"id": "id", "name": "name"}
	maps.Copy(all, attrs)
	seen := make(map[string]bool)
	for attr, body := range all {
		spec.Attrs[attr] = body
		spec.Attrs[body] = body
		if !seen[body] {
			seen[body] = true
			spec.Fields = append(spec.Fields, body)
		}
		if _, server := spec.Query[attr]; !server {
			spec.Filters[attr] = body
		}
	}
	slices.Sort(spec.Fields)
	return spec
}

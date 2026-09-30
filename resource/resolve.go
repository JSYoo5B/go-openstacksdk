package resource

import "context"

// ResolveID binds a reference to a stable identifier. Explicit IDs require no
// request; names use the collection's exact lookup and duplicate-name policy.
// Scoped resources and cross-service operations use this before constructing URLs.
func (c *Collection[T]) ResolveID(ctx context.Context, ref Ref) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", c.wrap("resolve", err)
	}
	if err := ref.Validate(); err != nil {
		return "", err
	}
	if !ref.IsName() {
		return ref.String(), nil
	}
	if c.binding.ID == nil {
		return "", c.wrap("resolve", ErrUnsupported)
	}
	value, err := c.Find(ctx, ref)
	if err != nil {
		return "", err
	}
	if value == nil {
		return "", &NotFoundError{Resource: c.binding.Kind, Reference: ref.String()}
	}
	id := c.binding.ID(value)
	if err := ID(id).Validate(); err != nil {
		return "", c.wrap("resolve", err)
	}
	return id, nil
}

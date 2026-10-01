package resource

import (
	"context"
	"errors"
)

// WaitDeleted resolves a name once and polls its ID until Get returns 404.
// An already missing resource succeeds; authorization and transport errors fail.
func (c *Collection[T]) WaitDeleted(ctx context.Context, ref Ref, opts ...WaitOption) error {
	if err := c.validateRef(ref); err != nil {
		return err
	}
	o, err := parseWait(opts)
	if err != nil {
		return err
	}
	ctx, cancel := o.context(ctx)
	defer cancel()
	id := ref.String()
	if ref.IsName() {
		value, err := c.Find(ctx, ref, WithIgnoreMissing())
		if err != nil {
			return err
		}
		if value == nil {
			return nil
		}
		id = c.binding.ID(value)
	}
	for {
		if err := ctx.Err(); err != nil {
			return c.wrap("wait deleted", err)
		}
		_, err := c.Get(ctx, id)
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		if err != nil {
			return c.wrap("wait deleted", err)
		}
		if err := o.pause(ctx); err != nil {
			return c.wrap("wait deleted", err)
		}
	}
}

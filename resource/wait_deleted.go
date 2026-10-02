package resource

import (
	"context"
	"errors"
	"strings"
)

// WaitDeleted resolves a name once and polls its ID until a 404, nil resource
// or deleted status. A selected attribute replaces the binding's status field.
// An already missing resource succeeds; authorization and transport errors fail.
func (c *Collection[T]) WaitDeleted(ctx context.Context, ref Ref, opts ...WaitOption) error {
	if err := c.validateRef(ref); err != nil {
		return err
	}
	o, err := parseWait(opts)
	if err != nil {
		return err
	}
	statusField, progressField, err := waitFields[T](o)
	if err != nil {
		return c.wrap("wait deleted", err)
	}
	ctx, cancel := o.context(ctx)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return c.wrap("wait deleted", err)
	}
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
		value, err := c.Get(ctx, id)
		if ctx.Err() != nil {
			return c.wrap("wait deleted", ctx.Err())
		}
		if !terminalReadError(err) && errors.Is(err, ErrNotFound) {
			return nil
		}
		if err != nil {
			return c.wrap("wait deleted", err)
		}
		if value == nil {
			return nil
		}
		if c.binding.Status != nil || statusField != nil {
			state, err := waitStatus(value, statusField, c.binding.Status)
			if err != nil {
				return c.wrap("wait deleted", err)
			}
			if strings.EqualFold(state, "deleted") {
				return nil
			}
		}
		if err := reportWaitProgress(o, value, progressField); err != nil {
			return c.wrap("wait deleted", err)
		}
		if err := o.pause(ctx); err != nil {
			return c.wrap("wait deleted", err)
		}
	}
}

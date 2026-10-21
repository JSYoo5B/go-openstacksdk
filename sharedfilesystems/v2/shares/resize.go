package shares

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ResizeAction reports what Resize did.
type ResizeAction string

const (
	// ResizeNone means no action was sent: the size already matched or the
	// direction was disabled.
	ResizeNone ResizeAction = "none"
	// ResizeExtend means an extend action was acknowledged.
	ResizeExtend ResizeAction = "extend"
	// ResizeShrink means a shrink action was acknowledged.
	ResizeShrink ResizeAction = "shrink"
)

type resizeConfig struct {
	noShrink, noExtend, force bool
}

// ResizeOption configures Resize.
type ResizeOption func(*resizeConfig) error

// WithResizeNoShrink makes Resize skip a request when the target is smaller.
func WithResizeNoShrink() ResizeOption {
	return func(c *resizeConfig) error { c.noShrink = true; return nil }
}

// WithResizeNoExtend makes Resize skip a request when the target is larger.
func WithResizeNoExtend() ResizeOption {
	return func(c *resizeConfig) error { c.noExtend = true; return nil }
}

// WithResizeForce adds "force": true to an extend action (Manila 2.64 and
// later, administrators only). It never affects shrink.
func WithResizeForce() ResizeOption {
	return func(c *resizeConfig) error { c.force = true; return nil }
}

// Resize reads the share with GET, compares its current size with newSize and
// then sends at most one extend or shrink action. Equal sizes and a disabled
// direction send no action and return ResizeNone. The size read is not locked,
// so a concurrent resize can make the decision stale, and the acknowledged
// action does not mean the share finished resizing: wait with WaitFor. The
// action is never replayed. Prefer Extend or Shrink when the direction is known.
func (a *API) Resize(ctx context.Context, id string, newSize int, options ...ResizeOption) (ResizeAction, error) {
	var cfg resizeConfig
	for _, option := range options {
		if option == nil {
			return ResizeNone, request.Wrap("Resize", "shares", fmt.Errorf("%w: nil resize option", resource.ErrInvalidOption))
		}
		if err := option(&cfg); err != nil {
			return ResizeNone, request.Wrap("Resize", "shares", err)
		}
	}
	if err := resource.ID(id).Validate(); err != nil {
		return ResizeNone, request.Wrap("Resize", "shares", err)
	}
	current, err := a.Get(ctx, id)
	if err != nil {
		return ResizeNone, err
	}
	switch {
	case newSize > current.Size && !cfg.noExtend:
		var extra []ExtendOption
		if cfg.force {
			extra = append(extra, WithExtendField("force", true))
		}
		if err := a.Extend(ctx, id, ExtendOpts{NewSize: newSize}, extra...); err != nil {
			return ResizeNone, err
		}
		return ResizeExtend, nil
	case newSize < current.Size && !cfg.noShrink:
		if err := a.Shrink(ctx, id, ShrinkOpts{NewSize: newSize}); err != nil {
			return ResizeNone, err
		}
		return ResizeShrink, nil
	}
	return ResizeNone, nil
}

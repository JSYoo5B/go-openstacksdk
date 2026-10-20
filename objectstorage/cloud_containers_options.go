package objectstorage

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type ContainerCreateOpts struct {
	Public  bool
	Headers map[string]string
}
type ContainerCreateOption func(*ContainerCreateOpts) error
type ContainerAccessOpts struct {
	Refresh bool
	Headers map[string]string
}
type ContainerAccessOption func(*ContainerAccessOpts) error
type ContainerReadOpts struct{ Headers map[string]string }
type ContainerReadOption func(*ContainerReadOpts) error

func WithContainerPublic(value bool) ContainerCreateOption {
	return func(c *ContainerCreateOpts) error { c.Public = value; return nil }
}
func WithContainerCreateOpts(value ContainerCreateOpts) ContainerCreateOption {
	value = copyContainerCreate(value)
	return func(c *ContainerCreateOpts) error { *c = copyContainerCreate(value); return nil }
}
func WithContainerCreateHeaders(value map[string]string) ContainerCreateOption {
	value = maps.Clone(value)
	return func(c *ContainerCreateOpts) error { c.Headers = containerMergeHeaders(c.Headers, value); return nil }
}
func WithContainerAccessRefresh(value bool) ContainerAccessOption {
	return func(c *ContainerAccessOpts) error { c.Refresh = value; return nil }
}
func WithContainerAccessOpts(value ContainerAccessOpts) ContainerAccessOption {
	value = copyContainerAccess(value)
	return func(c *ContainerAccessOpts) error { *c = copyContainerAccess(value); return nil }
}
func WithContainerAccessHeaders(value map[string]string) ContainerAccessOption {
	value = maps.Clone(value)
	return func(c *ContainerAccessOpts) error { c.Headers = containerMergeHeaders(c.Headers, value); return nil }
}
func WithContainerReadOpts(value ContainerReadOpts) ContainerReadOption {
	value = copyContainerRead(value)
	return func(c *ContainerReadOpts) error { *c = copyContainerRead(value); return nil }
}
func WithContainerReadHeaders(value map[string]string) ContainerReadOption {
	value = maps.Clone(value)
	return func(c *ContainerReadOpts) error { c.Headers = containerMergeHeaders(c.Headers, value); return nil }
}
func containerMergeHeaders(left, right map[string]string) map[string]string {
	owned := maps.Clone(left)
	if owned == nil {
		owned = map[string]string{}
	}
	for k, v := range right {
		owned[k] = v
	}
	return owned
}
func copyContainerCreate(c ContainerCreateOpts) ContainerCreateOpts {
	c.Headers = containerMergeHeaders(nil, c.Headers)
	return c
}
func copyContainerAccess(c ContainerAccessOpts) ContainerAccessOpts {
	c.Headers = containerMergeHeaders(nil, c.Headers)
	return c
}
func copyContainerRead(c ContainerReadOpts) ContainerReadOpts {
	c.Headers = containerMergeHeaders(nil, c.Headers)
	return c
}
func containerOptions[T any, O ~func(*T) error](ctx context.Context, p *containers.CloudRecords, value T, copyValue func(T) T, options []O) (T, error) {
	value = copyValue(value)
	for _, option := range slices.Clone(options) {
		if err := p.Check(ctx); err != nil {
			return value, err
		}
		if option == nil {
			return value, fmt.Errorf("%w: nil container option", resource.ErrInvalidOption)
		}
		callback := copyValue(value)
		if err := errors.Join(option(&callback), p.Check(ctx)); err != nil {
			return value, err
		}
		value = copyValue(callback)
	}
	return value, p.Check(ctx)
}
func applyContainerCreateOptions(ctx context.Context, p *containers.CloudRecords, options []ContainerCreateOption) (ContainerCreateOpts, error) {
	return containerOptions(ctx, p, ContainerCreateOpts{}, copyContainerCreate, options)
}
func applyContainerAccessOptions(ctx context.Context, p *containers.CloudRecords, options []ContainerAccessOption) (ContainerAccessOpts, error) {
	return containerOptions(ctx, p, ContainerAccessOpts{}, copyContainerAccess, options)
}
func applyContainerReadOptions(ctx context.Context, p *containers.CloudRecords, options []ContainerReadOption) (ContainerReadOpts, error) {
	return containerOptions(ctx, p, ContainerReadOpts{}, copyContainerRead, options)
}

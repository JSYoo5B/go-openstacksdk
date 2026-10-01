package stacks

import (
	"context"
	"fmt"
	"net/url"

	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// StackScope retains the complete canonical identity across mutations and waits.
type StackScope struct {
	api       *API
	identity  StackIdentity
	values    *resource.Collection[StackResource]
	deletions *resource.Collection[StackResource]
}

// ForStack binds a known identity without issuing a lookup request.
func (a *API) ForStack(identity StackIdentity) (*StackScope, error) {
	if err := identity.Validate(); err != nil {
		return nil, err
	}
	scope := &StackScope{api: a, identity: identity}
	binding := resource.Adapter[StackResource]{
		Kind:       "stacks",
		ValidateID: validateSegment,
		ID:         func(value *StackResource) string { return value.ID },
		Get: func(ctx context.Context, id string) (*StackResource, error) {
			if id != identity.ID {
				return nil, fmt.Errorf("%w: ID conflicts with the fixed stack identity", resource.ErrInvalidOption)
			}
			return scope.Get(ctx)
		},
		Delete: func(ctx context.Context, id string) error {
			if id != identity.ID {
				return fmt.Errorf("%w: ID conflicts with the fixed stack identity", resource.ErrInvalidOption)
			}
			return a.Delete(ctx, url.PathEscape(identity.Name), url.PathEscape(id))
		},
		Status: func(value *StackResource) string { return value.Status },
		Failed: failedStatus,
	}
	scope.values = resource.NewCollection(binding)
	binding.Get = func(ctx context.Context, id string) (*StackResource, error) {
		if id != identity.ID {
			return nil, fmt.Errorf("%w: ID conflicts with the fixed stack identity", resource.ErrInvalidOption)
		}
		value, err := scope.Get(ctx)
		return deletionState(value, err)
	}
	scope.deletions = resource.NewCollection(binding)
	return scope, nil
}

// InStack resolves a reference once. Names are exact and duplicate names fail;
// an ID uses Heat's native identity lookup, including its canonical redirect.
func (a *API) InStack(ctx context.Context, ref resource.Ref) (*StackScope, error) {
	identity, err := a.Resources().ResolveIdentity(ctx, ref)
	if err != nil {
		return nil, err
	}
	return a.ForStack(identity)
}

// Identity returns a copy; modifying it cannot change the bound scope.
func (s *StackScope) Identity() StackIdentity { return s.identity }

func (s *StackScope) Get(ctx context.Context, options ...GetOption) (*StackResource, error) {
	if err := ctx.Err(); err != nil {
		return nil, request.Wrap("get", "stacks", err)
	}
	endpoint, err := retrievalURL(s.api.client.ServiceURL("stacks", url.PathEscape(s.identity.Name), url.PathEscape(s.identity.ID)), options)
	if err != nil {
		return nil, err
	}
	value, err := fetchResource(ctx, s.api.client, endpoint)
	if err != nil {
		return nil, err
	}
	if value.Name != s.identity.Name || value.ID != s.identity.ID {
		return nil, request.Wrap("get", "stacks", fmt.Errorf("stack response does not match fixed identity %q/%q", s.identity.Name, s.identity.ID))
	}
	return value, nil
}

// Delete ignores missing stacks by default; WithMissingError makes 404 fail.
func (s *StackScope) Delete(ctx context.Context, options ...resource.LookupOption) error {
	return s.values.Delete(ctx, resource.ID(s.identity.ID), options...)
}

func (s *StackScope) Wait(ctx context.Context, status string, options ...resource.WaitOption) (*StackResource, error) {
	return s.values.Wait(ctx, resource.ID(s.identity.ID), status, options...)
}

// WaitDeleted succeeds on HTTP 404 or DELETE_COMPLETE. A *_FAILED state returns
// ErrFailedState immediately; authorization and other HTTP errors are preserved.
func (s *StackScope) WaitDeleted(ctx context.Context, options ...resource.WaitOption) error {
	return s.deletions.WaitDeleted(ctx, resource.ID(s.identity.ID), options...)
}

func (s *StackScope) Update(ctx context.Context, opts UpdateOpts, options ...UpdateOption) error {
	if err := ctx.Err(); err != nil {
		return request.Wrap("update", "stacks", err)
	}
	return s.api.Update(ctx, url.PathEscape(s.identity.Name), url.PathEscape(s.identity.ID), opts, options...)
}

func (s *StackScope) UpdatePatch(ctx context.Context, opts UpdateOpts, options ...UpdatePatchOption) error {
	if err := ctx.Err(); err != nil {
		return request.Wrap("update patch", "stacks", err)
	}
	return s.api.UpdatePatch(ctx, url.PathEscape(s.identity.Name), url.PathEscape(s.identity.ID), opts, options...)
}

func (s *StackScope) Abandon(ctx context.Context) (*AbandonedStack, error) {
	if err := ctx.Err(); err != nil {
		return nil, request.Wrap("abandon", "stacks", err)
	}
	return s.api.Abandon(ctx, url.PathEscape(s.identity.Name), url.PathEscape(s.identity.ID))
}

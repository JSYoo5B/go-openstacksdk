package limits

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/gophercloudsdk/internal/project"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type projectOptions struct{ identity *gophercloud.ServiceClient }

type ProjectOption func(*projectOptions) error

// WithIdentityClient supplies a separate Keystone v3 client for exact names.
func WithIdentityClient(client *gophercloud.ServiceClient) ProjectOption {
	return func(options *projectOptions) error {
		if err := project.ValidateIdentityClient(client); err != nil {
			return err
		}
		options.identity = client
		return nil
	}
}

// ProjectLimitsScope fixes the project for Nova's read-only limits singleton.
type ProjectLimitsScope struct {
	api       *API
	projectID string
}

// InProject binds IDs without HTTP, or resolves exact names once with Keystone.
func (a *API) InProject(ctx context.Context, ref resource.Ref, options ...ProjectOption) (*ProjectLimitsScope, error) {
	if err := a.validateClient(ctx); err != nil {
		return nil, limitsError("InProject", ref.String(), err)
	}
	var config projectOptions
	for _, apply := range options {
		if apply == nil {
			return nil, limitsError("InProject", ref.String(), fmt.Errorf("%w: nil project option", resource.ErrInvalidOption))
		}
		if err := apply(&config); err != nil {
			return nil, limitsError("InProject", ref.String(), err)
		}
	}
	id, err := project.Resolve(ctx, a.client, ref, config.identity)
	if err != nil {
		return nil, limitsError("InProject", ref.String(), err)
	}
	return &ProjectLimitsScope{api: a, projectID: id}, nil
}

// CurrentProject binds recorded Keystone v2/v3 scope without authenticating,
// refreshing the token, or guessing the project from an endpoint or token.
func (a *API) CurrentProject(ctx context.Context) (*ProjectLimitsScope, error) {
	if err := a.validateClient(ctx); err != nil {
		return nil, limitsError("CurrentProject", "", err)
	}
	id, err := project.Current(ctx, a.client)
	if err != nil {
		return nil, limitsError("CurrentProject", "", err)
	}
	return &ProjectLimitsScope{api: a, projectID: id}, nil
}

func (s *ProjectLimitsScope) ProjectID() string {
	if s == nil {
		return ""
	}
	return s.projectID
}

func (a *API) validateClient(ctx context.Context) error {
	var client *gophercloud.ServiceClient
	if a != nil {
		client = a.client
	}
	return project.ValidateClient(ctx, client)
}

func (s *ProjectLimitsScope) validate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil {
		return fmt.Errorf("%w: project limits scope is required", resource.ErrInvalidOption)
	}
	if err := s.api.validateClient(ctx); err != nil {
		return err
	}
	return resource.ID(s.projectID).Validate()
}

func limitsError(operation, projectID string, err error) error {
	if gophercloud.ResponseCodeIs(err, 404) {
		err = &resource.NotFoundError{Resource: "compute limits", Reference: projectID, Cause: err}
	}
	return request.Wrap(operation, "compute limits", err)
}

package quotas

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/internal/project"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type projectOptions struct {
	identity    *gophercloud.ServiceClient
	allProjects *bool
}

// ProjectOption configures quota scope construction, not a resource lookup.
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

// WithAllProjects explicitly sets Designate's cross-project authorization
// header. An omitted option preserves the service client's configured header.
func WithAllProjects(enabled bool) ProjectOption {
	return func(options *projectOptions) error {
		value := enabled
		options.allProjects = &value
		return nil
	}
}

// ProjectQuotaScope fixes the project for the DNS quota singleton.
type ProjectQuotaScope struct {
	api         *API
	projectID   string
	allProjects *bool
}

// InProject resolves exact names once with Keystone, or binds IDs without HTTP.
func (a *API) InProject(ctx context.Context, ref resource.Ref, options ...ProjectOption) (*ProjectQuotaScope, error) {
	if err := a.validateQuotaClient(ctx); err != nil {
		return nil, quotaError("InProject", ref.String(), err)
	}
	var config projectOptions
	for _, apply := range options {
		if apply == nil {
			return nil, quotaError("InProject", ref.String(), fmt.Errorf("%w: nil project option", resource.ErrInvalidOption))
		}
		if err := apply(&config); err != nil {
			return nil, quotaError("InProject", ref.String(), err)
		}
	}
	id, err := project.Resolve(ctx, a.client, ref, config.identity)
	if err != nil {
		return nil, quotaError("InProject", ref.String(), err)
	}
	return &ProjectQuotaScope{api: a, projectID: id, allProjects: config.allProjects}, nil
}

// CurrentProject binds recorded Keystone scope without refreshing authentication
// or guessing the project from a token or endpoint. The user may apply the same
// header options as InProject; project-name resolution is not needed here.
func (a *API) CurrentProject(ctx context.Context, options ...ProjectOption) (*ProjectQuotaScope, error) {
	if err := a.validateQuotaClient(ctx); err != nil {
		return nil, quotaError("CurrentProject", "", err)
	}
	id, err := project.Current(ctx, a.client)
	if err != nil {
		return nil, quotaError("CurrentProject", "", err)
	}
	scope, err := a.InProject(ctx, resource.ID(id), options...)
	return scope, quotaError("CurrentProject", id, err)
}

func (s *ProjectQuotaScope) ProjectID() string {
	if s == nil {
		return ""
	}
	return s.projectID
}

func (a *API) validateQuotaClient(ctx context.Context) error {
	var client *gophercloud.ServiceClient
	if a != nil {
		client = a.client
	}
	return project.ValidateClient(ctx, client)
}

func (s *ProjectQuotaScope) validate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil {
		return fmt.Errorf("%w: DNS quota scope is required", resource.ErrInvalidOption)
	}
	if err := s.api.validateQuotaClient(ctx); err != nil {
		return err
	}
	return resource.ID(s.projectID).Validate()
}

func quotaError(operation, projectID string, err error) error {
	if gophercloud.ResponseCodeIs(err, 404) {
		err = &resource.NotFoundError{Resource: "DNS quota", Reference: projectID, Cause: err}
	}
	return request.Wrap(operation, "DNS quota", err)
}

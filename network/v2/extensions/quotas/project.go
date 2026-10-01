package quotas

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/project"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

type projectOptions struct{ identity *gophercloud.ServiceClient }

// ProjectOption configures the one-time project resolution for a quota scope.
type ProjectOption func(*projectOptions) error

// WithIdentityClient supplies the Keystone v3 client used for exact project
// name resolution. The Network client is never used to query Keystone.
func WithIdentityClient(client *gophercloud.ServiceClient) ProjectOption {
	return func(options *projectOptions) error {
		if err := project.ValidateIdentityClient(client); err != nil {
			return err
		}
		options.identity = client
		return nil
	}
}

// ProjectQuotaScope fixes one Neutron project's quota singleton. Delete resets
// its quota overrides; the scope does not expose resource creation or listing.
type ProjectQuotaScope struct {
	api       *API
	projectID string
}

// InProject fixes the target once. IDs require no lookup; names use a separate
// Keystone client with the shared exact-name and ambiguity policy.
func (a *API) InProject(ctx context.Context, ref resource.Ref, options ...ProjectOption) (*ProjectQuotaScope, error) {
	if err := a.validateQuotaClient(ctx); err != nil {
		return nil, request.Wrap("InProject", "network quota", err)
	}
	if err := ref.Validate(); err != nil {
		return nil, request.Wrap("InProject", "network quota", err)
	}
	var config projectOptions
	for _, apply := range options {
		if apply == nil {
			return nil, request.Wrap("InProject", "network quota", fmt.Errorf("%w: nil project option", resource.ErrInvalidOption))
		}
		if err := apply(&config); err != nil {
			return nil, request.Wrap("InProject", "network quota", err)
		}
	}
	id, err := project.Resolve(ctx, a.client, ref, config.identity)
	if err != nil {
		return nil, request.Wrap("InProject", "network quota", err)
	}
	return &ProjectQuotaScope{api: a, projectID: id}, nil
}

// CurrentProject reads the recorded Keystone v3 project or v2 token tenant.
// It never guesses a project from the endpoint URL or a manual token string.
func (a *API) CurrentProject(ctx context.Context) (*ProjectQuotaScope, error) {
	if err := a.validateQuotaClient(ctx); err != nil {
		return nil, request.Wrap("CurrentProject", "network quota", err)
	}
	id, err := project.Current(ctx, a.client)
	if err != nil {
		return nil, request.Wrap("CurrentProject", "network quota", err)
	}
	return a.InProject(ctx, resource.ID(id))
}

// ProjectID returns the fixed request target without a request.
func (s *ProjectQuotaScope) ProjectID() string { return s.projectID }

func (a *API) validateQuotaClient(ctx context.Context) error {
	if a == nil {
		if err := ctx.Err(); err != nil {
			return err
		}
		return fmt.Errorf("%w: network quotas require a service client", resource.ErrInvalidOption)
	}
	return project.ValidateClient(ctx, a.client)
}

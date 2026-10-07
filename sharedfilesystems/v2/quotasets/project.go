package quotasets

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

// WithIdentityClient supplies the separate Keystone v3 client used for exact
// project and user names. Explicit IDs do not issue lookup requests.
func WithIdentityClient(client *gophercloud.ServiceClient) ProjectOption {
	return func(options *projectOptions) error {
		if err := project.ValidateIdentityClient(client); err != nil {
			return err
		}
		options.identity = client
		return nil
	}
}

type ProjectQuotaScope struct {
	api       *API
	projectID string
	identity  *gophercloud.ServiceClient
}

func (a *API) InProject(ctx context.Context, ref resource.Ref, options ...ProjectOption) (*ProjectQuotaScope, error) {
	if err := a.validate(ctx); err != nil {
		return nil, quotaError("InProject", ref.String(), err)
	}
	if _, err := a.minorVersion(); err != nil {
		return nil, quotaError("InProject", ref.String(), err)
	}
	if err := ref.Validate(); err != nil {
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
	return &ProjectQuotaScope{api: a, projectID: id, identity: config.identity}, nil
}

// CurrentProject reads the recorded Keystone v2/v3 project, without refreshing
// authentication or inferring a project from a URL or manual token.
func (a *API) CurrentProject(ctx context.Context) (*ProjectQuotaScope, error) {
	if err := a.validate(ctx); err != nil {
		return nil, quotaError("CurrentProject", "", err)
	}
	id, err := project.Current(ctx, a.client)
	if err != nil {
		return nil, quotaError("CurrentProject", "", err)
	}
	return a.InProject(ctx, resource.ID(id))
}

func (s *ProjectQuotaScope) ProjectID() string {
	if s == nil {
		return ""
	}
	return s.projectID
}

func (s *ProjectQuotaScope) validate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil {
		return fmt.Errorf("%w: project quota scope is required", resource.ErrInvalidOption)
	}
	if err := s.api.validate(ctx); err != nil {
		return err
	}
	return resource.ID(s.projectID).Validate()
}

func quotaError(operation, projectID string, err error) error {
	if gophercloud.ResponseCodeIs(err, 404) {
		err = &resource.NotFoundError{Resource: "shared file system quota", Reference: projectID, Cause: err}
	}
	return request.Wrap(operation, "shared file system quota", err)
}

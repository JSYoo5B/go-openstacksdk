package quotas

import (
	"context"
	"fmt"
	"strings"

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

// ProjectQuotaScope fixes the project for quota reads, updates and resets.
type ProjectQuotaScope struct {
	api       *API
	projectID string
}

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
	if err == nil {
		err = validateProjectID(id)
	}
	if err != nil {
		return nil, quotaError("InProject", ref.String(), err)
	}
	return &ProjectQuotaScope{api: a, projectID: id}, nil
}

// CurrentProject reads recorded Keystone v2/v3 scope, without refreshing auth.
func (a *API) CurrentProject(ctx context.Context) (*ProjectQuotaScope, error) {
	if err := a.validateQuotaClient(ctx); err != nil {
		return nil, quotaError("CurrentProject", "", err)
	}
	id, err := project.Current(ctx, a.client)
	if err == nil {
		err = validateProjectID(id)
	}
	if err != nil {
		return nil, quotaError("CurrentProject", "", err)
	}
	return &ProjectQuotaScope{api: a, projectID: id}, nil
}

func (s *ProjectQuotaScope) ProjectID() string { return s.projectID }

func validateProjectID(id string) error {
	if err := resource.ID(id).Validate(); err != nil {
		return err
	}
	if id == "defaults" || strings.ContainsRune(id, '\x00') {
		return fmt.Errorf("%w: project ID conflicts with the defaults route or contains NUL", resource.ErrInvalidOption)
	}
	return nil
}

func (a *API) validateQuotaClient(ctx context.Context) error {
	var client *gophercloud.ServiceClient
	if a != nil {
		client = a.client
	}
	return project.ValidateClient(ctx, client)
}

// The pinned native quota URL omits lbaas although its client base is v2.0.
// Scoped SDK operations use Octavia's actual lbaas routes. A caller-supplied
// ResourceBase already ending in lbaas is respected without double insertion.
func (a *API) quotaEndpoint(parts ...string) string {
	prefix := []string{"lbaas", "quotas"}
	if strings.HasSuffix(strings.TrimSuffix(a.client.ServiceURL(), "/"), "/lbaas") {
		prefix = []string{"quotas"}
	}
	return a.client.ServiceURL(append(prefix, parts...)...)
}

func quotaError(operation, projectID string, err error) error {
	if gophercloud.ResponseCodeIs(err, 404) {
		err = &resource.NotFoundError{Resource: "load balancer quota", Reference: projectID, Cause: err}
	}
	return request.Wrap(operation, "load balancer quota", err)
}

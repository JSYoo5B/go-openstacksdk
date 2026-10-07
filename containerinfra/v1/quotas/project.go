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

// ResourceName is Magnum's case-sensitive quota resource, not a quota row ID.
type ResourceName string

const Cluster ResourceName = "Cluster"

// ProjectQuotaScope resolves and fixes one project without fetching its quotas.
type ProjectQuotaScope struct {
	api       *API
	projectID string
}

// ResourceQuotaScope fixes both parts of Magnum's compound quota identity.
type ResourceQuotaScope struct {
	api          *API
	projectID    string
	resourceName ResourceName
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
		err = validateSegment(id)
	}
	if err != nil {
		return nil, quotaError("InProject", ref.String(), err)
	}
	return &ProjectQuotaScope{api: a, projectID: id}, nil
}

// CurrentProject reads the recorded Keystone v2/v3 scope without refreshing it.
func (a *API) CurrentProject(ctx context.Context) (*ProjectQuotaScope, error) {
	if err := a.validateQuotaClient(ctx); err != nil {
		return nil, quotaError("CurrentProject", "", err)
	}
	id, err := project.Current(ctx, a.client)
	if err == nil {
		err = validateSegment(id)
	}
	if err != nil {
		return nil, quotaError("CurrentProject", "", err)
	}
	return &ProjectQuotaScope{api: a, projectID: id}, nil
}

func (s *ProjectQuotaScope) ProjectID() string {
	if s == nil {
		return ""
	}
	return s.projectID
}

// ForResource selects a quota resource without HTTP. Cluster is the documented
// Magnum resource; additional deployment resource names retain server validation.
func (s *ProjectQuotaScope) ForResource(name ResourceName) (*ResourceQuotaScope, error) {
	if s == nil || s.api == nil || s.projectID == "" {
		return nil, quotaError("ForResource", "", fmt.Errorf("%w: project scope is required", resource.ErrInvalidOption))
	}
	if err := validateSegment(string(name)); err != nil {
		return nil, quotaError("ForResource", s.projectID, err)
	}
	return &ResourceQuotaScope{api: s.api, projectID: s.projectID, resourceName: name}, nil
}

func (s *ResourceQuotaScope) ProjectID() string {
	if s == nil {
		return ""
	}
	return s.projectID
}
func (s *ResourceQuotaScope) ResourceName() ResourceName {
	if s == nil {
		return ""
	}
	return s.resourceName
}

func validateSegment(value string) error {
	if err := resource.ID(value).Validate(); err != nil {
		return err
	}
	if strings.ContainsAny(value, "\x00\r\n\t") {
		return fmt.Errorf("%w: quota identity contains a control character", resource.ErrInvalidOption)
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

func quotaError(operation, reference string, err error) error {
	if gophercloud.ResponseCodeIs(err, 404) {
		err = &resource.NotFoundError{Resource: "container infrastructure quota", Reference: reference, Cause: err}
	}
	return request.Wrap(operation, "container infrastructure quota", err)
}

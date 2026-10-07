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

// ProjectOption configures project/user resolution when constructing a quota scope.
// Normal Connection helpers supply the correct Identity client themselves.
type ProjectOption func(*projectOptions) error

// WithIdentityClient supplies a Keystone v3 client for exact project/user name
// lookup. InUser inherits this client unless an option overrides it.
// The Nova client is never reused for Keystone requests.
func WithIdentityClient(client *gophercloud.ServiceClient) ProjectOption {
	return func(options *projectOptions) error {
		if err := project.ValidateIdentityClient(client); err != nil {
			return err
		}
		options.identity = client
		return nil
	}
}

// ProjectQuotaScope binds Nova's quota singleton to one resolved project.
// It exposes Get, Defaults, Detail, Update, Reset and an explicit InUser scope;
// quotas are not a CRUD list.
type ProjectQuotaScope struct {
	api       *API
	projectID string
	identity  *gophercloud.ServiceClient
}

// InProject fixes the project once. Explicit IDs require no lookup. Names use
// the supplied Identity v3 client and the shared exact-name project policy.
func (a *API) InProject(ctx context.Context, ref resource.Ref, options ...ProjectOption) (*ProjectQuotaScope, error) {
	if err := a.validateQuotaClient(ctx); err != nil {
		return nil, request.Wrap("InProject", "compute quota", err)
	}
	if err := ref.Validate(); err != nil {
		return nil, request.Wrap("InProject", "compute quota", err)
	}
	var config projectOptions
	for _, apply := range options {
		if apply == nil {
			return nil, request.Wrap("InProject", "compute quota", fmt.Errorf("%w: nil project option", resource.ErrInvalidOption))
		}
		if err := apply(&config); err != nil {
			return nil, request.Wrap("InProject", "compute quota", err)
		}
	}
	id, err := project.Resolve(ctx, a.client, ref, config.identity)
	if err != nil {
		return nil, request.Wrap("InProject", "compute quota", err)
	}
	return &ProjectQuotaScope{api: a, projectID: id, identity: config.identity}, nil
}

// CurrentProject binds the scope from the recorded Keystone authentication
// result. Manual tokens, system/domain scopes and missing auth results cannot
// identify a project; the endpoint URL and token string are never guessed.
func (a *API) CurrentProject(ctx context.Context) (*ProjectQuotaScope, error) {
	if err := a.validateQuotaClient(ctx); err != nil {
		return nil, request.Wrap("CurrentProject", "compute quota", err)
	}
	id, err := project.Current(ctx, a.client)
	if err != nil {
		return nil, request.Wrap("CurrentProject", "compute quota", err)
	}
	return a.InProject(ctx, resource.ID(id))
}

// ProjectID returns the resolved project without a request.
func (s *ProjectQuotaScope) ProjectID() string { return s.projectID }

func (a *API) validateQuotaClient(ctx context.Context) error {
	var client *gophercloud.ServiceClient
	if a != nil {
		client = a.client
	}
	return project.ValidateClient(ctx, client)
}

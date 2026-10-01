package quotasets

import (
	"context"
	"fmt"
	"reflect"

	"github.com/gophercloud/gophercloud/v2"
	tokens2 "github.com/gophercloud/gophercloud/v2/openstack/identity/v2/tokens"
	tokens3 "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
	"gophercloudsdk/identity/v3/projects"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

type projectOptions struct{ identity *gophercloud.ServiceClient }

// ProjectOption configures project resolution when constructing a quota scope.
// Normal Connection helpers supply the correct Identity client themselves.
type ProjectOption func(*projectOptions) error

// WithIdentityClient supplies a Keystone v3 client for an exact project name
// lookup. The Cinder client is never reused for Keystone requests.
func WithIdentityClient(client *gophercloud.ServiceClient) ProjectOption {
	return func(options *projectOptions) error {
		if client == nil || client.ProviderClient == nil || client.Type != "identity" {
			return fmt.Errorf("%w: project names require an Identity v3 service client", resource.ErrInvalidOption)
		}
		options.identity = client
		return nil
	}
}

// ProjectQuotaScope binds Cinder's quota singleton to one resolved project.
// It exposes only Get, Defaults, Usage, Update and Reset; quotas are not a CRUD list.
type ProjectQuotaScope struct {
	api       *API
	projectID string
}

// InProject fixes the project once. Explicit IDs require no lookup. Names use
// the supplied Identity v3 client and the shared exact-name project policy.
func (a *API) InProject(ctx context.Context, ref resource.Ref, options ...ProjectOption) (*ProjectQuotaScope, error) {
	if err := a.validateQuotaClient(ctx); err != nil {
		return nil, request.Wrap("InProject", "block storage quota", err)
	}
	if err := ref.Validate(); err != nil {
		return nil, request.Wrap("InProject", "block storage quota", err)
	}
	var config projectOptions
	for _, apply := range options {
		if apply == nil {
			return nil, request.Wrap("InProject", "block storage quota", fmt.Errorf("%w: nil project option", resource.ErrInvalidOption))
		}
		if err := apply(&config); err != nil {
			return nil, request.Wrap("InProject", "block storage quota", err)
		}
	}
	id := ref.String()
	if ref.IsName() {
		if config.identity == nil {
			return nil, request.Wrap("InProject", "block storage quota", fmt.Errorf("%w: project name resolution needs an Identity v3 client", resource.ErrUnsupported))
		}
		if config.identity == a.client {
			return nil, request.Wrap("InProject", "block storage quota", fmt.Errorf("%w: the Block Storage client cannot resolve Keystone project names", resource.ErrInvalidOption))
		}
		var err error
		id, err = projects.New(config.identity).Resources.ResolveID(ctx, ref)
		if err != nil {
			return nil, request.Wrap("InProject", "block storage quota", err)
		}
	}
	return &ProjectQuotaScope{api: a, projectID: id}, nil
}

// CurrentProject binds the scope from the recorded Keystone authentication
// result. Manual tokens, system/domain scopes and missing auth results cannot
// identify a project; the endpoint URL and token string are never guessed.
func (a *API) CurrentProject(ctx context.Context) (*ProjectQuotaScope, error) {
	if err := a.validateQuotaClient(ctx); err != nil {
		return nil, request.Wrap("CurrentProject", "block storage quota", err)
	}
	auth := a.client.ProviderClient.GetAuthResult()
	if auth == nil || reflect.ValueOf(auth).Kind() == reflect.Pointer && reflect.ValueOf(auth).IsNil() {
		return nil, request.Wrap("CurrentProject", "block storage quota", fmt.Errorf("%w: no project authentication result; supply an explicit project ID", resource.ErrUnsupported))
	}
	var id string
	switch result := auth.(type) {
	case interface {
		ExtractProject() (*tokens3.Project, error)
	}:
		project, err := result.ExtractProject()
		if err != nil {
			return nil, request.Wrap("CurrentProject", "block storage quota", err)
		}
		if project != nil {
			id = project.ID
		}
	case interface {
		ExtractToken() (*tokens2.Token, error)
	}:
		token, err := result.ExtractToken()
		if err != nil {
			return nil, request.Wrap("CurrentProject", "block storage quota", err)
		}
		if token != nil {
			id = token.Tenant.ID
		}
	default:
		return nil, request.Wrap("CurrentProject", "block storage quota", fmt.Errorf("%w: authentication result %T does not expose a Keystone project", resource.ErrUnsupported, auth))
	}
	if id == "" {
		return nil, request.Wrap("CurrentProject", "block storage quota", fmt.Errorf("%w: authentication is not project-scoped; supply an explicit project ID", resource.ErrUnsupported))
	}
	return a.InProject(ctx, resource.ID(id))
}

// ProjectID returns the resolved project without a request.
func (s *ProjectQuotaScope) ProjectID() string { return s.projectID }

func (a *API) validateQuotaClient(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if a == nil || a.client == nil || a.client.ProviderClient == nil {
		return fmt.Errorf("%w: block storage quotas require a service client", resource.ErrInvalidOption)
	}
	return nil
}

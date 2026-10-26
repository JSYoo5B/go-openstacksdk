package openstack

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/projectcleanup"
	"github.com/gophercloud/gophercloud/v2"
	v2 "github.com/gophercloud/gophercloud/v2/openstack/identity/v2/tokens"
	v3 "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
)

// PlanProjectCleanup discovers the resources of the current token's project
// that project_cleanup would delete, without sending any mutation. A plan
// without WithCreatedBefore, WithUpdatedBefore or WithSelector requires
// WithAllResources. A discovery failure of one service returns the partial
// plan together with an error naming the service. See the projectcleanup
// package for coverage, exclusions and ordering.
func (c *Connection) PlanProjectCleanup(ctx context.Context, options ...projectcleanup.PlanOption) (*projectcleanup.Plan, error) {
	if c == nil || c.provider == nil {
		return nil, invalid("Connection is required")
	}
	return projectcleanup.Discover(ctx, c.projectCleanupDependencies(), options...)
}

// ExecuteProjectCleanup deletes exactly the candidates of a plan from
// PlanProjectCleanup, in stage order, without discovering again. It refuses a
// modified plan, a plan with discovery errors (unless WithAllowIncompletePlan)
// and a Connection whose current project differs from the plan's. The Result
// has one Outcome per candidate; any Failed or Uncertain outcome also returns
// an error. Cancellation stops new requests only.
func (c *Connection) ExecuteProjectCleanup(ctx context.Context, plan *projectcleanup.Plan, options ...projectcleanup.ExecuteOption) (*projectcleanup.Result, error) {
	if c == nil || c.provider == nil {
		return nil, invalid("Connection is required")
	}
	return projectcleanup.Execute(ctx, c.projectCleanupDependencies(), plan, options...)
}

func (c *Connection) projectCleanupDependencies() projectcleanup.Dependencies {
	return projectcleanup.Dependencies{
		Project:        func(context.Context) (projectcleanup.Project, error) { return recordedProject(c.provider) },
		ServiceEnabled: c.ServiceEnabled,
		Endpoint: func(serviceType string) (bool, error) {
			_, ok, err := c.EndpointFor(serviceType)
			return ok, err
		},
		Client: c.projectCleanupClient,
	}
}

func (c *Connection) projectCleanupClient(ctx context.Context, service string) (*gophercloud.ServiceClient, error) {
	type raw interface {
		RawClient() *gophercloud.ServiceClient
	}
	var value raw
	var err error
	switch service {
	case "compute":
		value, err = c.Compute(ctx)
	case "block_storage":
		value, err = c.BlockStorage(ctx)
	case "network":
		value, err = c.Network(ctx)
	case "image":
		value, err = c.Image(ctx)
	case "object_store":
		value, err = c.ObjectStorage(ctx)
	case "key_manager":
		value, err = c.KeyManager(ctx)
	default:
		return nil, unsupported("project cleanup", service)
	}
	if err != nil {
		return nil, err
	}
	return value.RawClient(), nil
}

// recordedProject reads the project scope of the recorded Keystone token,
// never a configured or guessed project.
func recordedProject(provider *gophercloud.ProviderClient) (projectcleanup.Project, error) {
	var project projectcleanup.Project
	fromV3 := func(extract func() (*v3.Project, error)) error {
		value, err := extract()
		if err != nil {
			return err
		}
		if value != nil {
			project = projectcleanup.Project{ID: value.ID, Name: value.Name, DomainID: value.Domain.ID, DomainName: value.Domain.Name}
		}
		return nil
	}
	fromV2 := func(result v2.CreateResult) error {
		token, err := result.ExtractToken()
		if err != nil {
			return err
		}
		if token != nil {
			project = projectcleanup.Project{ID: token.Tenant.ID, Name: token.Tenant.Name}
		}
		return nil
	}
	var err error
	switch result := provider.GetAuthResult().(type) {
	case v3.CreateResult:
		err = fromV3(result.ExtractProject)
	case *v3.CreateResult:
		if result != nil {
			err = fromV3(result.ExtractProject)
		}
	case v3.GetResult:
		err = fromV3(result.ExtractProject)
	case *v3.GetResult:
		if result != nil {
			err = fromV3(result.ExtractProject)
		}
	case v2.CreateResult:
		err = fromV2(result)
	case *v2.CreateResult:
		if result != nil {
			err = fromV2(*result)
		}
	case v2.GetResult:
		err = fromV2(result.CreateResult)
	case *v2.GetResult:
		if result != nil {
			err = fromV2(result.CreateResult)
		}
	}
	if err != nil {
		return projectcleanup.Project{}, fmt.Errorf("recorded authentication project: %w", err)
	}
	if project.ID == "" {
		return projectcleanup.Project{}, projectcleanup.ErrUnscoped
	}
	return project, nil
}

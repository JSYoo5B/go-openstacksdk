package openstack

import (
	"context"
	"fmt"
	"slices"

	blockquotas "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets"
	"github.com/JSYoo5B/go-openstacksdk/identity"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/projectcleanup"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"

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
	case "orchestration":
		value, err = c.Orchestration(ctx)
	case "load_balancer":
		value, err = c.LoadBalancer(ctx)
	case "dns":
		value, err = c.DNS(ctx)
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

// findQuotaProjectID resolves a project identity the way a cloud helper does:
// one Keystone v3 lookup that tries the identity as an ID first and falls back
// to an exact name search, and fails when no project matches. A lookup is made
// even for strings that already are project IDs, so a missing project is
// reported before any quota request is sent.
func (c *Connection) findQuotaProjectID(ctx context.Context, operation, nameOrID string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", request.Wrap(operation, "project quota", err)
	}
	identity, err := c.IdentityV3(ctx)
	if err != nil {
		return "", request.Wrap(operation, "project quota", err)
	}
	found, err := identity.Projects.FindIdentity(ctx, nameOrID)
	if err != nil {
		return "", request.Wrap(operation, "project quota", err)
	}
	if found == nil || found.ID == "" {
		return "", request.Wrap(operation, "project quota", fmt.Errorf("%w: project %q has no ID", resource.ErrNotFound, nameOrID))
	}
	return found.ID, nil
}

// FindProjectComputeQuotas binds Nova quotas to the project selected by a
// Keystone ID-or-name lookup. Unlike ProjectQuotas it always performs the
// lookup and accepts an unqualified identity string; a missing project is an
// error and no Nova request is made.
func (c *Connection) FindProjectComputeQuotas(ctx context.Context, nameOrID string) (*quotasets.ProjectQuotaScope, error) {
	id, err := c.findQuotaProjectID(ctx, "FindProjectComputeQuotas", nameOrID)
	if err != nil {
		return nil, err
	}
	return c.ProjectQuotas(ctx, resource.ID(id))
}

// GetComputeQuotas looks up the project once and reads its Nova quota set.
func (c *Connection) GetComputeQuotas(ctx context.Context, nameOrID string) (*quotasets.QuotaResource, error) {
	scope, err := c.FindProjectComputeQuotas(ctx, nameOrID)
	if err != nil {
		return nil, err
	}
	return scope.Get(ctx)
}

// SetComputeQuotas looks up the project once and updates its Nova quotas with
// force=true, as the cloud helper does. The forced value is applied after the
// caller's options and replaces an earlier WithUpdateForce choice; the concrete
// UpdateOpts still decides which limits are sent.
func (c *Connection) SetComputeQuotas(ctx context.Context, nameOrID string, opts quotasets.UpdateOpts, options ...quotasets.UpdateOption) (*quotasets.QuotaResource, error) {
	scope, err := c.FindProjectComputeQuotas(ctx, nameOrID)
	if err != nil {
		return nil, err
	}
	forced := append(append([]quotasets.UpdateOption(nil), options...), quotasets.WithUpdateForce(true))
	return scope.Update(ctx, opts, forced...)
}

// DeleteComputeQuotas looks up the project once and resets its Nova quotas.
func (c *Connection) DeleteComputeQuotas(ctx context.Context, nameOrID string, options ...quotasets.ResetOption) (*quotasets.ResetResponse, error) {
	scope, err := c.FindProjectComputeQuotas(ctx, nameOrID)
	if err != nil {
		return nil, err
	}
	return scope.Reset(ctx, options...)
}

// FindProjectBlockStorageQuotas binds Cinder quotas to the project selected by
// a Keystone ID-or-name lookup (see FindProjectComputeQuotas).
func (c *Connection) FindProjectBlockStorageQuotas(ctx context.Context, nameOrID string) (*blockquotas.ProjectQuotaScope, error) {
	id, err := c.findQuotaProjectID(ctx, "FindProjectBlockStorageQuotas", nameOrID)
	if err != nil {
		return nil, err
	}
	return c.BlockStorageProjectQuotas(ctx, resource.ID(id))
}

// GetBlockStorageQuotas looks up the project once and reads its Cinder quota set.
func (c *Connection) GetBlockStorageQuotas(ctx context.Context, nameOrID string) (*blockquotas.QuotaResource, error) {
	scope, err := c.FindProjectBlockStorageQuotas(ctx, nameOrID)
	if err != nil {
		return nil, err
	}
	return scope.Get(ctx)
}

// SetBlockStorageQuotas looks up the project once and updates its Cinder quotas.
// No force flag is added: Cinder has none.
func (c *Connection) SetBlockStorageQuotas(ctx context.Context, nameOrID string, opts blockquotas.UpdateOpts, options ...blockquotas.UpdateOption) (*blockquotas.QuotaResource, error) {
	scope, err := c.FindProjectBlockStorageQuotas(ctx, nameOrID)
	if err != nil {
		return nil, err
	}
	return scope.Update(ctx, opts, options...)
}

// DeleteBlockStorageQuotas looks up the project once and resets its Cinder quotas.
func (c *Connection) DeleteBlockStorageQuotas(ctx context.Context, nameOrID string, options ...blockquotas.ResetOption) (*blockquotas.ResetResponse, error) {
	scope, err := c.FindProjectBlockStorageQuotas(ctx, nameOrID)
	if err != nil {
		return nil, err
	}
	return scope.Reset(ctx, options...)
}

// projectReadFacade binds the cached Keystone v3 Projects API and adds captured
// Connection facts after caller options, while preserving a per-call location.
func (c *Connection) projectReadFacade(ctx context.Context, options []identity.ProjectReadOption) (*identity.ProjectReadCloud, []identity.ProjectReadOption, error) {
	if err := c.recordListContext(ctx); err != nil {
		return nil, nil, err
	}
	service, err := c.IdentityV3(ctx)
	if err != nil {
		return nil, nil, err
	}
	location, err := c.CurrentLocation()
	if err != nil {
		return nil, nil, err
	}
	if _, err := location.ForResource(nil, nil); err != nil {
		return nil, nil, err
	}
	owned := append(slices.Clone(options), func(cfg *identity.ProjectReadOpts) error {
		if cfg.Location == nil {
			copy := location.Clone()
			cfg.Location = &copy
		}
		return nil
	})
	return identity.NewProjectReadCloud(service.Projects), owned, nil
}

// ListProjects lists projects visible to the selected system/domain reader.
// It keeps Source list_projects' server name/domain argument precedence and
// polymorphic Proxy list results, using the cached Keystone v3 service.
func (c *Connection) ListProjects(ctx context.Context, options ...identity.ProjectReadOption) (*identity.ProjectListResult, error) {
	facade, owned, err := c.projectReadFacade(ctx, options)
	if err == nil {
		var result *identity.ProjectListResult
		result, err = facade.ListProjects(ctx, owned...)
		return result, request.Wrap("ListProjects", "projects", cloudread.ContextError(ctx, err))
	}
	return nil, request.Wrap("ListProjects", "projects", cloudread.ContextError(ctx, err))
}

// SearchProjects supplies Connection location before local exact/glob and
// nested mapping selection. Keystone v3 absence remains a configuration error.
func (c *Connection) SearchProjects(ctx context.Context, nameOrID string, options ...identity.ProjectReadOption) (*identity.ProjectSearchResult, error) {
	facade, owned, err := c.projectReadFacade(ctx, options)
	if err == nil {
		var result *identity.ProjectSearchResult
		result, err = facade.SearchProjects(ctx, nameOrID, owned...)
		return result, request.Wrap("SearchProjects", "projects", cloudread.ContextError(ctx, err))
	}
	return nil, request.Wrap("SearchProjects", "projects", cloudread.ContextError(ctx, err))
}

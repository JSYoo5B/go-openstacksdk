package limits

import (
	"context"
	"fmt"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// WithGetReserved uses Nova's integer reserved query: 0 excludes reservations,
// 1 includes them. It works with Fetch, scoped Get and the native generated Get.
func WithGetReserved(include bool) GetOption {
	value := "0"
	if include {
		value = "1"
	}
	return WithGetQuery("reserved", value)
}

// prepareQuery fixes a single explicit tenant_id. Scoped calls do not accept
// request options that replace their project; other service query extensions
// keep their native values, including reserved's integer/server semantics.
func prepareQuery(fixedProject string, options ...GetOption) (url.Values, string, error) {
	config, err := request.Apply(GetOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, true, false)
	}
	if err != nil {
		return nil, "", err
	}
	if _, exists := config.Query["project_id"]; exists {
		return nil, "", fmt.Errorf("%w: project_id is not a Nova limits wire query; use InProject or TenantID", resource.ErrInvalidOption)
	}
	tenant, queryTenant := config.Query["tenant_id"]
	if fixedProject != "" && (config.Options.TenantID != "" || queryTenant) {
		return nil, "", fmt.Errorf("%w: request options cannot replace a fixed limits project", resource.ErrInvalidOption)
	}
	if queryTenant && (len(tenant) != 1 || config.Options.TenantID != "") {
		return nil, "", fmt.Errorf("%w: limits requires one unambiguous tenant_id", resource.ErrInvalidOption)
	}
	projectID := fixedProject
	if projectID == "" {
		projectID = config.Options.TenantID
		if queryTenant {
			projectID = tenant[0]
		}
		if projectID != "" || queryTenant {
			if err := resource.ID(projectID).Validate(); err != nil {
				return nil, "", err
			}
		}
	}
	query := make(url.Values, len(config.Query)+1)
	for key, values := range config.Query {
		query[key] = append([]string(nil), values...)
	}
	if projectID != "" {
		query.Set("tenant_id", projectID)
	}
	return query, projectID, nil
}

// Fetch reads Nova's limits singleton, retaining all response fields. Without
// an explicit tenant query it uses the server's current-project behavior and
// leaves ProjectID empty; CurrentProject binds recorded authentication instead.
func (a *API) Fetch(ctx context.Context, options ...GetOption) (*LimitsResource, error) {
	if err := a.validateClient(ctx); err != nil {
		return nil, limitsError("Fetch", "", err)
	}
	query, projectID, err := prepareQuery("", options...)
	if err != nil {
		return nil, limitsError("Fetch", "", err)
	}
	value, err := a.fetchLimits(ctx, query, projectID)
	return value, limitsError("Fetch", projectID, err)
}

// Get reads limits for the fixed project using Nova's admin-only tenant_id
// query. It never drops that query or falls back to the authenticated project.
func (s *ProjectLimitsScope) Get(ctx context.Context, options ...GetOption) (*LimitsResource, error) {
	if err := s.validate(ctx); err != nil {
		return nil, limitsError("Get", s.ProjectID(), err)
	}
	query, _, err := prepareQuery(s.projectID, options...)
	if err != nil {
		return nil, limitsError("Get", s.projectID, err)
	}
	value, err := s.api.fetchLimits(ctx, query, s.projectID)
	return value, limitsError("Get", s.projectID, err)
}

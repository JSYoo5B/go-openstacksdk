package limits

import (
	"context"
	"fmt"
	"net/url"

	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

type GetOpts struct {
	ProjectID string `q:"project_id"`
}
type GetOption = request.Option[GetOpts]

func WithGetOptions(value GetOpts) GetOption   { return request.WithOptions(value) }
func WithGetQuery(key, value string) GetOption { return request.WithQuery[GetOpts](key, value) }

func prepareQuery(fixedProject string, options ...GetOption) (url.Values, string, error) {
	config, err := request.Apply(GetOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, true, false)
	}
	if err != nil {
		return nil, "", err
	}
	if _, exists := config.Query["tenant_id"]; exists {
		return nil, "", fmt.Errorf("%w: Cinder limits uses project_id, not tenant_id", resource.ErrInvalidOption)
	}
	values, queryProject := config.Query["project_id"]
	if fixedProject != "" && (config.Options.ProjectID != "" || queryProject) {
		return nil, "", fmt.Errorf("%w: options cannot replace a fixed limits project", resource.ErrInvalidOption)
	}
	if queryProject && (len(values) != 1 || config.Options.ProjectID != "") {
		return nil, "", fmt.Errorf("%w: Cinder limits requires one unambiguous project_id", resource.ErrInvalidOption)
	}
	projectID := fixedProject
	if projectID == "" {
		projectID = config.Options.ProjectID
		if queryProject {
			projectID = values[0]
		}
		if projectID != "" || queryProject {
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
		query.Set("project_id", projectID)
	}
	return query, projectID, nil
}

// Fetch preserves Cinder's implicit authenticated-project behavior when no
// filter is supplied. ProjectID is then empty, rather than guessed or refreshed.
// An explicit project filter requires the caller's selected version to be 3.39+.
func (a *API) Fetch(ctx context.Context, options ...GetOption) (*LimitsResource, error) {
	if err := a.validateClient(ctx); err != nil {
		return nil, limitsError("Fetch", "", err)
	}
	query, projectID, err := prepareQuery("", options...)
	if err != nil {
		return nil, limitsError("Fetch", "", err)
	}
	if projectID != "" {
		if err := a.requireProjectFilter(); err != nil {
			return nil, limitsError("Fetch", projectID, err)
		}
	}
	value, err := a.fetchLimits(ctx, query, projectID)
	return value, limitsError("Fetch", projectID, err)
}

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

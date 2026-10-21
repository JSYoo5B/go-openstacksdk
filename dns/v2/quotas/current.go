package quotas

import (
	"context"
	"fmt"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/dns/v2/internal/designate"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type currentOptions struct {
	allProjects *bool
	sudoProject *string
}

// CurrentOption configures the project-less GET /quotas request.
type CurrentOption func(*currentOptions) error

// WithCurrentAllProjects sends X-Auth-All-Projects explicitly, including false.
// Python sends the header only for a truthy all_projects.
func WithCurrentAllProjects(enabled bool) CurrentOption {
	return func(o *currentOptions) error { o.allProjects = &enabled; return nil }
}

// WithCurrentSudoProject sends X-Auth-Sudo-Project-ID (Python project_id).
func WithCurrentSudoProject(projectID string) CurrentOption {
	return func(o *currentOptions) error {
		if err := resource.ID(projectID).Validate(); err != nil {
			return err
		}
		o.sudoProject = &projectID
		return nil
	}
}

// Current sends GET /quotas, the documented current-project quota object, and
// accepts only 200. This is Python quotas(): Designate returns one flat object,
// so there is no collection or paging; with a sudo project and all-projects
// header it reports that project's quota. ProjectID on the result is the sudo
// project when supplied and empty otherwise (the response carries no identity).
func (a *API) Current(ctx context.Context, options ...CurrentOption) (*QuotaResource, error) {
	var config currentOptions
	for _, apply := range options {
		if apply == nil {
			return nil, quotaError("Current", "", fmt.Errorf("%w: nil current option", resource.ErrInvalidOption))
		}
		if err := apply(&config); err != nil {
			return nil, quotaError("Current", "", err)
		}
	}
	if err := a.validateQuotaClient(ctx); err != nil {
		return nil, quotaError("Current", "", err)
	}
	sudo := ""
	if config.sudoProject != nil {
		sudo = *config.sudoProject
	}
	response, err := designate.Do(ctx, a.client, http.MethodGet, a.client.ServiceURL("quotas"), nil, designate.Headers(config.allProjects, sudo), http.StatusOK)
	if err != nil {
		return nil, quotaError("Current", sudo, err)
	}
	value, err := decodeQuota(quotaWire{body: response.Body, header: response.Header, status: response.StatusCode}, sudo)
	return value, quotaError("Current", sudo, err)
}

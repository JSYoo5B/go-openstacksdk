package quotasets

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/users"
	"github.com/JSYoo5B/go-openstacksdk/internal/project"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// UserQuotaScope fixes both the project and user. It has no Defaults method
// and cannot promote a project reset by embedding ProjectQuotaScope.
type UserQuotaScope struct{ scope quotaBinding }

func (s *ProjectQuotaScope) InUser(ctx context.Context, ref resource.Ref, options ...ProjectOption) (*UserQuotaScope, error) {
	if err := s.validate(ctx); err != nil {
		return nil, quotaError("InUser", s.ProjectID(), err)
	}
	if err := ref.Validate(); err != nil {
		return nil, quotaError("InUser", s.projectID, err)
	}
	config := projectOptions{identity: s.identity}
	for _, apply := range options {
		if apply == nil {
			return nil, quotaError("InUser", s.projectID, fmt.Errorf("%w: nil user resolution option", resource.ErrInvalidOption))
		}
		if err := apply(&config); err != nil {
			return nil, quotaError("InUser", s.projectID, err)
		}
	}
	id := ref.String()
	if ref.IsName() {
		if config.identity == nil {
			return nil, quotaError("InUser", s.projectID, fmt.Errorf("%w: user names require an Identity v3 client", resource.ErrUnsupported))
		}
		if err := project.ValidateIdentityClient(config.identity); err != nil {
			return nil, quotaError("InUser", s.projectID, err)
		}
		if config.identity == s.api.client {
			return nil, quotaError("InUser", s.projectID, fmt.Errorf("%w: Manila cannot resolve Keystone user names", resource.ErrInvalidOption))
		}
		var err error
		id, err = users.New(config.identity).Resources.ResolveID(ctx, ref)
		if err != nil {
			return nil, quotaError("InUser", s.projectID, err)
		}
	}
	return &UserQuotaScope{scope: quotaBinding{api: s.api, projectID: s.projectID, selector: "user_id", id: id}}, nil
}

func (s *UserQuotaScope) binding() quotaBinding {
	if s == nil {
		return quotaBinding{}
	}
	return s.scope
}
func (s *UserQuotaScope) ProjectID() string { return s.binding().projectID }
func (s *UserQuotaScope) UserID() string    { return s.binding().id }
func (s *UserQuotaScope) Get(ctx context.Context) (*QuotaResource, error) {
	return s.binding().get(ctx)
}
func (s *UserQuotaScope) Detail(ctx context.Context) (*QuotaDetailResource, error) {
	return s.binding().detail(ctx)
}
func (s *UserQuotaScope) Update(ctx context.Context, opts UpdateOpts, options ...UpdateOption) (*QuotaResource, error) {
	return s.binding().update(ctx, opts, options...)
}
func (s *UserQuotaScope) Reset(ctx context.Context, options ...ResetOption) (*ResetResponse, error) {
	return s.binding().reset(ctx, options...)
}

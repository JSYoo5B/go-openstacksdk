package quotasets

import (
	"context"

	"gophercloudsdk/resource"
	"gophercloudsdk/sharedfilesystems/v2/sharetypes"
)

// ShareTypeQuotaScope fixes project/share-type IDs, using only share_type as
// the query selector. Manila makes this mutually exclusive with user_id.
type ShareTypeQuotaScope struct{ scope quotaBinding }

// InShareType validates API 2.39 before any name lookup. IDs require no HTTP;
// exact names use this Manila service's existing share-type resource collection.
func (s *ProjectQuotaScope) InShareType(ctx context.Context, ref resource.Ref) (*ShareTypeQuotaScope, error) {
	if err := s.validate(ctx); err != nil {
		return nil, quotaError("InShareType", s.ProjectID(), err)
	}
	if err := s.api.requireVersion(39); err != nil {
		return nil, quotaError("InShareType", s.projectID, err)
	}
	if err := ref.Validate(); err != nil {
		return nil, quotaError("InShareType", s.projectID, err)
	}
	id, err := sharetypes.New(s.api.client).Resources.ResolveID(ctx, ref)
	if err != nil {
		return nil, quotaError("InShareType", s.projectID, err)
	}
	return &ShareTypeQuotaScope{scope: quotaBinding{api: s.api, projectID: s.projectID, selector: "share_type", id: id}}, nil
}

func (s *ShareTypeQuotaScope) binding() quotaBinding {
	if s == nil {
		return quotaBinding{}
	}
	return s.scope
}
func (s *ShareTypeQuotaScope) ProjectID() string   { return s.binding().projectID }
func (s *ShareTypeQuotaScope) ShareTypeID() string { return s.binding().id }
func (s *ShareTypeQuotaScope) Get(ctx context.Context) (*QuotaResource, error) {
	return s.binding().get(ctx)
}
func (s *ShareTypeQuotaScope) Detail(ctx context.Context) (*QuotaDetailResource, error) {
	return s.binding().detail(ctx)
}
func (s *ShareTypeQuotaScope) Update(ctx context.Context, opts UpdateOpts, options ...UpdateOption) (*QuotaResource, error) {
	return s.binding().update(ctx, opts, options...)
}
func (s *ShareTypeQuotaScope) Reset(ctx context.Context, options ...ResetOption) (*ResetResponse, error) {
	return s.binding().reset(ctx, options...)
}

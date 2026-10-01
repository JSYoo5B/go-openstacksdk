package quotasets

import "context"

// Defaults fetches the server-provided defaults for this project without
// changing its quota overrides. Nova does not define user_id for this endpoint.
func (s *ProjectQuotaScope) Defaults(ctx context.Context) (*QuotaResource, error) {
	if err := s.api.validateQuotaClient(ctx); err != nil {
		return nil, quotaError("Defaults", s.projectID, err)
	}
	result := s.api.getQuotaResponse(ctx, s.projectID, "", "defaults")
	value, err := decodeQuota(result, s.projectID)
	return value, quotaError("Defaults", s.projectID, err)
}

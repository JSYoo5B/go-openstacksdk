package quotas

import "context"

// Defaults fetches the project's default limits, independently of its quota
// overrides. Neutron returns these from /quotas/{project}/default, not Get.
func (s *ProjectQuotaScope) Defaults(ctx context.Context) (*QuotaResource, error) {
	if err := s.api.validateQuotaClient(ctx); err != nil {
		return nil, quotaError("Defaults", s.projectID, err)
	}
	result, statusCode := s.api.getQuota(ctx, s.api.client.ServiceURL("quotas", s.projectID, "default"))
	value, err := decodeQuota(result, s.projectID, statusCode)
	return value, quotaError("Defaults", s.projectID, err)
}

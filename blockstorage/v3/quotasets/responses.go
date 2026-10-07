package quotasets

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/quotasets"
)

// QuotaResource preserves native limits and every quota_set response field.
// ProjectID is the fixed request target; ID is the server's response value.
type QuotaResource struct {
	QuotaSet
	ProjectID string
	Body      map[string]json.RawMessage
	Header    http.Header
}

// QuotaUsageResource retains limit/in-use/reserved/allocated values and raw usage
// fields, including extensions whose quota name or nested values are unknown.
type QuotaUsageResource struct {
	QuotaUsageSet
	ProjectID string
	Body      map[string]json.RawMessage
	Header    http.Header
}

// ResetResponse retains the reset response headers. Reset returns no quota
// object and does not automatically fetch defaults or the resulting limits.
type ResetResponse struct {
	ProjectID string
	Header    http.Header
}

func quotaObject(result gophercloud.Result) (json.RawMessage, map[string]json.RawMessage, error) {
	var envelope struct {
		Quota json.RawMessage `json:"quota_set"`
	}
	if err := result.ExtractInto(&envelope); err != nil {
		return nil, nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(envelope.Quota, &fields); err != nil {
		return nil, nil, err
	}
	if fields == nil {
		return nil, nil, gophercloud.ErrUnexpectedType{Expected: "quota_set JSON object", Actual: "null"}
	}
	return envelope.Quota, fields, nil
}

func decodeQuota(result gophercloud.Result, projectID string) (*QuotaResource, error) {
	body, fields, err := quotaObject(result)
	if err != nil {
		return nil, err
	}
	var quota QuotaSet
	if err := json.Unmarshal(body, &quota); err != nil {
		return nil, err
	}
	return &QuotaResource{QuotaSet: quota, ProjectID: projectID, Body: fields, Header: result.Header.Clone()}, nil
}

func quotaError(operation, projectID string, err error) error {
	if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		err = &resource.NotFoundError{Resource: "block storage quota", Reference: projectID, Cause: err}
	}
	return request.Wrap(operation, "block storage quota", err)
}

// Get fetches the project's limits, preserving response extensions and headers.
func (s *ProjectQuotaScope) Get(ctx context.Context) (*QuotaResource, error) {
	if err := s.api.validateQuotaClient(ctx); err != nil {
		return nil, quotaError("Get", s.projectID, err)
	}
	result := upstream.Get(ctx, s.api.client, s.projectID)
	value, err := decodeQuota(result.Result, s.projectID)
	return value, quotaError("Get", s.projectID, err)
}

// Usage fetches limits, usage, reservations and nested-quota allocations.
func (s *ProjectQuotaScope) Usage(ctx context.Context) (*QuotaUsageResource, error) {
	if err := s.api.validateQuotaClient(ctx); err != nil {
		return nil, quotaError("Usage", s.projectID, err)
	}
	result := upstream.GetUsage(ctx, s.api.client, s.projectID)
	body, fields, err := quotaObject(result.Result)
	if err != nil {
		return nil, quotaError("Usage", s.projectID, err)
	}
	var quota QuotaUsageSet
	if err := json.Unmarshal(body, &quota); err != nil {
		return nil, quotaError("Usage", s.projectID, err)
	}
	return &QuotaUsageResource{QuotaUsageSet: quota, ProjectID: s.projectID, Body: fields, Header: result.Header.Clone()}, nil
}

// Defaults fetches the project's server-provided defaults without changing its
// configured quota overrides. It uses Cinder's /defaults endpoint.
func (s *ProjectQuotaScope) Defaults(ctx context.Context) (*QuotaResource, error) {
	if err := s.api.validateQuotaClient(ctx); err != nil {
		return nil, quotaError("Defaults", s.projectID, err)
	}
	result := upstream.GetDefaults(ctx, s.api.client, s.projectID)
	value, err := decodeQuota(result.Result, s.projectID)
	return value, quotaError("Defaults", s.projectID, err)
}

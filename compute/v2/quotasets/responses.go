package quotasets

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/quotasets"
)

// QuotaResource preserves native limits and every quota_set response field.
// ProjectID is the fixed request target; ID is the server's response value.
type QuotaResource struct {
	QuotaSet
	ProjectID string
	// UserID is set only for an explicitly bound UserQuotaScope.
	UserID string
	Body   map[string]json.RawMessage
	Header http.Header
}

// QuotaDetailResource retains limit/in-use/reserved values and raw detail
// fields, including extensions whose quota name or nested values are unknown.
type QuotaDetailResource struct {
	QuotaDetailSet
	ProjectID string
	// UserID is set only for an explicitly bound UserQuotaScope.
	UserID string
	Body   map[string]json.RawMessage
	Header http.Header
}

// ResetResponse retains the reset response headers. Reset returns no quota
// object and does not automatically fetch defaults or the resulting limits.
type ResetResponse struct {
	ProjectID string
	// UserID is set only for an explicitly bound UserQuotaScope.
	UserID string
	Header http.Header
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
		err = &resource.NotFoundError{Resource: "compute quota", Reference: projectID, Cause: err}
	}
	return request.Wrap(operation, "compute quota", err)
}

func (a *API) quotaURL(projectID, userID, suffix string) string {
	parts := []string{"os-quota-sets", projectID}
	if suffix != "" {
		parts = append(parts, suffix)
	}
	endpoint := a.client.ServiceURL(parts...)
	if userID != "" {
		endpoint += "?" + url.Values{"user_id": []string{userID}}.Encode()
	}
	return endpoint
}

func (a *API) getQuotaResponse(ctx context.Context, projectID, userID, suffix string) (result gophercloud.Result) {
	response, err := a.client.Get(ctx, a.quotaURL(projectID, userID, suffix), &result.Body, &gophercloud.RequestOpts{OkCodes: []int{http.StatusOK}})
	_, result.Header, result.Err = gophercloud.ParseResponse(response, err)
	return
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

// Detail fetches limit, in-use and reserved values from Nova's detail endpoint.
func (s *ProjectQuotaScope) Detail(ctx context.Context) (*QuotaDetailResource, error) {
	if err := s.api.validateQuotaClient(ctx); err != nil {
		return nil, quotaError("Detail", s.projectID, err)
	}
	result := upstream.GetDetail(ctx, s.api.client, s.projectID)
	body, fields, err := quotaObject(result.Result)
	if err != nil {
		return nil, quotaError("Detail", s.projectID, err)
	}
	var quota QuotaDetailSet
	if err := json.Unmarshal(body, &quota); err != nil {
		return nil, quotaError("Detail", s.projectID, err)
	}
	return &QuotaDetailResource{QuotaDetailSet: quota, ProjectID: s.projectID, Body: fields, Header: result.Header.Clone()}, nil
}

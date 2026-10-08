package quotas

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// QuotaResource retains native limits, raw quota fields and response headers.
// A scope's ProjectID remains its request target even if the response names
// another ID. ListProjects instead uses the validated project identity in the row.
type QuotaResource struct {
	Quota
	ProjectID  string
	Body       map[string]json.RawMessage
	Header     http.Header
	StatusCode int
}

// QuotaDetailResource retains native Used/Reserved/Limit values and raw detail
// fields, including unknown extensions and reserved values returned as strings.
type QuotaDetailResource struct {
	QuotaDetailSet
	ProjectID  string
	Body       map[string]json.RawMessage
	Header     http.Header
	StatusCode int
}

// DeleteResponse contains the reset response metadata; Delete does not fetch
// defaults or infer the resulting quota limits.
type DeleteResponse struct {
	ProjectID  string
	Header     http.Header
	StatusCode int
}

func quotaObject(result gophercloud.Result) (json.RawMessage, map[string]json.RawMessage, error) {
	var envelope struct {
		Quota json.RawMessage `json:"quota"`
	}
	if err := result.ExtractInto(&envelope); err != nil {
		return nil, nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(envelope.Quota, &fields); err != nil {
		return nil, nil, err
	}
	if fields == nil {
		return nil, nil, gophercloud.ErrUnexpectedType{Expected: "quota JSON object", Actual: "null"}
	}
	return envelope.Quota, fields, nil
}

func decodeQuota(result gophercloud.Result, projectID string, statusCode int) (*QuotaResource, error) {
	body, fields, err := quotaObject(result)
	if err != nil {
		return nil, err
	}
	var quota Quota
	if err := json.Unmarshal(body, &quota); err != nil {
		return nil, err
	}
	return &QuotaResource{Quota: quota, ProjectID: projectID, Body: fields, Header: result.Header.Clone(), StatusCode: statusCode}, nil
}

// RawMessage avoids a float64 round trip through the native result's any Body,
// preserving extension integers without depending on the native JSON decoder.
func (a *API) getQuota(ctx context.Context, endpoint string) (gophercloud.Result, int) {
	var result gophercloud.Result
	var raw json.RawMessage
	response, err := a.client.Get(ctx, endpoint, &raw, nil)
	result.Body = raw
	_, result.Header, result.Err = gophercloud.ParseResponse(response, err)
	statusCode := 0
	if response != nil {
		statusCode = response.StatusCode
	}
	return result, statusCode
}

func quotaError(operation, projectID string, err error) error {
	if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		err = &resource.NotFoundError{Resource: "network quota", Reference: projectID, Cause: err}
	}
	return request.Wrap(operation, "network quota", err)
}

// Get fetches the project's quota limits using the native member endpoint.
func (s *ProjectQuotaScope) Get(ctx context.Context) (*QuotaResource, error) {
	if err := s.api.validateQuotaClient(ctx); err != nil {
		return nil, quotaError("Get", s.projectID, err)
	}
	result, statusCode := s.api.getQuota(ctx, s.api.client.ServiceURL("quotas", s.projectID))
	value, err := decodeQuota(result, s.projectID, statusCode)
	return value, quotaError("Get", s.projectID, err)
}

// Detail fetches quota usage from /quotas/{project}/details.json. The server
// determines whether the quota_details extension is available.
func (s *ProjectQuotaScope) Detail(ctx context.Context) (*QuotaDetailResource, error) {
	if err := s.api.validateQuotaClient(ctx); err != nil {
		return nil, quotaError("Detail", s.projectID, err)
	}
	result, statusCode := s.api.getQuota(ctx, s.api.client.ServiceURL("quotas", s.projectID, "details.json"))
	body, fields, err := quotaObject(result)
	if err != nil {
		return nil, quotaError("Detail", s.projectID, err)
	}
	var quota QuotaDetailSet
	if err := json.Unmarshal(body, &quota); err != nil {
		return nil, quotaError("Detail", s.projectID, err)
	}
	return &QuotaDetailResource{QuotaDetailSet: quota, ProjectID: s.projectID, Body: fields, Header: result.Header.Clone(), StatusCode: statusCode}, nil
}

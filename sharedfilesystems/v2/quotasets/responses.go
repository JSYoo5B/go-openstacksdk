package quotasets

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/fixedrequest"
)

// QuotaSet uses exact int64 limits. Nil means omitted or JSON null; Body on the
// response distinguishes those cases and preserves every extension field.
type QuotaSet struct {
	ID                  string `json:"id"`
	Gigabytes           *int64 `json:"gigabytes,omitempty"`
	Snapshots           *int64 `json:"snapshots,omitempty"`
	Shares              *int64 `json:"shares,omitempty"`
	SnapshotGigabytes   *int64 `json:"snapshot_gigabytes,omitempty"`
	ShareGroups         *int64 `json:"share_groups,omitempty"`
	ShareGroupSnapshots *int64 `json:"share_group_snapshots,omitempty"`
	ShareNetworks       *int64 `json:"share_networks,omitempty"`
	ShareReplicas       *int64 `json:"share_replicas,omitempty"`
	ReplicaGigabytes    *int64 `json:"replica_gigabytes,omitempty"`
	PerShareGigabytes   *int64 `json:"per_share_gigabytes,omitempty"`
	Backups             *int64 `json:"backups,omitempty"`
	BackupGigabytes     *int64 `json:"backup_gigabytes,omitempty"`
}

type QuotaUsage struct {
	Limit    *int64 `json:"limit,omitempty"`
	InUse    *int64 `json:"in_use,omitempty"`
	Reserved *int64 `json:"reserved,omitempty"`
}

type QuotaDetailSet struct {
	ID                  string      `json:"id"`
	Gigabytes           *QuotaUsage `json:"gigabytes,omitempty"`
	Snapshots           *QuotaUsage `json:"snapshots,omitempty"`
	Shares              *QuotaUsage `json:"shares,omitempty"`
	SnapshotGigabytes   *QuotaUsage `json:"snapshot_gigabytes,omitempty"`
	ShareGroups         *QuotaUsage `json:"share_groups,omitempty"`
	ShareGroupSnapshots *QuotaUsage `json:"share_group_snapshots,omitempty"`
	ShareNetworks       *QuotaUsage `json:"share_networks,omitempty"`
	ShareReplicas       *QuotaUsage `json:"share_replicas,omitempty"`
	ReplicaGigabytes    *QuotaUsage `json:"replica_gigabytes,omitempty"`
	PerShareGigabytes   *QuotaUsage `json:"per_share_gigabytes,omitempty"`
	Backups             *QuotaUsage `json:"backups,omitempty"`
	BackupGigabytes     *QuotaUsage `json:"backup_gigabytes,omitempty"`
}

type QuotaResource struct {
	QuotaSet
	ProjectID   string
	UserID      string
	ShareTypeID string
	Body        map[string]json.RawMessage
	Header      http.Header
	StatusCode  int
}

type QuotaDetailResource struct {
	QuotaDetailSet
	ProjectID   string
	UserID      string
	ShareTypeID string
	Body        map[string]json.RawMessage
	Header      http.Header
	StatusCode  int
}

type ResetResponse struct {
	ProjectID   string
	UserID      string
	ShareTypeID string
	Header      http.Header
	StatusCode  int
}

type quotaResponse struct {
	body   map[string]json.RawMessage
	header http.Header
	status int
}

func (a *API) read(ctx context.Context, endpoint string) (quotaResponse, error) {
	client, err := fixedrequest.New(a.client, http.MethodGet, endpoint)
	if err != nil {
		return quotaResponse{}, err
	}
	var envelope map[string]json.RawMessage
	response, err := client.Get(ctx, endpoint, &envelope, &gophercloud.RequestOpts{OkCodes: []int{http.StatusOK}})
	return parseQuotaResponse(response, envelope, err)
}

func parseQuotaResponse(response *http.Response, envelope map[string]json.RawMessage, err error) (quotaResponse, error) {
	if err != nil {
		return quotaResponse{}, err
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(envelope["quota_set"], &body); err != nil {
		return quotaResponse{}, err
	}
	if body == nil {
		return quotaResponse{}, fmt.Errorf("quota_set must be a JSON object")
	}
	return quotaResponse{body: body, header: response.Header.Clone(), status: response.StatusCode}, nil
}

func decodeQuota(result quotaResponse, projectID string) (*QuotaResource, error) {
	data, err := json.Marshal(result.body)
	if err != nil {
		return nil, err
	}
	var quota QuotaSet
	if err := json.Unmarshal(data, &quota); err != nil {
		return nil, err
	}
	return &QuotaResource{QuotaSet: quota, ProjectID: projectID, Body: result.body, Header: result.header, StatusCode: result.status}, nil
}

func decodeDetail(result quotaResponse, projectID string) (*QuotaDetailResource, error) {
	data, err := json.Marshal(result.body)
	if err != nil {
		return nil, err
	}
	var quota QuotaDetailSet
	if err := json.Unmarshal(data, &quota); err != nil {
		return nil, err
	}
	return &QuotaDetailResource{QuotaDetailSet: quota, ProjectID: projectID, Body: result.body, Header: result.header, StatusCode: result.status}, nil
}

func (s *ProjectQuotaScope) get(ctx context.Context, operation, suffix string) (*QuotaResource, error) {
	if err := s.validate(ctx); err != nil {
		return nil, quotaError(operation, s.ProjectID(), err)
	}
	endpoint, err := s.api.quotaURL(s.projectID, suffix)
	if err != nil {
		return nil, quotaError(operation, s.projectID, err)
	}
	result, err := s.api.read(ctx, endpoint)
	if err != nil {
		return nil, quotaError(operation, s.projectID, err)
	}
	value, err := decodeQuota(result, s.projectID)
	return value, quotaError(operation, s.projectID, err)
}

func (s *ProjectQuotaScope) Get(ctx context.Context) (*QuotaResource, error) {
	return s.get(ctx, "Get", "")
}
func (s *ProjectQuotaScope) Defaults(ctx context.Context) (*QuotaResource, error) {
	return s.get(ctx, "Defaults", "defaults")
}

// Detail reads limit, in-use and reserved fields from Manila 2.25's /detail
// endpoint. The shared client's configured microversion is never upgraded.
func (s *ProjectQuotaScope) Detail(ctx context.Context) (*QuotaDetailResource, error) {
	if err := s.validate(ctx); err != nil {
		return nil, quotaError("Detail", s.ProjectID(), err)
	}
	if err := s.api.requireVersion(25); err != nil {
		return nil, quotaError("Detail", s.projectID, err)
	}
	endpoint, err := s.api.quotaURL(s.projectID, "detail")
	if err != nil {
		return nil, quotaError("Detail", s.projectID, err)
	}
	result, err := s.api.read(ctx, endpoint)
	if err != nil {
		return nil, quotaError("Detail", s.projectID, err)
	}
	value, err := decodeDetail(result, s.projectID)
	return value, quotaError("Detail", s.projectID, err)
}

package quotas

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func decodeProjectQuota(raw json.RawMessage, header http.Header, statusCode int) (*QuotaResource, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	if fields == nil {
		return nil, gophercloud.ErrUnexpectedType{Expected: "project quota JSON object", Actual: "null"}
	}
	projectID, err := projectQuotaIdentity(fields)
	if err != nil {
		return nil, err
	}
	var quota Quota
	if err := json.Unmarshal(raw, &quota); err != nil {
		return nil, err
	}
	return &QuotaResource{Quota: quota, ProjectID: projectID, Body: fields, Header: header.Clone(), StatusCode: statusCode}, nil
}

func projectQuotaIdentity(fields map[string]json.RawMessage) (string, error) {
	var projectID, tenantID string
	for _, field := range []struct {
		key    string
		target *string
	}{{"project_id", &projectID}, {"tenant_id", &tenantID}} {
		raw, present := fields[field.key]
		if !present {
			continue
		}
		if err := json.Unmarshal(raw, field.target); err != nil {
			return "", fmt.Errorf("quota %s: %w", field.key, err)
		}
		if err := resource.ID(*field.target).Validate(); err != nil {
			return "", fmt.Errorf("quota %s: %w", field.key, err)
		}
	}
	if projectID != "" && tenantID != "" && projectID != tenantID {
		return "", fmt.Errorf("%w: quota project_id and tenant_id disagree", resource.ErrInvalidOption)
	}
	if projectID != "" {
		return projectID, nil
	}
	if tenantID != "" {
		return tenantID, nil
	}
	return "", fmt.Errorf("%w: quota row has no project_id or tenant_id", resource.ErrInvalidOption)
}

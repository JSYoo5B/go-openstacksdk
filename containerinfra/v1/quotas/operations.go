package quotas

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

type QuotaUpdateOpts = QuotaCreateOpts
type QuotaUpdateOption = request.Option[QuotaUpdateOpts]

// WithQuotaUpdateOptions snapshots the hard limit at construction, replacing
// typed update attributes. WithHardLimit can also be used for an update.
func WithQuotaUpdateOptions(value QuotaUpdateOpts) QuotaUpdateOption {
	return WithQuotaCreateOptions(value)
}

func WithQuotaUpdateField(key string, value any) QuotaUpdateOption {
	return WithQuotaCreateField(key, value)
}

func (s *ResourceQuotaScope) validate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.api == nil {
		return fmt.Errorf("%w: quota scope is required", resource.ErrInvalidOption)
	}
	return s.api.validateQuotaClient(ctx)
}

func (s *ResourceQuotaScope) quotaEndpoint() string {
	return s.api.client.ServiceURL("quotas", s.projectID, string(s.resourceName))
}

// Get reads the fixed project/resource pair. Magnum may return its deployment
// default with no row ID or resource field when no explicit quota exists.
func (s *ResourceQuotaScope) Get(ctx context.Context) (*QuotaResource, error) {
	if err := s.validate(ctx); err != nil {
		return nil, quotaError("Get", s.ProjectID(), err)
	}
	wire, err := s.api.requestQuota(ctx, http.MethodGet, s.quotaEndpoint(), nil, []int{200})
	if err != nil {
		return nil, quotaError("Get", s.projectID, err)
	}
	value, err := decodeQuota(wire, s.projectID, s.resourceName)
	return value, quotaError("Get", s.projectID, err)
}

// Update uses Magnum's flat-object PATCH rather than JSON Patch operations.
// The hard limit is required, and identity is always taken from this scope.
func (s *ResourceQuotaScope) Update(ctx context.Context, options ...QuotaUpdateOption) (*QuotaResource, error) {
	if err := s.validate(ctx); err != nil {
		return nil, quotaError("Update", s.ProjectID(), err)
	}
	config, err := request.Apply(QuotaUpdateOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, true, false, false)
	}
	if err == nil && config.Options.HardLimit == nil {
		err = fmt.Errorf("%w: quota hard limit must be supplied explicitly", resource.ErrInvalidOption)
	}
	if err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	for _, key := range []string{"project_id", "resource", "id"} {
		if _, exists := config.Fields[key]; exists {
			return nil, quotaError("Update", s.projectID, fmt.Errorf("%w: extension %q is quota identity", resource.ErrInvalidOption, key))
		}
	}
	body, err := request.MergeFieldsFor(map[string]any{
		"project_id": s.projectID,
		"resource":   string(s.resourceName),
		"hard_limit": *config.Options.HardLimit,
	}, config.Fields, config.Options)
	if err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	wire, err := s.api.requestQuota(ctx, http.MethodPatch, s.quotaEndpoint(), encoded, []int{202})
	if err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	value, err := decodeQuota(wire, s.projectID, s.resourceName)
	return value, quotaError("Update", s.projectID, err)
}

type deleteOptions struct{ ignoreMissing bool }
type DeleteOption func(*deleteOptions) error

func WithDeleteIgnoreMissing(ignore bool) DeleteOption {
	return func(options *deleteOptions) error { options.ignoreMissing = ignore; return nil }
}

type DeleteResponse struct {
	ProjectID    string
	ResourceName ResourceName
	Header       http.Header
	StatusCode   int
}

// Delete removes the explicit quota for this project/resource pair. It sends
// no request body and no implicit GET; successful metadata comes from HTTP 204.
func (s *ResourceQuotaScope) Delete(ctx context.Context, options ...DeleteOption) (*DeleteResponse, error) {
	if err := s.validate(ctx); err != nil {
		return nil, quotaError("Delete", s.ProjectID(), err)
	}
	var config deleteOptions
	for _, apply := range options {
		if apply == nil {
			return nil, quotaError("Delete", s.projectID, fmt.Errorf("%w: nil delete option", resource.ErrInvalidOption))
		}
		if err := apply(&config); err != nil {
			return nil, quotaError("Delete", s.projectID, err)
		}
	}
	wire, err := s.api.requestQuota(ctx, http.MethodDelete, s.quotaEndpoint(), nil, []int{204})
	if config.ignoreMissing && gophercloud.ResponseCodeIs(err, 404) {
		return nil, nil
	}
	if err != nil {
		return nil, quotaError("Delete", s.projectID, err)
	}
	return &DeleteResponse{ProjectID: s.projectID, ResourceName: s.resourceName, Header: wire.header.Clone(), StatusCode: wire.status}, nil
}

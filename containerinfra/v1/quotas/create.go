package quotas

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// QuotaCreateOpts contains mutable quota attributes; the scope owns identity.
// HardLimit must be supplied explicitly, including when its value is zero.
type QuotaCreateOpts struct {
	HardLimit *int `json:"hard_limit"`
}

type QuotaCreateOption = request.Option[QuotaCreateOpts]

// WithHardLimit supplies an exact integer without requiring a pointer variable.
func WithHardLimit(value int) QuotaCreateOption {
	return func(config *request.Config[QuotaCreateOpts]) error {
		copy := value
		config.Options.HardLimit = &copy
		return nil
	}
}

// WithQuotaCreateOptions snapshots its pointer at option construction and gives
// each application a private copy. It replaces all typed create attributes.
func WithQuotaCreateOptions(value QuotaCreateOpts) QuotaCreateOption {
	copy := cloneCreateOptions(value)
	return func(config *request.Config[QuotaCreateOpts]) error {
		config.Options = cloneCreateOptions(copy)
		return nil
	}
}

func cloneCreateOptions(value QuotaCreateOpts) QuotaCreateOpts {
	if value.HardLimit != nil {
		copy := *value.HardLimit
		value.HardLimit = &copy
	}
	return value
}

// WithQuotaCreateField snapshots additional deployment JSON fields. Core quota
// attributes and scope identity cannot be replaced through extension fields.
func WithQuotaCreateField(key string, value any) QuotaCreateOption {
	return request.WithField[QuotaCreateOpts](key, value)
}

// Create posts a quota with this scope's fixed project and resource. It retains
// the pinned native flat body and 201 policy; limit validity is server-owned.
func (s *ResourceQuotaScope) Create(ctx context.Context, options ...QuotaCreateOption) (*QuotaResource, error) {
	if s == nil || s.api == nil {
		return nil, quotaError("Create", "", fmt.Errorf("%w: quota scope is required", resource.ErrInvalidOption))
	}
	if err := s.api.validateQuotaClient(ctx); err != nil {
		return nil, quotaError("Create", s.projectID, err)
	}
	config, err := request.Apply(QuotaCreateOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, true, false, false)
	}
	if err == nil && config.Options.HardLimit == nil {
		err = fmt.Errorf("%w: quota hard limit must be supplied explicitly", resource.ErrInvalidOption)
	}
	if err != nil {
		return nil, quotaError("Create", s.projectID, err)
	}
	for _, key := range []string{"project_id", "resource", "id"} {
		if _, exists := config.Fields[key]; exists {
			return nil, quotaError("Create", s.projectID, fmt.Errorf("%w: extension %q is quota identity", resource.ErrInvalidOption, key))
		}
	}
	// Raw JSON numbers avoid the native builder's float64 map conversion. The
	// complete body is encoded once, so retries never observe changed pointers.
	body, err := request.MergeFieldsFor(map[string]any{
		"project_id": s.projectID,
		"resource":   string(s.resourceName),
		"hard_limit": *config.Options.HardLimit,
	}, config.Fields, config.Options)
	if err != nil {
		return nil, quotaError("Create", s.projectID, err)
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, quotaError("Create", s.projectID, err)
	}
	wire, err := s.api.requestQuota(ctx, http.MethodPost, s.api.client.ServiceURL("quotas"), encoded, []int{201})
	if err != nil {
		return nil, quotaError("Create", s.projectID, err)
	}
	value, err := decodeQuota(wire, s.projectID, s.resourceName)
	return value, quotaError("Create", s.projectID, err)
}

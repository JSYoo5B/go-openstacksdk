package quotasets

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/quotasets"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

const scopeForceArgument = "project_quota_force"

// WithUpdateForce makes force explicit, including false. This option belongs
// to ProjectQuotaScope.Update; the lower-level API.Update retains native opts.
func WithUpdateForce(force bool) UpdateOption {
	return request.WithArgument[UpdateOpts](scopeForceArgument, force)
}

type preparedUpdate map[string]any

func (p preparedUpdate) ToComputeQuotaUpdateMap() (map[string]any, error) { return p, nil }

// Update changes only supplied limits. Nil limits are omitted, zero is sent,
// and -1 means unlimited. Force defaults to the native/server default false.
func (s *ProjectQuotaScope) Update(ctx context.Context, opts UpdateOpts, options ...UpdateOption) (*QuotaResource, error) {
	if err := s.api.validateQuotaClient(ctx); err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	config, err := request.Apply(opts, options...)
	if err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	if err := request.ValidateCapabilities(config, true, false, false, scopeForceArgument); err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	if err := validateLimits(config.Options); err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	force, explicitForce, err := request.Argument[bool](config, scopeForceArgument)
	if err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	body, err := (updateOptsBuilder{base: config.Options, config: config}).ToComputeQuotaUpdateMap()
	if err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	if explicitForce {
		body["quota_set"].(map[string]any)["force"] = force
	}
	result := upstream.Update(ctx, s.api.client, s.projectID, preparedUpdate(body))
	value, err := decodeQuota(result.Result, s.projectID)
	return value, quotaError("Update", s.projectID, err)
}

func validateLimits(options UpdateOpts) error {
	value := reflect.ValueOf(options)
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		if field.Kind() == reflect.Pointer && field.Type().Elem().Kind() == reflect.Int && !field.IsNil() && field.Elem().Int() < -1 {
			key := strings.Split(value.Type().Field(i).Tag.Get("json"), ",")[0]
			return fmt.Errorf("%w: quota %s must be -1 or non-negative", resource.ErrInvalidOption, key)
		}
	}
	return nil
}

type resetOptions struct{ ignoreMissing bool }

// ResetOption configures the reset's missing-project policy.
type ResetOption func(*resetOptions) error

// WithResetIgnoreMissing makes only HTTP 404 return nil, nil. The default is
// false, matching Python's revert_quota_set; the last option wins.
func WithResetIgnoreMissing(ignore bool) ResetOption {
	return func(options *resetOptions) error { options.ignoreMissing = ignore; return nil }
}

// Reset deletes the project's quota overrides, restoring Nova defaults.
// Successful DELETE responses carry headers, not a quota object; no GET follows.
func (s *ProjectQuotaScope) Reset(ctx context.Context, options ...ResetOption) (*ResetResponse, error) {
	if err := s.api.validateQuotaClient(ctx); err != nil {
		return nil, quotaError("Reset", s.projectID, err)
	}
	var config resetOptions
	for _, apply := range options {
		if apply == nil {
			return nil, quotaError("Reset", s.projectID, fmt.Errorf("%w: nil reset option", resource.ErrInvalidOption))
		}
		if err := apply(&config); err != nil {
			return nil, quotaError("Reset", s.projectID, err)
		}
	}
	result := upstream.Delete(ctx, s.api.client, s.projectID)
	if config.ignoreMissing && gophercloud.ResponseCodeIs(result.Err, http.StatusNotFound) {
		return nil, nil
	}
	if result.Err != nil {
		return nil, quotaError("Reset", s.projectID, result.Err)
	}
	return &ResetResponse{ProjectID: s.projectID, Header: result.Header.Clone()}, nil
}

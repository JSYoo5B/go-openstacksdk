package quotasets

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/quotasets"
)

const scopeForceArgument = "project_quota_force"

// WithUpdateForce makes force explicit, including false. This option belongs
// to the project/user scope Update; lower-level API.Update retains native opts.
func WithUpdateForce(force bool) UpdateOption {
	return request.WithArgument[UpdateOpts](scopeForceArgument, force)
}

// WithQuotaOptions snapshots every typed limit at option construction. It
// replaces the entire concrete update input and can be reused across calls.
func WithQuotaOptions(value UpdateOpts) UpdateOption {
	snapshot := snapshotOptions(value)
	return func(config *request.Config[UpdateOpts]) error {
		config.Options = snapshotOptions(snapshot)
		return nil
	}
}

func snapshotOptions(value UpdateOpts) UpdateOpts {
	copy := value
	pointers := reflect.ValueOf(&copy).Elem()
	for i := 0; i < pointers.NumField(); i++ {
		field := pointers.Field(i)
		if field.Kind() == reflect.Pointer && field.Type().Elem().Kind() == reflect.Int && !field.IsNil() {
			clone := reflect.New(field.Type().Elem())
			clone.Elem().SetInt(field.Elem().Int())
			field.Set(clone)
		}
	}
	return copy
}

type preparedUpdate map[string]any

func (p preparedUpdate) ToComputeQuotaUpdateMap() (map[string]any, error) { return p, nil }

// Update changes only supplied limits. Nil limits are omitted, zero is sent,
// and -1 means unlimited. Force defaults to the native/server default false.
func (s *ProjectQuotaScope) Update(ctx context.Context, opts UpdateOpts, options ...UpdateOption) (*QuotaResource, error) {
	if err := s.api.validateQuotaClient(ctx); err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	body, err := prepareQuotaUpdate(opts, options...)
	if err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	result := upstream.Update(ctx, s.api.client, s.projectID, preparedUpdate(body))
	value, err := decodeQuota(result.Result, s.projectID)
	return value, quotaError("Update", s.projectID, err)
}

func prepareQuotaUpdate(opts UpdateOpts, options ...UpdateOption) (map[string]any, error) {
	config, err := request.Apply(opts, options...)
	if err != nil {
		return nil, err
	}
	if err := request.ValidateCapabilities(config, true, false, false, scopeForceArgument); err != nil {
		return nil, err
	}
	config.Options = snapshotOptions(config.Options)
	if err := validateLimits(config.Options); err != nil {
		return nil, err
	}
	force, explicitForce, err := request.Argument[bool](config, scopeForceArgument)
	if err != nil {
		return nil, err
	}
	body, err := quotaUpdateBody(config)
	if err != nil {
		return nil, err
	}
	if explicitForce {
		body["quota_set"].(map[string]any)["force"] = force
	}
	return body, nil
}

func quotaUpdateBody(config request.Config[UpdateOpts]) (map[string]any, error) {
	// Native BuildRequestBody decodes through float64. Preserve typed integer
	// limits as raw JSON, including exact values larger than 2^53.
	encoded, err := json.Marshal(config.Options)
	if err != nil {
		return nil, fmt.Errorf("%w: quota options: %v", resource.ErrInvalidOption, err)
	}
	var core map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &core); err != nil {
		return nil, fmt.Errorf("%w: quota options: %v", resource.ErrInvalidOption, err)
	}
	fields := make(map[string]any, len(core))
	for key, value := range core {
		fields[key] = value
	}
	return request.MergeFieldsFor(map[string]any{"quota_set": fields}, config.Fields, config.Options)
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

// ResetOption configures the project/user reset's missing-target policy.
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

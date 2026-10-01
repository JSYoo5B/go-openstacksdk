package quotas

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

func WithQuotaOptions(options UpdateOpts) UpdateOption {
	snapshot := cloneQuotaOptions(options)
	return func(config *request.Config[UpdateOpts]) error {
		config.Options = cloneQuotaOptions(snapshot)
		return nil
	}
}
func cloneQuotaOptions(options UpdateOpts) UpdateOpts {
	value := reflect.ValueOf(&options).Elem()
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		if field.Kind() == reflect.Pointer && !field.IsNil() {
			copy := reflect.New(field.Type().Elem())
			copy.Elem().Set(field.Elem())
			field.Set(copy)
		}
	}
	return options
}

type QuotaLimit string

const (
	LimitLoadBalancers  QuotaLimit = "loadbalancer"
	LimitListeners      QuotaLimit = "listener"
	LimitMembers        QuotaLimit = "member"
	LimitPools          QuotaLimit = "pool"
	LimitHealthMonitors QuotaLimit = "healthmonitor"
	LimitL7Policies     QuotaLimit = "l7policy"
	LimitL7Rules        QuotaLimit = "l7rule"
)
const defaultLimitsArgument = "quota_default_limits"

// WithDefaultLimit sends explicit null for one limit, inheriting deployment
// defaults. A supplied typed value for the same limit is a conflicting input.
func WithDefaultLimit(limit QuotaLimit) UpdateOption {
	return func(config *request.Config[UpdateOpts]) error {
		if err := validateQuotaLimit(limit); err != nil {
			return err
		}
		limits, _, err := request.Argument[[]QuotaLimit](*config, defaultLimitsArgument)
		if err != nil {
			return err
		}
		for _, current := range limits {
			if current == limit {
				return nil
			}
		}
		config.Arguments[defaultLimitsArgument] = append(append([]QuotaLimit(nil), limits...), limit)
		return nil
	}
}

func validateQuotaLimit(limit QuotaLimit) error {
	switch limit {
	case LimitLoadBalancers, LimitListeners, LimitMembers, LimitPools, LimitHealthMonitors, LimitL7Policies, LimitL7Rules:
		return nil
	default:
		return fmt.Errorf("%w: unknown quota limit %q", resource.ErrInvalidOption, limit)
	}
}

func (s *ProjectQuotaScope) Update(ctx context.Context, options UpdateOpts, with ...UpdateOption) (*QuotaResource, error) {
	if err := s.api.validateQuotaClient(ctx); err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	config, err := request.Apply(options, with...)
	if err == nil {
		err = request.ValidateCapabilities(config, true, false, false, defaultLimitsArgument)
	}
	if err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	value := reflect.ValueOf(config.Options)
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		if field.Kind() == reflect.Pointer && !field.IsNil() && field.Elem().Int() < -1 {
			return nil, quotaError("Update", s.projectID, fmt.Errorf("%w: quota limit must be -1 or non-negative", resource.ErrInvalidOption))
		}
	}
	for _, alias := range []string{"load_balancer", "health_monitor", "project_id", "id"} {
		if _, exists := config.Fields[alias]; exists {
			return nil, quotaError("Update", s.projectID, fmt.Errorf("%w: %s is a quota identity or core alias", resource.ErrInvalidOption, alias))
		}
	}
	encoded, err := json.Marshal(config.Options)
	if err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	var core map[string]json.RawMessage
	if err = json.Unmarshal(encoded, &core); err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	defaults, _, err := request.Argument[[]QuotaLimit](config, defaultLimitsArgument)
	if err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	for _, limit := range defaults {
		if err := validateQuotaLimit(limit); err != nil {
			return nil, quotaError("Update", s.projectID, err)
		}
		key := string(limit)
		if _, exists := core[key]; exists {
			return nil, quotaError("Update", s.projectID, fmt.Errorf("%w: quota %s has both a value and default inheritance", resource.ErrInvalidOption, key))
		}
		core[key] = json.RawMessage("null")
	}
	fields := make(map[string]any, len(core))
	for key, value := range core {
		fields[key] = value
	}
	body, err := request.MergeFieldsFor(map[string]any{"quota": fields}, config.Fields, config.Options)
	if err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	encoded, err = json.Marshal(body)
	if err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	wire, err := s.api.requestQuota(ctx, http.MethodPut, s.api.quotaEndpoint(s.projectID), encoded, []int{200, 202})
	if err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	result, err := decodeQuota(wire, s.projectID)
	return result, quotaError("Update", s.projectID, err)
}

type resetOptions struct{ ignoreMissing bool }
type ResetOption func(*resetOptions) error

func WithResetIgnoreMissing(ignore bool) ResetOption {
	return func(options *resetOptions) error { options.ignoreMissing = ignore; return nil }
}

type ResetResponse struct {
	ProjectID  string
	Header     http.Header
	StatusCode int
}

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
	response, err := s.api.client.Delete(ctx, s.api.quotaEndpoint(s.projectID), nil)
	if config.ignoreMissing && gophercloud.ResponseCodeIs(err, 404) {
		return nil, nil
	}
	if err != nil {
		return nil, quotaError("Reset", s.projectID, err)
	}
	return &ResetResponse{ProjectID: s.projectID, Header: response.Header.Clone(), StatusCode: response.StatusCode}, nil
}

func validateFields(fields []string) error {
	for _, field := range fields {
		if strings.TrimSpace(field) == "" {
			return fmt.Errorf("%w: field must not be empty", resource.ErrInvalidOption)
		}
	}
	return nil
}

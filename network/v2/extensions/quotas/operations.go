package quotas

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
)

const scopeCheckLimitArgument = "project_quota_check_limit"

// WithQuotaOptions snapshots native typed limits when the option is created.
// Every application gets fresh pointers, so an option remains reusable.
func WithQuotaOptions(options UpdateOpts) UpdateOption {
	snapshot := cloneQuotaOptions(options)
	return func(config *request.Config[UpdateOpts]) error {
		config.Options = cloneQuotaOptions(snapshot)
		return nil
	}
}

func cloneQuotaOptions(options UpdateOpts) UpdateOpts {
	value := reflect.ValueOf(&options).Elem()
	// Pinned Neutron UpdateOpts contains only optional *int limit fields.
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		if field.Kind() == reflect.Pointer && field.Type().Elem().Kind() == reflect.Int && !field.IsNil() {
			copy := reflect.New(field.Type().Elem())
			copy.Elem().Set(field.Elem())
			field.Set(copy)
		}
	}
	return options
}

// WithUpdateCheckLimit explicitly sends check_limit, including false. This is
// a ProjectQuotaScope.Update option; the lower-level API retains native opts.
// Server extension support and usage-check policy are not inferred by the SDK.
func WithUpdateCheckLimit(check bool) UpdateOption {
	return request.WithArgument[UpdateOpts](scopeCheckLimitArgument, check)
}

// Update sends only supplied limits: nil omits, zero is explicit and -1 is
// unlimited. The body is serialized before the first HTTP request or retry.
func (s *ProjectQuotaScope) Update(ctx context.Context, opts UpdateOpts, options ...UpdateOption) (*QuotaResource, error) {
	if err := s.api.validateQuotaClient(ctx); err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	config, err := request.Apply(opts, options...)
	if err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	if err := request.ValidateCapabilities(config, true, false, false, scopeCheckLimitArgument); err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	if err := validateLimits(config.Options); err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	if _, exists := config.Fields["check_limit"]; exists {
		return nil, quotaError("Update", s.projectID, fmt.Errorf("%w: check_limit uses WithUpdateCheckLimit", resource.ErrInvalidOption))
	}
	check, explicitCheck, err := request.Argument[bool](config, scopeCheckLimitArgument)
	if err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	// Snapshot concrete pointer values as RawMessage, without native builder's
	// any/float64 round trip rounding large integer limits.
	encoded, err := json.Marshal(config.Options)
	if err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	limits := make(map[string]any, len(fields))
	for key, value := range fields {
		limits[key] = value
	}
	body, err := request.MergeFieldsFor(map[string]any{"quota": limits}, config.Fields, config.Options)
	if err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	if explicitCheck {
		body["quota"].(map[string]any)["check_limit"] = check
	}
	encoded, err = json.Marshal(body)
	if err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	var result gophercloud.Result
	var raw json.RawMessage
	response, err := s.api.client.Put(ctx, s.api.client.ServiceURL("quotas", s.projectID), json.RawMessage(encoded), &raw, &gophercloud.RequestOpts{OkCodes: []int{http.StatusOK}})
	result.Body = raw
	_, result.Header, result.Err = gophercloud.ParseResponse(response, err)
	statusCode := 0
	if response != nil {
		statusCode = response.StatusCode
	}
	value, err := decodeQuota(result, s.projectID, statusCode)
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

type deleteOptions struct{ ignoreMissing bool }

// DeleteOption changes the missing-project policy when resetting overrides.
type DeleteOption func(*deleteOptions) error

// WithDeleteIgnoreMissing suppresses only HTTP 404. The default is strict;
// later options override earlier ones, including an explicit false.
func WithDeleteIgnoreMissing(ignore bool) DeleteOption {
	return func(options *deleteOptions) error { options.ignoreMissing = ignore; return nil }
}

// Delete resets project overrides. It preserves native accepted 202/204 codes
// and does not automatically fetch the resulting defaults.
func (s *ProjectQuotaScope) Delete(ctx context.Context, options ...DeleteOption) (*DeleteResponse, error) {
	if err := s.api.validateQuotaClient(ctx); err != nil {
		return nil, quotaError("Delete", s.projectID, err)
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
	// Use the native request directly to retain its actual success code, which
	// upstream.Delete's ErrResult does not expose. nil opts retain native codes.
	response, err := s.api.client.Delete(ctx, s.api.client.ServiceURL("quotas", s.projectID), nil)
	if config.ignoreMissing && gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, quotaError("Delete", s.projectID, err)
	}
	return &DeleteResponse{ProjectID: s.projectID, Header: response.Header.Clone(), StatusCode: response.StatusCode}, nil
}

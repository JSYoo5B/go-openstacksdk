package quotas

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// WithQuotaOptions snapshots typed limit pointers at option construction. The
// option can be reused and gives each call its own copy of those values.
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

// Update uses Designate's PATCH route and unwrapped quota object. Nil typed
// limits are omitted, zero is sent, and large native ints remain exact.
func (s *ProjectQuotaScope) Update(ctx context.Context, options UpdateOpts, with ...UpdateOption) (*QuotaResource, error) {
	if err := s.validate(ctx); err != nil {
		return nil, quotaError("Update", s.ProjectID(), err)
	}
	config, err := request.Apply(options, with...)
	if err == nil {
		err = request.ValidateCapabilities(config, true, false, false)
	}
	if err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	config.Options = cloneQuotaOptions(config.Options)
	// Identity fields are never extension inputs to this fixed singleton.
	for _, key := range []string{"id", "project", "project_id"} {
		if _, exists := config.Fields[key]; exists {
			return nil, quotaError("Update", s.projectID, fmt.Errorf("%w: %s is a quota identity", resource.ErrInvalidOption, key))
		}
	}
	encoded, err := json.Marshal(config.Options)
	if err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	var core map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &core); err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	fields := make(map[string]any, len(core))
	for key, value := range core {
		fields[key] = value
	}
	body, err := request.MergeFieldsFor(fields, config.Fields, config.Options)
	if err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	encoded, err = json.Marshal(body)
	if err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	wire, err := s.requestQuota(ctx, http.MethodPatch, encoded, []int{http.StatusOK})
	if err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	value, err := decodeQuota(wire, s.projectID)
	return value, quotaError("Update", s.projectID, err)
}

type resetOptions struct{ ignoreMissing bool }

type ResetOption func(*resetOptions) error

// WithResetIgnoreMissing suppresses only HTTP 404. Reset is strict by default;
// the last option wins, and no read follows a successful reset.
func WithResetIgnoreMissing(enabled bool) ResetOption {
	return func(options *resetOptions) error { options.ignoreMissing = enabled; return nil }
}

// ResetResponse represents Designate's bodyless reset acknowledgement.
type ResetResponse struct {
	ProjectID  string
	Header     http.Header
	StatusCode int
}

// Reset deletes this project's custom quotas, restoring deployment defaults.
func (s *ProjectQuotaScope) Reset(ctx context.Context, options ...ResetOption) (*ResetResponse, error) {
	if err := s.validate(ctx); err != nil {
		return nil, quotaError("Reset", s.ProjectID(), err)
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
	wire, err := s.requestQuota(ctx, http.MethodDelete, nil, []int{http.StatusNoContent})
	if config.ignoreMissing && gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, quotaError("Reset", s.projectID, err)
	}
	return &ResetResponse{ProjectID: s.projectID, Header: wire.header.Clone(), StatusCode: wire.status}, nil
}

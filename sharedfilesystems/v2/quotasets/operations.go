package quotasets

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/JSYoo5B/gophercloudsdk/internal/fixedrequest"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// UpdateOpts is a concrete partial update: nil omits, zero sends zero, and -1
// is unlimited. Force is explicit, including false. No interface builder is
// required for either typed limits or service extension fields.
type UpdateOpts struct {
	Gigabytes           *int64         `json:"gigabytes,omitempty"`
	Snapshots           *int64         `json:"snapshots,omitempty"`
	Shares              *int64         `json:"shares,omitempty"`
	SnapshotGigabytes   *int64         `json:"snapshot_gigabytes,omitempty"`
	ShareGroups         *int64         `json:"share_groups,omitempty"`
	ShareGroupSnapshots *int64         `json:"share_group_snapshots,omitempty"`
	ShareNetworks       *int64         `json:"share_networks,omitempty"`
	ShareReplicas       *int64         `json:"share_replicas,omitempty"`
	ReplicaGigabytes    *int64         `json:"replica_gigabytes,omitempty"`
	PerShareGigabytes   *int64         `json:"per_share_gigabytes,omitempty"`
	Backups             *int64         `json:"backups,omitempty"`
	BackupGigabytes     *int64         `json:"backup_gigabytes,omitempty"`
	Force               *bool          `json:"force,omitempty"`
	Extra               map[string]any `json:"-"`
}

type UpdateOption = request.Option[UpdateOpts]

// WithQuotaOptions snapshots pointers and nested Extra values when constructed.
// Reusing the option cannot observe later changes to caller-owned data.
func WithQuotaOptions(value UpdateOpts) UpdateOption {
	snapshot, err := snapshotOptions(value)
	return func(config *request.Config[UpdateOpts]) error {
		if err != nil {
			return err
		}
		copy, err := snapshotOptions(snapshot)
		if err != nil {
			return err
		}
		config.Options = copy
		return nil
	}
}

func WithUpdateForce(force bool) UpdateOption {
	return func(config *request.Config[UpdateOpts]) error {
		copy := force
		config.Options.Force = &copy
		return nil
	}
}

// WithUpdateField captures an extension value as JSON. Declared limits, scope
// selectors and response-only metadata cannot be replaced through extensions.
func WithUpdateField(key string, value any) UpdateOption {
	return request.WithField[UpdateOpts](key, value)
}

type protectedFields struct {
	UpdateOpts
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	UserID      string `json:"user_id"`
	ShareType   string `json:"share_type"`
	Usage       any    `json:"usage"`
	Reservation any    `json:"reservation"`
	Envelope    any    `json:"quota_set"`
}

func snapshotOptions(value UpdateOpts) (UpdateOpts, error) {
	copy := value
	pointers := reflect.ValueOf(&copy).Elem()
	for i := 0; i < pointers.NumField(); i++ {
		field := pointers.Field(i)
		if field.Kind() == reflect.Pointer && !field.IsNil() {
			clone := reflect.New(field.Type().Elem())
			clone.Elem().Set(field.Elem())
			field.Set(clone)
		}
	}
	if value.Extra != nil {
		encoded, err := json.Marshal(value.Extra)
		if err != nil {
			return UpdateOpts{}, fmt.Errorf("%w: quota extensions: %v", resource.ErrInvalidOption, err)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(encoded, &fields); err != nil {
			return UpdateOpts{}, fmt.Errorf("%w: quota extensions: %v", resource.ErrInvalidOption, err)
		}
		if _, err := request.MergeFieldsFor(map[string]any{"quota_set": map[string]any{}}, fields, protectedFields{}); err != nil {
			return UpdateOpts{}, err
		}
		copy.Extra = make(map[string]any, len(fields))
		for key, data := range fields {
			if strings.TrimSpace(key) == "" {
				return UpdateOpts{}, fmt.Errorf("%w: empty quota extension key", resource.ErrInvalidOption)
			}
			copy.Extra[key] = append(json.RawMessage(nil), data...)
		}
	}
	return copy, nil
}

func prepareUpdate(opts UpdateOpts, options ...UpdateOption) (map[string]any, error) {
	config, err := request.Apply(opts, options...)
	if err != nil {
		return nil, err
	}
	if err := request.ValidateCapabilities(config, true, false, false); err != nil {
		return nil, err
	}
	config.Options, err = snapshotOptions(config.Options)
	if err != nil {
		return nil, err
	}
	value := reflect.ValueOf(config.Options)
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		if field.Kind() == reflect.Pointer && field.Type().Elem().Kind() == reflect.Int64 && !field.IsNil() && field.Elem().Int() < -1 {
			key := strings.Split(value.Type().Field(i).Tag.Get("json"), ",")[0]
			return nil, fmt.Errorf("%w: quota %s must be -1 or non-negative", resource.ErrInvalidOption, key)
		}
	}
	encoded, err := json.Marshal(config.Options)
	if err != nil {
		return nil, fmt.Errorf("%w: quota options: %v", resource.ErrInvalidOption, err)
	}
	var core map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &core); err != nil {
		return nil, err
	}
	fields := make(map[string]any, len(core))
	for key, data := range core {
		fields[key] = data
	}
	body := map[string]any{"quota_set": fields}
	extra := make(map[string]json.RawMessage, len(config.Options.Extra))
	for key, value := range config.Options.Extra {
		extra[key] = value.(json.RawMessage)
	}
	body, err = request.MergeFieldsFor(body, extra, protectedFields{})
	if err != nil {
		return nil, err
	}
	return request.MergeFieldsFor(body, config.Fields, protectedFields{})
}

func (a *API) update(ctx context.Context, endpoint string, body map[string]any) (quotaResponse, error) {
	client, err := fixedrequest.New(a.client, http.MethodPut, endpoint)
	if err != nil {
		return quotaResponse{}, err
	}
	var envelope map[string]json.RawMessage
	response, err := client.Put(ctx, endpoint, body, &envelope, &gophercloud.RequestOpts{OkCodes: []int{http.StatusOK}})
	return parseQuotaResponse(response, envelope, err)
}

func (s *ProjectQuotaScope) Update(ctx context.Context, opts UpdateOpts, options ...UpdateOption) (*QuotaResource, error) {
	if err := s.validate(ctx); err != nil {
		return nil, quotaError("Update", s.ProjectID(), err)
	}
	body, err := prepareUpdate(opts, options...)
	if err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	endpoint, err := s.api.quotaURL(s.projectID, "")
	if err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	result, err := s.api.update(ctx, endpoint, body)
	if err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	value, err := decodeQuota(result, s.projectID)
	return value, quotaError("Update", s.projectID, err)
}

type resetOptions struct{ ignoreMissing bool }
type ResetOption func(*resetOptions) error

// WithResetIgnoreMissing suppresses only HTTP 404. The default is strict.
func WithResetIgnoreMissing(ignore bool) ResetOption {
	return func(config *resetOptions) error { config.ignoreMissing = ignore; return nil }
}

func prepareReset(options ...ResetOption) (resetOptions, error) {
	var config resetOptions
	for _, apply := range options {
		if apply == nil {
			return config, fmt.Errorf("%w: nil reset option", resource.ErrInvalidOption)
		}
		if err := apply(&config); err != nil {
			return config, err
		}
	}
	return config, nil
}

func (a *API) reset(ctx context.Context, endpoint string, ignoreMissing bool) (*ResetResponse, error) {
	client, err := fixedrequest.New(a.client, http.MethodDelete, endpoint)
	if err != nil {
		return nil, err
	}
	response, err := client.Delete(ctx, endpoint, &gophercloud.RequestOpts{OkCodes: []int{http.StatusAccepted}})
	if ignoreMissing && gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ResetResponse{Header: response.Header.Clone(), StatusCode: response.StatusCode}, nil
}

// Reset deletes only this project's overrides using Manila's DELETE 202
// contract, without a follow-up GET or an invented quota list operation.
func (s *ProjectQuotaScope) Reset(ctx context.Context, options ...ResetOption) (*ResetResponse, error) {
	if err := s.validate(ctx); err != nil {
		return nil, quotaError("Reset", s.ProjectID(), err)
	}
	config, err := prepareReset(options...)
	if err != nil {
		return nil, quotaError("Reset", s.projectID, err)
	}
	endpoint, err := s.api.quotaURL(s.projectID, "")
	if err != nil {
		return nil, quotaError("Reset", s.projectID, err)
	}
	value, err := s.api.reset(ctx, endpoint, config.ignoreMissing)
	if value != nil {
		value.ProjectID = s.projectID
	}
	return value, quotaError("Reset", s.projectID, err)
}

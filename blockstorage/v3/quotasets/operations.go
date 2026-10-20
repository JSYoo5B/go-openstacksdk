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
	upstream "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/quotasets"
)

const scopeForceArgument = "project_quota_force"

// WithUpdateForce preserves an explicit false. It belongs to scope.Update;
// the lower-level API.Update retains native Force bool omission semantics.
func WithUpdateForce(force bool) UpdateOption {
	return request.WithArgument[UpdateOpts](scopeForceArgument, force)
}

// WithQuotaOptions snapshots every typed limit and nested Extra value when the
// option is constructed. It replaces the update's entire concrete input.
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

// VolumeTypeQuota selects one of Cinder's per-volume-type quota families.
type VolumeTypeQuota string

const (
	VolumeTypeVolumes   VolumeTypeQuota = "volumes"
	VolumeTypeSnapshots VolumeTypeQuota = "snapshots"
	VolumeTypeGigabytes VolumeTypeQuota = "gigabytes"
)

// WithVolumeTypeQuota sends a limit for an exact volume type name. It does not
// resolve a volume type ID: Cinder quota keys contain the type's name.
func WithVolumeTypeQuota(quota VolumeTypeQuota, name string, limit int) UpdateOption {
	field := WithUpdateField(string(quota)+"_"+name, limit)
	return func(config *request.Config[UpdateOpts]) error {
		if quota != VolumeTypeVolumes && quota != VolumeTypeSnapshots && quota != VolumeTypeGigabytes {
			return fmt.Errorf("%w: unsupported volume type quota %q", resource.ErrInvalidOption, quota)
		}
		if strings.TrimSpace(name) == "" || limit < -1 {
			return fmt.Errorf("%w: volume type quota requires a name and a limit of -1 or greater", resource.ErrInvalidOption)
		}
		return field(config)
	}
}

func snapshotOptions(value UpdateOpts) (UpdateOpts, error) {
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
	if value.Extra != nil {
		encoded, err := json.Marshal(value.Extra)
		if err != nil {
			return UpdateOpts{}, fmt.Errorf("%w: quota extensions: %v", resource.ErrInvalidOption, err)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(encoded, &fields); err != nil {
			return UpdateOpts{}, fmt.Errorf("%w: quota extensions: %v", resource.ErrInvalidOption, err)
		}
		// Native Extra merges without checking core fields, including omitted
		// fields. Use the common reserved-field policy before it can overwrite.
		if _, err := request.MergeFieldsFor(map[string]any{"quota_set": map[string]any{}}, fields, value); err != nil {
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

type preparedUpdate map[string]any

func (p preparedUpdate) ToBlockStorageQuotaUpdateMap() (map[string]any, error) { return p, nil }

// Update changes only supplied limits. Nil is omitted, zero is sent and -1
// means unlimited. The request body is detached from caller-owned Extra data.
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
	config.Options, err = snapshotOptions(config.Options)
	if err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	if err := validateLimits(config.Options); err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	force, explicitForce, err := request.Argument[bool](config, scopeForceArgument)
	if err != nil {
		return nil, quotaError("Update", s.projectID, err)
	}
	body, err := quotaUpdateBody(config)
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

func quotaUpdateBody(config request.Config[UpdateOpts]) (map[string]any, error) {
	// Native BuildRequestBody decodes through float64. Keep typed integer
	// limits as raw JSON, including values larger than 2^53.
	encoded, err := json.Marshal(config.Options)
	if err != nil {
		return nil, fmt.Errorf("%w: quota options: %v", resource.ErrInvalidOption, err)
	}
	var core map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &core); err != nil {
		return nil, fmt.Errorf("%w: quota options: %v", resource.ErrInvalidOption, err)
	}
	fields := make(map[string]any, len(core))
	for key, data := range core {
		fields[key] = data
	}
	body := map[string]any{"quota_set": fields}
	extra := make(map[string]json.RawMessage, len(config.Options.Extra))
	for key, value := range config.Options.Extra {
		// snapshotOptions has already detached and encoded every Extra value.
		extra[key] = value.(json.RawMessage)
	}
	body, err = request.MergeFieldsFor(body, extra, config.Options)
	if err != nil {
		return nil, err
	}
	return request.MergeFieldsFor(body, config.Fields, config.Options)
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

// ResetOption configures missing-project behavior for quota resets.
type ResetOption func(*resetOptions) error

// WithResetIgnoreMissing makes only HTTP 404 return nil, nil. Default false
// matches Python's revert_quota_set; the last option wins.
func WithResetIgnoreMissing(ignore bool) ResetOption {
	return func(options *resetOptions) error { options.ignoreMissing = ignore; return nil }
}

// Reset deletes quota overrides with Cinder's DELETE 200 contract. It returns
// response headers and never follows the reset with an automatic GET.
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

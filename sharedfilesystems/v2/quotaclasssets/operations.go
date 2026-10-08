package quotaclasssets

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/fixedrequest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// UpdateOpts has only quota-class limits. Force and project/user/share-type
// selectors belong to other APIs and cannot be promoted through this input.
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
	Extra               map[string]any `json:"-"`
}

type UpdateOption = request.Option[UpdateOpts]

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

func WithUpdateField(key string, value any) UpdateOption {
	return request.WithField[UpdateOpts](key, value)
}

type protectedFields struct {
	UpdateOpts
	ID             string `json:"id"`
	ClassName      string `json:"class_name"`
	QuotaClassName string `json:"quota_class_name"`
	ProjectID      string `json:"project_id"`
	UserID         string `json:"user_id"`
	ShareType      string `json:"share_type"`
	Force          bool   `json:"force"`
	Usage          any    `json:"usage"`
	Reservation    any    `json:"reservation"`
	Envelope       any    `json:"quota_class_set"`
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
			return UpdateOpts{}, fmt.Errorf("%w: quota class extensions: %v", resource.ErrInvalidOption, err)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(encoded, &fields); err != nil {
			return UpdateOpts{}, fmt.Errorf("%w: quota class extensions: %v", resource.ErrInvalidOption, err)
		}
		if _, err := request.MergeFieldsFor(map[string]any{"quota_class_set": map[string]any{}}, fields, protectedFields{}); err != nil {
			return UpdateOpts{}, err
		}
		copy.Extra = make(map[string]any, len(fields))
		for key, data := range fields {
			if strings.TrimSpace(key) == "" {
				return UpdateOpts{}, fmt.Errorf("%w: empty quota class extension key", resource.ErrInvalidOption)
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
		if field.Kind() == reflect.Pointer && !field.IsNil() && field.Elem().Int() < -1 {
			key := strings.Split(value.Type().Field(i).Tag.Get("json"), ",")[0]
			return nil, fmt.Errorf("%w: quota class %s must be -1 or non-negative", resource.ErrInvalidOption, key)
		}
	}
	encoded, err := json.Marshal(config.Options)
	if err != nil {
		return nil, fmt.Errorf("%w: quota class options: %v", resource.ErrInvalidOption, err)
	}
	var core map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &core); err != nil {
		return nil, err
	}
	fields := make(map[string]any, len(core))
	for key, data := range core {
		fields[key] = data
	}
	body := map[string]any{"quota_class_set": fields}
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

// Update uses PUT 200 and sends only supplied fields. Manila may create the
// named class through PUT; the SDK does not invent a separate Create operation.
func (s *QuotaClassScope) Update(ctx context.Context, opts UpdateOpts, options ...UpdateOption) (*QuotaClassResource, error) {
	if err := s.validate(ctx); err != nil {
		return nil, classError("Update", s.ClassName(), err)
	}
	body, err := prepareUpdate(opts, options...)
	if err != nil {
		return nil, classError("Update", s.className, err)
	}
	endpoint, err := s.endpoint()
	if err != nil {
		return nil, classError("Update", s.className, err)
	}
	client, err := fixedrequest.New(s.api.client, http.MethodPut, endpoint)
	if err != nil {
		return nil, classError("Update", s.className, err)
	}
	var envelope map[string]json.RawMessage
	response, err := client.Put(ctx, endpoint, body, &envelope, &gophercloud.RequestOpts{OkCodes: []int{http.StatusOK}})
	value, err := decodeClass(response, envelope, s.className, err)
	return value, classError("Update", s.className, err)
}

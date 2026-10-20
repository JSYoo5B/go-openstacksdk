package blockstorage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"time"
	"unicode/utf8"
)

// CreateVolumeAttributes provides common creation fields and known Cinder JSON
// attributes without a request builder. Nil omits a typed field; nonnil maps
// retain empty objects. Fields accepts pinned Volume attributes or wire names,
// with wire names taking precedence over aliases. Concrete fields override it.
type CreateVolumeAttributes struct {
	Name, Description, AvailabilityZone, ConsistencyGroupID *string
	SnapshotID, SourceVolumeID, SourceReplica, ImageID      *string
	BackupID, VolumeType, GroupID, VolumeTypeID             *string
	Multiattach                                             *bool
	Metadata                                                map[string]string
	SchedulerHints, Fields                                  map[string]json.RawMessage
}

// CreateVolumeOpts owns creation and completion policy. Wait nil selects true.
// Any nonnil Bootable forces waiting; only true adds a post-wait bootable action.
type CreateVolumeOpts struct {
	Wait, Bootable *bool
	WaitPolicy     CreateVolumeWaitOpts
	Attributes     CreateVolumeAttributes
}

// CreateVolumeWaitOpts applies an optional timeout after creation acknowledgement.
type CreateVolumeWaitOpts = AttachVolumeWaitOpts
type CreateVolumeOption func(*CreateVolumeOpts) error

func copyCreateVolumeRaw(values map[string]json.RawMessage) map[string]json.RawMessage {
	if values == nil {
		return nil
	}
	owned := make(map[string]json.RawMessage, len(values))
	for key, value := range values {
		owned[key] = append(json.RawMessage(nil), value...)
	}
	return owned
}

func copyCreateVolumeAttributes(value CreateVolumeAttributes) CreateVolumeAttributes {
	value.Name = copyAttachPointer(value.Name)
	value.Description = copyAttachPointer(value.Description)
	value.AvailabilityZone = copyAttachPointer(value.AvailabilityZone)
	value.ConsistencyGroupID = copyAttachPointer(value.ConsistencyGroupID)
	value.SnapshotID = copyAttachPointer(value.SnapshotID)
	value.SourceVolumeID = copyAttachPointer(value.SourceVolumeID)
	value.SourceReplica = copyAttachPointer(value.SourceReplica)
	value.ImageID = copyAttachPointer(value.ImageID)
	value.BackupID = copyAttachPointer(value.BackupID)
	value.VolumeType = copyAttachPointer(value.VolumeType)
	value.GroupID = copyAttachPointer(value.GroupID)
	value.VolumeTypeID = copyAttachPointer(value.VolumeTypeID)
	value.Multiattach = copyAttachPointer(value.Multiattach)
	value.Metadata = maps.Clone(value.Metadata)
	value.SchedulerHints = copyCreateVolumeRaw(value.SchedulerHints)
	value.Fields = copyCreateVolumeRaw(value.Fields)
	return value
}

func copyCreateVolumeOptions(value CreateVolumeOpts) CreateVolumeOpts {
	value.Wait = copyAttachPointer(value.Wait)
	value.Bootable = copyAttachPointer(value.Bootable)
	value.WaitPolicy = copyAttachWait(value.WaitPolicy)
	value.Attributes = copyCreateVolumeAttributes(value.Attributes)
	return value
}

func WithCreateVolumeOptions(value CreateVolumeOpts) CreateVolumeOption {
	owned := copyCreateVolumeOptions(value)
	return func(config *CreateVolumeOpts) error { *config = copyCreateVolumeOptions(owned); return nil }
}
func WithCreateVolumeAttributes(value CreateVolumeAttributes) CreateVolumeOption {
	owned := copyCreateVolumeAttributes(value)
	return func(config *CreateVolumeOpts) error {
		config.Attributes = copyCreateVolumeAttributes(owned)
		return nil
	}
}
func WithCreateVolumeWait(value bool) CreateVolumeOption {
	return func(config *CreateVolumeOpts) error { owned := value; config.Wait = &owned; return nil }
}
func WithCreateVolumeBootable(value bool) CreateVolumeOption {
	return func(config *CreateVolumeOpts) error { owned := value; config.Bootable = &owned; return nil }
}
func WithCreateVolumeWaitPolicy(value CreateVolumeWaitOpts) CreateVolumeOption {
	owned := copyAttachWait(value)
	return func(config *CreateVolumeOpts) error { config.WaitPolicy = copyAttachWait(owned); return nil }
}
func WithCreateVolumeName(value string) CreateVolumeOption {
	return func(config *CreateVolumeOpts) error { owned := value; config.Attributes.Name = &owned; return nil }
}
func WithCreateVolumeDescription(value string) CreateVolumeOption {
	return func(config *CreateVolumeOpts) error {
		owned := value
		config.Attributes.Description = &owned
		return nil
	}
}
func WithCreateVolumeSnapshot(value string) CreateVolumeOption {
	return func(config *CreateVolumeOpts) error {
		owned := value
		config.Attributes.SnapshotID = &owned
		return nil
	}
}
func WithCreateVolumeType(value string) CreateVolumeOption {
	return func(config *CreateVolumeOpts) error {
		owned := value
		config.Attributes.VolumeType = &owned
		return nil
	}
}
func WithCreateVolumeMetadata(value map[string]string) CreateVolumeOption {
	owned := maps.Clone(value)
	return func(config *CreateVolumeOpts) error { config.Attributes.Metadata = maps.Clone(owned); return nil }
}
func WithCreateVolumeSchedulerHints(value map[string]json.RawMessage) CreateVolumeOption {
	owned := copyCreateVolumeRaw(value)
	return func(config *CreateVolumeOpts) error {
		config.Attributes.SchedulerHints = copyCreateVolumeRaw(owned)
		return nil
	}
}
func WithCreateVolumeFields(value map[string]json.RawMessage) CreateVolumeOption {
	owned := copyCreateVolumeRaw(value)
	return func(config *CreateVolumeOpts) error {
		config.Attributes.Fields = copyCreateVolumeRaw(owned)
		return nil
	}
}

type preparedCreateVolumeOptions struct {
	policy            CreateVolumeOpts
	wait              bool
	interval, timeout time.Duration
	failures          []string
	body              map[string]json.RawMessage
}

func applyCreateVolumeOptions(options []CreateVolumeOption, guard func() error) (preparedCreateVolumeOptions, error) {
	config := CreateVolumeOpts{}
	for _, apply := range append([]CreateVolumeOption(nil), options...) {
		if err := guard(); err != nil {
			return preparedCreateVolumeOptions{}, err
		}
		if apply == nil {
			return preparedCreateVolumeOptions{}, attachInvalid("nil volume creation option")
		}
		callback := copyCreateVolumeOptions(config)
		err := apply(&callback)
		config = copyCreateVolumeOptions(callback)
		if err = errors.Join(err, guard()); err != nil {
			return preparedCreateVolumeOptions{}, err
		}
	}
	common, err := applyAttachOptions([]AttachVolumeOption{WithAttachVolumeOptions(AttachVolumeOpts{Wait: config.Wait, WaitPolicy: config.WaitPolicy})}, guard)
	if err != nil {
		return preparedCreateVolumeOptions{}, err
	}
	body, err := createVolumeBody(config.Attributes)
	if err != nil {
		return preparedCreateVolumeOptions{}, err
	}
	wait := common.wait
	if config.Bootable != nil {
		wait = true
	}
	return preparedCreateVolumeOptions{policy: copyCreateVolumeOptions(config), wait: wait, interval: common.interval, timeout: common.timeout, failures: common.failures, body: body}, guard()
}

// PrepareCreateVolumeOptions validates and owns all effective defaults locally.
// Original callbacks run once; the returned policy can be reapplied with the
// complete-policy factory without rerunning them or selecting any service.
func PrepareCreateVolumeOptions(ctx context.Context, options ...CreateVolumeOption) (CreateVolumeOpts, error) {
	if err := attachContext(ctx); err != nil {
		return CreateVolumeOpts{}, wrapCreateVolumeError(ctx, err)
	}
	prepared, err := applyCreateVolumeOptions(options, func() error { return attachContext(ctx) })
	if err != nil {
		return CreateVolumeOpts{}, wrapCreateVolumeError(ctx, err)
	}
	config := copyCreateVolumeOptions(prepared.policy)
	config.Wait = copyAttachPointer(&prepared.wait)
	config.WaitPolicy.Timeout = copyAttachPointer(&prepared.timeout)
	config.WaitPolicy.PollInterval = copyAttachPointer(&prepared.interval)
	config.WaitPolicy.FailureStates = append(make([]string, 0, len(prepared.failures)), prepared.failures...)
	return config, nil
}

func validateCreateVolumeRaw(value json.RawMessage) (json.RawMessage, error) {
	if value == nil {
		return json.RawMessage("null"), nil
	}
	if !utf8.Valid(value) || !json.Valid(value) {
		return nil, attachInvalid("volume attribute must be valid UTF-8 JSON")
	}
	return append(json.RawMessage(nil), value...), nil
}

func createVolumeFalsey(raw json.RawMessage) bool {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&value) != nil {
		return false
	}
	switch v := value.(type) {
	case nil:
		return true
	case bool:
		return !v
	case string:
		return v == ""
	case json.Number:
		// Only deciding zero truthiness; preserve the original numeric bytes.
		for _, c := range v.String() {
			if c >= '1' && c <= '9' {
				return false
			}
			if c == 'e' || c == 'E' {
				break
			}
		}
		return true
	case []any:
		return len(v) == 0
	case map[string]any:
		return len(v) == 0
	}
	return false
}

func putCreateVolumeValue[T any](fields map[string]json.RawMessage, key string, value *T) error {
	if value == nil {
		return nil
	}
	if text, ok := any(*value).(string); ok && !utf8.ValidString(text) {
		return attachInvalid("volume field %s must be valid UTF-8", key)
	}
	raw, err := json.Marshal(*value)
	if err != nil {
		return err
	}
	fields[key] = raw
	return nil
}

func createVolumeBody(attributes CreateVolumeAttributes) (map[string]json.RawMessage, error) {
	fields := copyCreateVolumeRaw(attributes.Fields)
	if fields == nil {
		fields = make(map[string]json.RawMessage)
	}
	for key, value := range fields {
		raw, err := validateCreateVolumeRaw(value)
		if err != nil {
			return nil, err
		}
		fields[key] = raw
	}
	for _, field := range []struct {
		key   string
		value *string
	}{
		{"name", attributes.Name}, {"description", attributes.Description}, {"availability_zone", attributes.AvailabilityZone},
		{"consistencygroup_id", attributes.ConsistencyGroupID}, {"snapshot_id", attributes.SnapshotID},
		{"source_volid", attributes.SourceVolumeID}, {"source_replica", attributes.SourceReplica}, {"imageRef", attributes.ImageID},
		{"backup_id", attributes.BackupID}, {"volume_type", attributes.VolumeType}, {"group_id", attributes.GroupID}, {"volume_type_id", attributes.VolumeTypeID},
	} {
		if err := putCreateVolumeValue(fields, field.key, field.value); err != nil {
			return nil, err
		}
	}
	if err := putCreateVolumeValue(fields, "multiattach", attributes.Multiattach); err != nil {
		return nil, err
	}
	if attributes.Metadata != nil {
		for key, value := range attributes.Metadata {
			if !utf8.ValidString(key) || !utf8.ValidString(value) {
				return nil, attachInvalid("volume metadata must be valid UTF-8")
			}
		}
		raw, err := json.Marshal(attributes.Metadata)
		if err != nil {
			return nil, err
		}
		fields["metadata"] = raw
	}
	if attributes.SchedulerHints != nil {
		hints := copyCreateVolumeRaw(attributes.SchedulerHints)
		for key, value := range hints {
			if !utf8.ValidString(key) {
				return nil, attachInvalid("scheduler hint key must be valid UTF-8")
			}
			raw, err := validateCreateVolumeRaw(value)
			if err != nil {
				return nil, err
			}
			hints[key] = raw
		}
		raw, err := json.Marshal(hints)
		if err != nil {
			return nil, err
		}
		fields["OS-SCH-HNT:scheduler_hints"] = raw
	}
	for _, keys := range [][2]string{{"name", "display_name"}, {"description", "display_description"}} {
		selected, present := fields[keys[0]]
		if !present {
			selected, present = fields[keys[1]]
		}
		delete(fields, keys[0])
		delete(fields, keys[1])
		if present && !createVolumeFalsey(selected) {
			fields[keys[0]] = selected
		}
	}
	volume := make(map[string]json.RawMessage, len(fields))
	for key, value := range fields {
		wire, known := createVolumeAttributeNames[key]
		if !known {
			return nil, attachInvalid("unknown or route-changing volume attribute %q", key)
		}
		if key != wire {
			if _, exists := fields[wire]; exists {
				continue
			}
		}
		volume[wire] = append(json.RawMessage(nil), value...)
	}
	body := make(map[string]json.RawMessage)
	if hints, present := volume["OS-SCH-HNT:scheduler_hints"]; present {
		delete(volume, "OS-SCH-HNT:scheduler_hints")
		if !bytes.Equal(bytes.TrimSpace(hints), []byte("null")) {
			object, err := attachmentObject(hints)
			if err != nil {
				return nil, attachInvalid("scheduler hints must be an object or null")
			}
			if len(object) > 0 {
				body["OS-SCH-HNT:scheduler_hints"] = append(json.RawMessage(nil), hints...)
			}
		}
	}
	raw, err := json.Marshal(volume)
	if err != nil {
		return nil, err
	}
	body["volume"] = raw
	return body, nil
}

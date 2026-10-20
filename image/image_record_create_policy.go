package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"slices"
	"unicode/utf8"
)

// ImageCreatePolicy captures cloud image defaults without a caller builder.
// Nil format/tasks default to qcow2/false; explicit JSON null is retained.
// ObjectStoreEnabled nil delegates availability to the configured service.
type ImageCreatePolicy struct {
	ImageFormat, UseTasks json.RawMessage
	DisableVendorAgent    map[string]any
	// RawVendorAgent, when present, overrides the typed mapping. Shape checks
	// belong after duplicate reuse, when the vendor flag selects that branch.
	RawVendorAgent     json.RawMessage
	ObjectStoreEnabled *bool
	// RawObjectStoreEnabled defers configuration type checks until Task routing.
	RawObjectStoreEnabled json.RawMessage
	CloudName             string
}
type ImageCreatePolicyOption func(*ImageCreatePolicy) error

func WithImageCreatePolicyOpts(value ImageCreatePolicy) ImageCreatePolicyOption {
	snapshot, captureErr := copyImageCreatePolicy(nil, nil, value)
	return func(config *ImageCreatePolicy) error {
		if captureErr != nil {
			return captureErr
		}
		owned, err := copyImageCreatePolicy(nil, nil, snapshot)
		if err == nil {
			*config = owned
		}
		return err
	}
}
func WithImageCreatePolicyFormat(value any) ImageCreatePolicyOption {
	return withImageCreatePolicyValue(value, true)
}
func WithImageCreatePolicyTasks(value any) ImageCreatePolicyOption {
	return withImageCreatePolicyValue(value, false)
}
func withImageCreatePolicyValue(value any, format bool) ImageCreatePolicyOption {
	snapshot, captureErr := captureImageRecordImportJSON(nil, nil, value)
	return func(config *ImageCreatePolicy) error {
		if captureErr != nil {
			return captureErr
		}
		if format {
			config.ImageFormat = bytes.Clone(snapshot)
		} else {
			config.UseTasks = bytes.Clone(snapshot)
		}
		return nil
	}
}
func WithImageCreatePolicyVendorAgent(values map[string]any) ImageCreatePolicyOption {
	snapshot, captureErr := captureImageRecordImportMap(nil, nil, values, nil)
	return func(config *ImageCreatePolicy) error {
		if captureErr != nil {
			return captureErr
		}
		owned, err := captureImageRecordImportMap(nil, nil, snapshot, nil)
		if err == nil {
			config.DisableVendorAgent = owned
			config.RawVendorAgent = nil
		}
		return err
	}
}
func WithImageCreatePolicyObjectStoreEnabled(enabled bool) ImageCreatePolicyOption {
	return func(config *ImageCreatePolicy) error {
		owned := enabled
		config.ObjectStoreEnabled = &owned
		config.RawObjectStoreEnabled = nil
		return nil
	}
}

func copyImageCreatePolicy(ctx context.Context, check func(context.Context) error, value ImageCreatePolicy) (ImageCreatePolicy, error) {
	if err := imageRecordImportCheck(ctx, check); err != nil {
		return value, err
	}
	value.ImageFormat, value.UseTasks, value.RawVendorAgent, value.RawObjectStoreEnabled = bytes.Clone(value.ImageFormat), bytes.Clone(value.UseTasks), bytes.Clone(value.RawVendorAgent), bytes.Clone(value.RawObjectStoreEnabled)
	for _, raw := range []json.RawMessage{value.ImageFormat, value.UseTasks, value.RawVendorAgent, value.RawObjectStoreEnabled} {
		if raw != nil {
			if err := validateImageRecordImportJSON(raw); err != nil {
				return value, errors.Join(err, imageRecordImportCheck(ctx, check))
			}
		}
	}
	if value.ObjectStoreEnabled != nil {
		owned := *value.ObjectStoreEnabled
		value.ObjectStoreEnabled = &owned
	}
	if !utf8.ValidString(value.CloudName) {
		return value, uploadInvalid("image create cloud name must be UTF-8")
	}
	var err error
	value.DisableVendorAgent, err = captureImageRecordImportMap(ctx, check, value.DisableVendorAgent, nil)
	return value, errors.Join(err, imageRecordImportCheck(ctx, check))
}

// PrepareImageCreatePolicy builds concrete defaults. Values are JSON-domain
// Source values rather than client-side format enums or boolean restrictions.
func PrepareImageCreatePolicy(options ...ImageCreatePolicyOption) (ImageCreatePolicy, error) {
	var config ImageCreatePolicy
	for _, apply := range slices.Clone(options) {
		if apply == nil {
			return config, uploadInvalid("nil image create policy option")
		}
		candidate, err := copyImageCreatePolicy(nil, nil, config)
		if err != nil {
			return config, err
		}
		if err := apply(&candidate); err != nil {
			return config, err
		}
		config, err = copyImageCreatePolicy(nil, nil, candidate)
		if err != nil {
			return config, err
		}
	}
	if config.ImageFormat == nil {
		config.ImageFormat = json.RawMessage(`"qcow2"`)
	}
	if config.UseTasks == nil {
		config.UseTasks = json.RawMessage(`false`)
	}
	if config.DisableVendorAgent == nil {
		config.DisableVendorAgent = make(map[string]any)
	}
	return config, nil
}

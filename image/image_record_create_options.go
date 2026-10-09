package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"
)

// ImageRecordCreateOpts owns Source-domain option values. Nil remains omitted
// until the workflow applies its ordered defaults; explicit null, false, empty
// data/store selections and arbitrary JSON values retain their presence. Meta
// bypasses later property conversion; Attributes follows ordinary kwargs.
type ImageRecordCreateOpts struct {
	Container                                          *string
	MD5, SHA256, DiskFormat, ContainerFormat, Tags     json.RawMessage
	DisableVendorAgent, AllowDuplicates, Wait, Timeout json.RawMessage
	ValidateChecksum, UseImport, Size                  json.RawMessage
	Meta, Attributes                                   map[string]any
	Import                                             ImageRecordImportOpts
	Headers, SwiftHeaders                              map[string]string
	optionContext                                      context.Context
	optionCheck                                        func(context.Context) error
}
type ImageRecordCreateOption func(*ImageRecordCreateOpts) error

// WithImageRecordCreateOpts captures and replaces the entire option policy.
func WithImageRecordCreateOpts(value ImageRecordCreateOpts) ImageRecordCreateOption {
	snapshot, captureErr := copyImageRecordCreateOpts(nil, nil, value)
	snapshot.optionContext, snapshot.optionCheck = nil, nil
	return func(config *ImageRecordCreateOpts) error {
		if captureErr != nil {
			return captureErr
		}
		owned, err := copyImageRecordCreateOpts(config.optionContext, config.optionCheck, snapshot)
		if err == nil {
			owned.optionContext, owned.optionCheck = config.optionContext, config.optionCheck
			*config = owned
		}
		return err
	}
}
func WithImageRecordCreateContainer(value string) ImageRecordCreateOption {
	return func(config *ImageRecordCreateOpts) error {
		if !utf8.ValidString(value) {
			return uploadInvalid("image create container must be UTF-8")
		}
		owned := value
		config.Container = &owned
		return nil
	}
}
func WithImageRecordCreateMD5(value any) ImageRecordCreateOption {
	return withImageRecordCreateValue(value, func(c *ImageRecordCreateOpts, r json.RawMessage) { c.MD5 = r })
}
func WithImageRecordCreateSHA256(value any) ImageRecordCreateOption {
	return withImageRecordCreateValue(value, func(c *ImageRecordCreateOpts, r json.RawMessage) { c.SHA256 = r })
}
func WithImageRecordCreateDiskFormat(value any) ImageRecordCreateOption {
	return withImageRecordCreateValue(value, func(c *ImageRecordCreateOpts, r json.RawMessage) { c.DiskFormat = r })
}
func WithImageRecordCreateContainerFormat(value any) ImageRecordCreateOption {
	return withImageRecordCreateValue(value, func(c *ImageRecordCreateOpts, r json.RawMessage) { c.ContainerFormat = r })
}
func WithImageRecordCreateTags(value any) ImageRecordCreateOption {
	return withImageRecordCreateValue(value, func(c *ImageRecordCreateOpts, r json.RawMessage) { c.Tags = r })
}
func WithImageRecordCreateDisableVendorAgent(value any) ImageRecordCreateOption {
	return withImageRecordCreateValue(value, func(c *ImageRecordCreateOpts, r json.RawMessage) { c.DisableVendorAgent = r })
}
func WithImageRecordCreateAllowDuplicates(value any) ImageRecordCreateOption {
	return withImageRecordCreateValue(value, func(c *ImageRecordCreateOpts, r json.RawMessage) { c.AllowDuplicates = r })
}
func WithImageRecordCreateWait(value any) ImageRecordCreateOption {
	return withImageRecordCreateValue(value, func(c *ImageRecordCreateOpts, r json.RawMessage) { c.Wait = r })
}
func WithImageRecordCreateTimeout(value any) ImageRecordCreateOption {
	return withImageRecordCreateValue(value, func(c *ImageRecordCreateOpts, r json.RawMessage) { c.Timeout = r })
}
func WithImageRecordCreateValidateChecksum(value any) ImageRecordCreateOption {
	return withImageRecordCreateValue(value, func(c *ImageRecordCreateOpts, r json.RawMessage) { c.ValidateChecksum = r })
}
func WithImageRecordCreateUseImport(value any) ImageRecordCreateOption {
	return withImageRecordCreateValue(value, func(c *ImageRecordCreateOpts, r json.RawMessage) { c.UseImport = r })
}
func WithImageRecordCreateSize(value any) ImageRecordCreateOption {
	return withImageRecordCreateValue(value, func(c *ImageRecordCreateOpts, r json.RawMessage) { c.Size = r })
}
func withImageRecordCreateValue(value any, set func(*ImageRecordCreateOpts, json.RawMessage)) ImageRecordCreateOption {
	snapshot, captureErr := captureImageRecordImportJSON(nil, nil, value)
	return func(config *ImageRecordCreateOpts) error {
		if captureErr != nil {
			return captureErr
		}
		set(config, bytes.Clone(snapshot))
		return nil
	}
}

// WithImageRecordCreateMeta replaces the complete raw metadata mapping.
func WithImageRecordCreateMeta(values map[string]any) ImageRecordCreateOption {
	snapshot, captureErr := captureImageRecordImportMap(nil, nil, values, nil)
	return func(config *ImageRecordCreateOpts) error {
		if captureErr != nil {
			return captureErr
		}
		owned, err := captureImageRecordImportMap(config.optionContext, config.optionCheck, snapshot, nil)
		if err == nil {
			config.Meta = owned
		}
		return err
	}
}
func WithImageRecordCreateAttribute(field string, value any) ImageRecordCreateOption {
	return WithImageRecordCreateAttributes(map[string]any{field: value})
}

// WithImageRecordCreateAttributes merges exact kwargs keys; metadata controls
// and Source shape/conversion checks belong to the later selected branch.
func WithImageRecordCreateAttributes(values map[string]any) ImageRecordCreateOption {
	snapshot, captureErr := captureImageRecordImportMap(nil, nil, values, nil)
	return func(config *ImageRecordCreateOpts) error {
		if captureErr != nil {
			return captureErr
		}
		incoming, err := captureImageRecordImportMap(config.optionContext, config.optionCheck, snapshot, nil)
		if err != nil {
			return err
		}
		if config.Attributes == nil {
			config.Attributes = make(map[string]any)
		}
		for key, value := range incoming {
			config.Attributes[key] = value
		}
		return nil
	}
}

// WithImageRecordCreateImportOptions applies each captured nested callback once
// at operation preparation, before any branch-specific import default/check.
func WithImageRecordCreateImportOptions(options ...ImageRecordImportOption) ImageRecordCreateOption {
	owned := slices.Clone(options)
	return func(config *ImageRecordCreateOpts) error {
		ctx, check := config.optionContext, config.optionCheck
		imported, err := copyImageRecordImportOpts(ctx, check, config.Import)
		if err != nil {
			return err
		}
		for _, apply := range owned {
			if err := imageRecordImportCheck(ctx, check); err != nil {
				return err
			}
			if apply == nil {
				return uploadInvalid("nil image create import option")
			}
			candidate, err := copyImageRecordImportOpts(ctx, check, imported)
			if err != nil {
				return err
			}
			if err := errors.Join(apply(&candidate), imageRecordImportCheck(ctx, check)); err != nil {
				return err
			}
			imported, err = copyImageRecordImportOpts(ctx, check, candidate)
			if err != nil {
				return err
			}
		}
		config.Import = imported
		return imageRecordImportCheck(ctx, check)
	}
}
func WithImageRecordCreateHeader(key, value string) ImageRecordCreateOption {
	return func(config *ImageRecordCreateOpts) error {
		return mergeImageRecordHeaders(&config.Headers, map[string]string{key: value})
	}
}
func WithImageRecordCreateHeaders(values map[string]string) ImageRecordCreateOption {
	snapshot := copyImageRecordHeaders(values)
	return func(config *ImageRecordCreateOpts) error { return mergeImageRecordHeaders(&config.Headers, snapshot) }
}
func WithImageRecordCreateSwiftHeader(key, value string) ImageRecordCreateOption {
	return WithImageRecordCreateSwiftHeaders(map[string]string{key: value})
}
func WithImageRecordCreateSwiftHeaders(values map[string]string) ImageRecordCreateOption {
	snapshot := copyImageRecordHeaders(values)
	return func(config *ImageRecordCreateOpts) error {
		headers, err := imageRecordCreateSwiftHeaders(snapshot)
		if err != nil {
			return err
		}
		return mergeImageRecordHeaders(&config.SwiftHeaders, headers)
	}
}
func imageRecordCreateSwiftHeaders(values map[string]string) (map[string]string, error) {
	headers, err := imageMutationHeaders(values, false, "")
	if err != nil {
		return nil, err
	}
	for key := range headers {
		if strings.EqualFold(key, "X-Delete-After") || strings.EqualFold(key, "X-Object-Meta-X-Sdk-Autocreated") {
			return nil, uploadInvalid("image create Swift header %q is owned by the SDK", key)
		}
	}
	return headers, nil
}

func copyImageRecordCreateOpts(ctx context.Context, check func(context.Context) error, value ImageRecordCreateOpts) (ImageRecordCreateOpts, error) {
	if err := imageRecordImportCheck(ctx, check); err != nil {
		return value, err
	}
	if value.Container != nil {
		owned := *value.Container
		value.Container = &owned
		if !utf8.ValidString(owned) {
			return value, uploadInvalid("image create container must be UTF-8")
		}
	}
	for _, raw := range []*json.RawMessage{&value.MD5, &value.SHA256, &value.DiskFormat, &value.ContainerFormat, &value.Tags, &value.DisableVendorAgent, &value.AllowDuplicates, &value.Wait, &value.Timeout, &value.ValidateChecksum, &value.UseImport, &value.Size} {
		*raw = bytes.Clone(*raw)
		if *raw != nil {
			if err := validateImageRecordImportJSON(*raw); err != nil {
				return value, errors.Join(err, imageRecordImportCheck(ctx, check))
			}
		}
	}
	value.Headers = copyImageRecordHeaders(value.Headers)
	value.SwiftHeaders = copyImageRecordHeaders(value.SwiftHeaders)
	value.Meta, value.Attributes = maps.Clone(value.Meta), maps.Clone(value.Attributes)
	var err error
	value.Import, err = copyImageRecordImportOpts(ctx, check, value.Import)
	if err != nil {
		return value, err
	}
	value.Meta, err = captureImageRecordImportMap(ctx, check, value.Meta, nil)
	if err != nil {
		return value, err
	}
	value.Attributes, err = captureImageRecordImportMap(ctx, check, value.Attributes, nil)
	return value, errors.Join(err, imageRecordImportCheck(ctx, check))
}
func prepareImageRecordCreateOptions(ctx context.Context, check func(context.Context) error, options []ImageRecordCreateOption) (ImageRecordCreateOpts, error) {
	config, err := copyImageRecordCreateOpts(ctx, check, ImageRecordCreateOpts{})
	if err != nil {
		return config, err
	}
	for _, apply := range slices.Clone(options) {
		if err := imageRecordImportCheck(ctx, check); err != nil {
			return config, err
		}
		if apply == nil {
			return config, uploadInvalid("nil image record create option")
		}
		candidate, err := copyImageRecordCreateOpts(ctx, check, config)
		if err != nil {
			return config, err
		}
		candidate.optionContext, candidate.optionCheck = ctx, check
		if err := errors.Join(apply(&candidate), imageRecordImportCheck(ctx, check)); err != nil {
			return config, err
		}
		candidate.optionContext, candidate.optionCheck = nil, nil
		config, err = copyImageRecordCreateOpts(ctx, check, candidate)
		if err != nil {
			return config, err
		}
	}
	config.Headers, err = imageMutationHeaders(config.Headers, false, "")
	if err == nil {
		config.SwiftHeaders, err = imageRecordCreateSwiftHeaders(config.SwiftHeaders)
	}
	return config, errors.Join(err, imageRecordImportCheck(ctx, check))
}

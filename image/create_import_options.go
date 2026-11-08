package image

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"time"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/image/v2/imagedata"
	"github.com/JSYoo5B/go-openstacksdk/image/v2/imageimport"
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// CreateImportMetadataOpts owns creation fields. Empty formats select
// qcow2/bare; nil Visibility selects private. Other nil pointers are omitted.
// Properties extend the flat metadata object without replacing core fields.
type CreateImportMetadataOpts struct {
	DiskFormat      string
	ContainerFormat string
	Visibility      *Visibility
	Protected       *bool
	Hidden          *bool
	MinDisk         *int
	MinRAM          *int
	Tags            []string
	Properties      map[string]any
	Headers         map[string]string
}

// CreateImportOpts owns the distinct metadata, stage and import policies.
// Wait false leaves the asynchronous import acknowledgement as the final
// phase. WaitOptions holds shared resource wait options for the active-state
// wait: defaults are five minutes, two seconds and killed/deleted failures;
// WithTimeout, WithUnlimitedWait, WithPollInterval, WithWaitStrategy and
// WithFailureStates (no states disable failure detection) apply in order.
// Other shared controls fail with resource.ErrInvalidOption before HTTP.
type CreateImportOpts struct {
	Metadata    CreateImportMetadataOpts
	Stage       imagedata.StageOpts
	Import      imageimport.ImportOpts
	Wait        bool
	WaitOptions []resource.WaitOption

	stageOptions  []imagedata.StageOption
	importOptions []imageimport.ImportOption
}

type CreateImportOption func(*CreateImportOpts) error

// WithCreateImportOpts snapshots and replaces the complete workflow policy.
func WithCreateImportOpts(value CreateImportOpts) CreateImportOption {
	snapshot, err := copyCreateImportOpts(value)
	return func(config *CreateImportOpts) error {
		if err != nil {
			return err
		}
		owned, err := copyCreateImportOpts(snapshot)
		if err == nil {
			*config = owned
		}
		return err
	}
}

func WithCreateImportMetadata(value CreateImportMetadataOpts) CreateImportOption {
	snapshot, err := copyCreateImportMetadata(value)
	return func(config *CreateImportOpts) error {
		if err != nil {
			return err
		}
		owned, err := copyCreateImportMetadata(snapshot)
		if err == nil {
			config.Metadata = owned
		}
		return err
	}
}

func WithCreateImportStage(value imagedata.StageOpts) CreateImportOption {
	option := imagedata.WithStageOpts(value)
	return func(config *CreateImportOpts) error {
		config.stageOptions = nil
		return applyCustomCall(option, &config.Stage)
	}
}

func WithCreateImportImport(value imageimport.ImportOpts) CreateImportOption {
	option := imageimport.WithImportOpts(value)
	return func(config *CreateImportOpts) error {
		config.importOptions = nil
		return applyCustomCall(option, &config.Import)
	}
}

// WithCreateImportStageOptions replaces the stage callback batch. Callbacks
// apply to the concrete Stage policy exactly once during preflight.
func WithCreateImportStageOptions(options ...imagedata.StageOption) CreateImportOption {
	snapshot := append([]imagedata.StageOption(nil), options...)
	return func(config *CreateImportOpts) error {
		config.stageOptions = append([]imagedata.StageOption(nil), snapshot...)
		return nil
	}
}

// WithCreateImportImportOptions replaces the import callback batch. Callbacks
// apply to the concrete Import policy exactly once during preflight.
func WithCreateImportImportOptions(options ...imageimport.ImportOption) CreateImportOption {
	snapshot := append([]imageimport.ImportOption(nil), options...)
	return func(config *CreateImportOpts) error {
		config.importOptions = append([]imageimport.ImportOption(nil), snapshot...)
		return nil
	}
}

// WithCreateImportWait enables the active-state wait and appends shared
// resource wait options to WaitOptions. Repeated calls append; for each
// control the later option wins.
func WithCreateImportWait(options ...resource.WaitOption) CreateImportOption {
	snapshot := slices.Clone(options)
	return func(config *CreateImportOpts) error {
		config.Wait = true
		config.WaitOptions = append(slices.Clone(config.WaitOptions), snapshot...)
		return nil
	}
}

// WithoutCreateImportWait disables the wait and drops its options.
func WithoutCreateImportWait() CreateImportOption {
	return func(config *CreateImportOpts) error { config.Wait, config.WaitOptions = false, nil; return nil }
}

func copyCreateImportMetadata(value CreateImportMetadataOpts) (CreateImportMetadataOpts, error) {
	value.Visibility = copyCreateImportPointer(value.Visibility)
	value.Protected = copyCreateImportPointer(value.Protected)
	value.Hidden = copyCreateImportPointer(value.Hidden)
	value.MinDisk = copyCreateImportPointer(value.MinDisk)
	value.MinRAM = copyCreateImportPointer(value.MinRAM)
	if value.Tags != nil {
		value.Tags = append(make([]string, 0, len(value.Tags)), value.Tags...)
	}
	value.Headers = maps.Clone(value.Headers)
	if value.Properties != nil {
		properties := make(map[string]any, len(value.Properties))
		for key, property := range value.Properties {
			body, err := json.Marshal(property)
			if err != nil {
				return value, fmt.Errorf("%w: image property %q is not JSON serializable: %w", resource.ErrInvalidOption, key, err)
			}
			if !utf8.Valid(body) {
				return value, uploadInvalid("image property must be valid UTF-8")
			}
			properties[key] = append(json.RawMessage(nil), body...)
		}
		value.Properties = properties
	}
	return value, nil
}

func copyCreateImportPointer[T any](value *T) *T {
	if value == nil {
		return nil
	}
	owned := *value
	return &owned
}

func copyCreateImportOpts(value CreateImportOpts) (CreateImportOpts, error) {
	var err error
	value.Metadata, err = copyCreateImportMetadata(value.Metadata)
	if err != nil {
		return value, err
	}
	if err = applyCustomCall(imagedata.WithStageOpts(value.Stage), &value.Stage); err != nil {
		return value, err
	}
	if err = applyCustomCall(imageimport.WithImportOpts(value.Import), &value.Import); err != nil {
		return value, err
	}
	value.WaitOptions = slices.Clone(value.WaitOptions)
	value.stageOptions = append([]imagedata.StageOption(nil), value.stageOptions...)
	value.importOptions = append([]imageimport.ImportOption(nil), value.importOptions...)
	return value, nil
}

func parseCreateImportOptions(options []CreateImportOption) (CreateImportOpts, error) {
	var config CreateImportOpts
	for _, apply := range options {
		if apply == nil {
			return config, uploadInvalid("nil create/import option")
		}
		if err := apply(&config); err != nil {
			return config, err
		}
	}
	// Keep the retained custom-option configuration separate from the snapshot.
	return copyCreateImportOpts(config)
}

// createImportWaitSupport lists the shared controls the active-state wait honors.
const createImportWaitSupport = resource.WaitSupportFailureStates

// prepareCreateImportWait validates the wait options and returns them over
// the create/import defaults; nil means no wait.
func prepareCreateImportWait(config CreateImportOpts) ([]resource.WaitOption, error) {
	if !config.Wait {
		return nil, nil
	}
	settings, err := resource.ResolveWait(resource.WaitDefaults{Timeout: 5 * time.Minute, Strategy: resource.FixedWait(2 * time.Second), Supports: createImportWaitSupport}, config.WaitOptions...)
	if err != nil {
		return nil, err
	}
	if !settings.FailureStatesSet {
		settings.FailureStates, settings.FailureStatesSet = []string{"killed", "deleted"}, true
	}
	for _, state := range settings.FailureStates {
		if !utf8.ValidString(state) {
			return nil, uploadInvalid("invalid wait failure state")
		}
	}
	options := []resource.WaitOption{resource.WithWaitSettings(settings)}
	return options, resource.ValidateWaitOptionsFor[Image](options...)
}

// applyCustomCall applies one request.Custom call over T, such as a
// With<Op>Opts option of another image package, to state.
func applyCustomCall[T any, C request.Call](call C, state *T) error {
	return calls.Translate([]C{call}, calls.Spec[T]{})[0](state)
}

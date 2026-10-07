package image

import (
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/image/v2/imagedata"
	"github.com/JSYoo5B/gophercloudsdk/image/v2/imageimport"
	"github.com/JSYoo5B/gophercloudsdk/resource"
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

// CreateImportWaitOpts configures an optional active-state wait. Defaults are
// five minutes, two seconds and killed/deleted failures. Timeout zero removes
// the SDK deadline; a non-nil empty FailureStates disables state failures.
type CreateImportWaitOpts struct {
	Timeout       *time.Duration
	PollInterval  *time.Duration
	FailureStates []string
}

// CreateImportOpts owns the distinct metadata, stage and import policies.
// Wait nil leaves the asynchronous import acknowledgement as the final phase.
type CreateImportOpts struct {
	Metadata CreateImportMetadataOpts
	Stage    imagedata.StageOpts
	Import   imageimport.ImportOpts
	Wait     *CreateImportWaitOpts

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
		return option(&config.Stage)
	}
}

func WithCreateImportImport(value imageimport.ImportOpts) CreateImportOption {
	option := imageimport.WithImportOpts(value)
	return func(config *CreateImportOpts) error {
		config.importOptions = nil
		return option(&config.Import)
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

func WithCreateImportWait(value CreateImportWaitOpts) CreateImportOption {
	snapshot := copyCreateImportWait(value)
	return func(config *CreateImportOpts) error {
		owned := copyCreateImportWait(snapshot)
		config.Wait = &owned
		return nil
	}
}

func WithoutCreateImportWait() CreateImportOption {
	return func(config *CreateImportOpts) error { config.Wait = nil; return nil }
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

func copyCreateImportWait(value CreateImportWaitOpts) CreateImportWaitOpts {
	value.Timeout = copyCreateImportPointer(value.Timeout)
	value.PollInterval = copyCreateImportPointer(value.PollInterval)
	if value.FailureStates != nil {
		value.FailureStates = append(make([]string, 0, len(value.FailureStates)), value.FailureStates...)
	}
	return value
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
	if err = imagedata.WithStageOpts(value.Stage)(&value.Stage); err != nil {
		return value, err
	}
	if err = imageimport.WithImportOpts(value.Import)(&value.Import); err != nil {
		return value, err
	}
	if value.Wait != nil {
		wait := copyCreateImportWait(*value.Wait)
		value.Wait = &wait
	}
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

func prepareCreateImportWait(value *CreateImportWaitOpts) ([]resource.WaitOption, error) {
	if value == nil {
		return nil, nil
	}
	options := make([]resource.WaitOption, 0, 3)
	if value.Timeout != nil {
		if *value.Timeout < 0 {
			return nil, uploadInvalid("wait timeout must not be negative")
		}
		if *value.Timeout == 0 {
			options = append(options, resource.WithUnlimitedWait())
		} else {
			options = append(options, resource.WithTimeout(*value.Timeout))
		}
	}
	if value.PollInterval != nil {
		options = append(options, resource.WithPollInterval(*value.PollInterval))
	}
	if value.FailureStates != nil {
		options = append(options, resource.WithFailureStates(value.FailureStates...))
	} else {
		options = append(options, resource.WithFailureStates("killed", "deleted"))
	}
	for _, state := range value.FailureStates {
		if !utf8.ValidString(state) || strings.TrimSpace(state) == "" {
			return nil, uploadInvalid("invalid wait failure state")
		}
	}
	return options, resource.ValidateWaitOptionsFor[Image](options...)
}

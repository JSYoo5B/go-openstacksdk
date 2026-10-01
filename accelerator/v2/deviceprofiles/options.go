package deviceprofiles

import (
	"encoding/json"
	"fmt"
	"regexp"

	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// CreateOpts creates one device profile. Nil optional pointers are omitted;
// explicit empty descriptions are preserved.
type CreateOpts struct {
	Name        string  `json:"name"`
	Groups      []Group `json:"groups"`
	UUID        *string `json:"uuid,omitempty"`
	Description *string `json:"description,omitempty"`
}

type CreateOption = request.Option[CreateOpts]

// WithCreateOptions snapshots nested typed inputs when the option is created.
// Applying the option replaces the typed options with a fresh copy.
func WithCreateOptions(options CreateOpts) CreateOption {
	encoded, err := json.Marshal(options)
	return func(config *request.Config[CreateOpts]) error {
		if err != nil {
			return fmt.Errorf("%w: device profile options: %v", resource.ErrInvalidOption, err)
		}
		var value CreateOpts
		if err := json.Unmarshal(encoded, &value); err != nil {
			return fmt.Errorf("%w: device profile options: %v", resource.ErrInvalidOption, err)
		}
		config.Options = value
		return nil
	}
}

func WithCreateField(key string, value any) CreateOption {
	return request.WithField[CreateOpts](key, value)
}

func WithCreateHeader(key, value string) CreateOption {
	return request.WithHeader[CreateOpts](key, value)
}

var namePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func prepareCreate(opts CreateOpts, options ...CreateOption) (json.RawMessage, map[string]string, error) {
	config, err := request.Apply(opts, options...)
	if err != nil {
		return nil, nil, err
	}
	if err := request.ValidateCapabilities(config, true, false, true); err != nil {
		return nil, nil, err
	}
	if !namePattern.MatchString(config.Options.Name) {
		return nil, nil, fmt.Errorf("%w: device profile name must use letters, digits, hyphen or underscore", resource.ErrInvalidOption)
	}
	if len(config.Options.Groups) == 0 {
		return nil, nil, fmt.Errorf("%w: device profile requires at least one group", resource.ErrInvalidOption)
	}
	for _, group := range config.Options.Groups {
		if group == nil {
			return nil, nil, fmt.Errorf("%w: device profile groups must be JSON objects", resource.ErrInvalidOption)
		}
	}
	if config.Options.UUID != nil {
		if err := validateUUID(*config.Options.UUID); err != nil {
			return nil, nil, err
		}
	}
	encoded, err := json.Marshal(config.Options)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: device profile input: %v", resource.ErrInvalidOption, err)
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &body); err != nil {
		return nil, nil, err
	}
	fields := make(map[string]any, len(body))
	for key, value := range body {
		fields[key] = value
	}
	fields, err = request.MergeFieldsFor(fields, config.Fields, config.Options)
	if err != nil {
		return nil, nil, err
	}
	// The controller accepts an array containing exactly one profile.
	encoded, err = json.Marshal([]map[string]any{fields})
	return encoded, config.Headers, err
}

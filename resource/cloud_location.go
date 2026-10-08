package resource

import (
	"bytes"
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
)

// CloudProject describes a token scope or the project owning a resource.
// Nil ID means JSON null; raw IDs retain a service's untyped JSON value.
// Names describe configured authentication, rather than inferred catalog data.
type CloudProject struct {
	ID         json.RawMessage `json:"id"`
	Name       *string         `json:"name"`
	DomainID   *string         `json:"domain_id"`
	DomainName *string         `json:"domain_name"`
}

// CloudLocation is the SDK-owned location used in normalized cloud resources.
// Nil strings and nil raw fields encode null. Explicit empty strings survive.
type CloudLocation struct {
	Cloud      *string         `json:"cloud"`
	RegionName *string         `json:"region_name"`
	Zone       json.RawMessage `json:"zone"`
	Project    CloudProject    `json:"project"`
}

func (value CloudLocation) Clone() CloudLocation {
	copy := value
	copy.Cloud = cloneLocationString(value.Cloud)
	copy.RegionName = cloneLocationString(value.RegionName)
	copy.Zone = bytes.Clone(value.Zone)
	copy.Project.ID = bytes.Clone(value.Project.ID)
	copy.Project.Name = cloneLocationString(value.Project.Name)
	copy.Project.DomainID = cloneLocationString(value.Project.DomainID)
	copy.Project.DomainName = cloneLocationString(value.Project.DomainName)
	return copy
}

// ForResource returns an independent computed location. A truthy foreign
// project ID keeps its raw value and clears the current project's names.
// A missing, falsey or matching project uses the configured current scope.
func (value CloudLocation) ForResource(projectID, zone json.RawMessage) (json.RawMessage, error) {
	copy := value.Clone()
	if err := validateLocationJSON(copy.Project.ID); err != nil {
		return nil, err
	}
	if err := validateLocationJSON(projectID); err != nil {
		return nil, err
	}
	if err := validateLocationJSON(zone); err != nil {
		return nil, err
	}
	truthy := false
	if projectID != nil {
		var err error
		boolean, truthError := jsonfilter.BooleanJSON(projectID)
		err = truthError
		truthy = bytes.Equal(boolean, []byte("true"))
		if err != nil {
			return nil, fmt.Errorf("%w: project ID: %w", ErrInvalidOption, err)
		}
	}
	if truthy {
		current := copy.Project.ID
		if current == nil {
			current = json.RawMessage("null")
		}
		equal, err := jsonfilter.EqualPythonJSON(projectID, current)
		if err != nil {
			return nil, fmt.Errorf("%w: project equality: %w", ErrInvalidOption, err)
		}
		if !equal {
			copy.Project = CloudProject{ID: bytes.Clone(projectID)}
		}
	}
	copy.Zone = bytes.Clone(zone)
	raw, err := json.Marshal(copy)
	if err != nil {
		return nil, fmt.Errorf("%w: location JSON: %w", ErrInvalidOption, err)
	}
	if !utf8.Valid(raw) {
		return nil, fmt.Errorf("%w: location must be UTF-8", ErrInvalidOption)
	}
	return raw, nil
}

func validateLocationJSON(raw json.RawMessage) error {
	if raw == nil {
		return nil
	}
	if !utf8.Valid(raw) || !json.Valid(raw) {
		return fmt.Errorf("%w: location fields must be complete UTF-8 JSON", ErrInvalidOption)
	}
	return nil
}

func cloneLocationString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

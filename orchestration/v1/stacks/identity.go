package stacks

import (
	"fmt"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// StackIdentity keeps both canonical components required by Heat endpoints.
// ID remains the stack UUID; neither component contains an encoded URL path.
type StackIdentity struct {
	Name string
	ID   string
}

func (identity StackIdentity) Validate() error {
	for _, value := range []string{identity.Name, identity.ID} {
		if err := validateSegment(value); err != nil {
			return err
		}
	}
	return nil
}

func validateSegment(value string) error {
	if err := resource.ID(value).Validate(); err != nil {
		return err
	}
	if strings.ContainsRune(value, '\x00') {
		return fmt.Errorf("%w: stack identity must contain non-empty path segments", resource.ErrInvalidOption)
	}
	return nil
}

// StackResource shares the native detailed fields across list and GET results.
// Detailed is false for list summaries: zero-valued detailed fields in a summary
// do not claim the stack has no outputs, parameters or other detailed data.
type StackResource struct {
	RetrievedStack
	Detailed bool `json:"-"`
}

func (value *StackResource) Identity() (StackIdentity, error) {
	if value == nil {
		return StackIdentity{}, fmt.Errorf("%w: nil stack", resource.ErrInvalidOption)
	}
	identity := StackIdentity{Name: value.Name, ID: value.ID}
	return identity, identity.Validate()
}

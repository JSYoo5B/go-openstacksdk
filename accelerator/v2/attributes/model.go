// Package attributes manages Cyborg deployable key/value attributes.
package attributes

import (
	"encoding/json"
	"fmt"
	"regexp"

	"gophercloudsdk/accelerator/v2/common"
	"gophercloudsdk/resource"
)

var uuidPattern = regexp.MustCompile(`^[[:xdigit:]]{8}-[[:xdigit:]]{4}-[[:xdigit:]]{4}-[[:xdigit:]]{4}-[[:xdigit:]]{12}$`)

type Attribute struct {
	common.Metadata
	UUID         string      `json:"uuid"`
	ID           json.Number `json:"id"`
	DeployableID int64       `json:"deployable_id"`
	Key          string      `json:"key"`
	Value        string      `json:"value"`
}

func (a *Attribute) UnmarshalJSON(data []byte) error {
	type plain Attribute
	var value plain
	if err := common.Decode(data, &value, &value.Metadata); err != nil {
		return err
	}
	if err := validateUUID(value.UUID); err != nil {
		return fmt.Errorf("Cyborg attribute response: %w", err)
	}
	*a = Attribute(value)
	return nil
}

func validateUUID(id string) error {
	if !uuidPattern.MatchString(id) {
		return fmt.Errorf("%w: attribute ID must be a canonical UUID", resource.ErrInvalidOption)
	}
	return nil
}

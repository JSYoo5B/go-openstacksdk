// Package deviceprofiles manages Cyborg accelerator device profiles.
package deviceprofiles

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/JSYoo5B/gophercloudsdk/accelerator/v2/common"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

var uuidPattern = regexp.MustCompile(`^[[:xdigit:]]{8}-[[:xdigit:]]{4}-[[:xdigit:]]{4}-[[:xdigit:]]{4}-[[:xdigit:]]{12}$`)

// Group contains a concrete JSON object of resource, trait or accelerator
// requirements. Numeric values are decoded as json.Number, without rounding.
type Group map[string]any

func (g *Group) UnmarshalJSON(data []byte) error {
	if data = bytes.TrimSpace(data); len(data) == 0 || data[0] != '{' {
		return fmt.Errorf("Cyborg device profile group must be a JSON object")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value map[string]any
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	*g = value
	return nil
}

type DeviceProfile struct {
	common.Metadata
	UUID        string  `json:"uuid"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Groups      []Group `json:"groups"`
}

func (p *DeviceProfile) UnmarshalJSON(data []byte) error {
	type plain DeviceProfile
	var value plain
	if err := common.Decode(data, &value, &value.Metadata); err != nil {
		return err
	}
	if err := validateUUID(value.UUID); err != nil {
		return fmt.Errorf("Cyborg device profile response: %w", err)
	}
	*p = DeviceProfile(value)
	return nil
}

func validateUUID(id string) error {
	if !uuidPattern.MatchString(id) {
		return fmt.Errorf("%w: device profile ID must be a canonical UUID", resource.ErrInvalidOption)
	}
	return nil
}

// Package softwareconfigs manages Heat software configurations. Gophercloud
// v2.15.0 has no binding for /software_configs, so this package is SDK-owned.
package softwareconfigs

import (
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// SoftwareConfig is one Heat software configuration. Config, Inputs, Outputs
// and Options keep their exact JSON (the config may be a string or an object).
// List rows are summaries that omit most of these members. Metadata.Body keeps
// every original field, including those not declared here.
type SoftwareConfig struct {
	resource.Metadata
	ID   string `json:"id"`
	Name string `json:"name"`
	// Group selects the configuration tool namespace.
	Group   string          `json:"group"`
	Config  json.RawMessage `json:"config"`
	Inputs  json.RawMessage `json:"inputs"`
	Outputs json.RawMessage `json:"outputs"`
	Options json.RawMessage `json:"options"`
	// CreationTime is the original wire timestamp string (Python created_at).
	CreationTime string `json:"creation_time"`
}

func (c *SoftwareConfig) UnmarshalJSON(data []byte) error {
	type plain SoftwareConfig
	var value plain
	if err := resource.DecodeObject(data, &value, &value.Metadata); err != nil {
		return err
	}
	*c = SoftwareConfig(value)
	return nil
}

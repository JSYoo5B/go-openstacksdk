// Package portgroups manages Ironic port groups through SDK-owned /v1/portgroups routes.
// The pinned Gophercloud release has no binding for this resource.
package portgroups

import (
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// PortGroup retains the typed server fields, the original response object in Body,
// and the HTTP header and status code of the response that produced it.
type PortGroup struct {
	resource.Metadata
	UUID                     string          `json:"uuid"`
	Name                     *string         `json:"name"`
	Address                  *string         `json:"address"`
	NodeUUID                 string          `json:"node_uuid"`
	StandalonePortsSupported *bool           `json:"standalone_ports_supported"`
	Mode                     *string         `json:"mode"`
	Properties               map[string]any  `json:"properties"`
	Extra                    map[string]any  `json:"extra"`
	InternalInfo             map[string]any  `json:"internal_info"`
	Ports                    []resource.Link `json:"ports"`
}

func (v *PortGroup) UnmarshalJSON(data []byte) error {
	type plain PortGroup
	var value plain
	if err := resource.DecodeObject(data, &value, &value.Metadata); err != nil {
		return err
	}
	*v = PortGroup(value)
	return nil
}

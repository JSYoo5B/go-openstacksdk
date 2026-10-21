// Package volumeconnectors manages Ironic volume connectors through SDK-owned /v1/volume/connectors routes.
// The pinned Gophercloud release has no binding for this resource.
package volumeconnectors

import (
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// VolumeConnector retains the typed server fields, the original response object in Body,
// and the HTTP header and status code of the response that produced it.
type VolumeConnector struct {
	resource.Metadata
	UUID        string         `json:"uuid"`
	NodeUUID    string         `json:"node_uuid"`
	Type        string         `json:"type"`
	ConnectorID string         `json:"connector_id"`
	Extra       map[string]any `json:"extra"`
}

func (v *VolumeConnector) UnmarshalJSON(data []byte) error {
	type plain VolumeConnector
	var value plain
	if err := resource.DecodeObject(data, &value, &value.Metadata); err != nil {
		return err
	}
	*v = VolumeConnector(value)
	return nil
}

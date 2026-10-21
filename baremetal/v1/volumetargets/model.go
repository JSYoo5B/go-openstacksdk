// Package volumetargets manages Ironic volume targets through SDK-owned /v1/volume/targets routes.
// The pinned Gophercloud release has no binding for this resource.
package volumetargets

import (
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// VolumeTarget retains the typed server fields, the original response object in Body,
// and the HTTP header and status code of the response that produced it.
type VolumeTarget struct {
	resource.Metadata
	UUID       string         `json:"uuid"`
	NodeUUID   string         `json:"node_uuid"`
	VolumeType string         `json:"volume_type"`
	VolumeID   string         `json:"volume_id"`
	BootIndex  int            `json:"boot_index"`
	Properties map[string]any `json:"properties"`
	Extra      map[string]any `json:"extra"`
}

func (v *VolumeTarget) UnmarshalJSON(data []byte) error {
	type plain VolumeTarget
	var value plain
	if err := resource.DecodeObject(data, &value, &value.Metadata); err != nil {
		return err
	}
	*v = VolumeTarget(value)
	return nil
}

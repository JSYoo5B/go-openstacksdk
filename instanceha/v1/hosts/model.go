// Package hosts manages hosts inside a fixed Masakari failover segment.
package hosts

import (
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/instanceha/v1/segments"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type Host struct {
	resource.Metadata
	ID                json.RawMessage   `json:"id"`
	UUID              string            `json:"uuid"`
	Name              string            `json:"name"`
	Type              string            `json:"type"`
	ControlAttributes json.RawMessage   `json:"control_attributes"`
	OnMaintenance     *bool             `json:"on_maintenance"`
	Reserved          *bool             `json:"reserved"`
	FailoverSegmentID string            `json:"failover_segment_id"`
	FailoverSegment   *segments.Segment `json:"failover_segment"`
	// SegmentID is the fixed URI parent, independent of response body fields.
	SegmentID string `json:"-"`
}

func (h *Host) UnmarshalJSON(data []byte) error {
	type plain Host
	var value plain
	if err := resource.DecodeObject(data, &value, &value.Metadata); err != nil {
		return err
	}
	*h = Host(value)
	return nil
}

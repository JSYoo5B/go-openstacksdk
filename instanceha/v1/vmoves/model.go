// Package vmoves reads virtual-machine recovery moves inside a notification.
package vmoves

import (
	"encoding/json"

	"gophercloudsdk/resource"
)

type VMove struct {
	resource.Metadata
	ID               json.RawMessage `json:"id"`
	UUID             string          `json:"uuid"`
	NotificationUUID string          `json:"notification_uuid"`
	ServerID         string          `json:"instance_uuid"`
	ServerName       string          `json:"instance_name"`
	SourceHost       *string         `json:"source_host"`
	DestHost         *string         `json:"dest_host"`
	StartTime        *string         `json:"start_time"`
	EndTime          *string         `json:"end_time"`
	Status           string          `json:"status"`
	Type             string          `json:"type"`
	Message          *string         `json:"message"`
	// NotificationID is the fixed URI parent, not a server-selected body field.
	NotificationID string `json:"-"`
}

func (v *VMove) UnmarshalJSON(data []byte) error {
	type plain VMove
	var value plain
	if err := resource.DecodeObject(data, &value, &value.Metadata); err != nil {
		return err
	}
	*v = VMove(value)
	return nil
}

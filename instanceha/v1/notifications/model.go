// Package notifications manages Masakari failure notifications.
package notifications

import (
	"encoding/json"

	"gophercloudsdk/resource"
)

type ProgressDetail struct {
	resource.Metadata
	Timestamp *string `json:"timestamp"`
	Message   *string `json:"message"`
	// Progress may be a JSON number or string; it is not an integer percentage.
	Progress json.RawMessage `json:"progress"`
}

func (d *ProgressDetail) UnmarshalJSON(data []byte) error {
	type plain ProgressDetail
	var value plain
	if err := resource.DecodeObject(data, &value, &value.Metadata); err != nil {
		return err
	}
	*d = ProgressDetail(value)
	return nil
}

type RecoveryWorkflowDetail struct {
	resource.Metadata
	Name            string           `json:"name"`
	State           string           `json:"state"`
	Progress        json.RawMessage  `json:"progress"`
	ProgressDetails []ProgressDetail `json:"progress_details"`
}

func (d *RecoveryWorkflowDetail) UnmarshalJSON(data []byte) error {
	type plain RecoveryWorkflowDetail
	var value plain
	if err := resource.DecodeObject(data, &value, &value.Metadata); err != nil {
		return err
	}
	*d = RecoveryWorkflowDetail(value)
	return nil
}

// UUID decodes notification_uuid, the route identity; ID is the separate raw
// database ID. Payload retains its exact JSON without a float64 round trip.
type Notification struct {
	resource.Metadata
	ID                      json.RawMessage          `json:"id"`
	UUID                    string                   `json:"notification_uuid"`
	Type                    string                   `json:"type"`
	Hostname                string                   `json:"hostname"`
	Status                  string                   `json:"status"`
	GeneratedTime           string                   `json:"generated_time"`
	Payload                 json.RawMessage          `json:"payload"`
	SourceHostUUID          string                   `json:"source_host_uuid"`
	RecoveryWorkflowDetails []RecoveryWorkflowDetail `json:"recovery_workflow_details"`
}

func (n *Notification) UnmarshalJSON(data []byte) error {
	type plain Notification
	var value plain
	if err := resource.DecodeObject(data, &value, &value.Metadata); err != nil {
		return err
	}
	*n = Notification(value)
	return nil
}

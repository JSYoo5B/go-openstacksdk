// Package softwaredeployments manages Heat software deployments. Gophercloud
// v2.15.0 has no binding for /software_deployments, so this package is SDK-owned.
package softwaredeployments

import (
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// SoftwareDeployment is one Heat software deployment. InputValues and
// OutputValues keep their exact JSON object text. Metadata.Body keeps every
// original field, including those not declared here.
type SoftwareDeployment struct {
	resource.Metadata
	ID                 string          `json:"id"`
	Action             string          `json:"action"`
	ConfigID           string          `json:"config_id"`
	InputValues        json.RawMessage `json:"input_values"`
	OutputValues       json.RawMessage `json:"output_values"`
	ServerID           string          `json:"server_id"`
	StackUserProjectID string          `json:"stack_user_project_id"`
	Status             string          `json:"status"`
	StatusReason       string          `json:"status_reason"`
	// CreationTime and UpdatedTime are the original wire timestamp strings
	// (Python created_at and updated_at).
	CreationTime string `json:"creation_time"`
	UpdatedTime  string `json:"updated_time"`
}

func (d *SoftwareDeployment) UnmarshalJSON(data []byte) error {
	type plain SoftwareDeployment
	var value plain
	if err := resource.DecodeObject(data, &value, &value.Metadata); err != nil {
		return err
	}
	*d = SoftwareDeployment(value)
	return nil
}

// Package nodes manages Senlin nodes and retains asynchronous submissions.
package nodes

import (
	"encoding/json"

	"gophercloudsdk/clustering/v1/actions"
	"gophercloudsdk/resource"
)

type Node struct {
	resource.Metadata
	ID           string                     `json:"id"`
	Name         string                     `json:"name"`
	PhysicalID   string                     `json:"physical_id"`
	ClusterID    *string                    `json:"cluster_id"`
	ProfileID    string                     `json:"profile_id"`
	ProfileName  string                     `json:"profile_name"`
	ProjectID    string                     `json:"project"`
	DomainID     *string                    `json:"domain"`
	UserID       string                     `json:"user"`
	Index        *json.Number               `json:"index"`
	Role         *string                    `json:"role"`
	InitAt       *string                    `json:"init_at"`
	Status       string                     `json:"status"`
	StatusReason string                     `json:"status_reason"`
	UserMetadata map[string]json.RawMessage `json:"metadata"`
	Data         map[string]json.RawMessage `json:"data"`
	Details      map[string]json.RawMessage `json:"details"`
	Dependents   map[string]json.RawMessage `json:"dependents"`
	Tainted      *bool                      `json:"tainted"`
	// Operation describes the accepted submission. It does not mean that the
	// physical node is created, updated or deleted, and performs no polling.
	Operation *actions.Submission `json:"-"`
}

func (value *Node) UnmarshalJSON(data []byte) error {
	type plain Node
	var decoded plain
	if err := resource.DecodeObject(data, &decoded, &decoded.Metadata); err != nil {
		return err
	}
	*value = Node(decoded)
	return nil
}

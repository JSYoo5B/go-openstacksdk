package clusterpolicies

import (
	"encoding/json"

	"gophercloudsdk/resource"
)

// ClusterPolicy distinguishes the binding UUID from its policy UUID and the
// canonical response parent from the fixed URI scope that fetched the row.
type ClusterPolicy struct {
	resource.Metadata
	ID           string                     `json:"id"`
	PolicyID     string                     `json:"policy_id"`
	ClusterID    string                     `json:"cluster_id"`
	ClusterName  string                     `json:"cluster_name"`
	PolicyName   string                     `json:"policy_name"`
	PolicyType   string                     `json:"policy_type"`
	IsEnabled    bool                       `json:"enabled"`
	Data         map[string]json.RawMessage `json:"data"`
	URIClusterID string                     `json:"-"`
}

func (value *ClusterPolicy) UnmarshalJSON(data []byte) error {
	type plain ClusterPolicy
	var decoded plain
	if err := resource.DecodeObject(data, &decoded, &decoded.Metadata); err != nil {
		return err
	}
	*value = ClusterPolicy(decoded)
	return nil
}

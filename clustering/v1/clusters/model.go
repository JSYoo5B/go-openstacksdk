// Package clusters manages Senlin clusters and preserves asynchronous submissions.
package clusters

import (
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Cluster keeps the server identity independent of the requested route and
// preserves exact JSON numbers and nullable timestamps in response metadata.
type Cluster struct {
	resource.Metadata
	ID              string                     `json:"id"`
	Name            string                     `json:"name"`
	ProfileID       string                     `json:"profile_id"`
	ProfileName     *string                    `json:"profile_name"`
	UserID          string                     `json:"user"`
	ProjectID       string                     `json:"project"`
	DomainID        *string                    `json:"domain"`
	InitializedAt   *string                    `json:"init_at"`
	MinSize         *json.Number               `json:"min_size"`
	MaxSize         *json.Number               `json:"max_size"`
	DesiredCapacity *json.Number               `json:"desired_capacity"`
	Timeout         *json.Number               `json:"timeout"`
	Status          string                     `json:"status"`
	StatusReason    string                     `json:"status_reason"`
	Config          map[string]json.RawMessage `json:"config"`
	UserMetadata    map[string]json.RawMessage `json:"metadata"`
	Data            map[string]json.RawMessage `json:"data"`
	Dependents      map[string]json.RawMessage `json:"dependents"`
	NodeIDs         []string                   `json:"nodes"`
	PolicyIDs       []string                   `json:"policies"`
	ProfileOnly     *bool                      `json:"profile_only"`
	// Operation is the accepted request's action reference, when supplied. It
	// does not imply that the cluster operation has completed.
	Operation *actions.Submission `json:"-"`
}

func (value *Cluster) UnmarshalJSON(data []byte) error {
	type plain Cluster
	var decoded plain
	if err := resource.DecodeObject(data, &decoded, &decoded.Metadata); err != nil {
		return err
	}
	*value = Cluster(decoded)
	return nil
}

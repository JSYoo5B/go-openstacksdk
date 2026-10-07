package clusterattributes

import (
	"bytes"
	"encoding/json"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// ClusterAttribute retains a value of any JSON type. A nil Value means omitted;
// explicit null is the bytes "null". URI fields describe the fixed read scope.
type ClusterAttribute struct {
	resource.Metadata
	NodeID       string          `json:"id"`
	Value        json.RawMessage `json:"value"`
	URIClusterID string          `json:"-"`
	URIPath      string          `json:"-"`
}

func (value *ClusterAttribute) UnmarshalJSON(data []byte) error {
	type plain ClusterAttribute
	var decoded plain
	if err := resource.DecodeObject(data, &decoded, &decoded.Metadata); err != nil {
		return err
	}
	decoded.Value = bytes.Clone(decoded.Body["value"])
	*value = ClusterAttribute(decoded)
	return nil
}

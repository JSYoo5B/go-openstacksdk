// Package chassis manages Ironic chassis through SDK-owned /v1/chassis routes.
// The pinned Gophercloud release has no binding for this resource.
package chassis

import (
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Chassis retains the typed server fields, the original response object in Body,
// and the HTTP header and status code of the response that produced it.
type Chassis struct {
	resource.Metadata
	UUID        string          `json:"uuid"`
	Description *string         `json:"description"`
	Extra       map[string]any  `json:"extra"`
	Nodes       []resource.Link `json:"nodes"`
}

func (v *Chassis) UnmarshalJSON(data []byte) error {
	type plain Chassis
	var value plain
	if err := resource.DecodeObject(data, &value, &value.Metadata); err != nil {
		return err
	}
	*v = Chassis(value)
	return nil
}

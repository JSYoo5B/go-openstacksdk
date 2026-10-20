package blockstorage

import (
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type GetVolumeTypeAccessRequest struct{ NameOrID string }

// VolumeTypeAccessRequest sends ProjectID literally, including an empty or
// nonexistent project ID. The workflow resolves only the volume type.
type VolumeTypeAccessRequest struct{ NameOrID, ProjectID string }

// GetVolumeTypeAccessResult keeps lookup and access response evidence separate.
// Value preserves any JSON value; a missing volume_type_access field gives [].
// Accesses is an optional owned projection when every array item is an object.
// It adds no descriptors, location, project validation or pagination.
type GetVolumeTypeAccessResult struct {
	Resolved *GetVolumeTypeResult
	TypeID   string
	Value    json.RawMessage
	Accesses []*resource.RawResource
	Observed *VolumeTypesPage
}

// VolumeTypeAccessActionResult exposes an actual accepted acknowledgement.
// Applied may survive an error reading or closing the response. It establishes
// neither a membership change nor completion of the server's operation.
type VolumeTypeAccessActionResult struct {
	Resolved          *GetVolumeTypeResult
	TypeID, ProjectID string
	Applied           *VolumeTypesPage
}

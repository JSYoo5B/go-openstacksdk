package blockstorage

import (
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ListVolumesResult commits a full owned normalized array and original wire
// rows only after success. Pages preserve actual admitted responses on errors.
type ListVolumesResult struct {
	Value   json.RawMessage
	Volumes []*resource.RawResource
	Pages   []*GetVolumesPage
}

// VolumeExistsResult distinguishes a completed true/false lookup from failure.
// Exists is nil on an error. Value/Volume describe an actual found resource;
// Observed and Pages preserve the member/list responses used by the lookup.
type VolumeExistsResult struct {
	Exists   *bool
	Value    json.RawMessage
	Volume   *resource.RawResource
	Observed *GetVolumesPage
	Pages    []*GetVolumesPage
}

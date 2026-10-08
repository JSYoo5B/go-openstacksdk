package blockstorage

import (
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// GetVolumeIDResult owns the response ID independently of the normalized Value,
// original Volume and admitted response evidence. ID is nil on absence or error;
// a found null or missing ID is the literal JSON null. Volume distinguishes that
// found resource from absence. Observed and Pages preserve actual lookup reads.
type GetVolumeIDResult struct {
	ID       json.RawMessage
	Value    json.RawMessage
	Volume   *resource.RawResource
	Observed *GetVolumesPage
	Pages    []*GetVolumesPage
}

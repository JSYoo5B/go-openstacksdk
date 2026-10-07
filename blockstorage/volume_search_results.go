package blockstorage

import (
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// SearchVolumesResult owns the complete normalized result. Value can be any
// JSON shape. Volumes carries original wire rows only for ordinary selection;
// an expression never fabricates a source-row association. Pages survive errors.
type SearchVolumesResult struct {
	Value   json.RawMessage
	Volumes []*resource.RawResource
	Pages   []*GetVolumesPage
}

// GetVolumeResult distinguishes absent/null Value from selected false, zero or
// empty string. Volume is an actual source row only for ordinary selection.
// Observed preserves an admitted member GET; Pages preserve list responses.
type GetVolumeResult struct {
	Value    json.RawMessage
	Volume   *resource.RawResource
	Observed *GetVolumesPage
	Pages    []*GetVolumesPage
}

// VolumeSelectionError reports len(value)>1 without inventing resource IDs for
// an arbitrary expression result. It participates in shared ErrAmbiguous checks.
type VolumeSelectionError struct {
	NameOrID string
	Length   int
}

func (e *VolumeSelectionError) Error() string {
	return fmt.Sprintf("volume %q has multiple matches (length %d)", e.NameOrID, e.Length)
}
func (e *VolumeSelectionError) Unwrap() error { return resource.ErrAmbiguous }

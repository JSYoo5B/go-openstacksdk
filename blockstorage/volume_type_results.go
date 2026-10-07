package blockstorage

import (
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// VolumeTypesPage owns one admitted physical response independently of rows.
type VolumeTypesPage = GetVolumesPage

// ListVolumeTypesResult commits a full normalized array and wire rows only
// after complete success. Actual admitted Pages remain available on errors.
type ListVolumeTypesResult struct {
	Value json.RawMessage
	Types []*resource.RawResource
	Pages []*VolumeTypesPage
}

// SearchVolumeTypesResult may contain any JSON expression result. Types retains
// original rows only for ordinary selection, without invented associations.
type SearchVolumeTypesResult struct {
	Value json.RawMessage
	Types []*resource.RawResource
	Pages []*VolumeTypesPage
}

// GetVolumeTypeResult distinguishes absence from selected false, zero and empty
// string. Type belongs to an actual ordinary source row; proof survives errors.
type GetVolumeTypeResult struct {
	Value    json.RawMessage
	Type     *resource.RawResource
	Observed *VolumeTypesPage
	Pages    []*VolumeTypesPage
}

type VolumeTypeSelectionError struct {
	NameOrID string
	Length   int
}

func (e *VolumeTypeSelectionError) Error() string {
	return fmt.Sprintf("volume type %q has multiple matches (length %d)", e.NameOrID, e.Length)
}
func (e *VolumeTypeSelectionError) Unwrap() error { return resource.ErrAmbiguous }

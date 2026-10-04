package blockstorage

import (
	"context"
	"encoding/json"
)

// collectViews normalizes every actual row before continuation. Failure keeps
// physical pages but never commits a partially normalized logical list.
func (p *preparedVolumeSearch) collectViews(ctx context.Context) ([]getVolumesEntry, []json.RawMessage, *GetVolumesResult, error) {
	proof := &GetVolumesResult{}
	entries := make([]getVolumesEntry, 0)
	views := make([]json.RawMessage, 0)
	err := p.reader.readVolumes(ctx, proof, func(entry getVolumesEntry) (bool, error) {
		view, err := p.view(ctx, entry)
		if err != nil {
			return false, err
		}
		entries = append(entries, entry)
		views = append(views, view)
		return true, nil
	})
	if err != nil {
		return nil, nil, proof, err
	}
	return entries, views, proof, nil
}

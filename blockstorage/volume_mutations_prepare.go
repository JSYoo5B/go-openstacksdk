package blockstorage

import (
	"context"
	"encoding/json"
	"fmt"

	"gophercloudsdk/internal/cloudlocation"
	"gophercloudsdk/resource"
)

// Snapshot scope once after original options, before either physical stage.
// The reader's source and route stay captured while authentication remains live.
func prepareVolumeMutationRead(ctx context.Context, reader *preparedGetVolumes, location *resource.CloudLocation) (*preparedVolumeSearch, error) {
	if err := reader.guard(ctx); err != nil {
		return nil, err
	}
	var owned resource.CloudLocation
	if location != nil {
		owned = location.Clone()
	} else {
		project, err := cloudlocation.ProjectID(reader.cinder.provider)
		if err != nil {
			return nil, err
		}
		owned.Project.ID = project
	}
	p := &preparedVolumeSearch{reader: reader, options: VolumeSearchOpts{Location: &owned}}
	return p, reader.guard(ctx)
}

func resolveVolumeMutation(ctx context.Context, p *preparedVolumeSearch, nameOrID string) (*GetVolumeResult, *volumeIdentityRecord, string, error) {
	resolved, selected, err := p.findSelected(ctx, nameOrID)
	if err != nil {
		return resolved, nil, "", err
	}
	if resolved.Volume == nil {
		return resolved, nil, "", &resource.NotFoundError{Resource: "volume", Reference: nameOrID}
	}
	var id string
	if err := json.Unmarshal(resolved.Volume.Body["id"], &id); err != nil {
		return resolved, selected, "", fmt.Errorf("%w: resolved volume ID must be a nonempty string: %w", resource.ErrInvalidOption, err)
	}
	if err := validateAttachRef(resource.ID(id)); err != nil {
		return resolved, selected, "", fmt.Errorf("resolved volume ID: %w", err)
	}
	return resolved, selected, id, p.reader.guard(ctx)
}

package blockstorage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Resolve once through the captured ordinary type reader. The second phase
// uses this same source, so original options and lookup are never replayed.
func prepareVolumeTypeAccess(ctx context.Context, cinder *gophercloud.ServiceClient, nameOrID string, options []VolumeTypeReadOption) (*preparedVolumeTypes, *GetVolumeTypeResult, string, error) {
	p, err := captureVolumeTypesRead(ctx, cinder, options)
	if err != nil {
		return nil, nil, "", err
	}
	resolved, err := p.find(ctx, nameOrID)
	if err != nil {
		return p, resolved, "", err
	}
	if resolved.Type == nil {
		return p, resolved, "", &resource.NotFoundError{Resource: "volume type", Reference: nameOrID}
	}
	// A fresh canonical response never inherits the supplied lookup identity.
	// Python's seeded Resource and str(id or '') route policy are deliberately
	// replaced by a validated actual response ID for this fixed second phase.
	var id string
	if err := json.Unmarshal(resolved.Type.Body["id"], &id); err != nil {
		return p, resolved, "", fmt.Errorf("%w: resolved volume type ID must be a nonempty string: %w", resource.ErrInvalidOption, err)
	}
	if err := validateAttachRef(resource.ID(id)); err != nil {
		return p, resolved, "", fmt.Errorf("resolved volume type ID: %w", err)
	}
	return p, resolved, id, p.reader.guard(ctx)
}

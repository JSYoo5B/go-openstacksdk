package cinderaction

import (
	"context"
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// ValidateSnapshotID validates body text, without imposing route-ID rules.
func ValidateSnapshotID(id string) error {
	if !utf8.ValidString(id) {
		return fmt.Errorf("%w: snapshot ID body value must be UTF-8", resource.ErrInvalidOption)
	}
	return nil
}

// RevertToSnapshot sends one literal snapshot value after advertised support.
func RevertToSnapshot(ctx context.Context, client *gophercloud.ServiceClient, id, snapshotID string) (*Result, error) {
	const operation = "RevertVolumeToSnapshot"
	source, err := captureAction(ctx, client, id)
	if err == nil {
		err = ValidateSnapshotID(snapshotID)
	}
	if err != nil {
		return nil, WrapOperation(ctx, operation, err)
	}
	body, _ := json.Marshal(map[string]any{"revert": map[string]string{"snapshot_id": snapshotID}})
	return applyRequired(ctx, source, id, operation, body, "3.40")
}

// Unmanage keeps the SDK null action separate from the native empty object.
func Unmanage(ctx context.Context, client *gophercloud.ServiceClient, id string) (*Result, error) {
	const operation = "UnmanageVolume"
	source, err := captureAction(ctx, client, id)
	if err != nil {
		return nil, WrapOperation(ctx, operation, err)
	}
	body, _ := json.Marshal(map[string]any{"os-unmanage": nil})
	return applyPrepared(ctx, source, id, operation, body)
}

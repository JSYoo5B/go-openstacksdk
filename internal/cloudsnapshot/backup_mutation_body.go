package cloudsnapshot

import (
	"encoding/json"
	"unicode/utf8"
)

func ValidateBackupCreateInput(volumeID string, policy BackupCreateOptions) error {
	_, _, err := compileBackupCreateBody(volumeID, cloneBackupCreate(policy))
	return err
}

func compileBackupCreateBody(volumeID string, policy BackupCreateOptions) (json.RawMessage, backupMutationState, error) {
	var state backupMutationState
	for _, value := range []*string{&volumeID, policy.Name, policy.Description, policy.SnapshotID} {
		if value != nil && !utf8.ValidString(*value) {
			return nil, state, invalid("backup creation text must be UTF-8")
		}
	}
	force, incremental := false, false
	if policy.Force != nil {
		force = *policy.Force
	}
	if policy.Incremental != nil {
		incremental = *policy.Incremental
	}
	// Cloud input presence is explicit even for defaults. Resource construction
	// keeps is_incremental; only the POST body transforms it to incremental.
	seed, err := json.Marshal(map[string]any{
		"volume_id": volumeID, "name": policy.Name, "description": policy.Description,
		"force": force, "is_incremental": incremental, "snapshot_id": policy.SnapshotID,
	})
	if err != nil {
		return nil, state, err
	}
	if err := state.overlay(seed); err != nil {
		return nil, state, err
	}
	body, err := json.Marshal(map[string]any{"backup": map[string]any{
		"volume_id": volumeID, "name": policy.Name, "description": policy.Description,
		"force": force, "incremental": incremental, "snapshot_id": policy.SnapshotID,
	}})
	return body, state, err
}

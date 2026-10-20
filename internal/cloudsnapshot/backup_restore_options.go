package cloudsnapshot

import (
	"bytes"
	"context"
	"encoding/json"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Seed supplies cached Backup fields; its ID must match the fixed target.
// Empty optional strings are omitted, following Backup.restore truthiness.
type BackupRestoreOptions struct {
	VolumeID, Name *string
	Seed           json.RawMessage
	Location       *resource.CloudLocation
}
type BackupRestoreOption = Option[BackupRestoreOptions]

func WithBackupRestoreOptions(value BackupRestoreOptions) BackupRestoreOption {
	return withValue(value, cloneBackupRestore)
}
func WithBackupRestoreVolumeID(value string) BackupRestoreOption {
	return func(target *BackupRestoreOptions) error { target.VolumeID = clonePointer(&value); return nil }
}
func WithBackupRestoreName(value string) BackupRestoreOption {
	return func(target *BackupRestoreOptions) error { target.Name = clonePointer(&value); return nil }
}
func WithBackupRestoreSeed(value json.RawMessage) BackupRestoreOption {
	owned := bytes.Clone(value)
	return func(target *BackupRestoreOptions) error { target.Seed = bytes.Clone(owned); return nil }
}
func WithBackupRestoreLocation(value resource.CloudLocation) BackupRestoreOption {
	owned := value.Clone()
	return func(target *BackupRestoreOptions) error { target.Location = cloneLocation(&owned); return nil }
}
func PrepareBackupRestore(ctx context.Context, options ...BackupRestoreOption) (BackupRestoreOptions, error) {
	return prepare(ctx, options, cloneBackupRestore, nil)
}
func cloneBackupRestore(value BackupRestoreOptions) BackupRestoreOptions {
	value.VolumeID = clonePointer(value.VolumeID)
	value.Name = clonePointer(value.Name)
	value.Seed = bytes.Clone(value.Seed)
	value.Location = cloneLocation(value.Location)
	return value
}

func ValidateBackupRestoreInput(id string, policy BackupRestoreOptions) error {
	_, _, err := compileBackupRestore(id, cloneBackupRestore(policy))
	return err
}
func compileBackupRestore(id string, policy BackupRestoreOptions) (json.RawMessage, backupMutationState, error) {
	var state backupMutationState
	if err := ValidateBackupExportID(id); err != nil {
		return nil, state, err
	}
	body := map[string]string{}
	for key, value := range map[string]*string{"volume_id": policy.VolumeID, "name": policy.Name} {
		if value == nil {
			continue
		}
		if !utf8.ValidString(*value) {
			return nil, state, invalid("backup restore text must be UTF-8")
		}
		if *value != "" {
			body[key] = *value
		}
	}
	if len(body) == 0 {
		return nil, state, invalid("either name or volume_id must be specified")
	}
	seed, _ := json.Marshal(map[string]string{"id": id})
	if err := state.overlay(seed); err != nil {
		return nil, state, err
	}
	if policy.Seed != nil {
		if !utf8.Valid(policy.Seed) {
			return nil, state, invalid("backup restore seed must be UTF-8 JSON")
		}
		if err := state.overlay(policy.Seed); err != nil {
			return nil, state, err
		}
		seedID, err := state.routeID()
		if err != nil || seedID != id {
			return nil, state, invalid("backup restore seed ID must match the fixed backup ID")
		}
	}
	encoded, err := json.Marshal(map[string]any{"restore": body})
	return encoded, state, err
}

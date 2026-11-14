package blockstorage_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups"
)

func Example_volumeBackupActions() {
	if err := runVolumeBackupActions(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runVolumeBackupActions() error {
	cloudName, backupID := os.Getenv("CLOUD_NAME"), os.Getenv("BACKUP_ID")
	if cloudName == "" || backupID == "" {
		return fmt.Errorf("CLOUD_NAME and BACKUP_ID are required")
	}
	path, action := os.Getenv("BACKUP_PATH"), os.Getenv("BACKUP_ACTION")
	if path == "" {
		path = "connection"
	}
	if action == "" {
		action = "restore"
	}
	switch path {
	case "connection", "direct", "service", "native":
	default:
		return fmt.Errorf("unknown BACKUP_PATH %q", path)
	}
	volumeID, name := os.Getenv("RESTORE_VOLUME_ID"), os.Getenv("RESTORE_NAME")
	status, statusPresent := os.LookupEnv("BACKUP_STATUS")
	switch action {
	case "restore":
		if volumeID == "" && name == "" {
			return fmt.Errorf("RESTORE_VOLUME_ID or RESTORE_NAME is required")
		}
	case "reset":
		if !statusPresent {
			return fmt.Errorf("BACKUP_STATUS must be defined; an empty value is allowed")
		}
	default:
		return fmt.Errorf("unknown BACKUP_ACTION %q", action)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
	if err != nil {
		return err
	}
	if action == "reset" {
		input := blockstorage.ResetVolumeBackupStatusRequest{BackupID: backupID, Status: status}
		if path == "connection" {
			result, err := conn.ResetVolumeBackupStatus(ctx, input)
			inspectResetVolumeBackupActions(result)
			return err
		}
		service, err := conn.BlockStorageV3(ctx)
		if err != nil {
			return err
		}
		switch path {
		case "direct":
			result, err := blockstorage.ResetVolumeBackupStatus(ctx, service.RawClient(), input)
			inspectResetVolumeBackupActions(result)
			return err
		case "service":
			result, err := service.Backups.ResetBackupStatus(ctx, backupID, status)
			inspectResetVolumeBackupActions(result)
			return err
		default:
			return service.Backups.ResetStatus(ctx, backupID, backups.ResetStatusOpts{Status: status})
		}
	}
	options := []blockstorage.RestoreVolumeBackupOption{
		blockstorage.WithRestoreVolumeBackupVolumeID(volumeID),
		blockstorage.WithRestoreVolumeBackupName(name),
	}
	if seed, present := os.LookupEnv("BACKUP_SEED"); present {
		options = append(options, blockstorage.WithRestoreVolumeBackupSeed(json.RawMessage(seed)))
	}
	policy, err := blockstorage.PrepareRestoreVolumeBackupOptions(ctx, options...)
	if err != nil {
		return err
	}
	input := blockstorage.RestoreVolumeBackupRequest{BackupID: backupID}
	if path == "connection" {
		result, err := conn.RestoreVolumeBackup(ctx, input, blockstorage.WithRestoreVolumeBackupOptions(policy))
		inspectRestoreVolumeBackupActions(result)
		return err
	}
	service, err := conn.BlockStorageV3(ctx)
	if err != nil {
		return err
	}
	if path == "native" {
		result, err := service.Backups.RestoreFromBackup(ctx, backupID, backups.RestoreOpts{VolumeID: volumeID, Name: name})
		if result != nil {
			fmt.Println("native restore:", result.BackupID, result.VolumeID, result.VolumeName)
		}
		return err
	}
	location, err := conn.CurrentLocation()
	if err != nil {
		return err
	}
	policy, err = blockstorage.PrepareRestoreVolumeBackupOptions(ctx,
		blockstorage.WithRestoreVolumeBackupOptions(policy),
		func(c *blockstorage.RestoreVolumeBackupOpts) error {
			loc := location.Clone()
			c.Location = &loc
			return nil
		})
	if err != nil {
		return err
	}
	if path == "direct" {
		result, err := blockstorage.RestoreVolumeBackup(ctx, service.RawClient(), input, blockstorage.WithRestoreVolumeBackupOptions(policy))
		inspectRestoreVolumeBackupActions(result)
		return err
	}
	result, err := service.Backups.RestoreBackup(ctx, backupID, backups.WithRestoreBackupOptions(policy))
	inspectRestoreVolumeBackupActions(result)
	return err
}

func inspectRestoreVolumeBackupActions(result *blockstorage.RestoreVolumeBackupResult) {
	if result == nil {
		return
	}
	fmt.Println("merged logical backup ID JSON:", string(result.BackupID))
	inspectAppliedVolumeBackupActions(result.Applied)
	if result.Backup != nil {
		actual := result.Backup.Clone()
		fmt.Println("actual selected fields:", len(actual.Body), "physical backup_id:", string(actual.Body["backup_id"]))
	}
	if result.Value != nil {
		formatted, err := json.MarshalIndent(result.Value, "", "  ")
		if err == nil {
			fmt.Println("merged nullable Backup:", string(formatted))
		}
	}
}

func inspectResetVolumeBackupActions(result *blockstorage.ResetVolumeBackupStatusResult) {
	if result == nil {
		return
	}
	fmt.Println("requested backup ID:", result.BackupID, "helper completed:", result.Completed)
	inspectAppliedVolumeBackupActions(result.Applied)
}

func inspectAppliedVolumeBackupActions(page *blockstorage.VolumeBackupMutationPage) {
	if page != nil {
		fmt.Println("admitted HTTP:", page.StatusCode, "actual body bytes:", len(page.Body))
	}
}

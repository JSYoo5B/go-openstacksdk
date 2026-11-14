package blockstorage_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func InspectVolumeBackupExportRecord(response *blockstorage.VolumeBackupExportResponse, value json.RawMessage, err error) {
	if response != nil {
		fmt.Println("export HTTP:", response.StatusCode, "bytes:", len(response.Body))
		fmt.Println("content type:", response.Header.Get("Content-Type"))
	}
	if err != nil {
		fmt.Println("export error:", err)
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			fmt.Println("parent context interrupted export")
		}
		var physical *resource.ResponseError
		if errors.As(err, &physical) {
			fmt.Println("accepted error proof:", physical.StatusCode, len(physical.Body))
		}
		return
	}
	if value != nil {
		// Literal null is successful nonnil bytes "null"; nil means no Value.
		fmt.Printf("JSON record: %s\n", value)
	}
}

func ConfiguredRecordVolumeBackupExportRecord(parent context.Context, cloudName, backupID string) (*blockstorage.ExportVolumeBackupRecordResult, error) {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
	if err != nil {
		return nil, err
	}
	result, err := conn.ExportVolumeBackupRecord(ctx,
		blockstorage.ExportVolumeBackupRecordRequest{BackupID: backupID})
	if result != nil {
		InspectVolumeBackupExportRecord(result.Exported, result.Value, err)
	}
	return result, err
}

func ServiceRecordVolumeBackupExportRecord(ctx context.Context, conn *sdk.Connection, backupID string) (*backups.ExportBackupResult, error) {
	cinder, err := conn.BlockStorageV3(ctx)
	if err != nil {
		return nil, err
	}
	result, err := cinder.Backups.ExportBackup(ctx, backupID)
	if result != nil {
		InspectVolumeBackupExportRecord(result.Exported, result.Value, err)
	}
	return result, err
}

func ServiceRawVolumeBackupExportRecord(ctx context.Context, conn *sdk.Connection, backupID string) (*backups.ExportRecordResult, error) {
	cinder, err := conn.BlockStorageV3(ctx)
	if err != nil {
		return nil, err
	}
	result, err := cinder.Backups.ExportRecord(ctx, backupID)
	if result != nil {
		InspectVolumeBackupExportRecord(result.Exported, nil, err)
	}
	return result, err
}

func DirectRecordVolumeBackupExportRecord(ctx context.Context, cinder *gophercloud.ServiceClient, backupID string) (*blockstorage.ExportVolumeBackupRecordResult, error) {
	result, err := blockstorage.ExportVolumeBackupRecord(ctx, cinder,
		blockstorage.ExportVolumeBackupRecordRequest{BackupID: backupID})
	if result != nil {
		InspectVolumeBackupExportRecord(result.Exported, result.Value, err)
	}
	return result, err
}

func DirectServiceRecordVolumeBackupExportRecord(ctx context.Context, cinder *gophercloud.ServiceClient, backupID string) (*backups.ExportBackupResult, error) {
	return backups.New(cinder).ExportBackup(ctx, backupID)
}

// Native typed Export is an alternative for the native BackupRecord schema.
func NativeTypedVolumeBackupExportRecord(ctx context.Context, conn *sdk.Connection, backupID string) (*backups.BackupRecord, error) {
	cinder, err := conn.BlockStorageV3(ctx)
	if err != nil {
		return nil, err
	}
	return cinder.Backups.Export(ctx, backupID)
}

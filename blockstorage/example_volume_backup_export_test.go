package blockstorage_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func AdoptVolumeBackupExport(provider *gophercloud.ProviderClient) (*sdk.Connection, error) {
	return sdk.FromProvider(provider, sdk.WithRegion("RegionOne"))
}

func InspectVolumeBackupExport(result *blockstorage.ExportVolumeBackupResult, err error) {
	if result != nil {
		fmt.Println("requested backup ID:", result.BackupID)
		if result.Exported != nil {
			response := result.Exported
			fmt.Println("export HTTP:", response.StatusCode, "bytes:", len(response.Body))
			fmt.Println("content type:", response.Header.Get("Content-Type"))
		}
	}
	if err == nil {
		return
	}
	fmt.Println("export error:", err)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		fmt.Println("parent context interrupted export")
	}
	if errors.Is(err, resource.ErrInvalidOption) {
		fmt.Println("invalid local input or changed request source")
	}
}

func ConfiguredExportVolumeBackupExport(parent context.Context, cloudName, backupID string) (*blockstorage.ExportVolumeBackupResult, error) {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
	if err != nil {
		return nil, err
	}
	result, err := conn.ExportVolumeBackup(ctx,
		blockstorage.ExportVolumeBackupRequest{BackupID: backupID})
	InspectVolumeBackupExport(result, err)
	return result, err
}

func DirectExportVolumeBackupExport(ctx context.Context, cinder *gophercloud.ServiceClient, backupID string) (*blockstorage.ExportVolumeBackupResult, error) {
	result, err := blockstorage.ExportVolumeBackup(ctx, cinder,
		blockstorage.ExportVolumeBackupRequest{BackupID: backupID})
	InspectVolumeBackupExport(result, err)
	return result, err
}

// Optional application policy: call after a successful export only when
// the application expects JSON. An opaque export need not contain JSON.
func ApplicationJSONVolumeBackupExport(result *blockstorage.ExportVolumeBackupResult) (json.RawMessage, error) {
	if result == nil || result.Exported == nil {
		return nil, fmt.Errorf("no admitted export response")
	}
	var value json.RawMessage
	if err := json.Unmarshal(result.Exported.Body, &value); err != nil {
		return nil, err
	}
	return value, nil
}

package blockstorage_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_volumeBackupImport() {
	if err := runVolumeBackupImport(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runVolumeBackupImport() error {
	cloudName := os.Getenv("CLOUD_NAME")
	if cloudName == "" {
		return fmt.Errorf("CLOUD_NAME is required")
	}
	backupService, servicePresent := os.LookupEnv("BACKUP_SERVICE")
	backupURL, urlPresent := os.LookupEnv("BACKUP_URL")
	if !servicePresent || !urlPresent {
		return fmt.Errorf("BACKUP_SERVICE and BACKUP_URL must be defined; empty values are allowed")
	}
	path := os.Getenv("BACKUP_IMPORT_PATH")
	if path == "" {
		path = "connection"
	}
	switch path {
	case "connection", "direct", "service":
	default:
		return fmt.Errorf("unknown BACKUP_IMPORT_PATH %q", path)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
	if err != nil {
		return err
	}
	input := blockstorage.ImportVolumeBackupRequest{
		BackupService: backupService,
		BackupURL:     backupURL,
	}
	var result *blockstorage.ImportVolumeBackupResult
	if path == "connection" {
		result, err = conn.ImportVolumeBackup(ctx, input)
	} else {
		cinder, selectErr := conn.BlockStorageV3(ctx)
		if selectErr != nil {
			return selectErr
		}
		if path == "direct" {
			result, err = blockstorage.ImportVolumeBackup(ctx, cinder.RawClient(), input)
		} else {
			result, err = cinder.Backups.ImportBackup(ctx, backupService, backupURL)
		}
	}
	inspectVolumeBackupImport(result, err)
	return err
}

func inspectVolumeBackupImport(result *blockstorage.ImportVolumeBackupResult, err error) {
	if result != nil {
		fmt.Printf("logical backup ID JSON: %s\n", result.BackupID)
		fmt.Printf("import microversion: %q\n", result.Microversion)
		for i, page := range result.Discovery {
			if page != nil {
				fmt.Println("admitted discovery response:", i, page.StatusCode, len(page.Body))
			}
		}
		if result.Applied != nil {
			fmt.Println("admitted import response:", result.Applied.StatusCode, len(result.Applied.Body))
		}
		if result.Backup != nil {
			actual := result.Backup.Clone()
			fmt.Println("actual selected fields:", len(actual.Body))
			fmt.Printf("actual response ID JSON: %s\n", actual.Body["id"])
		}
	}
	if err != nil {
		fmt.Println("import error:", err)
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			fmt.Println("parent context interrupted import")
		}
		var physical *resource.ResponseError
		if errors.As(err, &physical) {
			fmt.Println("current admitted error response:", physical.StatusCode, len(physical.Body))
		}
		return
	}
	if result != nil && result.Value != nil {
		pretty, formatErr := json.MarshalIndent(result.Value, "", "  ")
		if formatErr == nil {
			fmt.Println("nullable Backup:", string(pretty))
		}
	}
}

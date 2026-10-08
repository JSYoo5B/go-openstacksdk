package cloudsnapshot

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Backup results expose backup rows, never snapshot rows. The shared reader's
// private collection is transferred once to its resource-specific result.
type BackupListResult struct {
	Value   json.RawMessage
	Backups []*resource.RawResource
	Pages   []*Page
}

type BackupSearchResult struct {
	Value   json.RawMessage
	Backups []*resource.RawResource
	Pages   []*Page
}

type BackupResult struct {
	Value       json.RawMessage
	Backup      *resource.RawResource
	Observed    *Page
	Pages       []*Page
	RequestedID string
	SeededID    bool
}

type BackupSelectionError struct {
	NameOrID string
	Length   int
}

func (e *BackupSelectionError) Error() string {
	return fmt.Sprintf("multiple backup matches found for %q: length %d", e.NameOrID, e.Length)
}
func (e *BackupSelectionError) Unwrap() error { return resource.ErrAmbiguous }

func ListBackups(ctx context.Context, client *gophercloud.ServiceClient, options ...ListOption) (*BackupListResult, error) {
	value, err := listResource(ctx, client, backupReadSchema(), options...)
	if value == nil {
		return nil, err
	}
	return &BackupListResult{Value: value.Value, Backups: value.Snapshots, Pages: value.Pages}, err
}

func SearchBackups(ctx context.Context, client *gophercloud.ServiceClient, nameOrID string, options ...SearchOption) (*BackupSearchResult, error) {
	value, err := searchResource(ctx, client, backupReadSchema(), nameOrID, options...)
	if value == nil {
		return nil, err
	}
	return &BackupSearchResult{Value: value.Value, Backups: value.Snapshots, Pages: value.Pages}, err
}

func GetBackup(ctx context.Context, client *gophercloud.ServiceClient, nameOrID string, options ...SearchOption) (*BackupResult, error) {
	value, err := getResource(ctx, client, backupReadSchema(), nameOrID, options...)
	if value == nil {
		return nil, err
	}
	return &BackupResult{Value: value.Value, Backup: value.Snapshot, Observed: value.Observed, Pages: value.Pages, RequestedID: value.RequestedID, SeededID: value.SeededID}, err
}

func ValidateBackupListOptions(value ListOptions) error {
	_, err := compileListFor(cloneList(value), backupReadSchema())
	return err
}

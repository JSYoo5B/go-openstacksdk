// Package cloudbackup provides named backup results over the shared Cinder reader.
package cloudbackup

import (
	"context"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudsnapshot"
	"github.com/gophercloud/gophercloud/v2"
)

type Page = cloudsnapshot.Page
type ListResult = cloudsnapshot.BackupListResult
type SearchResult = cloudsnapshot.BackupSearchResult
type Result = cloudsnapshot.BackupResult
type SelectionError = cloudsnapshot.BackupSelectionError

func List(ctx context.Context, client *gophercloud.ServiceClient, options ...cloudsnapshot.ListOption) (*ListResult, error) {
	return cloudsnapshot.ListBackups(ctx, client, options...)
}
func Search(ctx context.Context, client *gophercloud.ServiceClient, nameOrID string, options ...cloudsnapshot.SearchOption) (*SearchResult, error) {
	return cloudsnapshot.SearchBackups(ctx, client, nameOrID, options...)
}
func Get(ctx context.Context, client *gophercloud.ServiceClient, nameOrID string, options ...cloudsnapshot.SearchOption) (*Result, error) {
	return cloudsnapshot.GetBackup(ctx, client, nameOrID, options...)
}

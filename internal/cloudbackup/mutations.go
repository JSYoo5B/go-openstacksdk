package cloudbackup

import (
	"context"
	"github.com/JSYoo5B/gophercloudsdk/internal/cloudsnapshot"
	"github.com/gophercloud/gophercloud/v2"
)

type MutationPage = cloudsnapshot.MutationPage
type CreateResult = cloudsnapshot.BackupCreateResult
type DeleteResult = cloudsnapshot.BackupDeleteResult
type WaitTimeoutError = cloudsnapshot.BackupWaitTimeoutError

func Create(ctx context.Context, client *gophercloud.ServiceClient, volumeID string, options ...cloudsnapshot.BackupCreateOption) (*CreateResult, error) {
	return cloudsnapshot.CreateBackup(ctx, client, volumeID, options...)
}
func Delete(ctx context.Context, client *gophercloud.ServiceClient, nameOrID string, options ...cloudsnapshot.BackupDeleteOption) (*DeleteResult, error) {
	return cloudsnapshot.DeleteBackup(ctx, client, nameOrID, options...)
}

package blockstorage

import "github.com/JSYoo5B/go-openstacksdk/internal/cloudbackup"

// VolumeBackupMutationPage owns actual mutation or polling body/header/status.
type VolumeBackupMutationPage = cloudbackup.MutationPage

// CreateVolumeBackupResult separates request-seeded logical stages and HTTP proof.
type CreateVolumeBackupResult = cloudbackup.CreateResult

// DeleteVolumeBackupResult retains lookup, acknowledgement and optional wait proof.
type DeleteVolumeBackupResult = cloudbackup.DeleteResult

// BackupWaitTimeoutError unwraps context.DeadlineExceeded for SDK loop expiry.
type BackupWaitTimeoutError = cloudbackup.WaitTimeoutError

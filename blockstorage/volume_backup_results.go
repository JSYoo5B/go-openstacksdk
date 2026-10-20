package blockstorage

import "github.com/JSYoo5B/go-openstacksdk/internal/cloudbackup"

// VolumeBackupsPage owns an actual admitted backup response.
type VolumeBackupsPage = cloudbackup.Page

// ListVolumeBackupsResult publishes a complete logical list and its raw rows.
type ListVolumeBackupsResult = cloudbackup.ListResult

// SearchVolumeBackupsResult separates expression output from raw rows.
type SearchVolumeBackupsResult = cloudbackup.SearchResult

// GetVolumeBackupResult retains selected logical values and physical proof.
type GetVolumeBackupResult = cloudbackup.Result

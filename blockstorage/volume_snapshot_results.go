package blockstorage

import "github.com/JSYoo5B/go-openstacksdk/internal/cloudsnapshot"

// VolumeSnapshotsPage keeps the actual admitted HTTP response independently.
type VolumeSnapshotsPage = cloudsnapshot.Page

// ListVolumeSnapshotsResult commits an owned complete list after success.
type ListVolumeSnapshotsResult = cloudsnapshot.ListResult

// SearchVolumeSnapshotsResult also supports arbitrary JSON expressions.
type SearchVolumeSnapshotsResult = cloudsnapshot.SearchResult

// GetVolumeSnapshotResult keeps logical selection and current physical proof
// separately, without creating server fields for a source-seeded missing ID.
type GetVolumeSnapshotResult = cloudsnapshot.Result

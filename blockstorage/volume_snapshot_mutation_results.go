package blockstorage

import "gophercloudsdk/internal/cloudsnapshot"

// VolumeSnapshotMutationPage owns an actual mutation or polling response.
type VolumeSnapshotMutationPage = cloudsnapshot.MutationPage

// CreateVolumeSnapshotResult separates logical merged stages and HTTP proof.
type CreateVolumeSnapshotResult = cloudsnapshot.CreateResult

// DeleteVolumeSnapshotResult retains lookup, acknowledgement and wait phases.
type DeleteVolumeSnapshotResult = cloudsnapshot.DeleteResult

// SnapshotWaitTimeoutError unwraps context.DeadlineExceeded for SDK wait expiry.
type SnapshotWaitTimeoutError = cloudsnapshot.WaitTimeoutError

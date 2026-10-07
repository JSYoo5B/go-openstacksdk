package cloudsnapshot

import (
	"context"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// Backup creation has fixed cloud arguments rather than caller-defined fields.
type BackupCreateOptions struct {
	Name, Description, SnapshotID *string
	Force, Incremental, Wait      *bool
	WaitPolicy                    MutationWaitOptions
	Location                      *resource.CloudLocation
}
type BackupDeleteOptions struct {
	Force, Wait *bool
	WaitPolicy  MutationWaitOptions
	Location    *resource.CloudLocation
}
type BackupCreateOption = Option[BackupCreateOptions]
type BackupDeleteOption = Option[BackupDeleteOptions]

func WithBackupCreateOptions(value BackupCreateOptions) BackupCreateOption {
	return withValue(value, cloneBackupCreate)
}
func WithBackupDeleteOptions(value BackupDeleteOptions) BackupDeleteOption {
	return withValue(value, cloneBackupDelete)
}
func WithBackupCreateName(value string) BackupCreateOption {
	return func(target *BackupCreateOptions) error { target.Name = clonePointer(&value); return nil }
}
func WithBackupCreateDescription(value string) BackupCreateOption {
	return func(target *BackupCreateOptions) error { target.Description = clonePointer(&value); return nil }
}
func WithBackupCreateSnapshotID(value string) BackupCreateOption {
	return func(target *BackupCreateOptions) error { target.SnapshotID = clonePointer(&value); return nil }
}
func WithBackupCreateForce(value bool) BackupCreateOption {
	return func(target *BackupCreateOptions) error { target.Force = clonePointer(&value); return nil }
}
func WithBackupCreateIncremental(value bool) BackupCreateOption {
	return func(target *BackupCreateOptions) error { target.Incremental = clonePointer(&value); return nil }
}
func WithBackupCreateWait(value bool) BackupCreateOption {
	return func(target *BackupCreateOptions) error { target.Wait = clonePointer(&value); return nil }
}
func WithBackupDeleteForce(value bool) BackupDeleteOption {
	return func(target *BackupDeleteOptions) error { target.Force = clonePointer(&value); return nil }
}
func WithBackupDeleteWait(value bool) BackupDeleteOption {
	return func(target *BackupDeleteOptions) error { target.Wait = clonePointer(&value); return nil }
}
func WithBackupCreateWaitPolicy(value MutationWaitOptions) BackupCreateOption {
	owned := cloneMutationWait(value)
	return func(target *BackupCreateOptions) error { target.WaitPolicy = cloneMutationWait(owned); return nil }
}
func WithBackupDeleteWaitPolicy(value MutationWaitOptions) BackupDeleteOption {
	owned := cloneMutationWait(value)
	return func(target *BackupDeleteOptions) error { target.WaitPolicy = cloneMutationWait(owned); return nil }
}
func WithBackupCreateLocation(value resource.CloudLocation) BackupCreateOption {
	owned := value.Clone()
	return func(target *BackupCreateOptions) error { target.Location = cloneLocation(&owned); return nil }
}
func WithBackupDeleteLocation(value resource.CloudLocation) BackupDeleteOption {
	owned := value.Clone()
	return func(target *BackupDeleteOptions) error { target.Location = cloneLocation(&owned); return nil }
}
func PrepareBackupCreate(ctx context.Context, options ...BackupCreateOption) (BackupCreateOptions, error) {
	return prepare(ctx, options, cloneBackupCreate, nil)
}
func PrepareBackupDelete(ctx context.Context, options ...BackupDeleteOption) (BackupDeleteOptions, error) {
	return prepare(ctx, options, cloneBackupDelete, nil)
}
func cloneBackupCreate(value BackupCreateOptions) BackupCreateOptions {
	value.Name = clonePointer(value.Name)
	value.Description = clonePointer(value.Description)
	value.SnapshotID = clonePointer(value.SnapshotID)
	value.Force = clonePointer(value.Force)
	value.Incremental = clonePointer(value.Incremental)
	value.Wait = clonePointer(value.Wait)
	value.WaitPolicy = cloneMutationWait(value.WaitPolicy)
	value.Location = cloneLocation(value.Location)
	return value
}
func cloneBackupDelete(value BackupDeleteOptions) BackupDeleteOptions {
	value.Force = clonePointer(value.Force)
	value.Wait = clonePointer(value.Wait)
	value.WaitPolicy = cloneMutationWait(value.WaitPolicy)
	value.Location = cloneLocation(value.Location)
	return value
}

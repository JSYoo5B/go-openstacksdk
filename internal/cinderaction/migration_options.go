package cinderaction

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func clonePointer[T any](value *T) *T {
	if value == nil {
		return nil
	}
	owned := *value
	return &owned
}

// StatusResetOptions omits every nil or empty status independently.
type StatusResetOptions struct{ Status, AttachStatus, MigrationStatus *string }
type StatusResetOption func(*StatusResetOptions) error

func cloneStatusReset(value StatusResetOptions) StatusResetOptions {
	value.Status = clonePointer(value.Status)
	value.AttachStatus = clonePointer(value.AttachStatus)
	value.MigrationStatus = clonePointer(value.MigrationStatus)
	return value
}
func WithStatusResetOptions(value StatusResetOptions) StatusResetOption {
	owned := cloneStatusReset(value)
	return func(target *StatusResetOptions) error { *target = cloneStatusReset(owned); return nil }
}
func WithStatusResetStatus(value string) StatusResetOption {
	return func(target *StatusResetOptions) error { target.Status = clonePointer(&value); return nil }
}
func WithStatusResetAttachStatus(value string) StatusResetOption {
	return func(target *StatusResetOptions) error { target.AttachStatus = clonePointer(&value); return nil }
}
func WithStatusResetMigrationStatus(value string) StatusResetOption {
	return func(target *StatusResetOptions) error { target.MigrationStatus = clonePointer(&value); return nil }
}
func PrepareStatusReset(ctx context.Context, options ...StatusResetOption) (StatusResetOptions, error) {
	return prepareStatusReset(options, func() error { return cloudread.Context(ctx) })
}
func prepareStatusReset(options []StatusResetOption, guard func() error) (StatusResetOptions, error) {
	policy, err := prepareOptions(options, cloneStatusReset, func(*StatusResetOptions) {}, guard)
	if err == nil {
		err = validateBodyStrings(policy.Status, policy.AttachStatus, policy.MigrationStatus)
	}
	if err != nil {
		return StatusResetOptions{}, err
	}
	return policy, nil
}

// MigrationOptions preserves host/cluster presence, including empty strings.
// Boolean nil values become owned false values and only true flags are sent.
type MigrationOptions struct {
	Host, Cluster             *string
	ForceHostCopy, LockVolume *bool
}
type MigrationOption func(*MigrationOptions) error

func cloneMigration(value MigrationOptions) MigrationOptions {
	value.Host = clonePointer(value.Host)
	value.Cluster = clonePointer(value.Cluster)
	value.ForceHostCopy = clonePointer(value.ForceHostCopy)
	value.LockVolume = clonePointer(value.LockVolume)
	return value
}
func WithMigrationOptions(value MigrationOptions) MigrationOption {
	owned := cloneMigration(value)
	return func(target *MigrationOptions) error { *target = cloneMigration(owned); return nil }
}
func WithMigrationHost(value string) MigrationOption {
	return func(target *MigrationOptions) error { target.Host = clonePointer(&value); return nil }
}
func WithMigrationCluster(value string) MigrationOption {
	return func(target *MigrationOptions) error { target.Cluster = clonePointer(&value); return nil }
}
func WithMigrationForceHostCopy(value bool) MigrationOption {
	return func(target *MigrationOptions) error { target.ForceHostCopy = clonePointer(&value); return nil }
}
func WithMigrationLockVolume(value bool) MigrationOption {
	return func(target *MigrationOptions) error { target.LockVolume = clonePointer(&value); return nil }
}
func PrepareMigration(ctx context.Context, options ...MigrationOption) (MigrationOptions, error) {
	return prepareMigration(options, func() error { return cloudread.Context(ctx) })
}
func prepareMigration(options []MigrationOption, guard func() error) (MigrationOptions, error) {
	policy, err := prepareOptions(options, cloneMigration, func(value *MigrationOptions) {
		if value.ForceHostCopy == nil {
			value.ForceHostCopy = clonePointer(new(bool))
		}
		if value.LockVolume == nil {
			value.LockVolume = clonePointer(new(bool))
		}
	}, guard)
	if err == nil {
		err = validateBodyStrings(policy.Host, policy.Cluster)
	}
	if err != nil {
		return MigrationOptions{}, err
	}
	return policy, nil
}

// MigrationCompletionOptions always transmits an error bool, default false.
type MigrationCompletionOptions struct{ Error *bool }
type MigrationCompletionOption func(*MigrationCompletionOptions) error

func cloneMigrationCompletion(value MigrationCompletionOptions) MigrationCompletionOptions {
	value.Error = clonePointer(value.Error)
	return value
}
func WithMigrationCompletionOptions(value MigrationCompletionOptions) MigrationCompletionOption {
	owned := cloneMigrationCompletion(value)
	return func(target *MigrationCompletionOptions) error { *target = cloneMigrationCompletion(owned); return nil }
}
func WithMigrationCompletionError(value bool) MigrationCompletionOption {
	return func(target *MigrationCompletionOptions) error { target.Error = clonePointer(&value); return nil }
}
func PrepareMigrationCompletion(ctx context.Context, options ...MigrationCompletionOption) (MigrationCompletionOptions, error) {
	return prepareMigrationCompletion(options, func() error { return cloudread.Context(ctx) })
}
func prepareMigrationCompletion(options []MigrationCompletionOption, guard func() error) (MigrationCompletionOptions, error) {
	return prepareOptions(options, cloneMigrationCompletion, func(value *MigrationCompletionOptions) {
		if value.Error == nil {
			value.Error = clonePointer(new(bool))
		}
	}, guard)
}

func validateBodyStrings(values ...*string) error {
	for _, value := range values {
		if value != nil && !utf8.ValidString(*value) {
			return fmt.Errorf("%w: volume action body text must be UTF-8", resource.ErrInvalidOption)
		}
	}
	return nil
}

// ValidateNewVolume treats this argument as literal body text, not a route ID.
func ValidateNewVolume(value string) error { return validateBodyStrings(&value) }

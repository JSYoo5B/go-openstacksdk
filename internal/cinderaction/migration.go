package cinderaction

import (
	"context"
	"encoding/json"

	"github.com/gophercloud/gophercloud/v2"
)

func ResetStatus(ctx context.Context, client *gophercloud.ServiceClient, id string, options ...StatusResetOption) (*Result, error) {
	const operation = "ResetVolumeStatus"
	source, err := captureAction(ctx, client, id)
	if err != nil {
		return nil, WrapOperation(ctx, operation, err)
	}
	policy, err := prepareStatusReset(options, func() error { return source.Guard(ctx) })
	if err != nil {
		return nil, WrapOperation(ctx, operation, err)
	}
	fields := make(map[string]string)
	for key, value := range map[string]*string{"status": policy.Status, "attach_status": policy.AttachStatus, "migration_status": policy.MigrationStatus} {
		if value != nil && *value != "" {
			fields[key] = *value
		}
	}
	body, _ := json.Marshal(map[string]any{"os-reset_status": fields})
	return applyPrepared(ctx, source, id, operation, body)
}

func Migrate(ctx context.Context, client *gophercloud.ServiceClient, id string, options ...MigrationOption) (*Result, error) {
	const operation = "MigrateVolume"
	source, err := captureAction(ctx, client, id)
	if err != nil {
		return nil, WrapOperation(ctx, operation, err)
	}
	policy, err := prepareMigration(options, func() error { return source.Guard(ctx) })
	if err != nil {
		return nil, WrapOperation(ctx, operation, err)
	}
	fields := make(map[string]any)
	if policy.Host != nil {
		fields["host"] = *policy.Host
	}
	if policy.Cluster != nil {
		fields["cluster"] = *policy.Cluster
	}
	if *policy.ForceHostCopy {
		fields["force_host_copy"] = true
	}
	if *policy.LockVolume {
		fields["lock_volume"] = true
	}
	body, _ := json.Marshal(map[string]any{"os-migrate_volume": fields})
	required := ""
	if policy.Cluster != nil {
		required = "3.16"
	}
	return applyRequired(ctx, source, id, operation, body, required)
}

func CompleteMigration(ctx context.Context, client *gophercloud.ServiceClient, id, newVolume string, options ...MigrationCompletionOption) (*Result, error) {
	const operation = "CompleteVolumeMigration"
	source, err := captureAction(ctx, client, id)
	if err == nil {
		err = ValidateNewVolume(newVolume)
	}
	if err != nil {
		return nil, WrapOperation(ctx, operation, err)
	}
	policy, err := prepareMigrationCompletion(options, func() error { return source.Guard(ctx) })
	if err != nil {
		return nil, WrapOperation(ctx, operation, err)
	}
	body, _ := json.Marshal(map[string]any{"os-migrate_volume_completion": map[string]any{"new_volume": newVolume, "error": *policy.Error}})
	return applyPrepared(ctx, source, id, operation, body)
}

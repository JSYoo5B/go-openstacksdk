package cinderaction

import (
	"context"
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/cloudread"
	"gophercloudsdk/resource"
)

func captureAction(ctx context.Context, client *gophercloud.ServiceClient, id string) (*cloudread.Source, error) {
	source, err := cloudread.Capture(ctx, client, "volume")
	if err == nil {
		err = ValidateID(id)
	}
	return source, err
}

// ValidateNewType validates body text without imposing route or enum rules.
func ValidateNewType(value string) error {
	if !utf8.ValidString(value) {
		return fmt.Errorf("%w: volume type must be UTF-8", resource.ErrInvalidOption)
	}
	return nil
}

func Extend(ctx context.Context, client *gophercloud.ServiceClient, id string, newSize int) (*Result, error) {
	const operation = "ExtendVolume"
	source, err := captureAction(ctx, client, id)
	if err != nil {
		return nil, WrapOperation(ctx, operation, err)
	}
	body, _ := json.Marshal(map[string]any{"os-extend": map[string]int{"new_size": newSize}})
	return applyPrepared(ctx, source, id, operation, body)
}

func Retype(ctx context.Context, client *gophercloud.ServiceClient, id, newType string, options ...RetypeOption) (*Result, error) {
	const operation = "RetypeVolume"
	source, err := captureAction(ctx, client, id)
	if err == nil {
		err = ValidateNewType(newType)
	}
	if err != nil {
		return nil, WrapOperation(ctx, operation, err)
	}
	policy, err := prepareRetype(options, func() error { return source.Guard(ctx) })
	if err != nil {
		return nil, WrapOperation(ctx, operation, err)
	}
	fields := map[string]string{"new_type": newType}
	if *policy.MigrationPolicy != "" {
		fields["migration_policy"] = *policy.MigrationPolicy
	}
	body, _ := json.Marshal(map[string]any{"os-retype": fields})
	return applyPrepared(ctx, source, id, operation, body)
}

func CompleteExtend(ctx context.Context, client *gophercloud.ServiceClient, id string, options ...ExtendCompletionOption) (*Result, error) {
	const operation = "CompleteVolumeExtend"
	source, err := captureAction(ctx, client, id)
	if err != nil {
		return nil, WrapOperation(ctx, operation, err)
	}
	policy, err := prepareExtendCompletion(options, func() error { return source.Guard(ctx) })
	if err != nil {
		return nil, WrapOperation(ctx, operation, err)
	}
	body, _ := json.Marshal(map[string]any{"os-extend_volume_completion": map[string]bool{"error": *policy.Error}})
	return applyPrepared(ctx, source, id, operation, body)
}

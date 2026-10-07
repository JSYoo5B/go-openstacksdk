package cinderaction

import (
	"context"
	"encoding/json"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudread"
	"github.com/gophercloud/gophercloud/v2"
)

// Bootable retains the required Proxy bool, including an explicit false.
func Bootable(ctx context.Context, client *gophercloud.ServiceClient, id string, bootable bool) (*Result, error) {
	const operation = "SetVolumeBootableStatus"
	source, err := cloudread.Capture(ctx, client, "volume")
	if err == nil {
		err = ValidateID(id)
	}
	if err != nil {
		return nil, WrapOperation(ctx, operation, err)
	}
	body, _ := json.Marshal(map[string]any{"os-set_bootable": map[string]bool{"bootable": bootable}})
	return applyPrepared(ctx, source, id, operation, body)
}

// Readonly captures the selected source before running original options.
func Readonly(ctx context.Context, client *gophercloud.ServiceClient, id string, options ...ReadonlyOption) (*Result, error) {
	const operation = "SetVolumeReadonly"
	source, err := cloudread.Capture(ctx, client, "volume")
	if err == nil {
		err = ValidateID(id)
	}
	if err != nil {
		return nil, WrapOperation(ctx, operation, err)
	}
	policy, err := prepareReadonly(options, func() error { return source.Guard(ctx) })
	if err != nil {
		return nil, WrapOperation(ctx, operation, err)
	}
	body, _ := json.Marshal(map[string]any{"os-update_readonly_flag": map[string]bool{"readonly": *policy.Readonly}})
	return applyPrepared(ctx, source, id, operation, body)
}

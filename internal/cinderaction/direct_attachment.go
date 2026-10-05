package cinderaction

import (
	"context"
	"encoding/json"

	"github.com/gophercloud/gophercloud/v2"
)

// ValidateMountpoint admits literal body text, including empty strings.
func ValidateMountpoint(value string) error { return validateBodyStrings(&value) }

// ValidateAttachmentID admits body text without route-ID or UUID restrictions.
func ValidateAttachmentID(value string) error { return validateBodyStrings(&value) }

func DirectAttach(ctx context.Context, client *gophercloud.ServiceClient, id, mountpoint string, options ...DirectAttachOption) (*Result, error) {
	const operation = "AttachCinderVolume"
	source, err := captureAction(ctx, client, id)
	if err == nil {
		err = ValidateMountpoint(mountpoint)
	}
	if err != nil {
		return nil, WrapOperation(ctx, operation, err)
	}
	policy, err := prepareDirectAttach(options, func() error { return source.Guard(ctx) })
	if err != nil {
		return nil, WrapOperation(ctx, operation, err)
	}
	fields := map[string]string{"mountpoint": mountpoint}
	if policy.Instance != nil {
		fields["instance_uuid"] = *policy.Instance
	} else {
		fields["host_name"] = *policy.HostName
	}
	body, _ := json.Marshal(map[string]any{"os-attach": fields})
	return applyPrepared(ctx, source, id, operation, body)
}

func DirectDetach(ctx context.Context, client *gophercloud.ServiceClient, id, attachmentID string, options ...DirectDetachOption) (*Result, error) {
	const operation = "DetachCinderVolume"
	source, err := captureAction(ctx, client, id)
	if err == nil {
		err = ValidateAttachmentID(attachmentID)
	}
	if err != nil {
		return nil, WrapOperation(ctx, operation, err)
	}
	policy, err := prepareDirectDetach(options, func() error { return source.Guard(ctx) })
	if err != nil {
		return nil, WrapOperation(ctx, operation, err)
	}
	fields := map[string]any{"attachment_id": attachmentID}
	action := "os-detach"
	if *policy.Force {
		action = "os-force_detach"
		if len(policy.Connector) != 0 {
			fields["connector"] = policy.Connector
		}
	}
	body, _ := json.Marshal(map[string]any{action: fields})
	return applyPrepared(ctx, source, id, operation, body)
}

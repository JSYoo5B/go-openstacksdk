package blockstorage

import (
	"context"
	"net/http"
	"unicode/utf8"

	"github.com/gophercloud/gophercloud/v2"
)

func changeVolumeTypeAccess(ctx context.Context, cinder *gophercloud.ServiceClient, input VolumeTypeAccessRequest, action, operation string, options []VolumeTypeReadOption) (*VolumeTypeAccessActionResult, error) {
	p, resolved, id, err := prepareVolumeTypeAccess(ctx, cinder, input.NameOrID, options)
	if p == nil {
		return nil, wrapVolumeTypeError(ctx, operation, err)
	}
	result := &VolumeTypeAccessActionResult{Resolved: resolved, TypeID: id, ProjectID: input.ProjectID}
	if err != nil {
		return result, wrapVolumeTypeError(ctx, operation, err)
	}
	if !utf8.ValidString(input.ProjectID) {
		return result, wrapVolumeTypeError(ctx, operation, attachInvalid("project ID must be valid UTF-8 JSON text"))
	}
	body := map[string]any{action: map[string]string{"project": input.ProjectID}}
	response, err := p.accessExchange(ctx, http.MethodPost, id, "action", body)
	result.Applied = volumeTypeAccessProof(response)
	// The source action never decodes response JSON or verifies membership.
	return result, wrapVolumeTypeError(ctx, operation, err)
}

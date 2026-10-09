package image

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ImageRecordCloudPropertiesRequest mirrors Cloud update_image_properties'
// image and name_or_id arguments. Image may be an SDK Record or a literal ID
// string; NameOrID is used only when both Image forms are empty, and it is
// passed on as a literal identity without a name lookup.
type ImageRecordCloudPropertiesRequest struct {
	Record   *ImageRecord
	ID       string
	NameOrID string
}

// UpdateCloudImageProperties implements Cloud update_image_properties by
// selecting `image or name_or_id` and delegating to the owned Proxy
// UpdateImagePropertiesRecord helper with the same concrete options.
func (s *Service) UpdateCloudImageProperties(ctx context.Context, input ImageRecordCloudPropertiesRequest, options ...ImageRecordPropertiesOption) (*ImageRecordPropertiesResult, error) {
	const operation = "UpdateCloudImageProperties"
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrapImageMutationError(ctx, operation, err)
	}
	selected := ImageRecordPropertiesRequest{Record: input.Record}
	if input.Record == nil {
		selected.ID = input.ID
		if selected.ID == "" {
			selected.ID = input.NameOrID
		}
	}
	result, err := s.UpdateImagePropertiesRecord(ctx, selected, options...)
	if failure, ok := err.(*resource.OperationError); ok {
		// Name the Cloud entry point without nesting the delegated operation.
		renamed := *failure
		renamed.Operation = operation
		return result, &renamed
	}
	return result, err
}

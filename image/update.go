package image

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"gophercloudsdk/internal/rest"
	"gophercloudsdk/resource"
)

// UpdateImage applies the owned ordered changes and returns fresh image JSON.
// Empty changes still issue a PATCH; Glance returns metadata without saving.
func (s *Service) UpdateImage(ctx context.Context, ref resource.Ref, options ...UpdateImageOption) (*ImageInfo, error) {
	p, err := s.prepareImageUpdateReference(ctx, ref)
	if err != nil {
		return nil, wrapImageMutationError(ctx, "UpdateImage", err)
	}
	policy, err := parseUpdateImageOptions(append([]UpdateImageOption(nil), options...))
	id, err := finishImageUpdateReference(ctx, p, ref, policy.Headers, err)
	if err != nil {
		return nil, wrapImageMutationError(ctx, "UpdateImage", err)
	}
	return executeImageUpdate(ctx, p, id, policy.Changes, "UpdateImage")
}

// SetImageProperties sends sorted v2.1 add operations, which upsert root fields.
// Values stay literal JSON; Glance owns schema, state and permission checks.
func (s *Service) SetImageProperties(ctx context.Context, ref resource.Ref, options ...SetImagePropertiesOption) (*ImageInfo, error) {
	p, err := s.prepareImageUpdateReference(ctx, ref)
	if err != nil {
		return nil, wrapImageMutationError(ctx, "SetImageProperties", err)
	}
	policy, changes, err := parseSetImagePropertiesOptions(append([]SetImagePropertiesOption(nil), options...))
	id, err := finishImageUpdateReference(ctx, p, ref, policy.Headers, err)
	if err != nil {
		return nil, wrapImageMutationError(ctx, "SetImageProperties", err)
	}
	return executeImageUpdate(ctx, p, id, changes, "SetImageProperties")
}

func executeImageUpdate(ctx context.Context, p *preparedTaskSource, id string, changes []ImagePatch, operation string) (*ImageInfo, error) {
	response, err := rest.DoJSON(ctx, p.client, http.MethodPatch, p.base+"images/"+url.PathEscape(id), changes, nil, http.StatusOK)
	if err = checkTaskResponse(ctx, p, response, err); err != nil {
		return nil, wrapImageMutationError(ctx, operation, err)
	}
	var value ImageInfo
	if err = json.Unmarshal(response.Body, &value); err != nil {
		return nil, wrapImageMutationError(ctx, operation, response.Fail(err))
	}
	if err = p.check(ctx); err != nil {
		return nil, wrapImageMutationError(ctx, operation, response.Fail(err))
	}
	value.Header, value.StatusCode = response.Header.Clone(), response.StatusCode
	return &value, nil
}

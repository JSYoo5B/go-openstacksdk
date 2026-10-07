package image

import (
	"context"
	"net/http"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// AddTag adds one literal tag through Glance's dedicated endpoint. An ID needs
// no lookup; an explicit Name is resolved exactly before the fixed mutation.
// The acknowledgement does not synthesize an image or replace its tag set.
func (s *Service) AddTag(ctx context.Context, ref resource.Ref, tag string, options ...ImageMutationOption) (*ImageTagResult, error) {
	id, response, err := s.executeImageMutation(ctx, ref, &tag, "", http.MethodPut, append([]ImageMutationOption(nil), options...))
	return imageTagResult(id, tag, response), wrapImageMutationError(ctx, "AddTag", err)
}

// RemoveTag removes one literal tag. Missing images and tags remain errors.
// Only an actual 204 establishes an acknowledgement, retained with body errors.
func (s *Service) RemoveTag(ctx context.Context, ref resource.Ref, tag string, options ...ImageMutationOption) (*ImageTagResult, error) {
	id, response, err := s.executeImageMutation(ctx, ref, &tag, "", http.MethodDelete, append([]ImageMutationOption(nil), options...))
	return imageTagResult(id, tag, response), wrapImageMutationError(ctx, "RemoveTag", err)
}

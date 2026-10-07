package image

import (
	"context"
	"net/http"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// DeactivateImage requests Glance's deactivate action without fetching current
// metadata or waiting. The server owns permission and status transition policy.
func (s *Service) DeactivateImage(ctx context.Context, ref resource.Ref, options ...ImageMutationOption) (*ImageActionResult, error) {
	id, response, err := s.executeImageMutation(ctx, ref, nil, "deactivate", http.MethodPost, append([]ImageMutationOption(nil), options...))
	return imageActionResult(id, "deactivate", response), wrapImageMutationError(ctx, "DeactivateImage", err)
}

// ReactivateImage requests Glance's reactivate action without a metadata update,
// discovery gate or implicit wait. Missing images remain errors.
func (s *Service) ReactivateImage(ctx context.Context, ref resource.Ref, options ...ImageMutationOption) (*ImageActionResult, error) {
	id, response, err := s.executeImageMutation(ctx, ref, nil, "reactivate", http.MethodPost, append([]ImageMutationOption(nil), options...))
	return imageActionResult(id, "reactivate", response), wrapImageMutationError(ctx, "ReactivateImage", err)
}

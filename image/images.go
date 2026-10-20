package image

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// GetImage resolves an optional exact Name and then fetches fresh image JSON.
// Returned identifiers, links and custom properties are passive response data.
func (s *Service) GetImage(ctx context.Context, ref resource.Ref, options ...GetImageOption) (*ImageInfo, error) {
	p, id, err := s.prepareGetImage(ctx, ref, append([]GetImageOption(nil), options...))
	if err != nil {
		return nil, wrapImageMutationError(ctx, "GetImage", err)
	}
	response, err := rest.DoJSON(ctx, p.client, http.MethodGet, p.base+"images/"+url.PathEscape(id), nil, nil, http.StatusOK)
	if err = checkTaskResponse(ctx, p, response, err); err != nil {
		return nil, wrapImageMutationError(ctx, "GetImage", err)
	}
	var value ImageInfo
	if err = json.Unmarshal(response.Body, &value); err != nil {
		return nil, wrapImageMutationError(ctx, "GetImage", response.Fail(err))
	}
	if err = p.check(ctx); err != nil {
		return nil, wrapImageMutationError(ctx, "GetImage", response.Fail(err))
	}
	value.Header, value.StatusCode = response.Header.Clone(), response.StatusCode
	return &value, nil
}

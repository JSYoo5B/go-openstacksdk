package image

import (
	"context"
	"errors"
	"net/url"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func (s *Service) prepareImageMember(ctx context.Context, parent resource.Ref, memberID, status *string, apply func() (map[string]string, error)) (*preparedImageMutation, error) {
	var source *gophercloud.ServiceClient
	if s != nil {
		source = s.client
	}
	var provider *gophercloud.ProviderClient
	if source != nil {
		provider = source.ProviderClient
	}
	prepared, err := s.prepareImageMutation(ctx, parent, nil, []ImageMutationOption{func(config *ImageMutationOpts) error {
		if memberID != nil {
			if !utf8.ValidString(*memberID) {
				return uploadInvalid("member ID must be valid UTF-8")
			}
			if err := createImportID(*memberID); err != nil {
				return err
			}
		}
		if status != nil && *status != "pending" && *status != "accepted" && *status != "rejected" {
			return uploadInvalid("member status must be pending, accepted or rejected")
		}
		headers, err := apply()
		config.Headers = headers
		return err
	}})
	if err != nil && source != nil {
		var sourceErr error
		if s.client != source || source.ProviderClient != provider {
			sourceErr = uploadInvalid("image member source or provider changed")
		} else {
			_, sourceErr = validateImageMutationSource(ctx, source)
		}
		if sourceErr != nil && !errors.Is(err, sourceErr) {
			err = errors.Join(err, sourceErr)
		}
	}
	return prepared, err
}

func imageMemberEndpoint(prepared *preparedImageMutation, memberID *string) string {
	endpoint := prepared.base + "images/" + url.PathEscape(prepared.id) + "/members"
	if memberID != nil {
		endpoint += "/" + url.PathEscape(*memberID)
	}
	return endpoint
}

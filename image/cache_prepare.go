package image

import (
	"context"
	"errors"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const cacheTargetHeader = "X-Image-Cache-Clear-Target"

func cacheHeaders(values map[string]string, source bool, version string) (map[string]string, error) {
	for key := range values {
		if strings.EqualFold(key, cacheTargetHeader) {
			return nil, uploadInvalid("cache target header is owned by the SDK")
		}
	}
	return imageMutationHeaders(values, source, version)
}
func validateCacheSource(ctx context.Context, source *gophercloud.ServiceClient) (map[string]string, error) {
	headers, err := validateImageMutationSource(ctx, source)
	if err != nil {
		return nil, err
	}
	return cacheHeaders(headers, true, source.Microversion)
}
func checkCacheSource(ctx context.Context, prepared *preparedImageMutation) error {
	if err := prepared.check(ctx); err != nil {
		return err
	}
	_, err := validateCacheSource(ctx, prepared.source)
	return err
}

// The adapter parses cache policy before the unchanged exact Name resolver.
// Singleton requests capture the same source fields without inventing a Ref.
func (s *Service) prepareCache(ctx context.Context, ref *resource.Ref, apply func() (map[string]string, error)) (*preparedImageMutation, error) {
	var source *gophercloud.ServiceClient
	if s != nil {
		source = s.client
	}
	headers, err := validateCacheSource(ctx, source)
	if err != nil {
		return nil, err
	}
	if ref != nil {
		prepared, err := s.prepareImageMutation(ctx, *ref, nil, []ImageMutationOption{func(config *ImageMutationOpts) error {
			values, err := apply()
			if err != nil {
				return err
			}
			if _, err := validateCacheSource(ctx, source); err != nil {
				return err
			}
			config.Headers = values
			return nil
		}})
		if err != nil {
			if _, sourceErr := validateCacheSource(ctx, source); sourceErr != nil && !errors.Is(err, sourceErr) {
				err = errors.Join(err, sourceErr)
			}
			return nil, err
		}
		if err := checkCacheSource(ctx, prepared); err != nil {
			return nil, err
		}
		return prepared, nil
	}
	client := *source
	client.MoreHeaders = headers
	prepared := &preparedImageMutation{service: s, source: source, client: &client, base: source.ServiceURL(), provider: source.ProviderClient}
	values, err := apply()
	if err == nil {
		err = checkCacheSource(ctx, prepared)
	}
	if err != nil {
		return nil, err
	}
	for key, value := range values {
		client.MoreHeaders[key] = value
	}
	return prepared, nil
}

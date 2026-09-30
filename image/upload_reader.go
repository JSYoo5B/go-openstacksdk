package image

import (
	"context"
	"io"
	"maps"
	"net/http"
	"strconv"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/imagedata"
)

// borrowedUploadReader exposes only Read, so each net/http request wraps it in
// its own no-op closer and never closes or seeks the caller's input.
type borrowedUploadReader struct{ io.Reader }

// Gophercloud may reuse RawBody after backoff without rewinding it. Even a
// seekable reader can race with asynchronous Close of the previous request.
// Binary upload therefore uses one attempt, while metadata and wait requests
// keep the original provider's reauthentication and retry behavior.
func (s *Service) uploadDataOnce(ctx context.Context, id string, data io.Reader, size *int64) error {
	source := s.client.ProviderClient
	provider := &gophercloud.ProviderClient{HTTPClient: source.HTTPClient, UserAgent: source.UserAgent}
	// net/http can change PUT to GET on a 303 redirect, losing the binary body.
	// Keep the redirect response as the original HTTP error instead.
	provider.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	if !source.IsThrowaway() {
		provider.SetToken(source.Token())
	}
	client := *s.client
	client.ProviderClient = provider
	client.MoreHeaders = maps.Clone(s.client.MoreHeaders)
	if size != nil {
		if client.MoreHeaders == nil {
			client.MoreHeaders = make(map[string]string)
		}
		client.MoreHeaders["X-OpenStack-Image-Size"] = strconv.FormatInt(*size, 10)
	}
	return imagedata.Upload(ctx, &client, id, borrowedUploadReader{Reader: data}).ExtractErr()
}

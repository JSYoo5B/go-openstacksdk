package cyborg

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// guardedClient owns the HTTP redirect boundary without changing the shared
// provider. Tokens/auth results are copied under native locks; reauthentication
// delegates to the original provider, then copies its refreshed token back.
func guardedClient(source *gophercloud.ServiceClient) (*gophercloud.ServiceClient, error) {
	if source == nil || source.ProviderClient == nil {
		return nil, fmt.Errorf("%w: Cyborg service client is required", resource.ErrInvalidOption)
	}
	origin, err := url.Parse(source.Endpoint)
	if err != nil || origin.Host == "" || origin.User != nil || (origin.Scheme != "http" && origin.Scheme != "https") {
		return nil, fmt.Errorf("%w: invalid Cyborg service endpoint", resource.ErrInvalidOption)
	}
	parent := source.ProviderClient
	provider := &gophercloud.ProviderClient{
		HTTPClient: parent.HTTPClient, UserAgent: parent.UserAgent,
		IdentityBase: parent.IdentityBase, IdentityEndpoint: parent.IdentityEndpoint, EndpointLocator: parent.EndpointLocator,
		RetryBackoffFunc: parent.RetryBackoffFunc, MaxBackoffRetries: parent.MaxBackoffRetries, RetryFunc: parent.RetryFunc,
	}
	provider.UseTokenLock()
	provider.CopyTokenFrom(parent)
	provider.SetThrowaway(parent.IsThrowaway())
	// The pager may live across a token refresh by another service. At each
	// actual attempt, send the source's current locked token and remember what
	// the server received. Keep the clone's native token state unchanged here:
	// its 401 machinery must still invoke our delegate, which uses the actual
	// sent token rather than the clone's possibly older pre-request snapshot.
	var sentToken atomic.Value
	sentToken.Store(provider.Token())
	transport := parent.HTTPClient.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	provider.HTTPClient.Transport = transportFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.User != nil || req.URL.Scheme != origin.Scheme || !strings.EqualFold(req.URL.Host, origin.Host) {
			return nil, fmt.Errorf("Cyborg request changes service origin")
		}
		if err := req.Context().Err(); err != nil {
			return nil, err
		}
		copy := req.Clone(req.Context())
		headers, err := authenticatedHeaders(req.Context(), parent)
		if err != nil {
			return nil, err
		}
		token := headers["X-Auth-Token"]
		if token == "" {
			copy.Header.Del("X-Auth-Token")
		} else {
			copy.Header.Set("X-Auth-Token", token)
		}
		sentToken.Store(token)
		return transport.RoundTrip(copy)
	})
	if parent.ReauthFunc != nil {
		provider.ReauthFunc = func(ctx context.Context) error {
			if err := parent.Reauthenticate(ctx, sentToken.Load().(string)); err != nil {
				return err
			}
			provider.CopyTokenFrom(parent)
			return nil
		}
	}
	policy := parent.HTTPClient.CheckRedirect
	provider.HTTPClient.CheckRedirect = func(next *http.Request, via []*http.Request) error {
		if policy != nil {
			if err := policy(next, via); err != nil {
				return err
			}
		} else if len(via) >= 10 {
			return fmt.Errorf("stopped after 10 redirects")
		}
		if next.URL.User != nil || next.URL.Scheme != origin.Scheme || !strings.EqualFold(next.URL.Host, origin.Host) {
			return fmt.Errorf("Cyborg redirect changes service origin")
		}
		return nil
	}
	client := *source
	client.ProviderClient = provider
	client.MoreHeaders = maps.Clone(source.MoreHeaders)
	return &client, nil
}

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// The native public helper waits for ongoing shared reauthentication but has
// no context parameter. A canceled caller can leave that shared operation to
// its owner; the buffered result lets this waiter finish when it completes.
func authenticatedHeaders(ctx context.Context, parent *gophercloud.ProviderClient) (map[string]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if parent.ReauthFunc == nil {
		if parent.IsThrowaway() {
			return nil, nil
		}
		return map[string]string{"X-Auth-Token": parent.Token()}, nil
	}
	done := make(chan map[string]string, 1)
	go func() { done <- parent.AuthenticatedHeaders() }()
	select {
	case headers := <-done:
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return headers, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

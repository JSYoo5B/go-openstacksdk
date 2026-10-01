package quotasets

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"sync/atomic"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/resource"
)

// guardedQuotaClient preserves the fixed request target even if a redirect
// would otherwise drop user_id and widen a user update/reset to the project.
// It never copies the provider struct or changes the shared HTTP client.
func guardedQuotaClient(source *gophercloud.ServiceClient, method, target string) (*gophercloud.ServiceClient, error) {
	if source == nil || source.ProviderClient == nil {
		return nil, fmt.Errorf("%w: quota service client is required", resource.ErrInvalidOption)
	}
	expected, err := url.Parse(target)
	if err != nil || expected.User != nil || expected.Host == "" || expected.Fragment != "" || (expected.Scheme != "http" && expected.Scheme != "https") {
		return nil, fmt.Errorf("%w: invalid quota request endpoint", resource.ErrInvalidOption)
	}
	guard := func(req *http.Request) error {
		if req.Method != method || req.URL == nil || req.URL.User != nil || req.URL.Opaque != "" || req.URL.Scheme != expected.Scheme || req.URL.Host != expected.Host || (req.Host != "" && req.Host != expected.Host) || req.URL.EscapedPath() != expected.EscapedPath() || req.URL.RawQuery != expected.RawQuery {
			return fmt.Errorf("%w: quota request changes fixed project/user target or method", resource.ErrInvalidOption)
		}
		return req.Context().Err()
	}
	parent := source.ProviderClient
	explicitHeaders := make(http.Header, len(source.MoreHeaders))
	for key, value := range source.MoreHeaders {
		explicitHeaders.Set(key, value)
	}
	provider := &gophercloud.ProviderClient{
		HTTPClient: parent.HTTPClient, UserAgent: parent.UserAgent,
		RetryBackoffFunc: parent.RetryBackoffFunc, MaxBackoffRetries: parent.MaxBackoffRetries, RetryFunc: parent.RetryFunc,
	}
	provider.UseTokenLock()
	provider.CopyTokenFrom(parent)
	provider.SetThrowaway(parent.IsThrowaway())
	var sentToken atomic.Value
	sentToken.Store(provider.Token())
	transport := parent.HTTPClient.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	provider.HTTPClient.Transport = quotaTransportFunc(func(req *http.Request) (*http.Response, error) {
		if err := guard(req); err != nil {
			return nil, err
		}
		headers, err := quotaAuthenticatedHeaders(req.Context(), parent)
		if err != nil {
			return nil, err
		}
		copy := req.Clone(req.Context())
		copy.Header.Del("X-Auth-Token")
		if token, present := headers["X-Auth-Token"]; present {
			copy.Header.Set("X-Auth-Token", token)
		} else if values, present := explicitHeaders[http.CanonicalHeaderKey("X-Auth-Token")]; present {
			copy.Header[http.CanonicalHeaderKey("X-Auth-Token")] = append([]string(nil), values...)
		}
		sentToken.Store(copy.Header.Get("X-Auth-Token"))
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
		redirectContext := next.Context()
		if policy != nil {
			if err := policy(next, via); err != nil {
				return err
			}
			if err := next.Context().Err(); err != nil {
				return err
			}
		} else if len(via) >= 10 {
			return fmt.Errorf("stopped after 10 redirects")
		}
		// Keep the context installed by http.Client, including its timeout,
		// even if the caller's redirect policy replaces the request context.
		*next = *next.WithContext(redirectContext)
		return guard(next)
	}
	client := *source
	client.ProviderClient = provider
	client.MoreHeaders = maps.Clone(source.MoreHeaders)
	return &client, nil
}

type quotaTransportFunc func(*http.Request) (*http.Response, error)

func (f quotaTransportFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// Native AuthenticatedHeaders waits for shared reauthentication without a
// context parameter. A canceled quota caller need not wait for its completion.
func quotaAuthenticatedHeaders(ctx context.Context, parent *gophercloud.ProviderClient) (map[string]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if parent.ReauthFunc == nil {
		if parent.IsThrowaway() {
			return nil, nil
		}
		if token := parent.Token(); token != "" {
			return map[string]string{"X-Auth-Token": token}, nil
		}
		return nil, nil
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

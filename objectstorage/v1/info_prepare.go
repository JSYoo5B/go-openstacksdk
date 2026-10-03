package v1

import (
	"context"
	"net/http"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/fixedrequest"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/internal/swiftinfo"
)

type preparedInfo struct {
	base      *preparedTempURLKey
	target    string
	afterHTTP error
}

func (s *Service) captureInfo(ctx context.Context) (*preparedInfo, error) {
	base, err := s.captureTempURLKey(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := validateInfoHeaders(base.source.MoreHeaders); err != nil {
		return nil, tempURLKeyContextError(ctx, err)
	}
	target, err := infoTarget(base.endpoint)
	if err != nil {
		return nil, tempURLKeyContextError(ctx, err)
	}
	return &preparedInfo{base: base, target: target}, nil
}

func infoTarget(endpoint string) (string, error) {
	return swiftinfo.Target(endpoint)
}

func validateInfoHeaders(headers map[string]string) (map[string]string, error) {
	owned, err := validateTempURLKeyHeaders(headers)
	if err != nil {
		return nil, err
	}
	for key := range owned {
		if strings.EqualFold(key, "Cookie") || strings.EqualFold(key, "X-Service-Token") {
			return nil, tempURLKeyInvalid("Swift info header %q is SDK owned", key)
		}
	}
	return owned, nil
}

func (p *preparedInfo) check(ctx context.Context) error {
	if err := p.base.check(ctx); err != nil {
		return err
	}
	_, err := validateInfoHeaders(p.base.source.MoreHeaders)
	return tempURLKeyContextError(ctx, err)
}
func (p *preparedInfo) guard(ctx context.Context) error {
	return joinTempURLKeyErrors(p.afterHTTP, p.check(ctx))
}
func (p *preparedInfo) finish(ctx context.Context, headers map[string]string, err error) error {
	if err = joinTempURLKeyErrors(err, p.guard(ctx)); err != nil {
		return tempURLKeyContextError(ctx, err)
	}
	owned, err := validateInfoHeaders(headers)
	if err != nil {
		return tempURLKeyContextError(ctx, err)
	}
	for key, value := range owned {
		p.base.client.MoreHeaders[key] = value
	}
	return nil
}

type infoTransport func(*http.Request) (*http.Response, error)

func (f infoTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func (p *preparedInfo) read(ctx context.Context, codes ...int) (*rest.Response, error) {
	if err := p.guard(ctx); err != nil {
		return nil, err
	}
	client, err := fixedrequest.New(p.base.client, http.MethodGet, p.target)
	if err != nil {
		return nil, tempURLKeyContextError(ctx, err)
	}
	provider := client.ProviderClient
	transport := provider.HTTPClient.Transport
	provider.HTTPClient.Transport = infoTransport(func(req *http.Request) (*http.Response, error) {
		if err := p.guard(req.Context()); err != nil {
			return nil, err
		}
		wire, err := transport.RoundTrip(req)
		// Preserve the wire for the shared reader. Returning an error beside
		// an accepted response would make http.Client discard its evidence.
		p.afterHTTP = joinTempURLKeyErrors(p.afterHTTP, p.check(req.Context()))
		return wire, err
	})
	if retry := provider.RetryFunc; retry != nil {
		provider.RetryFunc = func(ctx context.Context, method, endpoint string, options *gophercloud.RequestOpts, original error, count uint) error {
			if err := p.guard(ctx); err != nil {
				return joinTempURLKeyErrors(original, err)
			}
			err := retry(ctx, method, endpoint, options, original, count)
			if guardErr := p.guard(ctx); err != nil || guardErr != nil {
				return joinTempURLKeyErrors(original, err, guardErr)
			}
			return nil
		}
	}
	if backoff := provider.RetryBackoffFunc; backoff != nil {
		provider.RetryBackoffFunc = func(ctx context.Context, response *gophercloud.ErrUnexpectedResponseCode, original error, count uint) error {
			if err := p.guard(ctx); err != nil {
				return joinTempURLKeyErrors(response, original, err)
			}
			err := backoff(ctx, response, original, count)
			if guardErr := p.guard(ctx); err != nil || guardErr != nil {
				return joinTempURLKeyErrors(response, original, err, guardErr)
			}
			return nil
		}
	}
	if reauth := provider.ReauthFunc; reauth != nil {
		provider.ReauthFunc = func(ctx context.Context) error {
			if err := p.guard(ctx); err != nil {
				return err
			}
			return joinTempURLKeyErrors(reauth(ctx), p.guard(ctx))
		}
	}
	redirect := provider.HTTPClient.CheckRedirect
	provider.HTTPClient.CheckRedirect = func(next *http.Request, via []*http.Request) error {
		if err := p.guard(next.Context()); err != nil {
			return err
		}
		return joinTempURLKeyErrors(redirect(next, via), p.guard(next.Context()))
	}
	response, err := rest.DoJSON(ctx, client, http.MethodGet, p.target, nil, nil, codes...)
	if guardErr := p.guard(ctx); guardErr != nil {
		err = joinTempURLKeyErrors(err, guardErr)
		if response != nil {
			err = response.Fail(err)
		}
	}
	return response, err
}

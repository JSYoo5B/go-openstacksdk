package objects

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/internal/fixedrequest"
	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/gophercloud/gophercloud/v2"
)

type preparedDeleteObject struct {
	metadata          *preparedMetadata
	target            string
	staticLargeObject *bool
	ignoreMissing     bool
	observed          error
}

func (a *API) captureDeleteObject(ctx context.Context, container, object string) (*preparedDeleteObject, error) {
	metadata, err := a.captureMetadata(ctx, container, object)
	if err != nil {
		return nil, err
	}
	if _, err := validateDeleteObjectHeaders(metadata.source.MoreHeaders); err != nil {
		return nil, metadataContextError(ctx, err)
	}
	return &preparedDeleteObject{metadata: metadata, target: metadata.target}, nil
}
func validateDeleteObjectHeaders(headers map[string]string) (map[string]string, error) {
	owned, err := validateMetadataHeaders(headers)
	if err != nil {
		return nil, err
	}
	for key := range owned {
		switch strings.ToLower(key) {
		case "cookie", "x-service-token", "x-static-large-object":
			return nil, metadataInvalid("object delete header %q is SDK owned", key)
		}
	}
	return owned, nil
}
func (p *preparedDeleteObject) check(ctx context.Context) error {
	baseErr := p.metadata.check(ctx)
	_, headerErr := validateDeleteObjectHeaders(p.metadata.source.MoreHeaders)
	return metadataContextError(ctx, joinMetadataErrors(baseErr, headerErr))
}
func (p *preparedDeleteObject) guard(ctx context.Context) error {
	return joinMetadataErrors(p.observed, p.check(ctx))
}
func (p *preparedDeleteObject) note(ctx context.Context) {
	p.observed = joinMetadataErrors(p.observed, p.check(ctx))
}
func (p *preparedDeleteObject) finish(ctx context.Context, cfg DeleteObjectOpts, err error) error {
	if err = joinMetadataErrors(err, p.guard(ctx)); err != nil {
		return metadataContextError(ctx, err)
	}
	headers, err := validateDeleteObjectHeaders(cfg.Headers)
	if err != nil {
		return metadataContextError(ctx, err)
	}
	if !utf8.ValidString(cfg.VersionID) {
		return metadataInvalid("invalid object delete version ID")
	}
	for _, b := range []byte(cfg.VersionID) {
		if b < 32 || b == 127 {
			return metadataInvalid("invalid control in object delete version ID")
		}
	}
	for key, value := range headers {
		p.metadata.client.MoreHeaders[key] = value
	}
	if cfg.Newest != nil {
		p.metadata.client.MoreHeaders["X-Newest"] = strconv.FormatBool(*cfg.Newest)
	}
	if cfg.VersionID != "" {
		p.target += "?" + url.Values{"version-id": {cfg.VersionID}}.Encode()
	}
	p.staticLargeObject = cloneDeleteObjectBool(cfg.StaticLargeObject)
	p.ignoreMissing = cfg.IgnoreMissing == nil || *cfg.IgnoreMissing
	return nil
}

func objectDeleteSLOFlag(headers http.Header) (bool, error) {
	var values []string
	for key, entries := range headers {
		if strings.EqualFold(key, "X-Static-Large-Object") {
			values = append(values, entries...)
		}
	}
	if len(values) == 0 {
		return false, nil
	}
	if len(values) != 1 || !utf8.ValidString(values[0]) {
		return false, metadataInvalid("object SLO flag must have one valid value")
	}
	switch strings.ToLower(strings.Trim(values[0], " \t")) {
	case "true", "1":
		return true, nil
	case "false", "0":
		return false, nil
	default:
		return false, metadataInvalid("object SLO flag must be true/false or 1/0")
	}
}

type objectDeleteTransport func(*http.Request) (*http.Response, error)

func (f objectDeleteTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type objectDeleteBody struct {
	io.ReadCloser
	prepared *preparedDeleteObject
	ctx      context.Context
}

func (b *objectDeleteBody) Read(data []byte) (int, error) {
	n, err := b.ReadCloser.Read(data)
	b.prepared.note(b.ctx)
	return n, err
}
func (b *objectDeleteBody) Close() error {
	err := b.ReadCloser.Close()
	b.prepared.note(b.ctx)
	return err
}

func (p *preparedDeleteObject) read(ctx context.Context, method, target string, slo bool, codes ...int) (*rest.Response, error) {
	if err := p.guard(ctx); err != nil {
		return nil, err
	}
	client, err := fixedrequest.New(p.metadata.client, method, target)
	if err != nil {
		return nil, metadataContextError(ctx, err)
	}
	provider := client.ProviderClient
	transport := provider.HTTPClient.Transport
	provider.HTTPClient.Transport = objectDeleteTransport(func(req *http.Request) (*http.Response, error) {
		if err := p.guard(req.Context()); err != nil {
			return nil, err
		}
		if slo {
			req = req.Clone(req.Context())
			if req.Header == nil {
				req.Header = make(http.Header)
			}
			for key := range req.Header {
				if strings.EqualFold(key, "Accept") {
					delete(req.Header, key)
				}
			}
			req.Header.Set("Accept", "application/json")
		}
		wire, err := transport.RoundTrip(req)
		// Defer accepted source faults to the shared reader to preserve the
		// wire even if a body callback restores the original source later.
		p.note(req.Context())
		if wire != nil && wire.Body != nil {
			wire.Body = &objectDeleteBody{ReadCloser: wire.Body, prepared: p, ctx: req.Context()}
		}
		return wire, err
	})
	if retry := provider.RetryFunc; retry != nil {
		provider.RetryFunc = func(ctx context.Context, method, target string, options *gophercloud.RequestOpts, original error, count uint) error {
			if err := p.guard(ctx); err != nil {
				return joinMetadataErrors(original, err)
			}
			err := retry(ctx, method, target, options, original, count)
			if guardErr := p.guard(ctx); err != nil || guardErr != nil {
				return joinMetadataErrors(original, err, guardErr)
			}
			return nil
		}
	}
	if backoff := provider.RetryBackoffFunc; backoff != nil {
		provider.RetryBackoffFunc = func(ctx context.Context, response *gophercloud.ErrUnexpectedResponseCode, original error, count uint) error {
			if err := p.guard(ctx); err != nil {
				return joinMetadataErrors(response, original, err)
			}
			err := backoff(ctx, response, original, count)
			if guardErr := p.guard(ctx); err != nil || guardErr != nil {
				return joinMetadataErrors(response, original, err, guardErr)
			}
			return nil
		}
	}
	if reauth := provider.ReauthFunc; reauth != nil {
		provider.ReauthFunc = func(ctx context.Context) error {
			if err := p.guard(ctx); err != nil {
				return err
			}
			return joinMetadataErrors(reauth(ctx), p.guard(ctx))
		}
	}
	redirect := provider.HTTPClient.CheckRedirect
	provider.HTTPClient.CheckRedirect = func(next *http.Request, via []*http.Request) error {
		if err := p.guard(next.Context()); err != nil {
			return err
		}
		return joinMetadataErrors(redirect(next, via), p.guard(next.Context()))
	}
	response, err := rest.DoJSON(ctx, client, method, target, nil, nil, codes...)
	if guardErr := p.guard(ctx); guardErr != nil {
		err = joinMetadataErrors(err, guardErr)
		if response != nil {
			err = response.Fail(err)
		}
	}
	return response, err
}

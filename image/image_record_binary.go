package image

import (
	"context"
	"io"
	"maps"
	"net/http"
	"strconv"

	"github.com/JSYoo5B/go-openstacksdk/internal/fixedrequest"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/gophercloud/gophercloud/v2"
)

// The owned record and legacy metadata-upload flows share the binary engine.
// Each workflow supplies its own status policy and already-captured source.
func imageRecordDataOnce(p *preparedImageRecord, endpoint string, data io.Reader, size *int64) (*rest.Response, error) {
	return imageDataOnce(p.ctx, p.client, p.check, endpoint, data, size, imageRecordCodes())
}

func imageDataOnce(ctx context.Context, source *gophercloud.ServiceClient, check func(context.Context) error, endpoint string, data io.Reader, size *int64, codes []int) (*rest.Response, error) {
	checkSource := func() error {
		if check != nil {
			return check(ctx)
		}
		return ctx.Err()
	}
	fail := func(err error) error {
		return imageUploadContextError(ctx, imageUploadErrors(err, checkSource()))
	}
	if err := checkSource(); err != nil {
		return nil, fail(err)
	}
	if err := rest.ValidateTarget(source, endpoint); err != nil {
		return nil, fail(err)
	}
	client, err := fixedrequest.NewGuarded(source, http.MethodPut, endpoint, check)
	if err != nil {
		return nil, fail(err)
	}
	// A consumed reader cannot be replayed by retry, reauth, backoff or redirect.
	// The original transport/timeout and live parent token remain in use.
	client.ProviderClient.ReauthFunc = nil
	client.ProviderClient.RetryFunc = nil
	client.ProviderClient.RetryBackoffFunc = nil
	client.ProviderClient.MaxBackoffRetries = 0
	client.ProviderClient.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	client.MoreHeaders = maps.Clone(source.MoreHeaders)
	if client.MoreHeaders == nil {
		client.MoreHeaders = make(map[string]string)
	}
	client.MoreHeaders["Content-Type"], client.MoreHeaders["Accept"] = "application/octet-stream", ""
	if size != nil {
		client.MoreHeaders["X-OpenStack-Image-Size"] = strconv.FormatInt(*size, 10)
	}
	var faults rest.RejectedResponseFaults
	parent := client.HTTPClient.Transport
	client.HTTPClient.Transport = imageBinaryTransport(func(req *http.Request) (*http.Response, error) {
		response, err := parent.RoundTrip(req)
		if response != nil && response.Body != nil {
			response.Body = &imageBinaryResponseBody{ReadCloser: response.Body, faults: &faults, check: checkSource}
		}
		return response, err
	})
	options := &gophercloud.RequestOpts{KeepResponseBody: true, OkCodes: append([]int(nil), codes...)}
	if data != nil {
		// Preserve an empty Reader as an explicit stream, but None as an empty
		// request. This view exposes no Close, Seek, Len or GetBody capability.
		var reader io.Reader = data
		if check != nil {
			reader = imageBinaryReader{Reader: data, check: checkSource}
		}
		options.RawBody = borrowedUploadReader{Reader: reader}
	}
	wire, err := client.Request(ctx, http.MethodPut, endpoint, options)
	if err != nil {
		return nil, fail(imageUploadErrors(err, faults.Err()))
	}
	response := &rest.Response{Header: wire.Header.Clone(), StatusCode: wire.StatusCode}
	response.Body, err = io.ReadAll(wire.Body)
	readSourceErr := checkSource()
	closeErr := wire.Body.Close()
	err = fail(imageUploadErrors(err, readSourceErr, closeErr, faults.Err()))
	if err != nil {
		return response, response.Fail(err)
	}
	return response, nil
}

type imageBinaryTransport func(*http.Request) (*http.Response, error)

func (f imageBinaryTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

type imageBinaryResponseBody struct {
	io.ReadCloser
	faults *rest.RejectedResponseFaults
	check  func() error
}

func (b *imageBinaryResponseBody) Read(value []byte) (int, error) {
	n, err := b.ReadCloser.Read(value)
	if err != io.EOF {
		b.faults.Add(err)
	}
	b.faults.Add(b.check())
	return n, err
}

func (b *imageBinaryResponseBody) Close() error {
	err := b.ReadCloser.Close()
	b.faults.Add(err)
	b.faults.Add(b.check())
	return err
}

type imageBinaryReader struct {
	io.Reader
	check func() error
}

func (b imageBinaryReader) Read(value []byte) (int, error) {
	if err := b.check(); err != nil {
		return 0, err
	}
	n, err := b.Reader.Read(value)
	return n, imageUploadErrors(err, b.check())
}

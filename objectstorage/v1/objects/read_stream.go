package objects

import (
	"context"
	"io"
	"net/http"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/fixedrequest"
	"gophercloudsdk/resource"
)

// objectReadBody retains immutable wire evidence and serial body state. Public
// result fields cannot change its counters, cleanup outcome or error evidence.
type objectReadBody struct {
	prepared                     *preparedObjectRead
	ctx                          context.Context
	body                         io.ReadCloser
	header                       http.Header
	status                       int
	metadata                     *MetadataInfo
	count                        int64
	emptyReads                   int
	closed, done, complete       bool
	terminalCause, terminalError error
	closeCause, closeError       error
	result                       *StreamObjectResult
}

func (p *preparedObjectRead) open(ctx context.Context) (*objectReadBody, error) {
	if err := p.check(ctx); err != nil {
		return nil, err
	}
	client, err := fixedrequest.New(p.metadata.client, http.MethodGet, p.target)
	if err != nil {
		return nil, metadataContextError(ctx, err)
	}
	transport := client.ProviderClient.HTTPClient.Transport
	client.ProviderClient.HTTPClient.Transport = objectReadTransport(func(req *http.Request) (*http.Response, error) {
		if err := p.check(req.Context()); err != nil {
			return nil, err
		}
		return transport.RoundTrip(req)
	})
	if retry := client.ProviderClient.RetryFunc; retry != nil {
		client.ProviderClient.RetryFunc = func(ctx context.Context, method, target string, opts *gophercloud.RequestOpts, original error, count uint) error {
			if err := p.check(ctx); err != nil {
				return joinMetadataErrors(original, err)
			}
			callbackErr := retry(ctx, method, target, opts, original, count)
			var ownershipErr error
			if opts == nil || !opts.KeepResponseBody || opts.JSONBody != nil || opts.RawBody != nil || opts.JSONResponse != nil {
				ownershipErr = metadataInvalid("retry changes object read request or response body ownership")
			}
			checkErr := p.check(ctx)
			if err := joinMetadataErrors(callbackErr, ownershipErr, checkErr); err != nil {
				return metadataContextError(ctx, joinMetadataErrors(original, err))
			}
			return nil
		}
	}
	wire, err := client.Request(ctx, http.MethodGet, p.target, &gophercloud.RequestOpts{KeepResponseBody: true, OkCodes: []int{200, 206, 304}})
	if err != nil {
		// Native Request already owns rejected response payloads and cleanup.
		return nil, metadataContextError(ctx, joinMetadataErrors(err, p.check(ctx)))
	}
	b := &objectReadBody{prepared: p, ctx: ctx, body: wire.Body, header: wire.Header.Clone(), status: wire.StatusCode}
	if b.status != 200 && b.status != 206 && b.status != 304 {
		unexpected := gophercloud.ErrUnexpectedResponseCode{Method: http.MethodGet, URL: p.target, Expected: []int{200, 206, 304}, Actual: b.status, ResponseHeader: b.header.Clone()}
		return nil, joinMetadataErrors(unexpected, b.closeWire())
	}
	err = p.check(ctx)
	if err == nil {
		b.metadata, err = projectMetadata(b.header)
	}
	err = joinMetadataErrors(err, p.check(ctx))
	if err != nil {
		b.metadata = nil
		b.finish(err, false)
		return b, b.terminalError
	}
	if b.status == http.StatusNotModified {
		b.finish(nil, true)
		return b, b.terminalError
	}
	return b, nil
}

type objectReadTransport func(*http.Request) (*http.Response, error)

func (f objectReadTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func (b *objectReadBody) fail(cause error, body []byte) error {
	if cause == nil {
		return nil
	}
	return &resource.ResponseError{Body: append([]byte(nil), body...), Header: b.header.Clone(), StatusCode: b.status, Cause: cause}
}

func (b *objectReadBody) closeWire() error {
	if !b.closed {
		b.closed = true
		before := b.prepared.check(b.ctx)
		closeErr := b.body.Close()
		b.closeCause = joinMetadataErrors(before, closeErr, b.prepared.check(b.ctx))
		b.closeError = b.fail(b.closeCause, nil)
	}
	return b.closeCause
}

func (b *objectReadBody) finish(cause error, complete bool) {
	b.done, b.complete = true, complete
	b.terminalCause = joinMetadataErrors(cause, b.closeWire())
	b.terminalError = b.fail(b.terminalCause, nil)
	b.publish()
}

func (b *objectReadBody) publish() {
	if b.result != nil {
		b.result.BytesRead = b.count
		b.result.Complete = b.complete
		b.result.NotModified = b.status == http.StatusNotModified
	}
}

// read returns raw terminal causes for the buffered and borrowed-writer loops.
// An exact EOF is represented by done+complete, never hidden inside a join.
func (b *objectReadBody) read(buffer []byte) (int, error) {
	if b.done {
		return 0, b.terminalCause
	}
	if err := b.prepared.check(b.ctx); err != nil {
		b.finish(err, false)
		return 0, b.terminalCause
	}
	if len(buffer) == 0 {
		return 0, nil
	}
	n, readErr := b.body.Read(buffer)
	if n < 0 || n > len(buffer) {
		if readErr == io.EOF {
			readErr = nil
		}
		b.finish(joinMetadataErrors(metadataInvalid("invalid object body Read count %d", n), readErr), false)
		return 0, b.terminalCause
	}
	b.count += int64(n)
	checkErr := b.prepared.check(b.ctx)
	if n > 0 {
		b.emptyReads = 0
	} else if readErr == nil {
		b.emptyReads++
		if b.emptyReads >= 100 {
			readErr = io.ErrNoProgress
		}
	}
	if readErr != nil || checkErr != nil {
		complete := readErr == io.EOF
		if readErr == io.EOF {
			readErr = nil
		}
		b.finish(joinMetadataErrors(readErr, checkErr), complete)
		return n, b.terminalCause
	}
	b.publish()
	return n, nil
}

func (b *objectReadBody) Read(buffer []byte) (int, error) {
	n, _ := b.read(buffer)
	b.publish()
	if b.done {
		if b.terminalError != nil {
			return n, b.terminalError
		}
		return n, io.EOF
	}
	return n, nil
}

func (b *objectReadBody) Close() error {
	if !b.done {
		b.finish(io.ErrClosedPipe, false)
	} else {
		b.closeWire()
	}
	b.publish()
	return b.closeError
}

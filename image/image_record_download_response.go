package image

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"

	"github.com/JSYoo5B/go-openstacksdk/internal/fixedrequest"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/gophercloud/gophercloud/v2"
)

func openImageRecordDownload(p *preparedImageRecord, endpoint string) (*http.Response, error) {
	if err := p.check(p.ctx); err != nil {
		return nil, err
	}
	if err := rest.ValidateTarget(p.client, endpoint); err != nil {
		return nil, err
	}
	client, err := fixedrequest.NewGuarded(p.client, http.MethodGet, endpoint, p.check)
	if err != nil {
		return nil, err
	}
	// Authentication/retry is safe before accepting this read-only response.
	// Body consumption is outside Request, so no copied byte can be replayed.
	client.ProviderClient.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	var faults rest.RejectedResponseFaults
	parent := client.HTTPClient.Transport
	client.HTTPClient.Transport = imageBinaryTransport(func(req *http.Request) (*http.Response, error) {
		response, err := parent.RoundTrip(req)
		if response != nil && response.Body != nil {
			response.Body = &imageBinaryResponseBody{ReadCloser: response.Body, faults: &faults, check: func() error { return p.check(p.ctx) }}
		}
		return response, err
	})
	wire, err := client.Request(p.ctx, http.MethodGet, endpoint, &gophercloud.RequestOpts{KeepResponseBody: true, OkCodes: imageRecordCodes()})
	if wire != nil && (wire.StatusCode < 200 || wire.StatusCode >= 400) {
		// Native retry hooks can expand OkCodes. Keep this owned profile fixed
		// and clean up a response the native request then left unconsumed.
		if err == nil {
			body, readErr := io.ReadAll(wire.Body)
			closeErr := wire.Body.Close()
			rejection := &gophercloud.ErrUnexpectedResponseCode{URL: endpoint, Method: http.MethodGet,
				Expected: imageRecordCodes(), Actual: wire.StatusCode, Body: body, ResponseHeader: wire.Header.Clone()}
			err = errors.Join(rejection, readErr, closeErr)
		}
		wire = nil
	}
	return wire, errors.Join(err, faults.Err(), p.check(p.ctx))
}

// A returned stream retains the captured context and source through deferred
// reads. Close always releases the HTTP body, even after cancellation or a
// sticky binding failure; concurrent repeated Close cannot double-close it.
type imageRecordDownloadStream struct {
	body     io.ReadCloser
	ctx      context.Context
	check    func(context.Context) error
	once     sync.Once
	closeErr error
}

func (s *imageRecordDownloadStream) Read(buffer []byte) (int, error) {
	if err := s.check(s.ctx); err != nil {
		return 0, wrapImageMutationError(s.ctx, "DownloadImageRecord", err)
	}
	n, err := s.body.Read(buffer)
	if n < 0 || n > len(buffer) {
		return 0, wrapImageMutationError(s.ctx, "DownloadImageRecord", errors.Join(uploadInvalid("invalid image response Read count %d", n), err, s.check(s.ctx)))
	}
	guardErr := s.check(s.ctx)
	if err == io.EOF && guardErr == nil {
		return n, io.EOF
	}
	return n, wrapImageMutationError(s.ctx, "DownloadImageRecord", errors.Join(err, guardErr))
}
func (s *imageRecordDownloadStream) Close() error {
	s.once.Do(func() {
		s.closeErr = wrapImageMutationError(s.ctx, "DownloadImageRecord", errors.Join(s.check(s.ctx), s.body.Close(), s.check(s.ctx)))
	})
	return s.closeErr
}

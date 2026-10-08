package cloudsnapshot

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"

	"github.com/JSYoo5B/go-openstacksdk/internal/fixedrequest"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// Native Gophercloud discards read/Close errors on rejected responses. A404
// can complete this workflow, so observe those errors without admitting404
// in OkCodes: native retry/reauthentication behavior remains available.
func (p *reader) deletePoll(ctx context.Context, target string) (*rest.Response, error) {
	client, err := fixedrequest.NewGuarded(&p.source.Client, http.MethodGet, target, p.source.Guard)
	if err != nil {
		return nil, err
	}
	var faults rejectedPollFaults
	parent := client.ProviderClient.HTTPClient.Transport
	client.ProviderClient.HTTPClient.Transport = rejectedPollTransport(func(req *http.Request) (*http.Response, error) {
		response, err := parent.RoundTrip(req)
		if response != nil && response.StatusCode == http.StatusNotFound && response.Body != nil {
			response.Body = &rejectedPollBody{ReadCloser: response.Body, faults: &faults}
		}
		return response, err
	})
	guard := func(ctx context.Context) error {
		return errors.Join(p.source.Guard(ctx), faults.error())
	}
	return rest.DoJSONGuarded(ctx, client, guard, http.MethodGet, target, nil, nil, sourceCodes()...)
}

type rejectedPollTransport func(*http.Request) (*http.Response, error)

func (f rejectedPollTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

type rejectedPollFaults struct {
	mu    sync.Mutex
	cause error
}

func (f *rejectedPollFaults) add(err error) {
	if err == nil {
		return
	}
	f.mu.Lock()
	f.cause = errors.Join(f.cause, err)
	f.mu.Unlock()
}

func (f *rejectedPollFaults) error() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.cause
}

type rejectedPollBody struct {
	io.ReadCloser
	faults *rejectedPollFaults
}

func (b *rejectedPollBody) Read(data []byte) (int, error) {
	n, err := b.ReadCloser.Read(data)
	if err != io.EOF {
		b.faults.add(err)
	}
	return n, err
}

func (b *rejectedPollBody) Close() error {
	err := b.ReadCloser.Close()
	b.faults.add(err)
	return err
}
